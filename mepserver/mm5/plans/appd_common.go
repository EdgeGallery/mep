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
	"mepserver/common/arch/workspace"
	"mepserver/common/extif/backend"
	"mepserver/common/extif/dataplane"
	"mepserver/common/models"
	meputil "mepserver/common/util"
	"net/http"
	"reflect"

	"github.com/apache/servicecomb-service-center/pkg/log"
)

const DBFailure = "put app config rule to data-store failed"

type AppDCommon struct {
}

func (a *AppDCommon) IsAppInstanceIdAlreadyExists(appInstanceId string) (isExists bool) {

	/* This Database maintains local cache of all the Dataplane configurations. If record exist here means that
	   operation is completed and this APP has an existing rule configured*/
	record, errCode := backend.GetRecord(meputil.AppDConfigKeyPath + appInstanceId)
	if errCode != 0 || record == nil {
		return false
	}
	return true
}

func (a *AppDCommon) IsAppNameAlreadyExists(appName string) (isExists bool) {

	records, errCode := backend.GetRecords(meputil.AppDConfigKeyPath)
	if errCode != 0 || records == nil || len(records) == 0 {
		return false
	}

	for _, record := range records {
		appDInStore := &models.AppDConfig{}
		if jsonErr := json.Unmarshal(record, appDInStore); jsonErr != nil {
			continue
		}
		if appDInStore.AppName == appName {
			return true
		}
	}
	return false
}

func (a *AppDCommon) IsAnyOngoingOperationExist(appInstanceId string) (isExists bool) {

	/* Jobs DB temporarily holds the ongoing operation of this APPInstanceId. If entry exist in this DB mean the operation
	   is going on and not complete. Once operation is completed this entry would be deleted from DB for this appInstanceId. */
	records, errCode := backend.GetRecords(meputil.AppDLCMJobsPath + appInstanceId)
	if errCode != 0 {
		return false
	}

	if len(records) > 0 {
		return true
	}
	return false
}

func (a *AppDCommon) generateTaskResponse(taskId string, appInstanceId string, result string, percent string,
	details string) (progress models.TaskProgress) {
	return models.TaskProgress{
		TaskId: taskId, AppInstanceId: appInstanceId, ConfigResult: result, ConfigPhase: percent, Details: details,
	}
}

func (a *AppDCommon) StageNewTask(appInstanceId string, taskId string,
	appDConfigInput *models.AppDConfig) (code workspace.ErrCode, msg string) {
	appDInStore := &models.AppDConfig{}
	// Table already exists for modify and delete request, hence reading db for non post scenarios
	if appDConfigInput.Operation != http.MethodPost {
		appDConfigEntry, errCode := backend.GetRecord(meputil.AppDConfigKeyPath + appInstanceId)
		if errCode != 0 {
			log.Errorf(nil, "App config (appId: %s) retrieval from data-store failed.", appInstanceId)
			return workspace.ErrCode(errCode), "get app config rule from data-store failed"
		}
		err := json.Unmarshal(appDConfigEntry, appDInStore)
		if err != nil {
			log.Errorf(err, "Failed to parse the appd config from data-store.")
			return meputil.OperateDataWithEtcdErr, "parsing app config rule from data-store failed"
		}
	}
	if appDConfigInput.Operation == http.MethodPut && appDConfigInput.AppName != appDInStore.AppName {
		log.Errorf(nil, "App-name miss-match.")
		return meputil.OperateDataWithEtcdErr, "app-name doesn't match"
	}

	var err error
	var appDConfigBytes []byte
	if appDConfigInput.Operation == http.MethodDelete {
		appDInStore.Operation = appDConfigInput.Operation
		appDConfigBytes, err = json.Marshal(appDInStore)

		// App name is required to build the url for data-plane
		// Required because delete doesn't have body and app name is in the body
		appDConfigInput.AppName = appDInStore.AppName
	} else {
		appDConfigBytes, err = json.Marshal(appDConfigInput)
	}
	if err != nil {
		log.Errorf(nil, "Can not marshal appDConfig info.")
		return meputil.ParseInfoErr, "can not marshal appDConfig info"
	}

	// Add to Jobs DB
	errCode := backend.PutRecord(meputil.AppDLCMJobsPath+appInstanceId, appDConfigBytes)
	if errCode != 0 {
		log.Errorf(nil, "App config (appId: %s) insertion on data-store failed.", appInstanceId)
		return workspace.ErrCode(errCode), DBFailure
	}

	errCode = backend.PutRecord(meputil.AppDLCMTasksPath+taskId, []byte(appInstanceId))
	if errCode != 0 {
		_ = backend.DeletePaths([]string{meputil.AppDLCMJobsPath + appInstanceId}, true)
		log.Errorf(nil, "App config (taskId: %s) insertion on data-store failed.", appInstanceId)
		return workspace.ErrCode(errCode), DBFailure
	}

	taskStatus := a.buildTaskStatus(appDConfigInput, appDInStore)
	if taskStatus.TrafficRuleStatusLst == nil && taskStatus.DNSRuleStatusLst == nil {
		_ = backend.DeletePaths([]string{meputil.AppDLCMJobsPath + appInstanceId, meputil.AppDLCMTasksPath + taskId},
			true)
		log.Errorf(nil, "No modification found.")
		return meputil.SubscriptionNotFound, "no modification data found in the input"
	}

	return a.putInDB(taskStatus, appInstanceId, taskId, appDConfigInput)
}

func (a *AppDCommon) putInDB(taskStatus *models.TaskStatus, appInstanceId string, taskId string,
	appDConfigInput *models.AppDConfig) (code workspace.ErrCode, msg string) {
	// Check any duplicate dns entry exists
	if a.isDuplicateDomainNameForCreateExists(appInstanceId, appDConfigInput, taskStatus) {
		_ = backend.DeletePaths([]string{meputil.AppDLCMJobsPath + appInstanceId, meputil.AppDLCMTasksPath + taskId},
			true)
		log.Errorf(nil, "Duplicate dns entry found in the request.")
		return meputil.DuplicateOperation, "duplicate dns entry"
	}

	statusBytes, err := json.Marshal(taskStatus)
	if err != nil {
		_ = backend.DeletePaths([]string{meputil.AppDLCMJobsPath + appInstanceId, meputil.AppDLCMTasksPath + taskId},
			true)
		log.Errorf(nil, "Can not marshal status info.")
		return meputil.ParseInfoErr, "can not marshal status info"
	}

	errCode := backend.PutRecord(meputil.AppDLCMTaskStatusPath+appInstanceId+"/"+taskId, statusBytes)
	if errCode != 0 {
		_ = backend.DeletePaths([]string{meputil.AppDLCMJobsPath + appInstanceId, meputil.AppDLCMTasksPath + taskId},
			true)
		log.Errorf(nil, "App config (taskId: %s) insertion on data-store failed.", appInstanceId)
		return workspace.ErrCode(errCode), DBFailure
	}

	return 0, ""
}

func (a *AppDCommon) buildTaskStatus(appDConfigInput *models.AppDConfig,
	appDInStore *models.AppDConfig) *models.TaskStatus {
	var taskStatus = models.TaskStatus{}
	taskStatus.Progress = 0

	if appDConfigInput.Operation == http.MethodPost {
		// create works with only the input data
		a.handleRuleCreateOrDelete(meputil.OperCreate, appDConfigInput, &taskStatus)
	} else if appDConfigInput.Operation == http.MethodDelete {
		// delete works with the in-store data only
		a.handleRuleCreateOrDelete(meputil.OperDelete, appDInStore, &taskStatus)
	} else if appDConfigInput.Operation == http.MethodPut {
		// modify works on both new and old data
		a.handleRuleUpdate(appDConfigInput, appDInStore, &taskStatus)
	}

	return &taskStatus
}

func (a *AppDCommon) fillDnsDomainNameMap(appInstanceId string, path string, dnsRuleMap *map[string]bool) {
	records, errCode := backend.GetRecords(path)
	if errCode == 0 && len(records) != 0 {
		for appId, record := range records {
			// This check is to exclude the current processing entry
			if appId == appInstanceId {
				continue
			}
			appDInStore := &models.AppDConfig{}
			if err := json.Unmarshal(record, appDInStore); err != nil {
				continue
			}
			for i := 0; i < len(appDInStore.AppDNSRule); i++ {
				(*dnsRuleMap)[appDInStore.AppDNSRule[i].DomainName] = true
			}
		}
	}
}

func (a *AppDCommon) isDuplicateDomainNameForCreateExists(appInstanceId string, appDConfigInput *models.AppDConfig,
	taskStatus *models.TaskStatus) bool {

	dnsInStoreDomainNameMap := make(map[string]bool)

	a.fillDnsDomainNameMap(appInstanceId, meputil.AppDConfigKeyPath, &dnsInStoreDomainNameMap)
	a.fillDnsDomainNameMap(appInstanceId, meputil.AppDLCMJobsPath, &dnsInStoreDomainNameMap)

	dnsInputRuleMap := make(map[string]*dataplane.DNSRule)
	dnsInputDomainNameMap := make(map[string]bool)
	for i, rule := range appDConfigInput.AppDNSRule {
		dnsInputRuleMap[rule.DNSRuleID] = &appDConfigInput.AppDNSRule[i]

		// Duplicate entry in the input request
		if _, found := dnsInputDomainNameMap[rule.DomainName]; found {
			return true
		}
		dnsInputDomainNameMap[rule.DomainName] = true
	}

	for _, ruleStatus := range taskStatus.DNSRuleStatusLst {
		if ruleStatus.Method == meputil.OperCreate {
			if _, found := dnsInStoreDomainNameMap[dnsInputRuleMap[ruleStatus.Id].DomainName]; found {
				return true
			}
		}
	}

	return false
}

func (a *AppDCommon) handleRuleCreateOrDelete(method meputil.OperType, appDConfig *models.AppDConfig,
	taskStatus *models.TaskStatus) {
	for _, dnsRule := range appDConfig.AppDNSRule {
		state := models.RuleStatus{
			Id:     dnsRule.DNSRuleID,
			State:  meputil.WaitMp2,
			Method: method,
		}
		taskStatus.DNSRuleStatusLst = append(taskStatus.DNSRuleStatusLst, state)
	}
	for _, trafficRule := range appDConfig.AppTrafficRule {
		state := models.RuleStatus{
			Id:     trafficRule.TrafficRuleID,
			State:  meputil.WaitMp2,
			Method: method,
		}
		taskStatus.TrafficRuleStatusLst = append(taskStatus.TrafficRuleStatusLst, state)
	}
}

type ruleData struct {
	isExistingRule bool
	rule           interface{}
}

// Update need to check with existing rule and make both create, modify and delete operations accordingly
func (a *AppDCommon) handleRuleUpdate(appDConfigInput, appDConfigOnStore *models.AppDConfig,
	taskStatus *models.TaskStatus) {

	// Handling the DNS first
	idDnsStoreMap := make(map[string]*ruleData)
	idDnsInputMap := make(map[string]*ruleData)

	for _, dnsRule := range appDConfigOnStore.AppDNSRule {
		idDnsStoreMap[dnsRule.DNSRuleID] = &ruleData{false, dnsRule}
	}

	for _, dnsRule := range appDConfigInput.AppDNSRule {
		if _, ok := idDnsStoreMap[dnsRule.DNSRuleID]; ok {
			idDnsInputMap[dnsRule.DNSRuleID] = &ruleData{true, dnsRule}
			// Update the other map also as existing
			idDnsStoreMap[dnsRule.DNSRuleID].isExistingRule = true
		} else {
			idDnsInputMap[dnsRule.DNSRuleID] = &ruleData{false, dnsRule}
		}
	}

	taskStatus.DNSRuleStatusLst = a.processRulesFromIDMap(idDnsInputMap, idDnsStoreMap)

	// Handling the Traffic rules
	idTrfStoreMap := make(map[string]*ruleData)
	idTrfInputMap := make(map[string]*ruleData)

	for _, trfRule := range appDConfigOnStore.AppTrafficRule {
		idTrfStoreMap[trfRule.TrafficRuleID] = &ruleData{false, trfRule}
	}

	for _, trfRule := range appDConfigInput.AppTrafficRule {
		if _, ok := idTrfStoreMap[trfRule.TrafficRuleID]; ok {
			idTrfInputMap[trfRule.TrafficRuleID] = &ruleData{true, trfRule}
			// Update the other map also as existing
			idTrfStoreMap[trfRule.TrafficRuleID].isExistingRule = true
		} else {
			idTrfInputMap[trfRule.TrafficRuleID] = &ruleData{false, trfRule}
		}
	}

	taskStatus.TrafficRuleStatusLst = a.processRulesFromIDMap(idTrfInputMap, idTrfStoreMap)
}

func (a *AppDCommon) processRulesFromIDMap(idInputMap, idStoreMap map[string]*ruleData) []models.RuleStatus {

	var ruleStatusList []models.RuleStatus

	// Handling Create and modify
	for id, ruleData := range idInputMap {
		// Existing rule is for modify
		if ruleData.isExistingRule {
			// Is same with old ignore
			if reflect.DeepEqual(ruleData.rule, idStoreMap[id].rule) {
				continue
			}
			ruleStatusList = append(ruleStatusList, models.RuleStatus{
				Id:     id,
				State:  meputil.WaitMp2,
				Method: meputil.OperModify,
			})
		} else {
			// New entries
			ruleStatusList = append(ruleStatusList, models.RuleStatus{
				Id:     id,
				State:  meputil.WaitMp2,
				Method: meputil.OperCreate,
			})
		}
	}

	// Handling delete
	for id, ruleData := range idStoreMap {
		// Existing rules are already handled in above case
		if ruleData.isExistingRule {
			continue
		}
		ruleStatusList = append(ruleStatusList, models.RuleStatus{
			Id:     id,
			State:  meputil.WaitMp2,
			Method: meputil.OperDelete,
		})
	}

	return ruleStatusList
}
