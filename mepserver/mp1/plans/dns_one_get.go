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
	"encoding/json"
	"mepserver/common/extif/dataplane"
	"mepserver/common/models"

	"github.com/apache/servicecomb-service-center/pkg/log"

	"mepserver/common/arch/workspace"
	"mepserver/common/extif/backend"
	"mepserver/common/util"
)

type DNSRuleGet struct {
	workspace.TaskBase
	AppInstanceId string      `json:"appInstanceId,in"`
	DNSRuleId     string      `json:"dnsRuleId,in"`
	HttpRsp       interface{} `json:"httpRsp,out"`
}

func (t *DNSRuleGet) OnRequest(data string) workspace.TaskCode {
	log.Debugf("query request arrived to fetch dns rule %s for appId %s.", t.DNSRuleId, t.AppInstanceId)

	if len(t.DNSRuleId) == 0 {
		log.Errorf(nil, "invalid dns id on query request")
		t.SetFirstErrorCode(util.ParseInfoErr, "invalid query request")
		return workspace.TaskFinish
	}

	appDConfigEntry, errCode := backend.GetRecord(util.AppDConfigKeyPath + t.AppInstanceId)
	if errCode != 0 {
		log.Errorf(nil, "get dns rule from data-store failed")
		t.SetFirstErrorCode(workspace.ErrCode(errCode), "dns rule retrieval failed")
		return workspace.TaskFinish
	}

	appDInStore := models.AppDConfig{}
	var dnsOnStore *dataplane.DNSRule
	if appDConfigEntry != nil {
		jsonErr := json.Unmarshal(appDConfigEntry, &appDInStore)
		if jsonErr != nil {
			log.Errorf(jsonErr, "failed to parse the dns entry from data-store on update request")
			t.SetFirstErrorCode(util.OperateDataWithEtcdErr, "parse dns rules failed")
			return workspace.TaskFinish
		}
		for _, rule := range appDInStore.AppDNSRule {
			if rule.DNSRuleID == t.DNSRuleId {
				dnsOnStore = &rule
				break
			}
		}
	}
	if dnsOnStore == nil {
		log.Error("Requested dns rule id doesn't exists.", nil)
		t.SetFirstErrorCode(util.SubscriptionNotFound, "dns rule retrieval failed")
		return workspace.TaskFinish
	}

	t.HttpRsp = dnsOnStore
	return workspace.TaskFinish
}
