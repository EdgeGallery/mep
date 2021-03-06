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

// Package plans implements mep server api plans
package plans

import (
	"context"
	"net/http"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/server/core/backend"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/registry"

	"mepserver/common/arch/workspace"
	"mepserver/common/util"
)

// DelOneSubscribe steps to delete a subscription
type DelOneSubscribe struct {
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

// WithType set type and return DelOneSubscribe
func (t *DelOneSubscribe) WithType(subType string) *DelOneSubscribe {
	t.SubscribeType = subType
	return t
}

// OnRequest handles subscription update
func (t *DelOneSubscribe) OnRequest(data string) workspace.TaskCode {

	appInstanceId := t.AppInstanceId
	subscribeId := t.SubscribeId
	log.Debugf("Delete request arrived with app subscription with appId %s and subscriptionId %s.",
		appInstanceId, subscribeId)
	appSubKeyPath := util.GetSubscribeKeyPath(t.SubscribeType) + appInstanceId + "/" + subscribeId
	opts := []registry.PluginOp{
		registry.OpGet(registry.WithStrKey(appSubKeyPath)),
	}
	resp, errGet := backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if errGet != nil {
		log.Errorf(nil, "Get subscription from etcd failed.")
		t.SetFirstErrorCode(util.OperateDataWithEtcdErr, "get subscription from etch failed")
		return workspace.TaskFinish
	}

	if len(resp.Kvs) == 0 {
		log.Errorf(nil, "Subscription does not exist.")
		t.SetFirstErrorCode(util.SubscriptionNotFound, "subscription not exist")
		return workspace.TaskFinish
	}

	opts = []registry.PluginOp{
		registry.OpDel(registry.WithStrKey(appSubKeyPath)),
	}
	_, err := backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if err != nil {
		log.Errorf(nil, "Delete subscription from etcd failed.")
		t.SetFirstErrorCode(util.OperateDataWithEtcdErr, "delete subscription from etch failed")
		return workspace.TaskFinish
	}

	t.HttpRsp = ""
	log.Debugf("App subscription with appId %s and subscriptionId %s is deleted successfully.",
		appInstanceId, subscribeId)
	return workspace.TaskFinish
}
