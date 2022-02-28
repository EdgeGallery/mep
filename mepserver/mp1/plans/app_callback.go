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
 *
 */

package plans

import (
	"errors"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	"github.com/astaxie/beego/httplib"
	"io/ioutil"
	"mepserver/common/arch/workspace"
	meputil "mepserver/common/util"
	"net"
	"net/http"
	"time"
)

// Callback to callback consumer app by provider app
type Callback struct {
	workspace.TaskBase
	R          *http.Request   `json:"r,in"`
	HttpErrInf *proto.Response `json:"httpErrInf,out"`
	HttpRsp    interface{}     `json:"httpRsp,out"`
}

var tp http.RoundTripper = &http.Transport{
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

// OnRequest callback the consumer app by header info
func (t *Callback) OnRequest(data string) workspace.TaskCode {
	log.Infof("Received message from ClientIP [%s] AppInstanceId [%s] Operation [%s] Resource [%s].",
		meputil.GetClientIp(t.R), meputil.GetAppInstanceId(t.R), meputil.GetMethodFromReq(t.R), meputil.GetHttpResourceInfo(t.R))

	callbackUrl := t.R.Header.Get("callbackReference")
	if callbackUrl == "" {
		t.SetFirstErrorCode(meputil.CallbackUrlNotFound, "Callback Url validation failed")
		return workspace.TaskFinish
	}

	msg, err := ioutil.ReadAll(t.R.Body)
	if err != nil {
		log.Error("Callback request read failed.", nil)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "read request body error")
		return workspace.TaskFinish
	}
	if len(msg) > meputil.RequestBodyLength {
		err = errors.New("request body too large")
		log.Errorf(err, "Callback request body too large %d.", len(msg))
		t.SetFirstErrorCode(meputil.RequestParamErr, "request body too large")
		return workspace.TaskFinish
	}

	req := httplib.Post(callbackUrl)
	req.Header("Content-Type", "application/json; charset=utf-8")
	req.Header(meputil.XRealIp, meputil.GetLocalIP())
	req.Header("Connection", "Keep-Alive")
	req.SetTransport(tp)
	req.Body(msg)

	resp, err := req.Response()
	if err != nil {
		log.Errorf(err, "Callback failed(result: %s).", resp)
		return workspace.TaskFinish
	}

	statusCode := resp.StatusCode
	if statusCode != http.StatusNoContent {
		log.Errorf(nil, "Callback failed(statusCode: %d).", resp.StatusCode)
		return workspace.TaskFinish
	}

	log.Info(resp.Status)
	t.HttpRsp = resp.Status
	return workspace.TaskFinish
}
