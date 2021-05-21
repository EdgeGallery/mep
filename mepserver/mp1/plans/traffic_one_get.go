/*
 * Copyright 2021 Huawei Technologies Co., Ltd.
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

// Package plans implements mep server traffic apis
package plans

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"github.com/go-playground/validator/v10"
	"mepserver/common/extif/dataplane"
	"mepserver/common/models"
	meputil "mepserver/common/util"
	"net/http"

	"mepserver/common/extif/backend"

	"io/ioutil"

	"github.com/apache/servicecomb-service-center/pkg/log"

	"mepserver/common/arch/workspace"
)

// DecodeTrafficRestReq step to decode the traffic request message
type DecodeTrafficRestReq struct {
	workspace.TaskBase
	R             *http.Request   `json:"r,in"`
	Ctx           context.Context `json:"ctx,out"`
	AppInstanceId string          `json:"appInstanceId,out"`
	TrafficRuleId string          `json:"trafficRuleId,out"`
	RestBody      interface{}     `json:"restBody,out"`
}

// OnRequest handles the decode request message
func (t *DecodeTrafficRestReq) OnRequest(data string) workspace.TaskCode {
	err := t.getParam(t.R)
	if err != nil {
		log.Error("Parameters validation failed on traffic request.", err)
		return workspace.TaskFinish
	}
	err = t.parseBody(t.R)
	if err != nil {
		log.Error("Parse rest body failed on traffic rule request.", err)
	}
	return workspace.TaskFinish
}

func (t *DecodeTrafficRestReq) parseBody(r *http.Request) error {
	if t.RestBody == nil {
		return nil
	}
	msg, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("Traffic rule request body failed.", err)
		t.SetFirstErrorCode(meputil.SerErrFailBase, err.Error())
		return err
	}

	newMsg, err := t.checkParam(msg)
	if err != nil {
		log.Error("Traffic rule param check failed.", err)
		t.SetFirstErrorCode(meputil.SerErrFailBase, err.Error())
		return err
	}

	err = json.Unmarshal(newMsg, t.RestBody)
	if err != nil {
		log.Errorf(err, "Traffic request marshall failed: %s.", util.BytesToStringWithNoCopy(newMsg))
		t.SetFirstErrorCode(meputil.SerErrFailBase, err.Error())
		return err
	}

	trafficInPut, _ := t.RestBody.(*dataplane.TrafficRule)
	var validate *validator.Validate
	validate = validator.New()
	verrs := validate.Struct(trafficInPut)
	if verrs != nil {
		for _, verr := range verrs.(validator.ValidationErrors) {
			log.Errorf(err, "Validation Error(namespace: %v, field: %v, struct namespace:%v, struct field: %v, "+
				"tag: %v, actual tag: %v, kind: %v, type: %v, value: %v, param: %v).", verr.Namespace(), verr.Field(),
				verr.StructNamespace(), verr.StructField(), verr.Tag(), verr.ActualTag(), verr.Kind(), verr.Type(),
				verr.Value(), verr.Param())
		}
		t.SetFirstErrorCode(meputil.SerErrFailBase, verrs.Error())
		return verrs
	}

	return nil
}

func (t *DecodeTrafficRestReq) checkParam(msg []byte) ([]byte, error) {

	var temp map[string]interface{}
	err := json.Unmarshal(msg, &temp)
	if err != nil {
		log.Errorf(err, "Invalid traffic request body: %s.", util.BytesToStringWithNoCopy(msg))
		t.SetFirstErrorCode(meputil.SerErrFailBase, err.Error())
		return nil, err
	}

	msg, err = json.Marshal(&temp)
	if err != nil {
		log.Errorf(err, "Traffic request encoding failed.")
		t.SetFirstErrorCode(meputil.SerErrFailBase, err.Error())
		return nil, err
	}

	return msg, nil
}

// WithBody initialize the traffic body message
func (t *DecodeTrafficRestReq) WithBody(body interface{}) *DecodeTrafficRestReq {
	t.RestBody = body
	return t
}

func (t *DecodeTrafficRestReq) getParam(r *http.Request) error {
	query, _ := meputil.GetHTTPTags(r)
	var err error

	t.AppInstanceId = query.Get(":appInstanceId")
	if len(t.AppInstanceId) == 0 {
		err = fmt.Errorf("invalid app instance id")
		t.SetFirstErrorCode(meputil.AuthorizationValidateErr, err.Error())
		return err
	}
	if err = meputil.ValidateAppInstanceIdWithHeader(t.AppInstanceId, r); err != nil {
		log.Error("Validate X-AppinstanceId failed.", err)
		t.SetFirstErrorCode(meputil.AuthorizationValidateErr, err.Error())
		return err
	}
	err = meputil.ValidateUUID(t.AppInstanceId)
	if err != nil {
		t.SetFirstErrorCode(meputil.RequestParamErr, "app Instance ID validation failed, invalid uuid")
		return err
	}
	t.TrafficRuleId = query.Get(":trafficRuleId")
	if len(t.TrafficRuleId) > meputil.MaxTrafficRuleIdLength {
		err = fmt.Errorf("traffic rule id validation failed, invalid length")
		t.SetFirstErrorCode(meputil.RequestParamErr, err.Error())
		return err
	}
	t.Ctx = util.SetTargetDomainProject(r.Context(), r.Header.Get("X-Domain-Name"), query.Get(":project"))
	return nil
}

// TrafficRuleGet steps to query the traffic rules
type TrafficRuleGet struct {
	workspace.TaskBase
	AppInstanceId string      `json:"appInstanceId,in"`
	TrafficRuleId string      `json:"trafficRuleId,in"`
	HttpRsp       interface{} `json:"httpRsp,out"`
}

// OnRequest handles the traffic rule query
func (t *TrafficRuleGet) OnRequest(data string) workspace.TaskCode {

	if len(t.AppInstanceId) == 0 || len(t.TrafficRuleId) == 0 {
		log.Errorf(nil, "Invalid app/traffic id on query request.")
		t.SetFirstErrorCode(meputil.ParseInfoErr, "invalid query request")
		return workspace.TaskFinish
	}

	appDEntry, errCode := backend.GetRecord(meputil.AppDConfigKeyPath + t.AppInstanceId)
	if errCode != 0 {
		log.Errorf(nil, "Get traffic rules from etcd failed.")
		t.SetFirstErrorCode(workspace.ErrCode(errCode), "traffic rule retrieval failed")
		return workspace.TaskFinish
	}

	appDConfig := models.AppDConfig{}
	var trafficRule *dataplane.TrafficRule
	if appDEntry != nil {
		jsonErr := json.Unmarshal(appDEntry, &appDConfig)
		if jsonErr != nil {
			log.Warn("Could not read the traffic rule properly from etcd.")
			t.SetFirstErrorCode(meputil.OperateDataWithEtcdErr, "parse traffic rules from etcd failed")
			return workspace.TaskFinish
		}
		for _, rule := range appDConfig.AppTrafficRule {
			if rule.TrafficRuleID == t.TrafficRuleId {
				trafficRule = &rule
				break
			}
		}
	}
	if trafficRule == nil {
		log.Error("Requested traffic rule id doesn't exists.", nil)
		t.SetFirstErrorCode(meputil.SubscriptionNotFound, "traffic rule does not exist")
		return workspace.TaskFinish
	}
	t.HttpRsp = trafficRule
	return workspace.TaskFinish
}
