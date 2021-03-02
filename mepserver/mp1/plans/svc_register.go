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

// Package path implements mep server api plans
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

// OnRequest
func (t *DecodeRestReq) OnRequest(data string) workspace.TaskCode {
	log.Infof("Received message from ClientIP [%s] AppInstanceId [%s] Operation [%s] Resource [%s]",
		meputil.GetClientIp(t.R), meputil.GetAppInstanceId(t.R), meputil.GetMethod(t.R), meputil.GetResourceInfo(t.R))

	err := t.GetParam(t.R)
	if err != nil {
		log.Error("parameters validation failed", err)
		return workspace.TaskFinish
	}

	err = t.ParseBody(t.R)
	if err != nil {
		log.Error("parse rest body failed", err)
	}
	return workspace.TaskFinish
}

// Parse request body
func (t *DecodeRestReq) ParseBody(r *http.Request) error {
	if t.RestBody == nil {
		return nil
	}
	msg, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("read failed", nil)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "read request body error")
		return errors.New("read failed")
	}
	if len(msg) > meputil.RequestBodyLength {
		err = errors.New("request body too large")
		log.Errorf(err, "request body too large %d", len(msg))
		t.SetFirstErrorCode(meputil.RequestParamErr, "request body too large")
		return err
	}
	newMsg, err := t.checkParam(msg)
	if err != nil {
		log.Error("check Param failed", err)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "check Param failed")
		return err
	}

	err = json.Unmarshal(newMsg, t.RestBody)
	if err != nil {
		log.Errorf(nil, "json unmarshalling failed")
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

// set body and return DecodeRestReq
func (t *DecodeRestReq) WithBody(body interface{}) *DecodeRestReq {
	t.RestBody = body
	return t
}

// get param
func (t *DecodeRestReq) GetParam(r *http.Request) error {
	query, _ := meputil.GetHTTPTags(r)
	var err error

	t.AppInstanceId = query.Get(":appInstanceId")
	if err := meputil.ValidateAppInstanceIdWithHeader(t.AppInstanceId, r); err != nil {
		log.Error("validate X-AppinstanceId failed", err)
		t.SetFirstErrorCode(meputil.AuthorizationValidateErr, err.Error())
		return err
	}

	err = meputil.ValidateUUID(t.AppInstanceId)
	if err != nil {
		log.Error("app Instance ID validation failed", err)
		t.SetFirstErrorCode(meputil.RequestParamErr, "app Instance ID validation failed, invalid uuid")
		return err
	}

	t.SubscribeId = query.Get(":subscriptionId")
	err = meputil.ValidateUUID(t.SubscribeId)
	if err != nil {
		log.Error("subscription ID validation failed", err)
		t.SetFirstErrorCode(meputil.RequestParamErr, "subscription ID validation failed, invalid uuid")
		return err
	}

	t.ServiceId = query.Get(":serviceId")
	if len(t.ServiceId) > 0 {
		err = meputil.ValidateServiceID(t.ServiceId)
		if err != nil {
			log.Error("invalid service ID", err)
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

// OnRequest
func (t *RegisterServiceId) OnRequest(data string) workspace.TaskCode {

	serviceInfo, ok := t.RestBody.(*models.ServiceInfo)
	if !ok {
		log.Error(meputil.ErrorRequestBodyMessage, nil)
		t.SetFirstErrorCode(1, meputil.ErrorRequestBodyMessage)
		return workspace.TaskFinish
	}
	_, err := json.Marshal(serviceInfo)
	if err != nil {
		log.Error("parse service info error", nil)
		t.SetFirstErrorCode(meputil.ParseInfoErr, "parse service info error")
		return workspace.TaskFinish
	}

	req := &proto.CreateServiceRequest{}
	serviceInfo.ToServiceRequest(req)
	resp, err := core.ServiceAPI.Create(t.Ctx, req)
	if err != nil {
		log.Error("service center service api create fail", nil)
		t.SetFirstErrorCode(meputil.SerErrServiceRegFailed, "service creation failed")
		return workspace.TaskFinish
	}

	if resp.ServiceId == "" {
		log.Error("service register failed", nil)
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

// OnRequest
func (t *RegisterServiceInst) OnRequest(data string) workspace.TaskCode {
	serviceInfo, ok := t.RestBody.(*models.ServiceInfo)
	if !ok {
		log.Error(meputil.ErrorRequestBodyMessage, nil)
		t.SetFirstErrorCode(1, meputil.ErrorRequestBodyMessage)
		return workspace.TaskFinish
	}
	req := &proto.RegisterInstanceRequest{}
	serviceInfo.ToRegisterInstance(req)
	req.Instance.ServiceId = t.ServiceId
	req.Instance.Properties["appInstanceId"] = t.AppInstanceId
	resp, err := core.InstanceAPI.Register(t.Ctx, req)
	if err != nil {
		log.Errorf(nil, "registerInstance fail: %s", t.ServiceId)
		t.SetFirstErrorCode(meputil.SerErrServiceInstanceFailed, "instance registration failed")
		return workspace.TaskFinish
	}
	t.InstanceId = resp.InstanceId
	if t.InstanceId == "" {
		log.Error("instance id is empty", nil)
		t.SetFirstErrorCode(meputil.SerErrServiceInstanceFailed, "instance id is empty")
		return workspace.TaskFinish
	}

	if serviceInfo.LivenessInterval != 0 {
		req.Instance.Properties["liveness"] = "/mepserver/mec_service_mgmt/v1/applications/" + t.AppInstanceId + "/services/" + t.ServiceId + t.InstanceId + "/liveness"
		serviceInfo.Links.Self.Href = "/mepserver/mec_service_mgmt/v1/applications/" + t.AppInstanceId + "/services/" + t.ServiceId + t.InstanceId + "/liveness"
	}
	reqs := &proto.UpdateInstancePropsRequest{
		ServiceId:  t.ServiceId,
		InstanceId: t.InstanceId,
		Properties: req.Instance.Properties,
	}
	_, err = core.InstanceAPI.UpdateInstanceProperties(t.Ctx, reqs)
	if err != nil {
		log.Error("service properties of heartbeat updation failed", nil)
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
		log.Errorf(nil, "service info encoding on registration failed")
		unResReq := &proto.UnregisterInstanceRequest{
			ServiceId:  t.ServiceId,
			InstanceId: t.InstanceId,
		}
		_, err := core.InstanceAPI.Unregister(t.Ctx, unResReq)
		if err != nil {
			log.Errorf(nil, "service delete failed")
		}
		t.SetFirstErrorCode(meputil.ParseInfoErr, "marshal service info failed")
		return workspace.TaskFinish
	}
	log.Debugf("response sent for service registration with appId %s", t.AppInstanceId)
	return workspace.TaskFinish
}

type RegisterLimit struct {
	workspace.TaskBase
	Ctx           context.Context `json:"ctx,in"`
	RestBody      interface{}     `json:"restBody,in"`
	AppInstanceId string          `json:"appInstanceId,in"`
}

// OnRequest
func (t *RegisterLimit) OnRequest(data string) workspace.TaskCode {
	var query url.Values
	instances, err := meputil.FindInstanceByKey(query)
	if err != nil {
		if err.Error() == "null" {
			log.Info("the service is empty")
			return workspace.TaskFinish
		}
		log.Error("find instance error", nil)
		t.SetFirstErrorCode(meputil.SerErrServiceRegFailed, "find instance error")
		return workspace.TaskFinish
	}
	if instances == nil {
		log.Info("the service is empty")
		return workspace.TaskFinish
	}
	var count int
	for _, instance := range instances.Instances {
		if instance.Properties["appInstanceId"] == t.AppInstanceId {
			count++
		}
	}
	if count >= meputil.ServicesMaxCount {
		log.Error("registered services have achieve the limit", nil)
		t.SetFirstErrorCode(meputil.SerErrServiceRegFailed, "registered services have achieve the limit")
	}
	return workspace.TaskFinish
}
