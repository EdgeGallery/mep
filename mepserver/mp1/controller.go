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

// Package path implements rest api route controller
package mp1

import (
	"net/http"

	"github.com/apache/servicecomb-service-center/pkg/rest"
	v4 "github.com/apache/servicecomb-service-center/server/rest/controller/v4"

	"mepserver/mp1/arch/workspace"
	"mepserver/mp1/models"
	"mepserver/mp1/plans"
	meputil "mepserver/mp1/util"
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
	rest.
		RegisterServant(&Mp1Service{})
}

type Mp1Service struct {
	v4.MicroServiceService
}

// url patterns
func (m *Mp1Service) URLPatterns() []rest.Route {
	return []rest.Route{
		// appSubscriptions
		{Method: rest.HTTP_METHOD_POST, Path: meputil.AppSubscribePath, Func: doAppSubscribe},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.AppSubscribePath, Func: getAppSubscribes},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.AppSubscribePath + "/:subscriptionId", Func: getOneAppSubscribe},
		{Method: rest.HTTP_METHOD_DELETE, Path: meputil.AppSubscribePath + "/:subscriptionId",
			Func: delOneAppSubscribe},
		// appServices
		{Method: rest.HTTP_METHOD_POST, Path: meputil.AppServicesPath, Func: serviceRegister},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.AppServicesPath, Func: serviceDiscover},
		{Method: rest.HTTP_METHOD_PUT, Path: meputil.AppServicesPath + "/:serviceId", Func: serviceUpdate},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.AppServicesPath + "/:serviceId", Func: getOneService},
		{Method: rest.HTTP_METHOD_DELETE, Path: meputil.AppServicesPath + "/:serviceId", Func: serviceDelete},
		// MEC Application Support API - appSubscriptions
		{Method: rest.HTTP_METHOD_POST, Path: meputil.EndAppSubscribePath, Func: appEndSubscribe},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.EndAppSubscribePath, Func: getAppEndSubscribes},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.EndAppSubscribePath + "/:subscriptionId",
			Func: getEndAppOneSubscribe},
		{Method: rest.HTTP_METHOD_DELETE, Path: meputil.EndAppSubscribePath + "/:subscriptionId",
			Func: delEndAppOneSubscribe},
	}
}

func appEndSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeRestReq{}).WithBody(&models.AppTerminationNotificationSubscription{}),
		(&plans.AppSubscribeLimit{}).WithType(meputil.AppTerminationNotificationSubscription),
		(&plans.SubscribeIst{}).WithType(meputil.AppTerminationNotificationSubscription))
	workPlan.Finally(&plans.SendHttpRsp{StatusCode: http.StatusCreated})

	workspace.WkRun(workPlan)
}

func getAppEndSubscribes(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.GetSubscribes{}).WithType(meputil.AppTerminationNotificationSubscription))
	workPlan.Finally(&plans.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func getEndAppOneSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.GetOneSubscribe{}).WithType(meputil.AppTerminationNotificationSubscription))
	workPlan.Finally(&plans.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func delEndAppOneSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.DelOneSubscribe{}).WithType(meputil.AppTerminationNotificationSubscription))
	workPlan.Finally(&plans.SendHttpRsp{StatusCode: http.StatusNoContent})

	workspace.WkRun(workPlan)
}

func doAppSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeRestReq{}).WithBody(&models.SerAvailabilityNotificationSubscription{}),
		(&plans.AppSubscribeLimit{}).WithType(meputil.SerAvailabilityNotificationSubscription),
		(&plans.SubscribeIst{}).WithType(meputil.SerAvailabilityNotificationSubscription))
	workPlan.Finally(&plans.SendHttpRsp{StatusCode: http.StatusCreated})

	workspace.WkRun(workPlan)
}

func getAppSubscribes(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.GetSubscribes{}).WithType(meputil.SerAvailabilityNotificationSubscription))
	workPlan.Finally(&plans.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func getOneAppSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.GetOneSubscribe{}).WithType(meputil.SerAvailabilityNotificationSubscription))
	workPlan.Finally(&plans.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func delOneAppSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.DelOneSubscribe{}).WithType(meputil.SerAvailabilityNotificationSubscription))
	workPlan.Finally(&plans.SendHttpRsp{StatusCode: http.StatusNoContent})

	workspace.WkRun(workPlan)
}

func serviceRegister(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeRestReq{}).WithBody(&models.ServiceInfo{}),
		&plans.RegisterLimit{},
		&plans.RegisterServiceId{},
		&plans.RegisterServiceInst{})
	workPlan.Finally(&plans.SendHttpRsp{StatusCode: http.StatusCreated})

	workspace.WkRun(workPlan)
}

func serviceDiscover(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&DiscoverDecode{},
		&DiscoverService{},
		&ToStrDiscover{},
		&RspHook{})
	workPlan.Finally(&plans.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func serviceUpdate(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeRestReq{}).WithBody(&models.ServiceInfo{}),
		&plans.UpdateInstance{})
	workPlan.Finally(&plans.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func getOneService(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.GetOneDecode{},
		&plans.GetOneInstance{})
	workPlan.Finally(&plans.SendHttpRsp{})

	workspace.WkRun(workPlan)

}

func serviceDelete(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		&plans.DeleteService{})
	workPlan.Finally(&plans.SendHttpRsp{StatusCode: http.StatusNoContent})

	workspace.WkRun(workPlan)
}