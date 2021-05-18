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

// Package plans implements mep server api plans
package plans

import (
	"context"
	"encoding/json"
	"mepserver/common/models"
	"net/http"

	"github.com/apache/servicecomb-service-center/pkg/log"

	"github.com/apache/servicecomb-service-center/pkg/util"
	"github.com/apache/servicecomb-service-center/server/core"
	"github.com/apache/servicecomb-service-center/server/core/proto"

	"mepserver/common/arch/workspace"
	meputil "mepserver/common/util"
)

type GetOneDecode struct {
	workspace.TaskBase
	R             *http.Request   `json:"r,in"`
	Ctx           context.Context `json:"ctx,out"`
	CoreRequest   interface{}     `json:"coreRequest,out"`
	AppInstanceId string          `json:"appInstanceId,out"`
}

// OnRequest
func (t *GetOneDecode) OnRequest(data string) workspace.TaskCode {
	var err error
	log.Infof("Received message from ClientIP [%s] AppInstanceId [%s] Operation [%s] Resource [%s].",
		meputil.GetClientIp(t.R), meputil.GetAppInstanceId(t.R), meputil.GetMethod(t.R), meputil.GetResourceInfo(t.R))
	t.Ctx, t.CoreRequest, err = t.getFindParam(t.R)
	if err != nil {
		log.Error("Parameters validation failed for service query request.", err)
		return workspace.TaskFinish
	}
	req, ok := t.CoreRequest.(*proto.GetOneInstanceRequest)
	if !ok {
		log.Error("Get one instance request error.", nil)
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "get one instance request error")
		return workspace.TaskFinish
	}
	log.Debugf("Query request arrived to fetch the service information with subscriptionId %s", req.ProviderServiceId)
	return workspace.TaskFinish

}

func (t *GetOneDecode) getFindParam(r *http.Request) (context.Context, *proto.GetOneInstanceRequest, error) {
	query, ids := meputil.GetHTTPTags(r)

	t.AppInstanceId = query.Get(":appInstanceId")
	if err := meputil.ValidateAppInstanceIdWithHeader(t.AppInstanceId, r); err != nil {
		log.Error("Validate X-AppInstanceId failed.", err)
		t.SetFirstErrorCode(meputil.AuthorizationValidateErr, err.Error())
		return nil, nil, err
	}

	mp1SrvId := query.Get(":serviceId")
	log.Infof("New service request(service id: %s).", mp1SrvId)
	var err = meputil.ValidateServiceID(mp1SrvId)
	if err != nil {
		log.Error("Invalid service id.", err)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "invalid service id")
		return nil, nil, err
	}

	serviceId := mp1SrvId[:len(mp1SrvId)/2]
	instanceId := mp1SrvId[len(mp1SrvId)/2:]
	req := &proto.GetOneInstanceRequest{
		ConsumerServiceId:  r.Header.Get("X-ConsumerId"),
		ProviderServiceId:  serviceId,
		ProviderInstanceId: instanceId,
		Tags:               ids,
	}

	ctx := util.SetTargetDomainProject(r.Context(), r.Header.Get("X-Domain-Name"), query.Get(":project"))
	return ctx, req, nil
}

type GetOneInstance struct {
	workspace.TaskBase
	HttpErrInf    *proto.Response `json:"httpErrInf,out"`
	Ctx           context.Context `json:"ctx,in"`
	CoreRequest   interface{}     `json:"coreRequest,in"`
	HttpRsp       interface{}     `json:"httpRsp,out"`
	AppInstanceId string          `json:"appInstanceId,in"`
}

// OnRequest
func (t *GetOneInstance) OnRequest(data string) workspace.TaskCode {
	req, ok := t.CoreRequest.(*proto.GetOneInstanceRequest)
	if !ok {
		log.Error("Get instance request error.", nil)
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "get instance request error")
		return workspace.TaskFinish
	}

	resp, errGetOneInstance := core.InstanceAPI.GetOneInstance(t.Ctx, req)
	if errGetOneInstance != nil {
		log.Error("Get one instance error.", nil)
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "get one instance error")
		return workspace.TaskFinish
	}
	t.HttpErrInf = resp.Response
	resp.Response = nil
	mp1Rsp := &models.ServiceInfo{}

	t.filterAppInstanceId(resp.Instance)
	if resp.Instance != nil {
		mp1Rsp.FromServiceInstance(resp.Instance)
	} else {
		log.Error("Service instance id not found.", nil)
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "service instance id not found")
		return workspace.TaskFinish
	}
	t.HttpRsp = mp1Rsp
	_, err := json.Marshal(mp1Rsp)
	if err != nil {
		log.Error("Service info marshalling failed.", nil)
		t.SetFirstErrorCode(meputil.ParseInfoErr, "marshal service info failed")
		return workspace.TaskFinish
	}
	log.Debugf("Response for service information with subscriptionId %s", req.ProviderServiceId)
	return workspace.TaskFinish
}

func (t *GetOneInstance) filterAppInstanceId(inst *proto.MicroServiceInstance) {
	if inst == nil || inst.Properties == nil {
		return
	}
	if t.AppInstanceId != inst.Properties["appInstanceId"] {
		inst = nil
	}
}
