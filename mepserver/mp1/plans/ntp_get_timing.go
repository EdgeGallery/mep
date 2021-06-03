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
	meputil "mepserver/common/util"
)

// CurrentTimeGet step to read a single dns rule
type CurrentTimeGet struct {
	workspace.TaskBase
	HttpRsp interface{} `json:"httpRsp,out"`
}

// OnRequest handles the traffic rule query
func (t *CurrentTimeGet) OnRequest(data string) workspace.TaskCode {

	// call if api to get current time
	timeData, secs, nanos, errCode := ntp.GetCurrentTime() // time data type should be in bytes
	if errCode != 0 {
		log.Errorf(nil, "Get current time from NTP server failed")
		t.SetFirstErrorCode(workspace.ErrCode(errCode), "current time get failed")
		return workspace.TaskFinish
	}

	log.Infof("Seconds %v nanos %v", secs, nanos)

	Timing := models.Timing{}
	var currentTime *ntp.CurrentTime
	if timeData != nil {
		//jsonErr := json.Unmarshal(timeData, &Timing.CurrentTime)
		//if jsonErr != nil {
		//	log.Warn("Could not read the traffic rule properly from etcd.")
		//	t.SetFirstErrorCode(meputil.OperateDataWithNtpErr, "parse traffic rules from etcd failed")
		//	return workspace.TaskFinish
		//}
		currentTime = &Timing.CurrentTime
	}

	if currentTime == nil {
		log.Error("Requested didnt get the current time.", nil)
		t.SetFirstErrorCode(meputil.SubscriptionNotFound, "didnt get the current time")
		return workspace.TaskFinish
	}
	t.HttpRsp = currentTime
	return workspace.TaskFinish
}
