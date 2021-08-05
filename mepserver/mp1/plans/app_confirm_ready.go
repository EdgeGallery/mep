package plans

import (
	"fmt"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	log "github.com/sirupsen/logrus"
	"mepserver/common/appd"
	"mepserver/common/arch/workspace"
	meputil "mepserver/common/util"
	"net/http"
)

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

type DecodeConfirmReadyReq struct {
	workspace.TaskBase
	R             *http.Request `json:"r,in"`
	AppInstanceId string        `json:"appInstanceId,out"`
}

// OnRequest decodes the service request messages
func (t *DecodeConfirmReadyReq) OnRequest(data string) workspace.TaskCode {
	log.Infof("Received message from ClientIP [%s] AppInstanceId [%s] Operation [%s] Resource [%s].",
		meputil.GetClientIp(t.R), meputil.GetAppInstanceId(t.R), meputil.GetMethodFromReq(t.R), meputil.GetHttpResourceInfo(t.R))

	err := t.getParam(t.R)
	if err != nil {
		log.Error("Parameters validation failed on service register request.", err)
		return workspace.TaskFinish
	}

	return workspace.TaskFinish
}

func (t *DecodeConfirmReadyReq) getParam(r *http.Request) error {
	query, _ := meputil.GetHTTPTags(r)

	var err error

	t.AppInstanceId = query.Get(meputil.AppInstanceIdStr)
	if len(t.AppInstanceId) == 0 {
		err = fmt.Errorf("invalid app instance id")
		t.SetFirstErrorCode(meputil.AuthorizationValidateErr, err.Error())
		return err
	}
	return nil
}

// ConfirmReady to confirm the application is up and running
type ConfirmReady struct {
	workspace.TaskBase
	appd.AppDCommon
	R             *http.Request   `json:"r,in"`
	HttpErrInf    *proto.Response `json:"httpErrInf,out"`
	HttpRsp       interface{}     `json:"httpRsp,out"`
	AppInstanceId string          `json:"appInstanceId,in"`
}

// OnRequest handles service delete request
func (t *ConfirmReady) OnRequest(data string) workspace.TaskCode {
	appInstanceId := t.AppInstanceId

	/*
		1. Check if AppInstanceId already exist and return error if not exist.(query from db)
		2. Check if any other ongoing operation for this AppInstance Id in the system.
		3. Send the response
	*/

	if !t.IsAppInstanceAlreadyCreated(t.AppInstanceId) {
		log.Errorf("App instance not found.")
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "app instance not found")
		return workspace.TaskFinish
	}

	// Check if any other ongoing operation for this AppInstance Id in the system.
	if t.IsAnyOngoingOperationExist(t.AppInstanceId) {
		log.Errorf("App instance has other operation in progress.")
		t.SetFirstErrorCode(meputil.ForbiddenOperation, "app instance has other operation in progress")
		return workspace.TaskFinish
	}

	//t.HttpErrInf.Code =
	t.HttpRsp = ""
	log.Debugf("Confirm ready recieved for %s .", appInstanceId)

	return workspace.TaskFinish
}
