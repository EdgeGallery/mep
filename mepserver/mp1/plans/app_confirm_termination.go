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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	"io/ioutil"
	"mepserver/common/appd"
	"mepserver/common/arch/workspace"
	"mepserver/common/extif/backend"
	"mepserver/common/models"
	meputil "mepserver/common/util"
	"net/http"
)

type DecodeConfirmTerminateReq struct {
	workspace.TaskBase
	R             *http.Request `json:"r,in"`
	AppInstanceId string        `json:"appInstanceId,out"`
	RestBody      interface{}   `json:"restBody,out"`
}

// OnRequest decodes the service request messages
func (t *DecodeConfirmTerminateReq) OnRequest(data string) workspace.TaskCode {
	log.Infof("Received message from ClientIP [%s] AppInstanceId [%s] Operation [%s] Resource [%s].",
		meputil.GetClientIp(t.R), meputil.GetAppInstanceId(t.R), meputil.GetMethodFromReq(t.R), meputil.GetHttpResourceInfo(t.R))

	err := t.getParam(t.R)
	if err != nil {
		log.Error("Parameters validation failed on confirm termination request.", err)
		return workspace.TaskFinish
	}

	err = t.ParseBody(t.R)
	if err != nil {
		log.Error("Confirm terminate request body parse failed.", err)
	}

	return workspace.TaskFinish
}

func (t *DecodeConfirmTerminateReq) getParam(r *http.Request) error {
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

func (t *DecodeConfirmTerminateReq) validateParam(msg []byte) error {

	var confirmTermination models.ConfirmTermination
	err := json.Unmarshal(msg, &confirmTermination)
	if err != nil {
		return errors.New("unmarshal msg error")
	}

	resp, errCode := backend.GetRecord(meputil.AppConfirmTerminationPath + t.AppInstanceId + "/")
	if errCode != 0 {
		t.SetFirstErrorCode(meputil.ServiceInactive, "no termination is going on")
		return errors.New("no termination is going on for this instance")
	}

	terminationConfirmRec := &models.ConfirmTerminationRecord{}
	jsonErr := json.Unmarshal(resp, terminationConfirmRec)
	if jsonErr != nil {
		log.Error("Subscription parsed failed.", nil)
		t.SetFirstErrorCode(meputil.RequestParamErr, "subscription parsed failed")
		return errors.New("subscription parsed failed")
	}

	if confirmTermination.OperationAction != terminationConfirmRec.OperationAction {
		t.SetFirstErrorCode(meputil.RequestParamErr, "operation action is not matching")
		return errors.New("operation action is not matching")
	}
	return nil
}

// ParseBody Parse request body
func (t *DecodeConfirmTerminateReq) ParseBody(r *http.Request) error {
	if t.RestBody == nil {
		return nil
	}
	msg, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("Confirm termination request read failed.", nil)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "read request body error")
		return errors.New("read failed")
	}
	if len(msg) > meputil.RequestBodyLength {
		err = errors.New("request body too large")
		log.Errorf(err, "Confirm termination request body too large %d.", len(msg))
		t.SetFirstErrorCode(meputil.RequestParamErr, "request body too large")
		return err
	}

	err = t.validateParam(msg)
	if err != nil {
		log.Error("Confirm terminate validate param failed.", err)
		return err
	}

	err = json.Unmarshal(msg, t.RestBody)
	if err != nil {
		log.Errorf(nil, "Service register request unmarshalling failed.")
		t.SetFirstErrorCode(meputil.ParseInfoErr, "unmarshal request body error")
		return errors.New("json unmarshalling failed")
	}

	return nil
}

// WithBody set body and return DecodeConfirmReadyReq
func (t *DecodeConfirmTerminateReq) WithBody(body interface{}) *DecodeConfirmTerminateReq {
	t.RestBody = body
	return t
}

// ConfirmTermination to confirm the application is up and running
type ConfirmTermination struct {
	workspace.TaskBase
	appd.AppDCommon
	R             *http.Request   `json:"r,in"`
	HttpErrInf    *proto.Response `json:"httpErrInf,out"`
	HttpRsp       interface{}     `json:"httpRsp,out"`
	AppInstanceId string          `json:"appInstanceId,in"`
}

// OnRequest handles service delete request
func (t *ConfirmTermination) OnRequest(data string) workspace.TaskCode {
	appInstanceId := t.AppInstanceId
	log.Infof("Confirm terminate received for %s.", appInstanceId)

	/*
	 1. Get the record from DB and match the operation type, if match, update the record.
	 2. Send the response
	*/

	resp, err := backend.GetRecord(meputil.AppConfirmTerminationPath + appInstanceId + "/")
	if err != 0 {
		t.SetFirstErrorCode(meputil.ServiceInactive, "no termination is going on")
		log.Warnf("No termination is going on for %s.", appInstanceId)
		return workspace.TaskFinish
	}

	terminationConfirm := &models.ConfirmTerminationRecord{}
	jsonErr := json.Unmarshal(resp, terminationConfirm)
	if jsonErr != nil {
		log.Error("json unmarshalling failed.", nil)
		return workspace.TaskFinish
	}

	// Update the status
	terminationConfirm.TerminationStatus = meputil.TerminationFinish

	termConfirmBytes, jsonErr := json.Marshal(terminationConfirm)
	if jsonErr != nil {
		log.Error("Json marshalling failed.", nil)
		return workspace.TaskFinish
	}

	err = backend.PutRecord(meputil.AppConfirmTerminationPath+appInstanceId+"/", termConfirmBytes)
	if err != 0 {
		t.SetFirstErrorCode(meputil.OperateDataWithEtcdErr, "put record failed for confirm termination")
		return workspace.TaskFinish
	}

	t.HttpRsp = ""
	return workspace.TaskFinish
}
