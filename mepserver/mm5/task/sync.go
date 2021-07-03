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
	"mepserver/common/extif/backend"
	"mepserver/common/extif/dataplane"
	"mepserver/common/extif/dns"
	"mepserver/common/models"
	"mepserver/common/util"
	"net/http"
	"runtime/debug"
	"sync"

	"github.com/apache/servicecomb-service-center/pkg/log"
)

// Worker keeps the asynchronous task parameters
type Worker struct {
	waitWorkerFinish sync.WaitGroup
	dnsTypeConfig    string
	dataPlane        dataplane.DataPlane
	dnsAgent         dns.DNSAgent
}

const dataInconsistentError = "Failed to revert the data, this will lead to data inconsistency."
const ExistRuleError = "existing rule expected"

// InitializeWorker initialize worker instance
func (w *Worker) InitializeWorker(dataPlane dataplane.DataPlane, dnsAgent dns.DNSAgent, dnsType string) *Worker {
	w.dataPlane = dataPlane
	w.dnsAgent = dnsAgent
	w.dnsTypeConfig = dnsType
	return w
}

// StartNewTask start new task for sync
func (w *Worker) StartNewTask(appName, appInstanceId, taskId string) {
	log.Infof("New appd sync task created(app-name: %s, app-id: %s, task-id: %s).", appName, appInstanceId, taskId)
	w.waitWorkerFinish.Add(1)
	go w.ProcessAppDConfigSync(appName, appInstanceId, taskId)
	return
}

// ProcessAppDConfigSync handles appd config sync
func (w *Worker) ProcessAppDConfigSync(appName, appInstanceId, taskId string) {
	defer w.waitWorkerFinish.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Errorf(nil, "Sync process panic: %v.\n %s", r, string(debug.Stack()))
		}
	}()
	w.ProcessDataPlaneSync(appName, appInstanceId, taskId)
}

// ProcessDataPlaneSync Go Routine function to handle the sync of traffic and dns to the data-plane over mp2
func (w *Worker) ProcessDataPlaneSync(appName, appInstanceId, taskId string) {

	syncJob := newTask(appName, appInstanceId, taskId, w.dataPlane, w.dnsAgent, w.dnsTypeConfig)
	if syncJob == nil {
		log.Error("Failed to process the task, something went wrong.", nil)
		_ = backend.DeletePaths([]string{util.AppDLCMJobsPath + appInstanceId}, true)
		taskStatus := newStatusDB(appInstanceId, taskId)
		if taskStatus != nil {
			taskStatus.status.Progress = util.TaskProgressFailure
			taskStatus.setFailureReason("Unexpected error in processing.")
			_ = taskStatus.pushDB()
		}
		return
	}
	err := syncJob.handleDNSRules(util.ApplyFunc)
	if err != nil {
		log.Error("Failed to process the task in dns rules.", err)
		syncJob.statusDb.setFailureReason("Internal error(failed to configure dns rules).")
		err = syncJob.handleErrorOnProcessing()
		if err != nil {
			log.Error(dataInconsistentError, err)
		}
		return
	}
	err = syncJob.handleTrafficRules(util.ApplyFunc)
	if err != nil {
		log.Error("Failed to process the task in traffic rules.", err)
		syncJob.statusDb.setFailureReason("Internal error(failed to configure traffic rules).")
		err = syncJob.handleErrorOnProcessing()
		if err != nil {
			log.Error(dataInconsistentError, err)
		}
		return
	}
	err = syncJob.handleConfigDBWriteOnSuccess()
	if err != nil {
		log.Error("Failed to save appd config.", err)
		syncJob.statusDb.setFailureReason("Internal error(failed to write appdconfig db).")
		err = syncJob.handleErrorOnProcessing()
		if err != nil {
			log.Error(dataInconsistentError, err)
		}
		return
	}

	_ = syncJob.cleanProcessingCache()

}

type ruleOperation struct {
	apply     func(ruleId string, newRule interface{}, existingRule interface{}) error
	revert    func(ruleId string, newRule interface{}, existingRule interface{}) error
	nextState util.AppDRuleStatus
}

type task struct {
	appName         string
	appInstanceId   string
	taskId          string
	appDJobDb       *appDJobDB
	appDConfigDb    *appDConfigDB
	statusDb        *statusDB
	dataPlane       dataplane.DataPlane
	dnsAgent        dns.DNSAgent
	dnsStateMachine [][]*ruleOperation
	trfStateMachine [][]*ruleOperation
}

func newTask(appName, appInstanceId, taskId string, dataPlane dataplane.DataPlane, dnsAgent dns.DNSAgent,
	dnsType string) *task {
	jobConfig := newAppDJobDB(appInstanceId)
	if jobConfig == nil {
		return nil
	}
	// No  need to check the return value, as this will fail for create request. Only required in modify and delete
	appDConfig := newAppDConfigDB(appInstanceId)

	taskStatus := newStatusDB(appInstanceId, taskId)
	if taskStatus == nil {
		_ = jobConfig.deleteEntry()
		return nil
	}

	j := &task{
		appName:       appName,
		appInstanceId: appInstanceId,
		taskId:        taskId,
		appDJobDb:     jobConfig,
		appDConfigDb:  appDConfig,
		statusDb:      taskStatus,
		dataPlane:     dataPlane,
		dnsAgent:      dnsAgent}

	j.dnsStateMachine = [][]*ruleOperation{
		util.OperCreate: {
			util.WaitMp2:           &ruleOperation{j.addDNSOnMp2, j.deleteDNSOnMp2, util.WaitLocal},
			util.WaitLocal:         &ruleOperation{j.addDNSOnLocalDns, j.deleteDNSOnLocalDns, util.WaitConfigDBWrite},
			util.WaitConfigDBWrite: &ruleOperation{nil, nil, 0},
			// DB wil be handled separately at the end for atomicity
		},
		util.OperModify: {
			util.WaitMp2:           &ruleOperation{j.setDNSOnMp2, j.setDNSOnMp2, util.WaitLocal},
			util.WaitLocal:         &ruleOperation{j.setDNSOnLocalDns, j.setDNSOnLocalDns, util.WaitConfigDBWrite},
			util.WaitConfigDBWrite: &ruleOperation{nil, nil, 0}, // DB wil be handled separately at the end for atomicity
		},
		util.OperDelete: {
			util.WaitMp2:           &ruleOperation{j.deleteDNSOnMp2, j.addDNSOnMp2, util.WaitLocal},
			util.WaitLocal:         &ruleOperation{j.deleteDNSOnLocalDns, j.addDNSOnLocalDns, util.WaitConfigDBWrite},
			util.WaitConfigDBWrite: &ruleOperation{nil, nil, 0}, // DB wil be handled separately at the end for atomicity
		},
	}

	// Cleanup states as per mep-server config
	if dnsType == util.DnsAgentTypeLocal {
		for oper := util.OperCreate; oper <= util.OperDelete; oper++ {
			j.dnsStateMachine[oper][util.WaitMp2] = nil
		}
	} else if dnsType == util.DnsAgentTypeDataPlane {
		for oper := util.OperCreate; oper <= util.OperDelete; oper++ {
			j.dnsStateMachine[oper][util.WaitLocal] = nil
		}
	}

	j.trfStateMachine = [][]*ruleOperation{
		util.OperCreate: {
			util.WaitMp2:           &ruleOperation{j.addTrafficOnMp2, j.deleteTrafficOnMp2, util.WaitConfigDBWrite},
			util.WaitConfigDBWrite: &ruleOperation{nil, nil, 0},
			// DB wil be handled separately at the end for atomicity
		},
		util.OperModify: {
			util.WaitMp2:           &ruleOperation{j.setTrafficOnMp2, j.setTrafficOnMp2, util.WaitConfigDBWrite},
			util.WaitConfigDBWrite: &ruleOperation{nil, nil, 0}, // DB wil be handled separately at the end for atomicity
		},
		util.OperDelete: {
			util.WaitMp2:           &ruleOperation{j.deleteTrafficOnMp2, j.addTrafficOnMp2, util.WaitConfigDBWrite},
			util.WaitConfigDBWrite: &ruleOperation{nil, nil, 0}, // DB wil be handled separately at the end for atomicity
		},
	}

	return j
}

// Generate a map of traffic rules based on the function type
func (t *task) generateTrafficRuleMap(funcType util.FuncType) (map[string]*dataplane.TrafficRule, map[string]*dataplane.TrafficRule) {
	var trfNewRuleMap = make(map[string]*dataplane.TrafficRule)
	var trfOldRuleMap = make(map[string]*dataplane.TrafficRule)
	var primaryRuleList []dataplane.TrafficRule
	// Secondary is required for the delete rule scenarios
	var secondaryRuleList []dataplane.TrafficRule
	if funcType == util.ApplyFunc {
		primaryRuleList = t.appDJobDb.appDConfig.AppTrafficRule
		if t.appDConfigDb != nil {
			secondaryRuleList = t.appDConfigDb.appDConfig.AppTrafficRule
		}
	} else {
		if t.appDConfigDb != nil {
			primaryRuleList = t.appDConfigDb.appDConfig.AppTrafficRule
		}
		secondaryRuleList = t.appDJobDb.appDConfig.AppTrafficRule
	}

	if primaryRuleList != nil {
		for i, trRule := range primaryRuleList {
			trfNewRuleMap[trRule.TrafficRuleID] = &primaryRuleList[i]
		}
	}

	// Below for is for delete case. Reading from the stored appDConfig and filling.
	if secondaryRuleList != nil {
		for i, trRule := range secondaryRuleList {
			// Update only if not found
			if _, found := trfNewRuleMap[trRule.TrafficRuleID]; !found {
				trfNewRuleMap[trRule.TrafficRuleID] = &secondaryRuleList[i]
			}
			trfOldRuleMap[trRule.TrafficRuleID] = &secondaryRuleList[i]
		}
	}
	return trfNewRuleMap, trfOldRuleMap
}

// Generate a map of DNS rules based on the function type
func (t *task) generateDnsRuleMap(funcType util.FuncType) (map[string]*dataplane.DNSRule, map[string]*dataplane.DNSRule) {

	var dnsNewRuleMap = make(map[string]*dataplane.DNSRule)
	var dnsOldRuleMap = make(map[string]*dataplane.DNSRule)
	var primaryRuleList []dataplane.DNSRule
	// Secondary is required for the delete rule scenarios
	var secondaryRuleList []dataplane.DNSRule
	if funcType == util.ApplyFunc {
		primaryRuleList = t.appDJobDb.appDConfig.AppDNSRule
		if t.appDConfigDb != nil {
			secondaryRuleList = t.appDConfigDb.appDConfig.AppDNSRule
		}
	} else {
		if t.appDConfigDb != nil {
			primaryRuleList = t.appDConfigDb.appDConfig.AppDNSRule
		}
		secondaryRuleList = t.appDJobDb.appDConfig.AppDNSRule
	}

	for i, dnsRule := range primaryRuleList {
		dnsNewRuleMap[dnsRule.DNSRuleID] = &primaryRuleList[i]
	}

	// Below for is for delete case. Reading from the stored appDConfig and filling.
	for i, dnsRule := range secondaryRuleList {
		// Update only if not found
		if _, found := dnsNewRuleMap[dnsRule.DNSRuleID]; !found {
			dnsNewRuleMap[dnsRule.DNSRuleID] = &secondaryRuleList[i]
		}
		dnsOldRuleMap[dnsRule.DNSRuleID] = &secondaryRuleList[i]
	}

	return dnsNewRuleMap, dnsOldRuleMap
}

// Handle dns rule related configurations
func (t *task) handleTrafficRules(funcType util.FuncType) error {
	var err error
	var trfNewRuleMap, trfOldRuleMap = t.generateTrafficRuleMap(funcType)

	for _, trRuleStatus := range t.statusDb.status.TrafficRuleStatusLst {
		if funcType == util.RevertFunc {
			err = t.processTrfEntryRevert(trfNewRuleMap[trRuleStatus.Id], trfOldRuleMap[trRuleStatus.Id], trRuleStatus)
		} else {
			err = t.processTrfEntryApply(trfNewRuleMap[trRuleStatus.Id], trfOldRuleMap[trRuleStatus.Id], trRuleStatus)
		}
		if err != nil {
			t.statusDb.setFailureReason("Internal error(invalid function type).")
			return err
		}
	}

	return nil
}

func (t *task) processTrfEntryApply(trfNewRule *dataplane.TrafficRule, trfOldRule *dataplane.TrafficRule,
	ruleStatus models.RuleStatus) error {

	for state := util.WaitMp2; state < util.WaitConfigDBWrite; state++ {
		operation := t.trfStateMachine[ruleStatus.Method][state]
		if state < ruleStatus.State {
			continue
		}
		var err error
		if operation != nil && operation.apply != nil {
			log.Debugf("Traffic apply(method:%v, state: %v).", ruleStatus.Method, state)
			err = operation.apply(ruleStatus.Id, trfNewRule, trfOldRule)
		}
		if err != nil {
			log.Errorf(err, "Traffic apply(method:%v, state: %v) failed in configuration.",
				ruleStatus.Method, state)
			t.statusDb.setFailureReason("Failed in configuring traffic rule on remote data-plane.")
			return err
		}
		// Set next state on db
		err = t.statusDb.setStateAndProgress(util.RuleTypeTraffic, ruleStatus.Id, state+1)
		if err != nil {
			log.Errorf(err, "Traffic apply(method:%v, state: %v) failed in setting status DB.",
				ruleStatus.Method, state)
			t.statusDb.setFailureReason("Failed in setting status for traffic rule.")
			return err
		}
	}

	return nil
}

func (t *task) processTrfEntryRevert(trfNewRule *dataplane.TrafficRule, trfOldRule *dataplane.TrafficRule,
	ruleStatus models.RuleStatus) error {

	for state := util.WaitConfigDBWrite - 1; state >= util.WaitMp2; state-- {
		operation := t.trfStateMachine[ruleStatus.Method][state]
		// On revert the current state is the failed one, so no need to process the current state also
		if state >= ruleStatus.State {
			continue
		}
		var err error
		if operation != nil && operation.revert != nil {
			log.Debugf("Traffic revert(method:%v, state: %v).", ruleStatus.Method, state)
			err = operation.revert(ruleStatus.Id, trfNewRule, trfOldRule)
		}
		if err != nil {
			log.Errorf(err, "Traffic revert(method:%v, state: %v) failed in configuration.",
				ruleStatus.Method, state)
			t.statusDb.setFailureReason("Failed in reverting traffic rule on remote data-plane.")
			return err
		}

		// No need to set the state if it already reached the first one
		if state == util.WaitMp2 {
			return nil
		}
		// Set next state on db
		err = t.statusDb.setStateAndProgress(util.RuleTypeTraffic, ruleStatus.Id, state-1)
		if err != nil {
			log.Errorf(err, "Traffic revert(method:%v, state: %v) failed in setting status DB.",
				ruleStatus.Method, state)
			t.statusDb.setFailureReason("Failed in setting status for traffic rule revert.")
			return err
		}
	}

	return nil
}

// Handle dns rule related configurations
func (t *task) handleDNSRules(funcType util.FuncType) error {
	var err error
	var dnsNewRuleMap, dnsOldRuleMap = t.generateDnsRuleMap(funcType)

	for _, dnsRuleStatus := range t.statusDb.status.DNSRuleStatusLst {
		if funcType == util.RevertFunc {
			err = t.processDNSEntryRevert(dnsNewRuleMap[dnsRuleStatus.Id], dnsOldRuleMap[dnsRuleStatus.Id],
				dnsRuleStatus)
		} else {
			err = t.processDNSEntryApply(dnsNewRuleMap[dnsRuleStatus.Id], dnsOldRuleMap[dnsRuleStatus.Id], dnsRuleStatus)
		}
		if err != nil {
			t.statusDb.setFailureReason("Internal error(invalid function type).")
			return err
		}
	}

	return nil
}

func (t *task) processDNSEntryApply(dnsNewRule *dataplane.DNSRule, dnsOldRule *dataplane.DNSRule,
	ruleStatus models.RuleStatus) error {

	for state := util.WaitMp2; state < util.WaitConfigDBWrite; state++ {
		operation := t.dnsStateMachine[ruleStatus.Method][state]
		if state < ruleStatus.State {
			continue
		}
		var err error
		if operation != nil && operation.apply != nil {
			log.Debugf("DNS apply(method:%v, state: %v).", ruleStatus.Method, state)
			err = operation.apply(ruleStatus.Id, dnsNewRule, dnsOldRule)
		}
		if err != nil {
			log.Errorf(err, "DNS apply(method:%v, state: %v) failed in configuration.", ruleStatus.Method, state)
			t.statusDb.setFailureReason("Failed in configuring dns rule on remote dns-server/data-plane.")
			return err
		}
		// Set next state on db
		err = t.statusDb.setStateAndProgress(util.RuleTypeDns, ruleStatus.Id, state+1)
		if err != nil {
			log.Errorf(err, "DNS apply(method:%v, state: %v) failed in setting dns status.",
				ruleStatus.Method, state)
			t.statusDb.setFailureReason("Failed in setting status for dns rule.")
			return err
		}
	}

	return nil
}

func (t *task) processDNSEntryRevert(dnsNewRule *dataplane.DNSRule, dnsOldRule *dataplane.DNSRule,
	ruleStatus models.RuleStatus) error {

	for state := util.WaitConfigDBWrite - 1; state >= util.WaitMp2; state-- {
		operation := t.dnsStateMachine[ruleStatus.Method][state]
		// On revert the current state is the failed one, so no need to process the current state also
		if state >= ruleStatus.State {
			continue
		}
		var err error
		if operation != nil && operation.revert != nil {
			log.Debugf("DNS revert(method:%v, state: %v).", ruleStatus.Method, state)
			err = operation.revert(ruleStatus.Id, dnsNewRule, dnsOldRule)
		}
		if err != nil {
			log.Errorf(err, "DNS revert(method:%v, state: %v) failed in configuration.", ruleStatus.Method, state)
			t.statusDb.setFailureReason("Failed in reverting dns rule on remote dns-server/data-plane.")
			return err
		}

		// No need to set the state if it already reached the first one
		if state == util.WaitMp2 {
			return nil
		}
		// Set next state on db
		err = t.statusDb.setStateAndProgress(util.RuleTypeDns, ruleStatus.Id, state-1)
		if err != nil {
			log.Errorf(err, "DNS revert(method:%v, state: %v) failed in setting dns revert status.",
				ruleStatus.Method, state)
			t.statusDb.setFailureReason("Failed in setting status for dns rule revert.")
			return err
		}
	}

	return nil
}

func (t *task) addDNSOnMp2(ruleId string, newRule interface{}, existingRule interface{}) error {
	dnsRule := newRule.(*dataplane.DNSRule)
	if dnsRule.State == "" {
		dnsRule.State = util.ActiveState
	}
	if dnsRule.State != util.ActiveState {
		// Send only if the state is active
		return nil
	}
	appInfo := dataplane.ApplicationInfo{
		Id:   t.appInstanceId,
		Name: t.appName,
	}
	return t.dataPlane.AddDNSRule(appInfo, ruleId, dnsRule.DomainName, dnsRule.IPAddressType,
		dnsRule.IPAddress, dnsRule.TTL)
}

func (t *task) setDNSOnMp2(ruleId string, newRule interface{}, existingRule interface{}) error {
	dnsRule := newRule.(*dataplane.DNSRule)
	if existingRule == nil {
		return fmt.Errorf(ExistRuleError)
	}
	appInfo := dataplane.ApplicationInfo{
		Id:   t.appInstanceId,
		Name: t.appName,
	}

	dnsExistingRule := existingRule.(*dataplane.DNSRule)

	if dnsRule.State == "" {
		dnsRule.State = util.ActiveState
	}
	if dnsExistingRule.State == "" {
		dnsExistingRule.State = util.ActiveState
	}

	if dnsExistingRule.State == util.InactiveState && dnsRule.State == util.ActiveState {
		// Add rule
		return t.dataPlane.AddDNSRule(appInfo, ruleId, dnsRule.DomainName, dnsRule.IPAddressType,
			dnsRule.IPAddress, dnsRule.TTL)
	} else if dnsExistingRule.State == util.ActiveState && dnsRule.State == util.InactiveState {
		// Delete rule
		return t.dataPlane.DeleteDNSRule(appInfo, ruleId)
	}

	return t.dataPlane.SetDNSRule(appInfo, ruleId, dnsRule.DomainName, dnsRule.IPAddressType,
		dnsRule.IPAddress, dnsRule.TTL)
}

func (t *task) deleteDNSOnMp2(ruleId string, newRule interface{}, existingRule interface{}) error {
	if existingRule == nil {
		return fmt.Errorf(ExistRuleError)
	}
	dnsExistingRule := existingRule.(*dataplane.DNSRule)
	if dnsExistingRule.State == "" {
		dnsExistingRule.State = util.ActiveState
	}
	if dnsExistingRule.State == util.InactiveState {
		// No need to delete as the state was already inactive and not available in the Mp2
		return nil
	}

	appInfo := dataplane.ApplicationInfo{
		Id:   t.appInstanceId,
		Name: t.appName,
	}
	return t.dataPlane.DeleteDNSRule(appInfo, ruleId)
}

func (t *task) addDNSOnLocalDns(ruleId string, newRule interface{}, existingRule interface{}) error {
	dnsRule := newRule.(*dataplane.DNSRule)
	if dnsRule.State == "" {
		dnsRule.State = util.ActiveState
	}
	if dnsRule.State != util.ActiveState {
		// Send only if the state is active
		return nil
	}

	rrType := util.RRTypeA
	if dnsRule.IPAddressType == util.IPv6Type {
		rrType = util.RRTypeAAAA
	}
	err := t.dnsAgent.SetResourceRecordTypeA(
		dnsRule.DomainName, rrType, util.RRClassIN, []string{dnsRule.IPAddress},
		dnsRule.TTL)
	if err != nil {
		return err
	}

	return err
}

func (t *task) setDNSOnLocalDns(ruleId string, newRule interface{}, existingRule interface{}) error {
	return t.addDNSOnLocalDns(ruleId, newRule, existingRule)
}

func (t *task) deleteDNSOnLocalDns(ruleId string, newRule interface{}, existingRule interface{}) error {
	if existingRule == nil {
		return fmt.Errorf(ExistRuleError)
	}
	dnsExistingRule := existingRule.(*dataplane.DNSRule)
	if dnsExistingRule.State == "" {
		dnsExistingRule.State = util.ActiveState
	}
	if dnsExistingRule.State == util.InactiveState {
		// No need to delete as the state was already inactive and not available in the remote server
		return nil
	}

	dnsRule := newRule.(*dataplane.DNSRule)
	rrType := util.RRTypeA
	if dnsRule.IPAddressType == util.IPv6Type {
		rrType = util.RRTypeAAAA
	}
	err := t.dnsAgent.DeleteResourceRecordTypeA(dnsRule.DomainName, rrType)
	if err != nil {
		return err
	}

	return err
}

func (t *task) addTrafficOnMp2(ruleId string, newRule interface{}, existingRule interface{}) error {
	trRule := newRule.(*dataplane.TrafficRule)
	if trRule.State == "" {
		trRule.State = util.ActiveState
	}
	if trRule.State != util.ActiveState {
		// Send only if the state is active
		return nil
	}
	appInfo := dataplane.ApplicationInfo{
		Id:   t.appInstanceId,
		Name: t.appName,
	}
	return t.dataPlane.AddTrafficRule(appInfo, ruleId, trRule.FilterType, trRule.Action,
		trRule.Priority, trRule.TrafficFilter)
}

func (t *task) setTrafficOnMp2(ruleId string, newRule interface{}, existingRule interface{}) error {
	trRule := newRule.(*dataplane.TrafficRule)

	if existingRule == nil {
		return fmt.Errorf(ExistRuleError)
	}
	trExistingRule := existingRule.(*dataplane.TrafficRule)

	appInfo := dataplane.ApplicationInfo{
		Id:   t.appInstanceId,
		Name: t.appName,
	}

	if trRule.State == "" {
		trRule.State = util.ActiveState
	}
	if trExistingRule.State == "" {
		trExistingRule.State = util.ActiveState
	}
	if trExistingRule.State == util.InactiveState && trRule.State == util.ActiveState {
		// Add rule
		return t.dataPlane.AddTrafficRule(appInfo, ruleId, trRule.FilterType, trRule.Action,
			trRule.Priority, trRule.TrafficFilter)
	} else if trExistingRule.State == util.ActiveState && trRule.State == util.InactiveState {
		// Delete rule
		return t.dataPlane.DeleteTrafficRule(appInfo, ruleId)
	}

	return t.dataPlane.SetTrafficRule(appInfo, ruleId, trRule.FilterType, trRule.Action,
		trRule.Priority, trRule.TrafficFilter)
}

func (t *task) deleteTrafficOnMp2(ruleId string, newRule interface{}, existingRule interface{}) error {
	if existingRule == nil {
		return fmt.Errorf(ExistRuleError)
	}
	trExistingRule := existingRule.(*dataplane.TrafficRule)
	if trExistingRule.State == "" {
		trExistingRule.State = util.ActiveState
	}
	if trExistingRule.State == util.InactiveState {
		// No need to delete as the state was already inactive and not available in the Mp2
		return nil
	}
	appInfo := dataplane.ApplicationInfo{
		Id:   t.appInstanceId,
		Name: t.appName,
	}
	return t.dataPlane.DeleteTrafficRule(appInfo, ruleId)
}

// Handle dns rule related configurations
func (t *task) handleConfigDBWriteOnSuccess() error {

	for _, dnsRuleStatus := range t.statusDb.status.DNSRuleStatusLst {
		if dnsRuleStatus.State != util.WaitConfigDBWrite {
			log.Errorf(nil, "Invalid state(%v) for dns rule(%s).", dnsRuleStatus.State, dnsRuleStatus.Id)
			t.statusDb.setFailureReason("Internal error(invalid dns rule state).")
			return fmt.Errorf("invalid state for dns rule(%s)", dnsRuleStatus.Id)
		}
	}
	for _, trfRuleStatus := range t.statusDb.status.TrafficRuleStatusLst {
		if trfRuleStatus.State != util.WaitConfigDBWrite {
			log.Errorf(nil, "Invalid state(%v) for traffic rule(%s).", trfRuleStatus.State, trfRuleStatus.Id)
			t.statusDb.setFailureReason("Internal error(invalid traffic rule state).")
			return fmt.Errorf("invalid state for traffic rule(%s)", trfRuleStatus.Id)
		}
	}

	operation := t.appDJobDb.appDConfig.Operation

	// Cleaning the operation field to avoid it in save
	t.appDJobDb.appDConfig.Operation = ""

	appDConfigBytes, err := json.Marshal(t.appDJobDb.appDConfig)
	if err != nil {
		log.Errorf(nil, "Can not marshal appd config info.")
		return err
	}

	var errCode int
	if operation == http.MethodPost || operation == http.MethodPut {
		errCode = backend.PutRecord(util.AppDConfigKeyPath+t.appInstanceId, appDConfigBytes)
	} else if operation == http.MethodDelete {
		errCode = backend.DeleteRecord(util.AppDConfigKeyPath + t.appInstanceId)
	}

	if errCode != 0 {
		log.Error("AppD config DB write error.", err)
		t.statusDb.setFailureReason("Internal error(failed to write appdconfig db).")
		return err
	}

	return nil
}

// Handle any error cases during the process
func (t *task) handleErrorOnProcessing() error {
	err := t.handleDNSRules(util.RevertFunc)
	if err != nil {
		log.Error("Failed to revert dns rules.", err)
		return err
	}

	err = t.handleTrafficRules(util.RevertFunc)
	if err != nil {
		log.Error("Failed to revert dns rules.", err)
		return err
	}

	t.statusDb.status.Progress = util.TaskProgressFailure
	err = t.statusDb.pushDB()
	if err != nil {
		log.Errorf(nil, "Couldn't update progress failure status, this will lead to data inconsistency.")
		return err
	}

	return t.cleanProcessingCache()
}

// Handle any error cases during the process
func (t *task) cleanProcessingCache() error {
	if errCode := backend.DeletePaths([]string{util.AppDLCMJobsPath + t.appInstanceId}, false); errCode != 0 {
		return fmt.Errorf("delete paths returned error(%d)", errCode)
	}
	return nil
}
