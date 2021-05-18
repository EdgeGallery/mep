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
	"net/url"
	"path"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/server/core/backend"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/registry"

	"mepserver/common/arch/workspace"
	"mepserver/common/util"
)

type GetSubscribes struct {
	workspace.TaskBase
	R             *http.Request       `json:"r,in"`
	HttpErrInf    *proto.Response     `json:"httpErrInf,out"`
	Ctx           context.Context     `json:"ctx,in"`
	W             http.ResponseWriter `json:"w,in"`
	AppInstanceId string              `json:"appInstanceId,in"`
	HttpRsp       interface{}         `json:"httpRsp,out"`
	SubscribeType string              `json:"subscribeType,out"`
}

// set type and return GetSubscribes
func (t *GetSubscribes) WithType(subType string) *GetSubscribes {
	t.SubscribeType = subType
	return t
}

// OnRequest
func (t *GetSubscribes) OnRequest(data string) workspace.TaskCode {

	subscribeKeyPath := util.GetSubscribeKeyPath(t.SubscribeType)

	appInstanceId := t.AppInstanceId
	log.Debugf("Query request arrived to fetch all the %s information for appId %s.", t.SubscribeType, appInstanceId)

	opts := []registry.PluginOp{
		registry.OpGet(registry.WithStrKey(subscribeKeyPath+appInstanceId), registry.WithPrefix()),
	}
	resp, err := backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if err != nil {
		log.Errorf(nil, "Get subscription from etcd failed.")
		t.SetFirstErrorCode(util.OperateDataWithEtcdErr, "get subscription from etcd failed")
		return workspace.TaskFinish
	}

	var subs []models.Subscription
	selfPath := t.R.URL.Path[len(util.RootPath):]
	for _, value := range resp.Kvs {
		u, err := url.Parse(string(value.Key))
		if err != nil {
			log.Error("Parse URL value failed.", nil)
			t.SetFirstErrorCode(util.ParseInfoErr, "parse value failed")
			return workspace.TaskFinish
		}
		subId := path.Base(u.Path)
		href := selfPath + "/" + subId
		subs = append(subs, models.Subscription{Href: href, Rel: t.SubscribeType})
	}
	if len(subs) == 0 {
		log.Errorf(nil, "Get subscription failed, subscription not exist.")
		t.SetFirstErrorCode(util.SubscriptionNotFound, "get subscription failed, subscription not exist")
		return workspace.TaskFinish
	}

	links := models.SubscriptionLinks{Self: models.Self{Href: selfPath}, Subscriptions: subs}
	subsResp := models.MecServiceMgmtApiSubscriptionLinkList{Links: links}

	t.HttpRsp = subsResp
	_, err = json.Marshal(subsResp)
	if err != nil {
		log.Error("Marshal subscription info failed.", nil)
		t.SetFirstErrorCode(util.ParseInfoErr, "marshal subscription info failed")
		return workspace.TaskFinish
	}
	log.Debugf("Response for all the app subscription information with appId %s", appInstanceId)

	return workspace.TaskFinish
}
