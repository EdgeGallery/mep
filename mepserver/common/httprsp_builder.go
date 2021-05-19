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

// Package common implements mep server common functionalities
package common

import (
	"encoding/json"
	"fmt"

	"mepserver/common/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/rest"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	error2 "github.com/apache/servicecomb-service-center/server/error"
	"github.com/apache/servicecomb-service-center/server/rest/controller"

	"mepserver/common/arch/workspace"
	"mepserver/common/util"
)

// SendHttpRsp holds the http response building parameters
type SendHttpRsp struct {
	HttpErrInf *proto.Response `json:"httpErrInf,in"`
	R          *http.Request   `json:"r,in"`
	workspace.TaskBase
	W          http.ResponseWriter `json:"w,in"`
	HttpRsp    interface{}         `json:"httpRsp,in"`
	StatusCode int
}

// OnRequest builds an http response based on the input provided
func (t *SendHttpRsp) OnRequest(data string) workspace.TaskCode {
	// remove service-center server header
	t.W.Header().Del(util.ServerHeader)

	failureEventLogFormat := "Response Message for ClientIP [%s] AppInstanceId [%s] Operation [%s] Resource " +
		"[%s] Result [Failure : %s]."
	successEventLogFormat := "Response Message for ClientIP [%s] AppInstanceId [%s] Operation [%s] Resource " +
		"[%s] Result [Success]."

	errInfo := t.GetSerErrInfo()
	if errInfo == nil {
		body := &models.ProblemDetails{
			Title:  "Internal Server Error",
			Status: uint32(util.RemoteServerErr),
			Detail: "server internal function failed",
		}
		log.Infof(failureEventLogFormat, util.GetClientIp(t.R), util.GetAppInstanceId(t.R), util.GetMethod(t.R),
			util.GetHttpResourceInfo(t.R), body.Detail)
		util.HttpErrResponse(t.W, util.RemoteServerErr, body)
		return workspace.TaskFinish
	}

	if errInfo.ErrCode == util.SerErrServiceNotFound && strings.EqualFold(errInfo.Message, "failed to find the instance") {
		t.writeResponse(t.W, t.HttpErrInf, make([]*models.ServiceInfo, 0))
		return workspace.TaskFinish
	}

	if errInfo.ErrCode >= int(workspace.TaskFail) {
		statusCode, httpBody := t.cvtHttpErrInfo(errInfo)
		util.HttpErrResponse(t.W, statusCode, httpBody)
		log.Infof(failureEventLogFormat, util.GetClientIp(t.R), util.GetAppInstanceId(t.R), util.GetMethod(t.R),
			util.GetHttpResourceInfo(t.R), errInfo.Message)
		return workspace.TaskFinish
	}
	log.Infof(successEventLogFormat, util.GetClientIp(t.R), util.GetAppInstanceId(t.R), util.GetMethod(t.R),
		util.GetHttpResourceInfo(t.R))
	t.writeResponse(t.W, t.HttpErrInf, t.HttpRsp)
	return workspace.TaskFinish
}

func (t *SendHttpRsp) writeResponse(w http.ResponseWriter, resp *proto.Response, obj interface{}) {
	if resp != nil && resp.GetCode() != proto.Response_SUCCESS {
		controller.WriteError(w, resp.GetCode(), resp.GetMessage())
		return
	}
	if obj == nil {
		w.Header().Set(rest.HEADER_RESPONSE_STATUS, strconv.Itoa(http.StatusExpectationFailed))
		w.Header().Set(rest.HEADER_CONTENT_TYPE, rest.CONTENT_TYPE_JSON)
		w.WriteHeader(http.StatusExpectationFailed)
		return
	}

	objJSON, err := json.Marshal(obj)
	if err != nil {
		controller.WriteError(w, error2.ErrInternal, err.Error())
		return
	}
	w.Header().Set(rest.HEADER_CONTENT_TYPE, rest.CONTENT_TYPE_JSON)
	if t.StatusCode == 0 {
		t.StatusCode = http.StatusOK
	}
	w.Header().Set(rest.HEADER_RESPONSE_STATUS, strconv.Itoa(t.StatusCode))
	w.WriteHeader(t.StatusCode)
	_, err = fmt.Fprintln(w, string(objJSON))
	if err != nil {
		return
	}
}

func (t *SendHttpRsp) cvtHttpErrInfo(errInfo *workspace.SerErrInfo) (int, interface{}) {
	statusCode := http.StatusBadRequest
	var httpBody interface{}
	body := &models.ProblemDetails{
		Title:  "",
		Status: uint32(errInfo.ErrCode),
		Detail: errInfo.Message,
	}
	switch workspace.ErrCode(errInfo.ErrCode) {
	case util.SerErrServiceNotFound:
		body.Title = "Can not found resource"
	case util.SerInstanceNotFound:
		fallthrough
	case util.HeartbeatServiceNotFound:
		fallthrough
	case util.SubscriptionNotFound:
		statusCode = http.StatusNotFound
		body.Title = "Can not found resource"
	case util.EtagMissMatchErr:
		statusCode = http.StatusPreconditionFailed
		body.Title = "Precondition failed"
	case util.RemoteServerErr:
		statusCode = http.StatusServiceUnavailable
		body.Title = "Remote server error"
	case util.AuthorizationValidateErr:
		statusCode = http.StatusUnauthorized
		body.Title = "UnAuthorization"
	case util.SerErrServiceRegFailed:
		body.Title = "Service register failed"
	case util.SerErrServiceInstanceFailed:
		body.Title = "Service instance failed"
	case util.RequestParamErr:
		body.Title = "Request parameter error"
	case util.SubscriptionErr:
		body.Title = "App subscription error"
	case util.ServiceInactive:
		statusCode = http.StatusConflict
		body.Title = "Service is in INACTIVE state"
	case util.ResourceExists:
		statusCode = http.StatusUnprocessableEntity
		body.Title = "Resource already exists"
	case util.DuplicateOperation:
		body.Title = "Duplicate request error"
	case util.ForbiddenOperation:
		statusCode = http.StatusForbidden
		body.Title = "Operation Not Allowed"

	default:
		body.Title = "Bad Request"
	}
	httpBody = body
	return statusCode, httpBody
}
