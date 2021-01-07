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

type DeleteAppDConfig struct {
	workspace.TaskBase
	Ctx           context.Context     `json:"ctx,in"`
	W             http.ResponseWriter `json:"w,in"`
	AppInstanceId string              `json:"appInstanceId,in"`
	RestBody      interface{}         `json:"restBody,in"`
	HttpRsp       interface{}         `json:"httpRsp,out"`
	worker        *task.Worker
}

func (t *DeleteAppDConfig) WithWorker(w *task.Worker) *DeleteAppDConfig {
	t.worker = w
	return t
}

func (t *DeleteAppDConfig) OnRequest(data string) workspace.TaskCode {

	/*
			1. Check if AppInstanceId already exist and return error if not exist.(query from db)
		    2. Check if any other ongoing operation for this AppInstance Id in the system.
			3. update the this request to DB (job, task and task status)
	*/
	if IsAppInstanceIdAlreadyExists(t.AppInstanceId) == false {
		log.Errorf(nil, "app instance not found")
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "app instance not found")
		return workspace.TaskFinish
	}

	// Check if any other ongoing operation for this AppInstance Id in the system.
	if IsAnyOngoingOperationExist(t.AppInstanceId) == true {
		log.Errorf(nil, "app instance has other operation in progress")
		t.SetFirstErrorCode(meputil.ForbiddenOperation, "app instance has other operation in progress")
		return workspace.TaskFinish
	}

	var appDConfig models.AppDConfig
	appDConfig.Operation = http.MethodDelete

	taskId := meputil.GenerateUniqueId()

	errCode, msg := UpdateProcessingDatabase(t.AppInstanceId, taskId, &appDConfig)
	if errCode != 0 {
		t.SetFirstErrorCode(errCode, msg)
		return workspace.TaskFinish
	}

	t.worker.StartNewTask(appDConfig.AppName, t.AppInstanceId, taskId)

	t.HttpRsp = GenerateTaskResponse(taskId, t.AppInstanceId, "PROCESSING", "0", "Operation In progress")
	return workspace.TaskFinish
}
