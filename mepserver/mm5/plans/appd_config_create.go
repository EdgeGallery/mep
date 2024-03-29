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

// Package plans implements mep server mm5 interfaces
package plans

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"mepserver/common/appd"
	"mepserver/common/models"
	meputil "mepserver/common/util"
	"mepserver/mm5/task"
	"net/http"

	"github.com/go-playground/validator/v10"

	"mepserver/common/arch/workspace"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/util"
)

// DecodeAppDRestReq handles appd config decode request
type DecodeAppDRestReq struct {
	workspace.TaskBase
	R             *http.Request   `json:"r,in"`
	Ctx           context.Context `json:"ctx,out"`
	AppInstanceId string          `json:"appInstanceId,out"`
	RestBody      interface{}     `json:"restBody,out"`
}

// OnRequest handles the appd request decoding
func (t *DecodeAppDRestReq) OnRequest(data string) workspace.TaskCode {
	err := t.getParam(t.R)
	if err != nil {
		log.Error("Parameters validation failed for appd request.", nil)
		return workspace.TaskFinish
	}
	err = t.parseBody(t.R)
	if err != nil {
		log.Error("Parse rest body failed.", nil)
	}
	return workspace.TaskFinish
}

// WithBody handle input body initialization
func (t *DecodeAppDRestReq) WithBody(body interface{}) *DecodeAppDRestReq {
	t.RestBody = body
	return t
}

func (t *DecodeAppDRestReq) getParam(r *http.Request) error {
	queryReq, _ := meputil.GetHTTPTags(r)
	t.AppInstanceId = queryReq.Get(meputil.AppInstanceIdStr)
	t.Ctx = util.SetTargetDomainProject(r.Context(), r.Header.Get("X-Domain-Name"), queryReq.Get(":project"))
	return nil
}

func (t *DecodeAppDRestReq) parseBody(r *http.Request) error {
	if t.RestBody == nil {
		return nil
	}
	msg, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("Input body read failed.", nil)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "read request body error")
		return err
	}

	/* We can have the total of 32 traffic Rule and 64 DNS Rules so need sufficient length*/
	if len(msg) > (meputil.RequestBodyLength * 8) {
		err = errors.New("request body too large")
		log.Errorf(nil, "Request body too large %d.", len(msg))
		t.SetFirstErrorCode(meputil.RequestParamErr, "request body too large")
		return err
	}

	newMsg, err := t.checkParam(msg)
	if err != nil {
		log.Error("Check param failed.", nil)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "check Param failed")
		return err
	}

	err = json.Unmarshal(newMsg, t.RestBody)
	if err != nil {
		log.Errorf(nil, "Request body unmarshalling failed.")
		t.SetFirstErrorCode(meputil.ParseInfoErr, "unmarshal request body error")
		return errors.New("json unmarshalling failed")
	}

	appDConfigInput, _ := t.RestBody.(*models.AppDConfig)

	validate := validator.New()
	verrs := validate.Struct(appDConfigInput)
	if verrs != nil {
		errorString := "Invalid value for input on: "
		for _, verr := range verrs.(validator.ValidationErrors) {
			log.Errorf(err, "Validation Error(namespace: %v, field: %v, struct namespace:%v, struct field: %v, "+
				"tag: %v, actual tag: %v, kind: %v, type: %v, value: %v, param: %v).", verr.Namespace(), verr.Field(),
				verr.StructNamespace(), verr.StructField(), verr.Tag(), verr.ActualTag(), verr.Kind(), verr.Type(),
				verr.Value(), verr.Param())
			errorString += fmt.Sprintf(" %v", verr.Field())
		}
		t.SetFirstErrorCode(meputil.SerErrFailBase, errorString)
		return verrs
	}
	log.Infof("AppD config received(method: %s, body:%s).", r.Method, string(msg))
	return nil
}

func (t *DecodeAppDRestReq) checkParam(msg []byte) ([]byte, error) {

	var temp map[string]interface{}
	err := json.Unmarshal(msg, &temp)
	if err != nil {
		log.Errorf(err, "Invalid json to map: %s.", util.BytesToStringWithNoCopy(msg))
		t.SetFirstErrorCode(meputil.SerErrFailBase, err.Error())
		return nil, err
	}

	meputil.SetMapValue(temp, "consumedLocalOnly", true)
	meputil.SetMapValue(temp, "isLocal", true)
	meputil.SetMapValue(temp, "scopeOfLocality", "MEC_HOST")

	msg, err = json.Marshal(&temp)
	if err != nil {
		log.Errorf(err, "Invalid map to json.")
		t.SetFirstErrorCode(meputil.SerErrFailBase, err.Error())
		return nil, err
	}

	return msg, nil
}

// CreateAppDConfig handle appd config create
type CreateAppDConfig struct {
	workspace.TaskBase
	appd.AppDCommon
	Ctx           context.Context     `json:"ctx,in"`
	W             http.ResponseWriter `json:"w,in"`
	AppInstanceId string              `json:"appInstanceId,in"`
	RestBody      interface{}         `json:"restBody,in"`
	HttpRsp       interface{}         `json:"httpRsp,out"`
	worker        *task.Worker
}

// WithWorker input worker instance
func (t *CreateAppDConfig) WithWorker(w *task.Worker) *CreateAppDConfig {
	t.worker = w
	return t
}

// OnRequest handles appd configuration create
func (t *CreateAppDConfig) OnRequest(data string) workspace.TaskCode {

	appDConfigInput, ok := t.RestBody.(*models.AppDConfig)
	if !ok {
		t.SetFirstErrorCode(1, "input body parse failed")
		t.SetSerErrInfo(&workspace.SerErrInfo{ErrCode: http.StatusBadRequest, Message: "Parse body error."})
		return workspace.TaskFinish
	}

	/*
			1. Check if AppInstanceId already exist and return error as duplicate.(query from db)
		    2. Also check if any other ongoing operation for this AppInstanceId
			2. Add the this request to DB (job, task and task status)
	*/
	if t.IsAppInstanceAlreadyCreated(t.AppInstanceId) {
		log.Errorf(nil, "Duplicate app instance.")
		t.SetFirstErrorCode(meputil.DuplicateOperation, "duplicate app instance")
		return workspace.TaskFinish
	}

	if t.IsDuplicateAppNameExists(appDConfigInput.AppName) {
		log.Errorf(nil, "Duplicate app name.")
		t.SetFirstErrorCode(meputil.DuplicateOperation, "duplicate app name")
		return workspace.TaskFinish
	}

	// Check if any other ongoing operation for this AppInstance Id in the system.
	if t.IsAnyOngoingOperationExist(t.AppInstanceId) {
		log.Errorf(nil, "App instance has other operation in progress.")
		t.SetFirstErrorCode(meputil.ForbiddenOperation, "app instance has other operation in progress")
		return workspace.TaskFinish
	}

	appDConfigInput.Operation = http.MethodPost

	// Change the IP Address type to type common for MP2 and MP1
	for i, _ := range appDConfigInput.AppDNSRule {
		if appDConfigInput.AppDNSRule[i].IPAddressType == "IPv4" {
			appDConfigInput.AppDNSRule[i].IPAddressType = "IP_V4"
		} else if appDConfigInput.AppDNSRule[i].IPAddressType == "IPv6" {
			appDConfigInput.AppDNSRule[i].IPAddressType = "IP_V6"
		}
	}

	// Add to Task InstanceID mapping DB
	taskId := meputil.GenerateUniqueId()

	errCode, msg := t.StageNewTask(t.AppInstanceId, taskId, appDConfigInput, false)

	if errCode != 0 {
		t.SetFirstErrorCode(errCode, msg)
		return workspace.TaskFinish
	}

	t.worker.StartNewTask(appDConfigInput.AppName, t.AppInstanceId, taskId)

	t.HttpRsp = t.GenerateTaskResponse(taskId, t.AppInstanceId, "PROCESSING", "0", "Operation In progress")
	return workspace.TaskFinish
}
