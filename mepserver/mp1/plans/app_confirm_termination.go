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
	"errors"
	"fmt"
	"github.com/apache/servicecomb-service-center/pkg/log"
	registry_backend "github.com/apache/servicecomb-service-center/server/core/backend"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/registry"
	"io/ioutil"
	"mepserver/common/appd"
	"mepserver/common/arch/workspace"
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
		log.Error("Parameters validation failed on confirm ready request.", err)
		return workspace.TaskFinish
	}

	err = t.ParseBody(t.R)
	if err != nil {
		log.Error("Confirm ready request body parse failed.", err)
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

// ParseBody Parse request body
func (t *DecodeConfirmTerminateReq) ParseBody(r *http.Request) error {
	if t.RestBody == nil {
		return nil
	}
	msg, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("Confirm ready request read failed.", nil)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "read request body error")
		return errors.New("read failed")
	}
	if len(msg) > meputil.RequestBodyLength {
		err = errors.New("request body too large")
		log.Errorf(err, "Confirm ready request body too large %d.", len(msg))
		t.SetFirstErrorCode(meputil.RequestParamErr, "request body too large")
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
	log.Debugf("Confirm terminate received for %s.", appInstanceId)

	/*
	 1. Get the record from DB and match the operation type, if match, update the record.
	 2. Send the response
	*/

	opts := []registry.PluginOp{
		registry.OpGet(registry.WithStrKey(meputil.GetSubscribeKeyPath(meputil.AppTerminationConfirmation) +
			appInstanceId + "/")),
	}

	resp, err := registry_backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if err != nil {
		t.SetFirstErrorCode(meputil.OperateDataWithEtcdErr, "get record failed for confirm termination")
		return workspace.TaskFinish
	}

	if len(resp.Kvs) == 0 {
		log.Warnf("Subscription doesn't exist.")
		t.SetFirstErrorCode(meputil.SubscriptionNotFound, "get record failed for confirm termination")
		return workspace.TaskFinish
	}

	terminationConfirm := &models.ConfirmTerminate{}
	jsonErr := json.Unmarshal(resp.Kvs[0].Value, terminationConfirm)
	if jsonErr != nil {
		log.Error("Subscription parsed failed.", nil)
		return workspace.TaskFinish
	}
	// TODO check the operateAction with DB and msg
	log.Infof("from Db (%v)", terminationConfirm.OperationAction)  // tobe removed
	log.Infof("from msg (%v)", terminationConfirm.OperationAction) //  tobe removed

	t.HttpRsp = ""
	return workspace.TaskFinish
}
