/*
 * Copyright 2020 Huawei Technologies Co., Ltd.
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

package mm5

import (
	"fmt"
	"mepserver/common/config"
	dpCommon "mepserver/common/extif/dataplane/common"
	"mepserver/common/extif/dns"
	"mepserver/common/models"
	"mepserver/mm5/task"
	"net/http"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/rest"
	v4 "github.com/apache/servicecomb-service-center/server/rest/controller/v4"

	"mepserver/common"
	"mepserver/common/arch/workspace"
	meputil "mepserver/common/util"
	"mepserver/mm5/plans"
)

func init() {
	initMm5Router()
}

func initMm5Router() {
	mm5 := &Mm5Service{}

	if err := mm5.Init(); err != nil {
		log.Errorf(err, "Mm5 interface initialization failed.")
		//os.Exit(1) # Init function cannot be mocked by test. Hence removed this.
	}
	rest.RegisterServant(mm5)
}

type Mm5Service struct {
	v4.MicroServiceService
	config    *config.MepServerConfig
	mp2Worker task.Worker
}

func (m *Mm5Service) Init() error {
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

	// select data plane as per configuration
	dataPlane := dpCommon.CreateDataPlane(mepConfig)
	if dataPlane == nil {
		return fmt.Errorf("error: unsupported data-plane")
	}

	if err := dataPlane.InitDataPlane(mepConfig); err != nil {
		return err
	}

	log.Infof("Data plane initialized to %s", m.config.DataPlane.Type)

	m.mp2Worker.InitializeWorker(dataPlane, dnsAgent, m.config.DNSAgent.Type)

	return nil
}

func (m *Mm5Service) URLPatterns() []rest.Route {

	return []rest.Route{
		// AppD Configurations
		{Method: rest.HTTP_METHOD_POST, Path: meputil.AppDConfigPath, Func: m.appDCreate},
		{Method: rest.HTTP_METHOD_PUT, Path: meputil.AppDConfigPath, Func: m.appDUpdate},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.AppDConfigPath, Func: m.getAppD},
		{Method: rest.HTTP_METHOD_DELETE, Path: meputil.AppDConfigPath, Func: m.appDDelete},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.AppDQueryResPath, Func: m.getResourceTasks},

		// Platform Capability Query
		{Method: rest.HTTP_METHOD_GET, Path: meputil.CapabilityPath, Func: m.getPlatformCapabilities},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.CapabilityPath + meputil.CapabilityIdPath, Func: m.getPlatformCapability},

		// App Termination
		{Method: rest.HTTP_METHOD_DELETE, Path: meputil.AppInsTerminationPath, Func: m.terminateAppInstance},

		// Monitor Interface
		{Method: rest.HTTP_METHOD_POST, Path: meputil.KongHttpLogPath, Func: m.insertHttpLog},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.KongHttpLogPath, Func: m.queryHttpLog},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.SubscribeStatisticPath, Func: m.querySubscribeStatistic},
	}
}

func (m *Mm5Service) getPlatformCapabilities(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeCapabilityQueryReq{},
		&plans.CapabilitiesGet{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mm5Service) getPlatformCapability(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeCapabilityQueryReq{},
		&plans.CapabilityGet{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}
func (m *Mm5Service) appDCreate(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeAppDRestReq{}).WithBody(&models.AppDConfig{}),
		(&plans.CreateAppDConfig{}).WithWorker(&m.mp2Worker))
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mm5Service) appDUpdate(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeAppDRestReq{}).WithBody(&models.AppDConfig{}),
		(&plans.UpdateAppDConfig{}).WithWorker(&m.mp2Worker))
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mm5Service) appDDelete(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeAppDRestReq{},
		(&plans.DeleteAppDConfig{}).WithWorker(&m.mp2Worker))
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)

}

func (m *Mm5Service) getAppD(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeAppDRestReq{},
		&plans.AppDConfigGet{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)

}

func (m *Mm5Service) getResourceTasks(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeTaskRestReq{},
		&plans.TaskStatusGet{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)

}

func (m *Mm5Service) terminateAppInstance(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeAppTerminationReq{},
		(&plans.DeleteAppDConfigWithSync{}).WithWorker(&m.mp2Worker),
		&plans.DeleteService{},
		&plans.DeleteFromMepauth{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mm5Service) insertHttpLog(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.CreateKongHttpLog{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mm5Service) queryHttpLog(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.GetKongHttpLog{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mm5Service) querySubscribeStatistic(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.SubscriptionInfoReq{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}
