/*
 * Copyright 2020-2021 Huawei Technologies Co., Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package path implements rest api route controller
package mp1

import (
	"fmt"
	"mepserver/common/config"
	"mepserver/common/extif/dataplane"
	dpCommon "mepserver/common/extif/dataplane/common"
	"mepserver/common/extif/dns"
	"mepserver/common/models"
	"net/http"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/rest"
	v4 "github.com/apache/servicecomb-service-center/server/rest/controller/v4"

	"mepserver/common"
	"mepserver/common/arch/workspace"
	meputil "mepserver/common/util"
	"mepserver/mp1/plans"
)

type APIHookFunc func() models.EndPointInfo

type APIGwHook struct {
	APIHook APIHookFunc
}

var apihook APIGwHook

// set api gw hook
func SetAPIHook(hook APIGwHook) {
	apihook = hook
}

func init() {
	initRouter()
}

func initRouter() {
	mp1 := &Mp1Service{}
	if err := mp1.Init(); err != nil {
		log.Errorf(err, "Mp1 interface initialization failed.")
		//os.Exit(1) # Init function cannot be mocked by test. Hence removed this.
	}
	rest.RegisterServant(mp1)
}

type Mp1Service struct {
	v4.MicroServiceService
	config    *config.MepServerConfig
	dnsAgent  dns.DNSAgent
	dataPlane dataplane.DataPlane
}

func (m *Mp1Service) Init() error {
	mepConfig, err := config.LoadMepServerConfig()
	if err != nil {
		return fmt.Errorf("error: reading configuration failed")
	}
	m.config = mepConfig

	// Checking if local or both is configured
	var dnsAgent dns.DNSAgent
	if m.config.DNSAgent.Type != meputil.DnsAgentTypeDataPlane {
		dnsAgent = dns.NewRestDNSAgent(mepConfig)
	}
	m.dnsAgent = dnsAgent
	// select data plane as per configuration
	dataPlane := dpCommon.CreateDataPlane(mepConfig)
	if dataPlane == nil {
		return fmt.Errorf("error: unsupported data-plane")
	}

	if err := dataPlane.InitDataPlane(mepConfig); err != nil {
		return err
	}
	m.dataPlane = dataPlane
	log.Infof("Data plane initialized to %s.", m.config.DataPlane.Type)

	return nil
}

// url patterns
func (m *Mp1Service) URLPatterns() []rest.Route {
	return []rest.Route{
		// appSubscriptions
		{Method: rest.HTTP_METHOD_POST, Path: meputil.AppSubscribePath, Func: m.doAppSubscribe},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.AppSubscribePath, Func: m.getAppSubscribes},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.AppSubscribePath + meputil.SubscriptionIdPath,
			Func: m.getOneAppSubscribe},
		{Method: rest.HTTP_METHOD_DELETE, Path: meputil.AppSubscribePath + meputil.SubscriptionIdPath,
			Func: m.delOneAppSubscribe},
		// appServices
		{Method: rest.HTTP_METHOD_POST, Path: meputil.AppServicesPath, Func: m.serviceRegister},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.AppServicesPath, Func: m.serviceDiscover},
		{Method: rest.HTTP_METHOD_PUT, Path: meputil.AppServicesPath + meputil.ServiceIdPath, Func: m.serviceUpdate},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.AppServicesPath + meputil.ServiceIdPath, Func: m.getOneService},
		{Method: rest.HTTP_METHOD_DELETE, Path: meputil.AppServicesPath + meputil.ServiceIdPath, Func: m.serviceDelete},
		// MEC Application Support API - appSubscriptions
		{Method: rest.HTTP_METHOD_POST, Path: meputil.EndAppSubscribePath, Func: m.appEndSubscribe},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.EndAppSubscribePath, Func: m.getAppEndSubscribes},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.EndAppSubscribePath + meputil.SubscriptionIdPath,
			Func: m.getEndAppOneSubscribe},
		{Method: rest.HTTP_METHOD_DELETE, Path: meputil.EndAppSubscribePath + meputil.SubscriptionIdPath,
			Func: m.delEndAppOneSubscribe},
		// DNS
		{Method: rest.HTTP_METHOD_GET, Path: meputil.DNSRulesPath, Func: m.getDnsRules},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.DNSRulesPath + meputil.DNSRuleIdPath, Func: m.getDnsRule},
		{Method: rest.HTTP_METHOD_PUT, Path: meputil.DNSRulesPath + meputil.DNSRuleIdPath, Func: m.dnsRuleUpdate},
		// HeartBeat
		{Method: rest.HTTP_METHOD_GET, Path: meputil.AppServicesPath + meputil.ServiceIdPath + meputil.Liveness,
			Func: m.getHeartbeat},
		{Method: rest.HTTP_METHOD_PUT, Path: meputil.AppServicesPath + meputil.ServiceIdPath + meputil.Liveness,
			Func: m.heartbeatService},
		//Liveness and readiness
		{Method: rest.HTTP_METHOD_GET, Path: "/health", Func: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("ok"))
		}},
		// services
		{Method: rest.HTTP_METHOD_GET, Path: meputil.ServicesPath, Func: m.serviceDiscover},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.ServicesPath + "/:serviceId", Func: m.getOneService},
		//traffic Rules
		{Method: rest.HTTP_METHOD_GET, Path: meputil.TrafficRulesPath, Func: m.getTrafficRules},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.TrafficRulesPath + meputil.TrafficRuleIdPath, Func: m.getTrafficRule},
		{Method: rest.HTTP_METHOD_PUT, Path: meputil.TrafficRulesPath + meputil.TrafficRuleIdPath, Func: m.trafficRuleUpdate},
	}
}

func (m *Mp1Service) appEndSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try((&plans.DecodeRestReq{}).WithBody(&models.AppTerminationNotificationSubscription{}),
		(&plans.AppSubscribeLimit{}).WithType(meputil.AppTerminationNotificationSubscription),
		(&plans.SubscribeIst{}).WithType(meputil.AppTerminationNotificationSubscription))
	workPlan.Finally(&common.SendHttpRsp{StatusCode: http.StatusCreated})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getAppEndSubscribes(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.GetSubscribes{}).WithType(meputil.AppTerminationNotificationSubscription))
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getEndAppOneSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.GetOneSubscribe{}).WithType(meputil.AppTerminationNotificationSubscription))
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) delEndAppOneSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.DelOneSubscribe{}).WithType(meputil.AppTerminationNotificationSubscription))
	workPlan.Finally(&common.SendHttpRsp{StatusCode: http.StatusNoContent})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) doAppSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeRestReq{}).WithBody(&models.SerAvailabilityNotificationSubscription{}),
		(&plans.AppSubscribeLimit{}).WithType(meputil.SerAvailabilityNotificationSubscription),
		(&plans.SubscribeIst{}).WithType(meputil.SerAvailabilityNotificationSubscription))
	workPlan.Finally(&common.SendHttpRsp{StatusCode: http.StatusCreated})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getAppSubscribes(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.GetSubscribes{}).WithType(meputil.SerAvailabilityNotificationSubscription))
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getOneAppSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.GetOneSubscribe{}).WithType(meputil.SerAvailabilityNotificationSubscription))
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) delOneAppSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.DelOneSubscribe{}).WithType(meputil.SerAvailabilityNotificationSubscription))
	workPlan.Finally(&common.SendHttpRsp{StatusCode: http.StatusNoContent})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) serviceRegister(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeRestReq{}).WithBody(&models.ServiceInfo{}),
		&plans.RegisterLimit{},
		&plans.RegisterServiceId{},
		&plans.RegisterServiceInst{})
	workPlan.Finally(&common.SendHttpRsp{StatusCode: http.StatusCreated})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) serviceDiscover(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&DiscoverDecode{},
		&DiscoverService{},
		&ToStrDiscover{},
		&RspHook{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) serviceUpdate(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeRestReq{}).WithBody(&models.ServiceInfo{}),
		&plans.UpdateInstance{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getOneService(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.GetOneDecode{},
		&plans.GetOneInstance{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)

}

func (m *Mp1Service) serviceDelete(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		&plans.DeleteService{})
	workPlan.Finally(&common.SendHttpRsp{StatusCode: http.StatusNoContent})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getDnsRules(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeDnsRestReq{},
		&plans.DNSRulesGet{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getDnsRule(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeDnsRestReq{},
		&plans.DNSRuleGet{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) dnsRuleUpdate(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeDnsRestReq{}).WithBody(&dataplane.DNSRule{}),
		(&plans.DNSRuleUpdate{}).WithDNSAgent(m.dnsAgent).WithDataPlane(m.dataPlane))
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getHeartbeat(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.GetOneDecodeHeartbeat{},
		&plans.GetOneInstanceHeartbeat{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) heartbeatService(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeHeartbeatRestReq{}).WithBodies(&models.ServiceLivenessUpdate{}),
		&plans.UpdateHeartbeat{})
	workPlan.Finally(&common.SendHttpRsp{StatusCode: http.StatusNoContent})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getTrafficRules(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeTrafficRestReq{},
		&plans.TrafficRulesGet{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getTrafficRule(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeTrafficRestReq{},
		&plans.TrafficRuleGet{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) trafficRuleUpdate(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeTrafficRestReq{}).WithBody(&dataplane.TrafficRule{}),
		(&plans.TrafficRuleUpdate{}).WithDataPlane(m.dataPlane))
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}
