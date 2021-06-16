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

// Package mp1 implements rest api route controller
package mp1

import (
	"net/http"
	"net/url"

	"github.com/apache/servicecomb-service-center/server/core/proto"
	"golang.org/x/net/context"

	"mepserver/common/arch/workspace"
)

// MepSpace base mep bus structure
type MepSpace struct {
	workspace.SpaceBase
	R *http.Request       `json:"r"`
	W http.ResponseWriter `json:"w"`

	Ctx           context.Context `json:"ctx"`
	ServiceId     string          `json:"serviceId"`
	RestBody      interface{}     `json:"restBody"`
	AppInstanceId string          `json:"appInstanceId"`
	InstanceId    string          `json:"instanceId"`
	SubscribeId   string          `json:"subscribeId"`
	DNSRuleId     string          `json:"dnsRuleId"`
	TrafficRuleId string          `json:"trafficRuleId"`
	Flag          bool            `json:"flag"`

	QueryParam url.Values `json:"queryParam"`

	CoreRequest interface{}     `json:"coreRequest"`
	CoreRsp     interface{}     `json:"coreRsp"`
	HttPErrInf  *proto.Response `json:"httpErrInf"`
	HttPRsp     interface{}     `json:"httpRsp"`
}

// NewWorkSpace new a work space
func NewWorkSpace(w http.ResponseWriter, r *http.Request) *MepSpace {
	var plan = MepSpace{
		W: w,
		R: r,
	}

	plan.Init()
	return &plan
}
