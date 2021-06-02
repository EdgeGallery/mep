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

// Package mp1 implements rest api route controller
package mp1

import (
	"context"
	"encoding/json"
	"mepserver/common/models"
	"net/http"
	"net/url"

	"github.com/apache/servicecomb-service-center/pkg/util"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/server/core"
	"github.com/apache/servicecomb-service-center/server/core/proto"

	"mepserver/common/arch/workspace"
	meputil "mepserver/common/util"
)

// DiscoverDecode step to handle the service discovery request
type DiscoverDecode struct {
	workspace.TaskBase
	R             *http.Request   `json:"r,in"`
	Ctx           context.Context `json:"ctx,out"`
	QueryParam    url.Values      `json:"queryParam,out"`
	CoreRequest   interface{}     `json:"coreRequest,out"`
	AppInstanceId string          `json:"appInstanceId,out"`
}

// OnRequest discover decode request
func (t *DiscoverDecode) OnRequest(data string) workspace.TaskCode {
	log.Infof("Received message from ClientIP [%s] AppInstanceId [%s] Operation [%s] Resource [%s].",
		meputil.GetClientIp(t.R), meputil.GetAppInstanceId(t.R), meputil.GetMethodFromReq(t.R), meputil.GetHttpResourceInfo(t.R))
	err := t.GetFindParam(t.R)
	if err != nil {
		log.Error("Validate service discovery message error.", nil)
	}
	return workspace.TaskFinish
}

// GetFindParam get find param by request
func (t *DiscoverDecode) GetFindParam(r *http.Request) error {

	query, ids := meputil.GetHTTPTags(r)
	if err := meputil.ValidateAppInstanceIdWithHeader(query.Get(":appInstanceId"), r); err != nil {
		t.SetFirstErrorCode(meputil.AuthorizationValidateErr, err.Error())
		return err
	}

	req := &proto.FindInstancesRequest{
		ConsumerServiceId: r.Header.Get("X-ConsumerId"),
		AppId:             query.Get("instance_id"),
		ServiceName:       query.Get("ser_name"),
		VersionRule:       query.Get("version"),
		Environment:       query.Get("env"),
		Tags:              ids,
	}

	if req.AppId == "" {
		req.AppId = "default"
	}
	if req.VersionRule == "" {
		req.VersionRule = "latest"
	}
	t.Ctx = util.SetTargetDomainProject(r.Context(), r.Header.Get("X-Domain-Name"), query.Get(":project"))
	t.CoreRequest = req
	t.QueryParam = query
	t.AppInstanceId = r.Header.Get("X-AppInstanceId")
	return nil
}

type DiscoverService struct {
	workspace.TaskBase
	Ctx           context.Context `json:"ctx,in"`
	QueryParam    url.Values      `json:"queryParam,in"`
	CoreRequest   interface{}     `json:"coreRequest,in"`
	AppInstanceId string          `json:"appInstanceId,in"`
	InstanceId    string          `json:"instanceId,out"`
	CoreRsp       interface{}     `json:"coreRsp,out"`
}

func (t *DiscoverService) checkInstanceId(req *proto.FindInstancesRequest) bool {
	instanceId := req.AppId
	if instanceId != "default" {
		value, ok := t.CoreRsp.(*proto.FindInstancesResponse)
		if !ok {
			log.Error("Interface cast is failed in service discovery.", nil)
			return false
		}
		instances := value.Instances
		for _, val := range instances {
			if val.ServiceId+val.InstanceId == instanceId {
				return true
			}
		}
		return false
	}
	return true
}

func (t *DiscoverService) filterAppInstanceId() {
	appInstanceId := t.QueryParam.Get(":appInstanceId")
	if appInstanceId == "" {
		return
	}

	value, ok := t.CoreRsp.(*proto.FindInstancesResponse)
	if !ok {
		log.Error("Interface cast failed.", nil)
		return
	}

	instances := value.Instances
	var result []*proto.MicroServiceInstance
	for _, instance := range instances {
		id := instance.Properties["appInstanceId"]
		if id == appInstanceId {
			result = append(result, instance)
		}
	}
	value.Instances = result
}

// OnRequest service discover request
func (t *DiscoverService) OnRequest(data string) workspace.TaskCode {
	req, ok := t.CoreRequest.(*proto.FindInstancesRequest)
	if !ok {
		log.Error("Cast input to find-instance-request failed.", nil)
		t.SetFirstErrorCode(meputil.SerErrServiceNotFound, "cast to instance request failed")
		return workspace.TaskFinish
	}
	log.Debugf("Query request arrived to fetch all the service information with appId %s.", req.AppId)
	t.InstanceId = t.AppInstanceId
	if req.ServiceName == "" {
		var errFindByKey error
		t.CoreRsp, errFindByKey = meputil.FindInstanceByKey(t.QueryParam)
		if errFindByKey != nil {
			log.Error("Failed to find instance.", nil)
			t.SetFirstErrorCode(meputil.SerErrServiceNotFound, "failed to find the instance")
			return workspace.TaskFinish
		}
		if t.CoreRsp == nil {
			log.Error("Failed to find any instance.", nil)
			t.SetFirstErrorCode(meputil.SerErrServiceNotFound, "could not find any instance")
			return workspace.TaskFinish
		}
		if !t.checkInstanceId(req) {
			log.Error("Instance id not found.", nil)
			t.SetFirstErrorCode(meputil.SerErrServiceNotFound, "instance id not found")
		}
		t.filterAppInstanceId()
		return workspace.TaskFinish
	}

	findInstance, err := core.InstanceAPI.Find(t.Ctx, req)
	if err != nil {
		log.Error("Failed to find instance request.", nil)
		t.SetFirstErrorCode(meputil.SerErrServiceNotFound, "failed to find instance request")
		return workspace.TaskFinish
	}
	t.CoreRsp = findInstance
	t.filterAppInstanceId()
	return workspace.TaskFinish
}

type ToStrDiscover struct {
	workspace.TaskBase
	CoreRsp    interface{}     `json:"coreRsp,in"`
	InstanceId string          `json:"instanceId,in"`
	HttpRsp    interface{}     `json:"httpRsp,out"`
	HttpErrInf *proto.Response `json:"httpErrInf,out"`
}

// OnRequest to string discover request
func (t *ToStrDiscover) OnRequest(data string) workspace.TaskCode {
	value, ok := t.CoreRsp.(*proto.FindInstancesResponse)
	if !ok {
		log.Error("Cast input to find-instance-response failed.", nil)
		t.SetFirstErrorCode(meputil.SerErrServiceNotFound, "cast to instance response failed")
		return workspace.TaskFinish
	}
	t.HttpErrInf, t.HttpRsp = Mp1CvtSrvDiscover(value, t.InstanceId)
	return workspace.TaskFinish
}

type RspHook struct {
	R *http.Request `json:"r,in"`
	workspace.TaskBase
	Ctx     context.Context `json:"ctx,in"`
	HttpRsp interface{}     `json:"httpRsp,in"`
	HookRsp interface{}     `json:"hookRsp,out"`
}

// OnRequest resp hook request
func (t *RspHook) OnRequest(data string) workspace.TaskCode {
	t.HookRsp = instanceHook(t.R, t.HttpRsp)
	_, err := json.Marshal(t.HttpRsp)
	if err != nil {
		log.Error("Http response marshal fail.", nil)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "http response marshal fail")
	}
	return workspace.TaskFinish
}

func instanceHook(r *http.Request, rspData interface{}) interface{} {
	rspBody, ok := rspData.([]*models.ServiceInfo)
	if !ok {
		return rspData
	}

	if len(rspBody) == 0 {
		return rspBody
	}
	consumerName := r.Header.Get("X-ConsumerName")
	if consumerName == "APIGW" {
		return rspBody
	}

	for _, v := range rspBody {
		if apihook.APIHook != nil {
			info := apihook.APIHook()
			if len(info.Addresses) == 0 && len(info.Uris) == 0 {
				return rspBody
			}
			v.TransportInfo.Endpoint = info
		}
	}
	return rspBody
}

// Mp1CvtSrvDiscover mp1 cvt all service discover
func Mp1CvtSrvDiscoverAll(findInsResp *proto.FindInstancesResponse) (*proto.Response, []*models.ServiceInfo) {
	resp := findInsResp.Response
	if resp != nil && resp.GetCode() != proto.Response_SUCCESS {
		return resp, nil
	}
	serviceInfos := make([]*models.ServiceInfo, 0, len(findInsResp.Instances))
	for _, ins := range findInsResp.Instances {
		serviceInfo := &models.ServiceInfo{}
		serviceInfo.FromServiceInstance(ins)
		serviceInfos = append(serviceInfos, serviceInfo)
	}
	return resp, serviceInfos

}

// Mp1CvtSrvDiscover mp1 cvt service discover by app instance id
func Mp1CvtSrvDiscover(findInsResp *proto.FindInstancesResponse, appInsId string) (*proto.Response, []*models.ServiceInfo) {
	resp := findInsResp.Response
	if resp != nil && resp.GetCode() != proto.Response_SUCCESS {
		return resp, nil
	}
	serviceInfos := make([]*models.ServiceInfo, 0, len(findInsResp.Instances))
	requiredServices, err := meputil.GetRequiredSerFromMepauth(appInsId)
	if err != nil {
		return resp, serviceInfos
	}
	for _, ins := range findInsResp.Instances {
		serviceInfo := &models.ServiceInfo{}
		serviceInfo.FromServiceInstance(ins)
		serviceName := serviceInfo.SerName
		if meputil.InArray(serviceName, requiredServices) {
			serviceInfos = append(serviceInfos, serviceInfo)
		}
	}
	return resp, serviceInfos

}
