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

// Package path implements mep server api plans
package plans

import (
	"context"
	"encoding/json"
	"mepserver/common/models"
	"net/http"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/server/core/backend"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/registry"

	"mepserver/common/arch/workspace"
	"mepserver/common/util"
)

type GetOneSubscribe struct {
	workspace.TaskBase
	R             *http.Request       `json:"r,in"`
	HttpErrInf    *proto.Response     `json:"httpErrInf,out"`
	Ctx           context.Context     `json:"ctx,in"`
	W             http.ResponseWriter `json:"w,in"`
	AppInstanceId string              `json:"appInstanceId,in"`
	SubscribeId   string              `json:"subscribeId,in"`
	HttpRsp       interface{}         `json:"httpRsp,out"`
	SubscribeType string              `json:"subscribeType,out"`
}

// set type and return GetOneSubscribe
func (t *GetOneSubscribe) WithType(subType string) *GetOneSubscribe {
	t.SubscribeType = subType
	return t
}

// OnRequest
func (t *GetOneSubscribe) OnRequest(data string) workspace.TaskCode {

	appInstanceId := t.AppInstanceId
	subscribeId := t.SubscribeId
	log.Debugf("Query request arrived to fetch the subscription information with  "+
		"appId %s and subscriptionId %s", appInstanceId, subscribeId)

	opts := []registry.PluginOp{
		registry.OpGet(registry.WithStrKey(util.GetSubscribeKeyPath(t.SubscribeType) +
			appInstanceId + "/" + subscribeId)),
	}
	resp, err := backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if err != nil {
		log.Errorf(nil, "get subscription from etcd failed")
		t.SetFirstErrorCode(util.OperateDataWithEtcdErr, "get subscription from etch failed")
		return workspace.TaskFinish
	}

	if len(resp.Kvs) == 0 {
		log.Errorf(nil, "subscription doesn't exist")
		t.SetFirstErrorCode(util.SubscriptionNotFound, "subscription not exist")
		return workspace.TaskFinish
	}

	var jsonErr error
	selfPath := t.R.URL.Path[len(util.RootPath):]
	if t.SubscribeType == util.SerAvailabilityNotificationSubscription {
		sub := &models.SerAvailabilityNotificationSubscription{}
		jsonErr = json.Unmarshal(resp.Kvs[0].Value, sub)
		sub.Links.Self.Href = selfPath
		t.HttpRsp = sub
	} else {
		sub := &models.AppTerminationNotificationSubscription{}
		jsonErr = json.Unmarshal(resp.Kvs[0].Value, sub)
		sub.Links.Self.Href = selfPath
		t.HttpRsp = sub
	}
	if jsonErr != nil {
		log.Error("subscription parsed fail", nil)
		t.SetFirstErrorCode(util.ParseInfoErr, "subscription parsed fail")
		return workspace.TaskFinish
	}
	log.Debugf("Response for app subscription information with appId %s and subscriptionId %s",
		appInstanceId, subscribeId)
	return workspace.TaskFinish
}
