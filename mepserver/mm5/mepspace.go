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

// Package path implements new workspace for Mm5
package mm5

import (
	"net/http"
	"net/url"

	"github.com/apache/servicecomb-service-center/server/core/proto"
	"golang.org/x/net/context"

	"mepserver/common/arch/workspace"
)

type MepSpace struct {
	workspace.SpaceBase
	R *http.Request       `json:"r"`
	W http.ResponseWriter `json:"w"`

	Ctx           context.Context `json:"ctx"`
	RestBody      interface{}     `json:"restBody"`
	AppInstanceId string          `json:"appInstanceId"`
	DNSRuleId     string          `json:"dnsRuleId"`
	CapabilityId  string          `json:"capabilityId"`
	TaskId        string          `json:"taskId"`
	QueryParam    url.Values      `json:"queryParam"`
	CoreRequest   interface{}     `json:"coreRequest"`
	CoreRsp       interface{}     `json:"coreRsp"`
	HttPErrInf    *proto.Response `json:"httpErrInf"`
	HttPRsp       interface{}     `json:"httpRsp"`
}

// new a work space
func NewWorkSpace(w http.ResponseWriter, r *http.Request) *MepSpace {
	var plan = MepSpace{
		W: w,
		R: r,
	}

	plan.Init()
	return &plan
}
