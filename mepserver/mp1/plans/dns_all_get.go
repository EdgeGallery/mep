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

// Package path implements mep server api plans
package plans

import (
	"encoding/json"
	"mepserver/common/extif/dataplane"
	"mepserver/common/models"

	"github.com/apache/servicecomb-service-center/pkg/log"

	"mepserver/common/arch/workspace"
	"mepserver/common/extif/backend"
	"mepserver/common/util"
)

type DNSRulesGet struct {
	workspace.TaskBase
	AppInstanceId string      `json:"appInstanceId,in"`
	HttpRsp       interface{} `json:"httpRsp,out"`
}

func (t *DNSRulesGet) OnRequest(data string) workspace.TaskCode {

	log.Debugf("query request arrived to fetch all dns rules for appId %s.", t.AppInstanceId)

	appDConfigEntry, errCode := backend.GetRecord(util.AppDConfigKeyPath + t.AppInstanceId)
	if errCode == util.SubscriptionNotFound {
		t.HttpRsp = []dataplane.DNSRule{}
		return workspace.TaskFinish
	} else if errCode != 0 {
		t.SetFirstErrorCode(workspace.ErrCode(errCode), "dns rules not found")
		return workspace.TaskFinish
	}

	appDInStore := models.AppDConfig{}
	jsonErr := json.Unmarshal(appDConfigEntry, &appDInStore)
	if jsonErr != nil {
		log.Errorf(nil, "Failed to parse the dns entries from data-store.")
		t.SetFirstErrorCode(util.OperateDataWithEtcdErr, "parse dns rules from data-store failed")
		return workspace.TaskFinish
	}

	t.HttpRsp = appDInStore.AppDNSRule
	return workspace.TaskFinish
}
