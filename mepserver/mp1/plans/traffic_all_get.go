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
	"mepserver/common/extif/backend"
	"mepserver/common/extif/dataplane"
	"mepserver/common/models"

	meputil "mepserver/common/util"

	"github.com/apache/servicecomb-service-center/pkg/log"

	"mepserver/common/arch/workspace"
)

type TrafficRulesGet struct {
	workspace.TaskBase
	AppInstanceId string      `json:"appInstanceId,in"`
	TrafficRuleId string      `json:"trafficRuleId,in"`
	HttpRsp       interface{} `json:"httpRsp,out"`
}

func (t *TrafficRulesGet) OnRequest(data string) workspace.TaskCode {

	if len(t.AppInstanceId) == 0 {
		log.Errorf(nil, "invalid app id on query request")
		t.SetFirstErrorCode(meputil.ParseInfoErr, "invalid query request")
		return workspace.TaskFinish
	}

	trafficRuleDB, errCode := backend.GetRecord(meputil.AppDConfigKeyPath + t.AppInstanceId)
	if errCode != 0 {
		t.HttpRsp = []dataplane.TrafficRule{}
		return workspace.TaskFinish
	}

	if trafficRuleDB == nil {
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "traffic rules not found")
		return workspace.TaskFinish
	}
	appDConfig := models.AppDConfig{}
	jsonErr := json.Unmarshal(trafficRuleDB, &appDConfig)
	if jsonErr != nil {
		log.Errorf(nil, "failed to parse the dns entries from data-store")
		t.SetFirstErrorCode(meputil.OperateDataWithEtcdErr, "parse dns rules from data-store failed")
		return workspace.TaskFinish
	}

	t.HttpRsp = appDConfig.AppTrafficRule
	return workspace.TaskFinish
}
