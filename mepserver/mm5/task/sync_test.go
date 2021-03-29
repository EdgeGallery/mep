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
	"github.com/agiledragon/gomonkey"
	uuid "github.com/satori/go.uuid"
	"math/rand"
	"mepserver/common/config"
	"mepserver/common/extif/backend"
	"mepserver/common/extif/dataplane"
	"mepserver/common/extif/dataplane/none"
	"mepserver/common/extif/dns"
	"mepserver/common/models"
	"mepserver/common/util"
	"testing"
)

const panicFormatString = "Panic: %v"
const defaultAppInstanceId = "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"
const ruleId = "7d71e54e-81f3-47bb-a2fc-b565a326d794"
const maxIPVal = 255
const ipAddFormatter = "%d.%d.%d.%d"

var exampleIPAddress = fmt.Sprintf(ipAddFormatter, rand.Intn(maxIPVal), rand.Intn(maxIPVal), rand.Intn(maxIPVal),
	rand.Intn(maxIPVal))

func TestProcessDataPlaneSync(t *testing.T) {

	patch1 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		dnsRule := dataplane.DNSRule{DNSRuleID: ruleId, IPAddress: exampleIPAddress, State: "ACTIVE"}
		var filters []dataplane.DNSRule
		filters = append(filters, dnsRule)
		entry := &models.AppDConfig{AppName: "AppName", AppDNSRule: filters}
		outBytesForDns, _ := json.Marshal(&entry)
		return outBytesForDns, 0
	})
	patch2 := gomonkey.ApplyFunc(backend.PutRecord, func(path string, value []byte) int {
		return 0
	})
	patch3 := gomonkey.ApplyFunc(backend.DeletePaths, func(value []string, continueOnFailure bool) int {
		return 0
	})
	patch4 := gomonkey.ApplyFunc(newStatusDB, func(appId, taskId string) *statusDB {
		ruleStatus := models.RuleStatus{Id: ruleId, State: 0, Method: 0}
		var ruleList []models.RuleStatus
		ruleList = append(ruleList, ruleStatus)

		taskStatus := &models.TaskStatus{Progress: 1, DNSRuleStatusLst: ruleList}
		statusDB := &statusDB{appInstanceId: defaultAppInstanceId, status: taskStatus}
		return statusDB
	})
	defer patch1.Reset()
	defer patch2.Reset()
	defer patch3.Reset()
	defer patch4.Reset()
	noneDataPlane := &none.NoneDataPlane{}
	dnsRules := dns.NewRestDNSAgent(&config.MepServerConfig{})
	worker := Worker{dataPlane: noneDataPlane, dnsAgent: dnsRules}
	worker.waitWorkerFinish.Add(1)
	taskId := uuid.NewV4().String()
	worker.ProcessDataPlane("AppName", defaultAppInstanceId, taskId)
}

func TestProcessDataPlaneSyncForError(t *testing.T) {

	patch1 := gomonkey.ApplyFunc(newTask, func(appName, appInstanceId string, taskId string,
		dataPlane dataplane.DataPlane, dnsAgent dns.DNSAgent, dnsType string) *task {
		return nil
	})
	patch2 := gomonkey.ApplyFunc(backend.DeletePaths, func(paths []string, continueOnFailure bool) int {
		return 0
	})
	patch3 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		ruleStatus := models.RuleStatus{Id: ruleId, State: 0, Method: 0}
		var ruleList []models.RuleStatus
		ruleList = append(ruleList, ruleStatus)
		entry := &models.TaskStatus{Progress: 1, DNSRuleStatusLst: ruleList, TrafficRuleStatusLst: ruleList}
		outBytesForDns, _ := json.Marshal(&entry)
		return outBytesForDns, 0
	})
	patch4 := gomonkey.ApplyFunc(backend.PutRecord, func(path string, value []byte) int {
		return 0
	})
	defer patch1.Reset()
	defer patch2.Reset()
	defer patch3.Reset()
	defer patch4.Reset()
	worker := Worker{}
	worker.waitWorkerFinish.Add(1)
	taskId := uuid.NewV4().String()
	worker.ProcessDataPlane("AppName", defaultAppInstanceId, taskId)

}

func TestHandleTrafficRules(t *testing.T) {

	ruleStatus := models.RuleStatus{Id: ruleId, State: 0, Method: 0}
	var ruleList []models.RuleStatus
	ruleList = append(ruleList, ruleStatus)
	trafficRule := dataplane.TrafficRule{TrafficRuleID: ruleId, FilterType: "FLOW", State: "ACTIVE"}
	var filters []dataplane.TrafficRule
	filters = append(filters, trafficRule)
	j := &task{appInstanceId: defaultAppInstanceId, taskId: ruleId, appDJobDb: &appDJobDB{appInstanceId: defaultAppInstanceId,
		appDConfig: &models.AppDConfig{AppName: "AppName", AppTrafficRule: filters}},
		statusDb: &statusDB{appInstanceId: defaultAppInstanceId,
			status: &models.TaskStatus{Progress: 1, DNSRuleStatusLst: ruleList, TrafficRuleStatusLst: ruleList}},
		dataPlane: &none.NoneDataPlane{}, dnsAgent: dns.NewRestDNSAgent(&config.MepServerConfig{})}

	j.trfStateMachine = [][]*ruleOperation{
		util.OperCreate: {
			util.WaitMp2:           &ruleOperation{j.addTrafficOnMp2, j.deleteTrafficOnMp2, util.WaitConfigDBWrite},
			util.WaitConfigDBWrite: &ruleOperation{nil, nil, 0},
		},
		util.OperModify: {
			util.WaitMp2:           &ruleOperation{j.setTrafficOnMp2, j.setTrafficOnMp2, util.WaitConfigDBWrite},
			util.WaitConfigDBWrite: &ruleOperation{nil, nil, 0},
		},
		util.OperDelete: {
			util.WaitMp2:           &ruleOperation{j.deleteTrafficOnMp2, j.addTrafficOnMp2, util.WaitConfigDBWrite},
			util.WaitConfigDBWrite: &ruleOperation{nil, nil, 0},
		},
	}

	patch1 := gomonkey.ApplyFunc(backend.PutRecord, func(path string, value []byte) int {
		return 0
	})
	defer patch1.Reset()
	j.handleTrafficRules(0)

}

func TestHandleDNSRules(t *testing.T) {

	ruleStatus := models.RuleStatus{Id: ruleId, State: 1, Method: 1}
	var ruleList []models.RuleStatus
	ruleList = append(ruleList, ruleStatus)
	dnsRule := dataplane.DNSRule{DNSRuleID: ruleId, IPAddressType: "IP_V6", IPAddress: exampleIPAddress, State: "ACTIVE"}
	var filters []dataplane.DNSRule
	filters = append(filters, dnsRule)
	j := &task{appInstanceId: defaultAppInstanceId, taskId: ruleId, appDJobDb: &appDJobDB{appInstanceId: defaultAppInstanceId,
		appDConfig: &models.AppDConfig{AppName: "AppName", AppDNSRule: filters}},
		statusDb: &statusDB{appInstanceId: defaultAppInstanceId,
			status: &models.TaskStatus{Progress: 1, DNSRuleStatusLst: ruleList, TrafficRuleStatusLst: ruleList}},
		dataPlane: &none.NoneDataPlane{}, dnsAgent: dns.NewRestDNSAgent(&config.MepServerConfig{})}

	j.dnsStateMachine = [][]*ruleOperation{
		util.OperModify: {
			util.WaitMp2:           &ruleOperation{j.setDNSOnMp2, j.setDNSOnMp2, util.WaitLocal},
			util.WaitLocal:         &ruleOperation{j.setDNSOnLocalDns, j.setDNSOnLocalDns, util.WaitConfigDBWrite},
			util.WaitConfigDBWrite: &ruleOperation{nil, nil, 0}, // DB wil be handled separately at the end for atomicity
		},
	}
	patch1 := gomonkey.ApplyFunc(j.setDNSOnLocalDns, func(ruleId string, rule interface{}, rule2 interface{}) error {
		return nil
	})

	patch2 := gomonkey.ApplyFunc(backend.PutRecord, func(path string, value []byte) int {
		return 0
	})
	defer patch1.Reset()
	defer patch2.Reset()
	j.handleDNSRules(0)
}

func TestHandleDNSRulesForDelete(t *testing.T) {

	ruleStatus := models.RuleStatus{Id: ruleId, State: 3, Method: 0}
	var ruleList []models.RuleStatus
	ruleList = append(ruleList, ruleStatus)
	ruleList = append(ruleList, ruleStatus)
	dnsRule := dataplane.DNSRule{DNSRuleID: ruleId, IPAddressType: "IP_V6", IPAddress: exampleIPAddress, State: "ACTIVE"}
	var filters []dataplane.DNSRule
	filters = append(filters, dnsRule)
	j := &task{appInstanceId: defaultAppInstanceId, taskId: ruleId, appDJobDb: &appDJobDB{appInstanceId: defaultAppInstanceId,
		appDConfig: &models.AppDConfig{AppName: "AppName", AppDNSRule: filters}},
		statusDb: &statusDB{appInstanceId: defaultAppInstanceId,
			status: &models.TaskStatus{Progress: 1, DNSRuleStatusLst: ruleList, TrafficRuleStatusLst: ruleList}},
		dataPlane: &none.NoneDataPlane{}, dnsAgent: dns.NewRestDNSAgent(&config.MepServerConfig{})}

	j.dnsStateMachine = [][]*ruleOperation{
		util.OperCreate: {
			util.WaitLocal:         &ruleOperation{j.addDNSOnLocalDns, j.deleteDNSOnLocalDns, util.WaitConfigDBWrite},
			util.WaitConfigDBWrite: &ruleOperation{nil, nil, 0},
		},
	}

	j.handleDNSRules(1)
}

func TestHandleConfigDBWriteOnSuccess(t *testing.T) {

	ruleStatus := models.RuleStatus{Id: ruleId, State: 2, Method: 0}
	var ruleList []models.RuleStatus
	ruleList = append(ruleList, ruleStatus)

	j := &task{appInstanceId: defaultAppInstanceId, taskId: ruleId, appDJobDb: &appDJobDB{appInstanceId: defaultAppInstanceId,
		appDConfig: &models.AppDConfig{AppName: "AppName"}},
		statusDb: &statusDB{appInstanceId: defaultAppInstanceId,
			status: &models.TaskStatus{Progress: 1, DNSRuleStatusLst: ruleList, TrafficRuleStatusLst: ruleList}},
		dataPlane: &none.NoneDataPlane{}, dnsAgent: dns.NewRestDNSAgent(&config.MepServerConfig{})}

	patch1 := gomonkey.ApplyFunc(backend.PutRecord, func(path string, value []byte) int {
		return 0
	})
	defer patch1.Reset()
	j.handleConfigDBWriteOnSuccess()

}

func TestNewTask(t *testing.T) {

	appJobDB := &appDJobDB{appInstanceId: defaultAppInstanceId, appDConfig: &models.AppDConfig{AppName: "AppName"}}
	patch1 := gomonkey.ApplyFunc(newAppDJobDB, func(path string) *appDJobDB {
		return appJobDB
	})

	patch2 := gomonkey.ApplyFunc(newStatusDB, func(appInstanceId, taskId string) *statusDB {
		return nil
	})
	patch3 := gomonkey.ApplyFunc(newAppDConfigDB, func(appInstanceId string) *appDConfigDB {
		return nil
	})
	patch4 := gomonkey.ApplyFunc(backend.DeleteRecord, func(path string) int {
		return 0
	})
	defer patch1.Reset()
	defer patch2.Reset()
	defer patch3.Reset()
	defer patch4.Reset()

	noneDataPlane := &none.NoneDataPlane{}
	dnsRules := dns.NewRestDNSAgent(&config.MepServerConfig{})
	worker := Worker{dataPlane: noneDataPlane, dnsAgent: dnsRules}
	newTask("AppName", defaultAppInstanceId, ruleId, worker.dataPlane, worker.dnsAgent, worker.dnsTypeConfig)

}

func TestSetDNSOnLocalDns(t *testing.T) {

	dnsRule := dataplane.DNSRule{DNSRuleID: ruleId, IPAddressType: "IP_V6", IPAddress: exampleIPAddress, State: "ACTIVE"}

	j := &task{appInstanceId: defaultAppInstanceId, taskId: ruleId, dnsAgent: dns.NewRestDNSAgent(&config.MepServerConfig{})}
	patch1 := gomonkey.ApplyFunc(dns.NewRestDNSAgent(&config.MepServerConfig{}).SetResourceRecordTypeA, func(host, rrtype, class string, pointTo []string, ttl uint32) error {
		return nil
	})
	defer patch1.Reset()

	j.setDNSOnLocalDns(dnsRule.DNSRuleID, &dnsRule, &dnsRule)
}

func TestProcessTrfEntryRevert(t *testing.T) {
	ruleStatus := models.RuleStatus{Id: ruleId, State: 2, Method: 0}
	var ruleList []models.RuleStatus
	ruleList = append(ruleList, ruleStatus)
	trafficRule := dataplane.TrafficRule{TrafficRuleID: ruleId, FilterType: "FLOW", State: "ACTIVE"}
	var filters []dataplane.TrafficRule
	filters = append(filters, trafficRule)
	j := &task{appInstanceId: defaultAppInstanceId, taskId: ruleId, appDJobDb: &appDJobDB{appInstanceId: defaultAppInstanceId,
		appDConfig: &models.AppDConfig{AppName: "AppName", AppTrafficRule: filters}},
		statusDb: &statusDB{appInstanceId: defaultAppInstanceId,
			status: &models.TaskStatus{Progress: 1, DNSRuleStatusLst: ruleList, TrafficRuleStatusLst: ruleList}},
		dataPlane: &none.NoneDataPlane{}, dnsAgent: dns.NewRestDNSAgent(&config.MepServerConfig{})}

	j.trfStateMachine = [][]*ruleOperation{
		util.OperCreate: {
			util.WaitMp2:           &ruleOperation{j.addTrafficOnMp2, j.deleteTrafficOnMp2, util.WaitConfigDBWrite},
			util.WaitConfigDBWrite: &ruleOperation{nil, nil, 0},
		},
		util.OperModify: {
			util.WaitMp2:           &ruleOperation{j.setTrafficOnMp2, j.setTrafficOnMp2, util.WaitConfigDBWrite},
			util.WaitConfigDBWrite: &ruleOperation{nil, nil, 0},
		},
		util.OperDelete: {
			util.WaitMp2:           &ruleOperation{j.deleteTrafficOnMp2, j.addTrafficOnMp2, util.WaitConfigDBWrite},
			util.WaitConfigDBWrite: &ruleOperation{nil, nil, 0},
		},
	}
	patch1 := gomonkey.ApplyFunc(j.deleteTrafficOnMp2, func(ruleId string, rule interface{}, rule2 interface{}) error {
		return nil
	})
	patch2 := gomonkey.ApplyFunc(backend.PutRecord, func(path string, value []byte) int {
		return 0
	})
	defer patch1.Reset()
	defer patch2.Reset()
	j.processTrfEntryRevert(&trafficRule, &trafficRule, ruleStatus)
}
