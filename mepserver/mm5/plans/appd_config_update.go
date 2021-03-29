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
	"context"
	"mepserver/common/models"
	meputil "mepserver/common/util"
	"mepserver/mm5/task"
	"net/http"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"mepserver/common/arch/workspace"
)

type UpdateAppDConfig struct {
	workspace.TaskBase
	Ctx           context.Context     `json:"ctx,in"`
	W             http.ResponseWriter `json:"w,in"`
	AppInstanceId string              `json:"appInstanceId,in"`
	RestBody      interface{}         `json:"restBody,in"`
	HttpRsp       interface{}         `json:"httpRsp,out"`
	worker        *task.Worker
}

func (t *UpdateAppDConfig) WithWorker(w *task.Worker) *UpdateAppDConfig {
	t.worker = w
	return t
}

func (t *UpdateAppDConfig) OnRequest(data string) workspace.TaskCode {

	appDConfigInput, ok := t.RestBody.(*models.AppDConfig)
	if !ok {
		t.SetFirstErrorCode(1, "input body parse failed")
		t.SetSerErrInfo(&workspace.SerErrInfo{ErrCode: http.StatusBadRequest, Message: "Parse body error!"})
		return workspace.TaskFinish
	}

	/*
			1. Check if AppInstanceId already exist and return error if not exist.(query from db)
		    2. Check if any other ongoing operation for this AppInstance Id in the system.
			3. update the this request to DB (job, task and task status)
	*/
	if !IsAppInstanceIdAlreadyExists(t.AppInstanceId) {
		log.Errorf(nil, "app instance not found")
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "app instance not found")
		return workspace.TaskFinish
	}

	// Check if any other ongoing operation for this AppInstance Id in the system.
	if IsAnyOngoingOperationExist(t.AppInstanceId) {
		log.Errorf(nil, "app instance has other operation in progress")
		t.SetFirstErrorCode(meputil.ForbiddenOperation, "app instance has other operation in progress")
		return workspace.TaskFinish
	}

	appDConfigInput.Operation = http.MethodPut
	taskId := meputil.GenerateUniqueId()

	// Change the IP Address type to type common for MP2 and MP1
	for i, _ := range appDConfigInput.AppDNSRule {
		if appDConfigInput.AppDNSRule[i].IPAddressType == "IPv4" {
			appDConfigInput.AppDNSRule[i].IPAddressType = "IP_V4"
		} else if appDConfigInput.AppDNSRule[i].IPAddressType == "IPv6" {
			appDConfigInput.AppDNSRule[i].IPAddressType = "IP_V6"
		}
	}

	errCode, msg := UpdateProcessingDatabase(t.AppInstanceId, taskId, appDConfigInput)
	if errCode != 0 {
		t.SetFirstErrorCode(errCode, msg)
		return workspace.TaskFinish
	}

	t.worker.StartNewTask(appDConfigInput.AppName, t.AppInstanceId, taskId)

	t.HttpRsp = GenerateTaskResponse(taskId, t.AppInstanceId, "PROCESSING", "0", "Operation In progress")
	return workspace.TaskFinish
}
