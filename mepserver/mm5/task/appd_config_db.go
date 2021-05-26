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

package task

import (
	"encoding/json"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"mepserver/common/extif/backend"
	"mepserver/common/extif/dataplane"
	"mepserver/common/models"
	"mepserver/common/util"
)

type appDConfigDB struct {
	appInstanceId string
	appDConfig    *models.AppDConfig
}

func newAppDConfigDB(appInstanceId string) *appDConfigDB {
	appDConfigEntry, errCode := backend.GetRecord(util.AppDConfigKeyPath + appInstanceId)
	if errCode != 0 {
		log.Warnf("retrieve jobs from temp-cache on data-store failed")
		return nil
	}
	appDConfig := &models.AppDConfig{}
	err := json.Unmarshal(appDConfigEntry, appDConfig)
	if err != nil {
		log.Warnf("failed to parse the appDConfig from data-store")
		return nil
	}
	return &appDConfigDB{appInstanceId, appDConfig}
}

// GetDnsRule retrieves dns rule based on the id input
func (a *appDConfigDB) GetDnsRule(ruleId string) *dataplane.DNSRule {
	for _, rule := range a.appDConfig.AppDNSRule {
		if ruleId == rule.DNSRuleID {
			return &rule
		}
	}
	return nil
}

// GetTrafficRule retrieves traffic rule based on the id input
func (a *appDConfigDB) GetTrafficRule(ruleId string) *dataplane.TrafficRule {
	for _, rule := range a.appDConfig.AppTrafficRule {
		if ruleId == rule.TrafficRuleID {
			return &rule
		}
	}
	return nil
}
