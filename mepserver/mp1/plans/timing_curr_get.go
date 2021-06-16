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

// Package plans implements mep server api plans
package plans

import (
	"github.com/apache/servicecomb-service-center/pkg/log"
	"mepserver/common/arch/workspace"
	"mepserver/common/extif/ntp"
	"mepserver/common/models"
)

// CurrentTimeGet step to get current time
type CurrentTimeGet struct {
	workspace.TaskBase
	HttpRsp interface{} `json:"httpRsp,out"`
}

// OnRequest handles get current timing query
func (t *CurrentTimeGet) OnRequest(data string) workspace.TaskCode {

	// call external if api to get current time
	currentTimeRsp, errCode := ntp.GetTimeStamp()
	if errCode != 0 {
		log.Errorf(nil, "Get current time from NTP server failed")
		t.SetFirstErrorCode(workspace.ErrCode(errCode), "Current time get failed")
		return workspace.TaskFinish
	}

	ct := models.CurrentTime{}
	ct.Seconds = currentTimeRsp.Seconds
	ct.NanoSeconds = currentTimeRsp.NanoSeconds
	ct.TimeSourceStatus = currentTimeRsp.TimeSourceStatus

	t.HttpRsp = ct
	return workspace.TaskFinish
}
