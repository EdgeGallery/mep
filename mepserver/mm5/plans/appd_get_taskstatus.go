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
	"encoding/json"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"mepserver/common/arch/workspace"
	"mepserver/common/extif/backend"
	"mepserver/common/models"
	meputil "mepserver/common/util"
	"net/http"
	"strconv"
)

type DecodeTaskRestReq struct {
	workspace.TaskBase
	R      *http.Request   `json:"r,in"`
	Ctx    context.Context `json:"ctx,out"`
	TaskId string          `json:"taskId,out"`
}

func (t *DecodeTaskRestReq) OnRequest(data string) workspace.TaskCode {
	err := t.getParam(t.R)
	if err != nil {
		log.Error("parameters validation failed", nil)
		return workspace.TaskFinish
	}
	return workspace.TaskFinish
}

func (t *DecodeTaskRestReq) getParam(r *http.Request) error {
	queryReq, _ := meputil.GetHTTPTags(r)

	t.TaskId = queryReq.Get(":taskId")

	t.Ctx = util.SetTargetDomainProject(r.Context(), r.Header.Get("X-Domain-Name"), queryReq.Get(":project"))
	return nil
}

type TaskStatusGet struct {
	workspace.TaskBase
	R       *http.Request       `json:"r,in"`
	W       http.ResponseWriter `json:"w,in"`
	TaskId  string              `json:"taskId,in"`
	HttpRsp interface{}         `json:"httpRsp,out"`
}

func (t *TaskStatusGet) OnRequest(inputData string) workspace.TaskCode {
	log.Debugf("query request arrived to fetch task status for taskId %s.", t.TaskId)

	taskEntry, err := backend.GetRecord(meputil.AppDLCMTasksPath + t.TaskId)
	if err != 0 {
		log.Errorf(nil, "get task rule from data-store failed")
		t.SetFirstErrorCode(workspace.ErrCode(err), "task rule retrieval failed")
		return workspace.TaskFinish
	}

	appInstInStore := string(taskEntry)

	taskStatus, err := backend.GetRecord(meputil.AppDLCMTaskStatusPath + appInstInStore + "/" + t.TaskId)
	if err != 0 {
		log.Errorf(nil, "get task status rule from data-store failed")
		t.SetFirstErrorCode(workspace.ErrCode(err), "task status rule retrieval failed")
		return workspace.TaskFinish
	}

	taskStatusInStore := &models.TaskStatus{}
	jsonErr := json.Unmarshal(taskStatus, taskStatusInStore)
	if jsonErr != nil {
		log.Errorf(nil, "failed to parse the task status from data-store")
		t.SetFirstErrorCode(meputil.OperateDataWithEtcdErr, "parse task status from data-store failed")
		return workspace.TaskFinish
	}

	progress := (taskStatusInStore.Progress * 100) / (len(taskStatusInStore.TrafficRuleStatusLst) + len(taskStatusInStore.DNSRuleStatusLst))

	var state string
	if taskStatusInStore.Progress == (len(taskStatusInStore.TrafficRuleStatusLst) + len(taskStatusInStore.DNSRuleStatusLst)) {
		state = meputil.TASK_STATE_SUCCESS
	} else if taskStatusInStore.Progress >= 0 {
		state = meputil.TASK_STATE_PROCESSING
	} else {
		state = meputil.TASK_STATE_FAILURE
		progress = 0
	}

	t.HttpRsp = GenerateTaskResponse(t.TaskId, appInstInStore, state,
		strconv.Itoa(progress), taskStatusInStore.Details)

	return workspace.TaskFinish
}
