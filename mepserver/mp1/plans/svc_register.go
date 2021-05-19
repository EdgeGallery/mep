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
	"errors"
	"fmt"
	"io/ioutil"
	"mepserver/common/models"
	"net/http"
	"net/url"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"github.com/apache/servicecomb-service-center/server/core"
	"github.com/apache/servicecomb-service-center/server/core/proto"

	"mepserver/common/arch/workspace"
	meputil "mepserver/common/util"
)

type DecodeRestReq struct {
	workspace.TaskBase
	R             *http.Request   `json:"r,in"`
	Ctx           context.Context `json:"ctx,out"`
	AppInstanceId string          `json:"appInstanceId,out"`
	SubscribeId   string          `json:"subscribeId,out"`
	ServiceId     string          `json:"serviceId,out"`
	RestBody      interface{}     `json:"restBody,out"`
}

// OnRequest decodes the service request messages
func (t *DecodeRestReq) OnRequest(data string) workspace.TaskCode {
	log.Infof("Received message from ClientIP [%s] AppInstanceId [%s] Operation [%s] Resource [%s].",
		meputil.GetClientIp(t.R), meputil.GetAppInstanceId(t.R), meputil.GetMethod(t.R), meputil.GetHttpResourceInfo(t.R))

	err := t.GetParam(t.R)
	if err != nil {
		log.Error("Parameters validation failed on service register request.", err)
		return workspace.TaskFinish
	}

	err = t.ParseBody(t.R)
	if err != nil {
		log.Error("Service register request body parse failed.", err)
	}
	return workspace.TaskFinish
}

// ParseBody Parse request body
func (t *DecodeRestReq) ParseBody(r *http.Request) error {
	if t.RestBody == nil {
		return nil
	}
	msg, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("Service register request read failed.", nil)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "read request body error")
		return errors.New("read failed")
	}
	if len(msg) > meputil.RequestBodyLength {
		err = errors.New("request body too large")
		log.Errorf(err, "Service register request body too large %d.", len(msg))
		t.SetFirstErrorCode(meputil.RequestParamErr, "request body too large")
		return err
	}
	newMsg, err := t.checkParam(msg)
	if err != nil {
		log.Error("Service register check param failed.", err)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "check Param failed")
		return err
	}

	err = json.Unmarshal(newMsg, t.RestBody)
	if err != nil {
		log.Errorf(nil, "Service register request unmarshalling failed.")
		t.SetFirstErrorCode(meputil.ParseInfoErr, "unmarshal request body error")
		return errors.New("json unmarshalling failed")
	}
	err = meputil.ValidateRestBody(t.RestBody)
	if err != nil {
		t.SetFirstErrorCode(meputil.RequestParamErr, "request param validation failed")
		return err
	}
	return nil
}

func (t *DecodeRestReq) checkParam(msg []byte) ([]byte, error) {

	var temp map[string]interface{}
	err := json.Unmarshal(msg, &temp)
	if err != nil {
		return nil, errors.New("unmarshal msg error")
	}

	meputil.SetMapValue(temp, "consumedLocalOnly", true)
	meputil.SetMapValue(temp, "isLocal", true)
	meputil.SetMapValue(temp, "scopeOfLocality", "MEC_HOST")

	msg, err = json.Marshal(&temp)
	if err != nil {
		return nil, errors.New("marshal map to json error")
	}

	return msg, nil
}

// WithBody set body and return DecodeRestReq
func (t *DecodeRestReq) WithBody(body interface{}) *DecodeRestReq {
	t.RestBody = body
	return t
}

// GetParam get url param and validates
func (t *DecodeRestReq) GetParam(r *http.Request) error {
	query, _ := meputil.GetHTTPTags(r)
	var err error

	t.AppInstanceId = query.Get(":appInstanceId")
	if err := meputil.ValidateAppInstanceIdWithHeader(t.AppInstanceId, r); err != nil {
		log.Error("Validate X-AppinstanceId failed.", err)
		t.SetFirstErrorCode(meputil.AuthorizationValidateErr, err.Error())
		return err
	}

	err = meputil.ValidateUUID(t.AppInstanceId)
	if err != nil {
		log.Error("App instance ID validation failed.", err)
		t.SetFirstErrorCode(meputil.RequestParamErr, "app Instance ID validation failed, invalid uuid")
		return err
	}

	t.SubscribeId = query.Get(":subscriptionId")
	err = meputil.ValidateUUID(t.SubscribeId)
	if err != nil {
		log.Error("Subscription ID validation failed.", err)
		t.SetFirstErrorCode(meputil.RequestParamErr, "subscription ID validation failed, invalid uuid")
		return err
	}

	t.ServiceId = query.Get(":serviceId")
	if len(t.ServiceId) > 0 {
		err = meputil.ValidateServiceID(t.ServiceId)
		if err != nil {
			log.Error("Invalid service ID on service register.", err)
			t.SetFirstErrorCode(meputil.SerErrFailBase, "invalid service ID")
			return err
		}
	}

	t.Ctx = util.SetTargetDomainProject(r.Context(), r.Header.Get("X-Domain-Name"), query.Get(":project"))
	return nil
}

type RegisterServiceId struct {
	HttpErrInf *proto.Response `json:"httpErrInf,out"`
	workspace.TaskBase
	Ctx       context.Context `json:"ctx,in"`
	ServiceId string          `json:"serviceId,out"`
	RestBody  interface{}     `json:"restBody,in"`
}

// OnRequest handles service registration id generations
func (t *RegisterServiceId) OnRequest(data string) workspace.TaskCode {

	serviceInfo, ok := t.RestBody.(*models.ServiceInfo)
	if !ok {
		log.Error(meputil.ErrorRequestBodyMessage, nil)
		t.SetFirstErrorCode(1, meputil.ErrorRequestBodyMessage)
		return workspace.TaskFinish
	}
	_, err := json.Marshal(serviceInfo)
	if err != nil {
		log.Error("Service register service info parse error", nil)
		t.SetFirstErrorCode(meputil.ParseInfoErr, "parse service info error")
		return workspace.TaskFinish
	}

	req := &proto.CreateServiceRequest{}
	serviceInfo.GenerateServiceRequest(req)
	resp, err := core.ServiceAPI.Create(t.Ctx, req)
	if err != nil {
		log.Error("Service center service api create fail.", nil)
		t.SetFirstErrorCode(meputil.SerErrServiceRegFailed, "service creation failed")
		return workspace.TaskFinish
	}

	if resp.ServiceId == "" {
		log.Error("Service register failed.", nil)
		t.SetFirstErrorCode(meputil.SerErrServiceRegFailed, "service register failed")
	}
	t.ServiceId = resp.ServiceId
	return workspace.TaskFinish
}

type RegisterServiceInst struct {
	HttpErrInf *proto.Response `json:"httpErrInf,out"`
	workspace.TaskBase
	W             http.ResponseWriter `json:"w,in"`
	Ctx           context.Context     `json:"ctx,in"`
	AppInstanceId string              `json:"appInstanceId,in"`
	ServiceId     string              `json:"serviceId,in"`
	InstanceId    string              `json:"instanceId,out"`
	RestBody      interface{}         `json:"restBody,in"`
	HttpRsp       interface{}         `json:"httpRsp,out"`
}

// OnRequest handles service instance registrations
func (t *RegisterServiceInst) OnRequest(data string) workspace.TaskCode {
	serviceInfo, ok := t.RestBody.(*models.ServiceInfo)
	if !ok {
		log.Error(meputil.ErrorRequestBodyMessage, nil)
		t.SetFirstErrorCode(1, meputil.ErrorRequestBodyMessage)
		return workspace.TaskFinish
	}
	req := &proto.RegisterInstanceRequest{}
	serviceInfo.GenerateRegisterInstance(req)
	req.Instance.ServiceId = t.ServiceId
	req.Instance.Properties["appInstanceId"] = t.AppInstanceId
	resp, err := core.InstanceAPI.Register(t.Ctx, req)
	if err != nil {
		log.Errorf(nil, "Register instance fail: %s.", t.ServiceId)
		t.SetFirstErrorCode(meputil.SerErrServiceInstanceFailed, "instance registration failed")
		return workspace.TaskFinish
	}
	t.InstanceId = resp.InstanceId
	if t.InstanceId == "" {
		log.Error("Instance id is empty on service registration.", nil)
		t.SetFirstErrorCode(meputil.SerErrServiceInstanceFailed, "instance id is empty")
		return workspace.TaskFinish
	}

	if serviceInfo.LivenessInterval != 0 {
		req.Instance.Properties["liveness"] = fmt.Sprintf(meputil.LivenessPath, t.AppInstanceId, t.ServiceId+t.InstanceId)
		serviceInfo.Links.Self.Href = fmt.Sprintf(meputil.LivenessPath, t.AppInstanceId, t.ServiceId+t.InstanceId)
	}
	reqs := &proto.UpdateInstancePropsRequest{
		ServiceId:  t.ServiceId,
		InstanceId: t.InstanceId,
		Properties: req.Instance.Properties,
	}
	_, err = core.InstanceAPI.UpdateInstanceProperties(t.Ctx, reqs)
	if err != nil {
		log.Error("Service properties of heartbeat update failed.", nil)
		t.SetFirstErrorCode(meputil.SerErrServiceInstanceFailed, "Status properties failed")
		return workspace.TaskFinish
	}
	// build response serviceComb use serviceId + InstanceId to mark a service instance
	mp1SerId := t.ServiceId + t.InstanceId
	serviceInfo.SerInstanceId = mp1SerId
	t.HttpRsp = serviceInfo

	location := fmt.Sprintf("/mep/mp1/v1/services/%s", mp1SerId)
	t.W.Header().Set("Location", location)
	_, err = json.Marshal(serviceInfo)
	if err != nil {
		log.Errorf(nil, "Service info encoding on registration failed.")
		unResReq := &proto.UnregisterInstanceRequest{
			ServiceId:  t.ServiceId,
			InstanceId: t.InstanceId,
		}
		_, err := core.InstanceAPI.Unregister(t.Ctx, unResReq)
		if err != nil {
			log.Errorf(nil, "Service delete failed.")
		}
		t.SetFirstErrorCode(meputil.ParseInfoErr, "marshal service info failed")
		return workspace.TaskFinish
	}
	log.Debugf("Response sent for service registration with appId %s.", t.AppInstanceId)
	return workspace.TaskFinish
}

type RegisterLimit struct {
	workspace.TaskBase
	Ctx           context.Context `json:"ctx,in"`
	RestBody      interface{}     `json:"restBody,in"`
	AppInstanceId string          `json:"appInstanceId,in"`
}

// OnRequest handles the max limit checking for the service registration
func (t *RegisterLimit) OnRequest(data string) workspace.TaskCode {
	var query url.Values
	instances, err := meputil.FindInstanceByKey(query)
	if err != nil {
		if err.Error() == "null" {
			return workspace.TaskFinish
		}
		log.Error("Find service instance failed.", nil)
		t.SetFirstErrorCode(meputil.SerErrServiceRegFailed, "find instance error")
		return workspace.TaskFinish
	}
	if instances == nil {
		return workspace.TaskFinish
	}
	var count int
	for _, instance := range instances.Instances {
		if instance.Properties["appInstanceId"] == t.AppInstanceId {
			count++
		}
	}
	if count >= meputil.ServicesMaxCount {
		log.Error("Registered services have reached the limit.", nil)
		t.SetFirstErrorCode(meputil.SerErrServiceRegFailed, "registered services have achieve the limit")
	}
	return workspace.TaskFinish
}
