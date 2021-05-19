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

// Package models implements mep server object models
package models

import (
	"mepserver/common/extif/dataplane"
	meputil "mepserver/common/util"
)

// AppDConfig holds the application configurations such as traffic and dns rules.
type AppDConfig struct {
	AppTrafficRule []dataplane.TrafficRule `json:"appTrafficRule" validate:"dive,max=16"`
	AppDNSRule     []dataplane.DNSRule     `json:"appDNSRule" validate:"dive,max=32"`
	AppSupportMp1  bool                    `json:"appSupportMp1"`
	AppName        string                  `json:"appName" validate:"required,min=1,max=63"`
	// Operation specifies the type of the request
	Operation string `json:"operation,omitempty"` // For local use in the DB only
}

// TaskStatus hold the status of asynchronous sync task for app configuration
type TaskStatus struct {
	Progress             int          `json:"progress"`
	TrafficRuleStatusLst []RuleStatus `json:"trafficRuleStatusList"`
	DNSRuleStatusLst     []RuleStatus `json:"dnsRuleStatusList"`
	Details              string       `json:"details" validate:"omitempty"`
}

// RuleStatus holds status of either traffic or dns rules on sync from eg to data-plane
type RuleStatus struct {
	Id     string                 `json:"id"`
	State  meputil.AppDRuleStatus `json:"state"`  //One of INIT, MP2_OK, LOCAL_OK, DB_OK
	Method meputil.OperType       `json:"method"` // Outgoing request method
}

// TaskProgress response model
type TaskProgress struct {
	TaskId        string `json:"taskId"`
	AppInstanceId string `json:"appInstanceId"`
	ConfigResult  string `json:"configResult"`
	ConfigPhase   string `json:"configPhase"`
	Details       string `json:"Detailed"`
}

//Use ProblemDetails struct for Returning task fail immediate response
/* type TaskFail struct {
	Type     string   `json:"type"`
	Title    string   `json:"title"`
	Status   int      `json:"status"`
	Detail   string   `json:"detail"`
}*/
