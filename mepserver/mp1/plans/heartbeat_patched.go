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

package plans

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"mepserver/common/arch/workspace"
	meputil "mepserver/common/util"
	"net/http"
	"strconv"
	"time"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"github.com/apache/servicecomb-service-center/server/core"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	"github.com/go-playground/validator/v10"
)

type DecodeHeartbeatRestReq struct {
	workspace.TaskBase
	R             *http.Request   `json:"r,in"`
	Ctx           context.Context `json:"ctx,out"`
	CoreRequest   interface{}     `json:"coreRequest,out"`
	AppInstanceId string          `json:"appInstanceId,out"`
	ServiceId     string          `json:"serviceId,out"`
	RestBody      interface{}     `json:"restBody,out"`
}

// OnRequest handles heartbeat request decode
func (t *DecodeHeartbeatRestReq) OnRequest(data string) workspace.TaskCode {
	var err error
	log.Infof("Received message from ClientIP [%s] AppInstanceId [%s] Operation [%s] Resource [%s] in heartbeat.",
		meputil.GetClientIp(t.R), meputil.GetAppInstanceId(t.R), meputil.GetMethod(t.R), meputil.GetResourceInfo(t.R))
	t.Ctx, t.CoreRequest, err = t.getFindParam(t.R)
	if err != nil {
		log.Error("Parameters validation for heartbeat failed.", err)
		return workspace.TaskFinish
	}
	req, ok := t.CoreRequest.(*proto.GetOneInstanceRequest)
	if !ok {
		log.Error("Error in casting the heartbeat patch.", nil)
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "get one instance request for heartbeat patch error")
		return workspace.TaskFinish
	}
	err = t.ParseBody(t.R)
	if err != nil {
		log.Error("Parse heartbeat body error.", err)
		return workspace.TaskFinish
	}
	log.Debugf("Query request arrived to fetch the service information to heartbeat patch with subscriptionId %s.",
		req.ProviderServiceId)
	return workspace.TaskFinish

}

// ParseBody Parse request body
func (t *DecodeHeartbeatRestReq) ParseBody(r *http.Request) error {
	var err error
	msg, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("Heart beat rest request body read failed.", nil)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "read request body error")
		return errors.New("read failed")
	}
	if len(msg) > meputil.RequestBodyLength {
		err = errors.New("request body too large")
		log.Errorf(err, "Request body too large %d.", len(msg))
		t.SetFirstErrorCode(meputil.RequestParamErr, "request body too large")
		return err
	}
	if len(msg) == 0 {
		err = errors.New("body is empty")
		log.Errorf(err, "Heart beat request body is empty.")
		t.SetFirstErrorCode(meputil.RequestParamErr, "request body is empty")
		return err
	}
	err = json.Unmarshal(msg, t.RestBody)
	if err != nil {
		log.Errorf(nil, "Heart beat request unmarshalling failed.")
		t.SetFirstErrorCode(meputil.ParseInfoErr, "unmarshal request body error")
		return errors.New("json unmarshalling failed")
	}
	err = validateRestBody(t.RestBody)
	if err != nil {
		t.SetFirstErrorCode(meputil.RequestParamErr, "request param validation failed")
		return err
	}
	return nil
}

func (t *DecodeHeartbeatRestReq) getFindParam(r *http.Request) (context.Context, *proto.GetOneInstanceRequest, error) {
	query, ids := meputil.GetHTTPTags(r)

	var err error
	t.AppInstanceId = query.Get(":appInstanceId")
	err = meputil.ValidateAppInstanceIdWithHeader(t.AppInstanceId, r)
	if err != nil {
		log.Error("Validate X-AppInstanceId failed.", err)
		t.SetFirstErrorCode(meputil.AuthorizationValidateErr, err.Error())
		return nil, nil, err
	}

	mp1SrvId := query.Get(":serviceId")
	err = meputil.ValidateServiceID(mp1SrvId)
	if err != nil {
		log.Error("Invalid service id.", err)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "Invalid service ID")
		return nil, nil, err
	}
	t.ServiceId = mp1SrvId
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

func (t *DecodeHeartbeatRestReq) WithBodies(body interface{}) *DecodeHeartbeatRestReq {
	t.RestBody = body
	return t
}

type UpdateHeartbeat struct {
	HttpErrInf *proto.Response `json:"httpErrInf,out"`
	workspace.TaskBase
	Ctx           context.Context `json:"ctx,in"`
	AppInstanceId string          `json:"appInstanceId,in"`
	CoreRequest   interface{}     `json:"coreRequest,in"`
	ServiceId     string          `json:"serviceId,in"`
	HttpRsp       interface{}     `json:"httpRsp,out"`
	RestBody      interface{}     `json:"restBody,in"`
}

func (t *UpdateHeartbeat) OnRequest(data string) workspace.TaskCode {
	if t.ServiceId == "" {
		log.Error("Heartbeat request service id is empty.", nil)
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "ServiceId is not found")
		return workspace.TaskFinish
	}
	serviceID := t.ServiceId[:len(t.ServiceId)/2]
	instanceID := t.ServiceId[len(t.ServiceId)/2:]
	reqs, ok := t.CoreRequest.(*proto.GetOneInstanceRequest)
	if !ok {
		log.Error("Get instance request error.", nil)
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "get instance request error")
		return workspace.TaskFinish
	}
	resp, errGetOneInstance := core.InstanceAPI.GetOneInstance(t.Ctx, reqs)
	if errGetOneInstance != nil {
		log.Error("Get one instance error.", nil)
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "get one instance error")
		return workspace.TaskFinish
	}
	t.HttpErrInf = resp.Response

	t.filterAppInstanceId(resp.Instance)
	if resp.Instance == nil {
		log.Error("Service instance id not found.", nil)
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "service instance id not found")
		return workspace.TaskFinish
	}
	properties := resp.Instance.Properties
	interval, err := strconv.Atoi(properties["livenessInterval"])
	if err != nil {
		log.Warn("time Interval is failing")
	}
	if interval == 0 {
		log.Error("Service instance is not avail the service of heartbeat. Invalid patch request.", nil)
		t.SetFirstErrorCode(meputil.HeartbeatServiceNotFound, "Invalid heartbeat update request")
		return workspace.TaskFinish
	}
	if properties["mecState"] == meputil.InactiveState {
		log.Error("Can't change the service from inactive to active state.", nil)
		t.SetFirstErrorCode(meputil.ServiceInactive, "The service is in INACTIVE state")
		return workspace.TaskFinish
	}
	meputil.InfoToProperties(properties, "mecState", meputil.ActiveState)
	secNanoSec := strconv.FormatInt(time.Now().UTC().UnixNano(), meputil.FormatIntBase)
	meputil.InfoToProperties(properties, "timestamp/seconds", secNanoSec[:len(secNanoSec)/2+1])
	meputil.InfoToProperties(properties, "timestamp/nanoseconds", secNanoSec[len(secNanoSec)/2+1:])
	req := &proto.UpdateInstancePropsRequest{
		ServiceId:  serviceID,
		InstanceId: instanceID,
		Properties: properties,
	}
	respns, err := core.InstanceAPI.UpdateInstanceProperties(t.Ctx, req)
	if err != nil {
		log.Error("Service properties of heartbeat update failed.", nil)
		t.SetFirstErrorCode(meputil.SerErrServiceInstanceFailed, "Status properties failed")
		return workspace.TaskFinish
	}
	t.HttpErrInf = respns.Response
	t.HttpRsp = ""
	log.Debugf("Status of service of heartbeat with serviceId %s is updated successfully.", serviceID)
	return workspace.TaskFinish
}

func (t *UpdateHeartbeat) filterAppInstanceId(inst *proto.MicroServiceInstance) {
	if inst == nil || inst.Properties == nil {
		return
	}
	if t.AppInstanceId != inst.Properties["appInstanceId"] {
		inst = nil
	}
}

// validate rest body
func validateRestBody(body interface{}) error {
	validate := validator.New()
	verrs := validate.Struct(body)
	if verrs != nil {
		for _, verr := range verrs.(validator.ValidationErrors) {
			log.Debugf("Namespace=%s, Field=%s, StructField=%s, Tag=%s, Kind =%s, Type=%s, Value=%s",
				verr.Namespace(), verr.Field(), verr.StructField(), verr.Tag(), verr.Kind(), verr.Type(),
				verr.Value())
		}
		return verrs
	}
	return nil
}
