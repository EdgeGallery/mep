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
	currentTimeRsp, errCode := ntp.GetCurrentTime()
	if errCode != 0 {
		log.Errorf(nil, "Get current time from NTP server failed")
		t.SetFirstErrorCode(workspace.ErrCode(errCode), "current time get failed")
		return workspace.TaskFinish
	}

	log.Infof("Seconds %v nanos %v", currentTimeRsp.Seconds, currentTimeRsp.NanoSeconds)

	ct := models.CurrentTime{}
	ct.Seconds = currentTimeRsp.Seconds
	ct.NanoSeconds = currentTimeRsp.NanoSeconds
	ct.TimeSourceStatus = currentTimeRsp.TimeSourceStatus

	t.HttpRsp = ct
	return workspace.TaskFinish
}

// TimingCaps to get timing capabilities
type TimingCaps struct {
	workspace.TaskBase
	HttpRsp interface{} `json:"httpRsp,out"`
}

// OnRequest handles to get timing capabilities query
func (t *TimingCaps) OnRequest(data string) workspace.TaskCode {

	// call external if api to get current time
	timingCapsRsp, errCode := ntp.GetTimingCaps()
	if errCode != 0 {
		log.Errorf(nil, "Get timing caps from NTP server failed")
		t.SetFirstErrorCode(workspace.ErrCode(errCode), "timing caps get failed")
		return workspace.TaskFinish
	}

	log.Infof("Seconds %v nanos %v", timingCapsRsp.Seconds, timingCapsRsp.NanoSeconds)

	tc := models.TimingCaps{}
	tc.TimeStamp.Seconds = timingCapsRsp.Seconds
	tc.TimeStamp.NanoSeconds = timingCapsRsp.NanoSeconds

	t.HttpRsp = tc
	return workspace.TaskFinish
}
