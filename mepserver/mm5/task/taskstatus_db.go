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
	"fmt"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"mepserver/common/extif/backend"
	"mepserver/common/models"
	"mepserver/common/util"
)

type statusDB struct {
	appInstanceId string
	taskId        string
	status        *models.TaskStatus
}

func newStatusDB(appInstanceId string, taskId string) *statusDB {
	path := util.AppDLCMTaskStatusPath + appInstanceId + "/" + taskId
	statusEntry, errCode := backend.GetRecord(path)
	if errCode != 0 {
		log.Errorf(nil, "retrieve task statusDb from temp-cache on data-store failed")
		return nil
	}
	status := &models.TaskStatus{}
	err := json.Unmarshal(statusEntry, status)
	if err != nil {
		log.Errorf(nil, "failed to parse the task statusDb from data-store")
		return nil
	}

	return &statusDB{appInstanceId: appInstanceId, taskId: taskId, status: status}
}

func (s *statusDB) searchRule(ruleList []models.RuleStatus, ruleId string) int {
	for index, rule := range ruleList {
		if rule.Id == ruleId {
			return index
		}
	}
	return -1
}

func (s *statusDB) setStateAndProgress(ruleType util.AppDRuleType, ruleId string, state util.AppDRuleStatus) error {
	var oldState util.AppDRuleStatus
	var ruleIndex int

	if ruleType == util.RuleTypeDns {
		ruleIndex = s.searchRule(s.status.DNSRuleStatusLst, ruleId)
		if ruleIndex == -1 {
			return fmt.Errorf("error: could not find the dns rule specified")
		}
		oldState = s.status.DNSRuleStatusLst[ruleIndex].State
		s.status.DNSRuleStatusLst[ruleIndex].State = state
	} else if ruleType == util.RuleTypeTraffic {
		ruleIndex = s.searchRule(s.status.TrafficRuleStatusLst, ruleId)
		if ruleIndex == -1 {
			return fmt.Errorf("error: could not find the traffic rule specified")
		}
		oldState = s.status.TrafficRuleStatusLst[ruleIndex].State
		s.status.TrafficRuleStatusLst[ruleIndex].State = state
	}

	// Set the progress
	if state == util.WaitConfigDBWrite {
		s.status.Progress++
	}

	err := s.pushDB()
	if err != nil {
		if ruleType == util.RuleTypeDns {
			s.status.DNSRuleStatusLst[ruleIndex].State = oldState
		} else if ruleType == util.RuleTypeTraffic {
			s.status.TrafficRuleStatusLst[ruleIndex].State = oldState
		}

		if state == util.WaitConfigDBWrite { // revert the modification on failure
			s.status.Progress--
		}
	}

	log.Debugf("Updated state as %v for rule %v", state, ruleId)

	return err
}

func (s *statusDB) pushDB() error {
	path := util.AppDLCMTaskStatusPath + s.appInstanceId + "/" + s.taskId

	statusBytes, err := json.Marshal(s.status)
	if err != nil {
		log.Errorf(nil, "can not marshal task statusDb info")
		return fmt.Errorf("error: failed to marshal task statusDb while writing to cache")
	}

	errCode := backend.PutRecord(path, statusBytes)
	if errCode != 0 {
		log.Errorf(nil, "update task statusDb on cache failed")
		return fmt.Errorf("error: update task statusDb on cache failed")
	}
	return nil
}

func (s *statusDB) setFailureReason(reason string) {
	// Set only the first failure reason
	if len(s.status.Details) == 0 {
		s.status.Details = reason
	}
}

func CheckErrorInDB(appInstanceId string, taskId string) error {
	path := util.AppDLCMTaskStatusPath + appInstanceId + "/" + taskId

	statusBytes, errCode := backend.GetRecord(path)
	if errCode != 0 {
		log.Errorf(nil, "update task statusDb on cache failed")
		return fmt.Errorf("error: update task statusDb on cache failed")
	}

	var statusDB *models.TaskStatus
	err := json.Unmarshal(statusBytes, &statusDB)
	if err != nil {
		log.Errorf(nil, "can not unmarshal task statusDb info")
		return fmt.Errorf("error: failed to unmarshal task statusDb while writing to cache")
	}

	if statusDB.Progress == util.TaskProgressFailure {
		return fmt.Errorf(statusDB.Details)
	}

	return nil
}
