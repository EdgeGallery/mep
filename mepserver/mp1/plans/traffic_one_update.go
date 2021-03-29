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

package plans

import (
	"encoding/json"
	"mepserver/common/extif/dataplane"
	"mepserver/common/models"
	"net/http"
	"reflect"

	meputil "mepserver/common/util"

	"mepserver/common/extif/backend"

	"github.com/apache/servicecomb-service-center/pkg/log"

	"mepserver/common/arch/workspace"
)

type TrafficRuleUpdate struct {
	workspace.TaskBase
	R             *http.Request       `json:"r,in"`
	W             http.ResponseWriter `json:"w,in"`
	RestBody      interface{}         `json:"restBody,in"`
	AppInstanceId string              `json:"appInstanceId,in"`
	TrafficRuleId string              `json:"trafficRuleId,in"`
	HttpRsp       interface{}         `json:"httpRsp,out"`
	dataPlane     dataplane.DataPlane
}

func (t *TrafficRuleUpdate) WithDataPlane(dataPlane dataplane.DataPlane) *TrafficRuleUpdate {
	t.dataPlane = dataPlane
	return t
}

func (t *TrafficRuleUpdate) OnRequest(data string) workspace.TaskCode {

	trafficInPut, ok := t.RestBody.(*dataplane.TrafficRule)
	if !ok {
		t.SetFirstErrorCode(meputil.ParseInfoErr, "rest-body failed")
		return workspace.TaskFinish
	}

	if len(t.TrafficRuleId) == 0 {
		log.Errorf(nil, "invalid app/traffic id on update request")
		t.SetFirstErrorCode(meputil.ParseInfoErr, "invalid update request")
		return workspace.TaskFinish
	}

	appDConfigDB, errCode := backend.GetRecord(meputil.AppDConfigKeyPath + t.AppInstanceId)
	if errCode != 0 {
		log.Errorf(nil, "Update traffic rules failed")
		t.SetFirstErrorCode(workspace.ErrCode(errCode), "update rule retrieval failed")
		return workspace.TaskFinish
	}

	appDConfig := models.AppDConfig{}
	var trafficRule *dataplane.TrafficRule
	var ruleIndex int

	jsonErr := json.Unmarshal(appDConfigDB, &appDConfig)
	if jsonErr != nil {
		log.Warn("Could not read the traffic rule properly from etcd.")
		t.SetFirstErrorCode(meputil.OperateDataWithEtcdErr, "parse traffic rules from etcd failed")
		return workspace.TaskFinish
	}
	for i, rule := range appDConfig.AppTrafficRule {
		if rule.TrafficRuleID == t.TrafficRuleId {
			trafficRule = &rule
			ruleIndex = i
			break
		}
	}

	if trafficRule == nil {
		log.Error("Requested traffic rule id doesn't exists.", nil)
		t.SetFirstErrorCode(meputil.SubscriptionNotFound, "traffic rule does not exist")
		return workspace.TaskFinish
	}

	if reflect.DeepEqual(trafficRule, trafficInPut) {
		t.HttpRsp = trafficInPut
		return workspace.TaskFinish
	}

	dataStoreEntryBytes, err := json.Marshal(trafficRule)
	if err != nil {
		log.Errorf(err, "Traffic rule parse failed")
		t.SetFirstErrorCode(meputil.ParseInfoErr, "internal error on data parsing")
		return workspace.TaskFinish
	}

	// Check for E-Tags precondition. More details could be found here: https://tools.ietf.org/html/rfc7232#section-2.3
	ifMatchTag := t.R.Header.Get("If-Match")
	if len(ifMatchTag) != 0 && ifMatchTag != meputil.GenerateStrongETag(dataStoreEntryBytes) {
		log.Warn("E-Tag miss-match.")
		t.SetFirstErrorCode(meputil.EtagMissMatchErr, "e-tag miss-match")
		return workspace.TaskFinish
	}

	if len(trafficInPut.TrafficRuleID) != 0 && trafficRule.TrafficRuleID != trafficInPut.TrafficRuleID {
		log.Warn("Traffic identifier miss-match.")
		t.SetFirstErrorCode(meputil.ParseInfoErr, "traffic identifier miss-match")
		return workspace.TaskFinish
	}

	errCode, errString := t.applyTrafficRule(trafficRule, appDConfig, ruleIndex, appDConfigDB)
	if errCode != 0 {
		t.SetFirstErrorCode(workspace.ErrCode(errCode), errString)
		return workspace.TaskFinish
	}
	return workspace.TaskFinish
}

func (t *TrafficRuleUpdate) applyTrafficRule(trafficRule *dataplane.TrafficRule, appDConfig models.AppDConfig,
	ruleIndex int, appDConfigDB []byte) (int, string) {
	trafficInPut, _ := t.RestBody.(*dataplane.TrafficRule)

	trafficInPut.TrafficRuleID = trafficRule.TrafficRuleID

	appDConfig.AppTrafficRule[ruleIndex] = *trafficInPut
	updateJSON, err := json.Marshal(appDConfig)
	if err != nil {
		log.Errorf(err, "Can not marshal the input traffic body")
		return meputil.ParseInfoErr, "can not marshal traffic info"
	}

	resultErr := backend.PutRecord(meputil.AppDConfigKeyPath+t.AppInstanceId, updateJSON)
	if resultErr != 0 {
		log.Errorf(nil, "Traffic rule(appId: %s, ruleId: %s) update on etcd failed, "+
			"this will lead to data inconsistency!", t.AppInstanceId,
			t.TrafficRuleId)
		return meputil.OperateDataWithEtcdErr, "put traffic rule to etcd failed"
	}

	appInfo := dataplane.ApplicationInfo{
		ApplicationId:   t.AppInstanceId,
		ApplicationName: appDConfig.AppName,
	}
	if trafficInPut.State != trafficRule.State {
		if trafficInPut.State == "ACTIVE" {
			err = t.dataPlane.AddTrafficRule(appInfo, t.TrafficRuleId, trafficInPut.FilterType,
				trafficInPut.Action, trafficInPut.Priority, trafficInPut.TrafficFilter)
		} else {
			err = t.dataPlane.DeleteTrafficRule(appInfo, t.TrafficRuleId)
		}
	} else if trafficInPut.State == "ACTIVE" {
		err = t.dataPlane.SetTrafficRule(appInfo, t.TrafficRuleId, trafficInPut.FilterType,
			trafficInPut.Action, trafficInPut.Priority, trafficInPut.TrafficFilter)
	}

	if err != nil {
		log.Errorf(err, "Traffic rule(appId: %s, dnsRuleId: %s) update fail on server: %s!",
			t.AppInstanceId, t.TrafficRuleId, err.Error())
		t.SetFirstErrorCode(meputil.RemoteServerErr, "failed to apply configuration on data-plane")

		resultErr := backend.PutRecord(meputil.AppDConfigKeyPath+t.AppInstanceId, appDConfigDB)
		if resultErr != 0 {
			log.Errorf(nil, "Traffic rule(appId: %s, ruleId: %s) update on etcd failed, "+
				"this will lead to data inconsistency!", t.AppInstanceId,
				t.TrafficRuleId)
		}
		return 0, ""
	}

	t.W.Header().Set("ETag", meputil.GenerateStrongETag(updateJSON))
	t.HttpRsp = trafficInPut
	return 0, ""
}
