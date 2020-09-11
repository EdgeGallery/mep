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

// Package path implements apigw hook
package hook

import (
	"context"
	"sync"
	"time"

	"mepserver/common/util"

	"github.com/apache/servicecomb-service-center/server/core"
	"github.com/apache/servicecomb-service-center/server/core/proto"

	"mepserver/mp1"
	"mepserver/mp1/models"
)

type endPoint struct {
	info models.EndPointInfo
	mu   sync.Mutex
}

var gEndPoint endPoint

func init() {
	var hook mp1.APIGwHook
	hook.APIHook = replaceAPIGwAddr
	mp1.SetAPIHook(hook)
	go timerTask()
}

func replaceAPIGwAddr() models.EndPointInfo {
	var ep models.EndPointInfo
	gEndPoint.mu.Lock()
	ep = gEndPoint.info
	gEndPoint.mu.Unlock()

	return ep
}

func timerTask() {
	for range time.Tick(util.HookTimerLimit * time.Second) {
		go refreshAPIGwAddr()
	}
}

func refreshAPIGwAddr() {
	req := &proto.FindInstancesRequest{
		ConsumerServiceId: "",
		AppId:             "",
		ServiceName:       "GatewayAdapter",
		VersionRule:       "",
		Tags:              nil,
		Environment:       "",
	}

	if req.AppId == "" {
		req.AppId = "default"
	}
	if req.VersionRule == "" {
		req.VersionRule = "latest"
	}

	resp, err := core.InstanceAPI.Find(context.TODO(), req)
	if err != nil {
		return
	}

	_, apiGw := mp1.Mp1CvtSrvDiscover(resp)
	if len(apiGw) == 0 {
		return
	}

	gEndPoint.mu.Lock()
	gEndPoint.info = apiGw[0].TransportInfo.Endpoint
	gEndPoint.mu.Unlock()
}
