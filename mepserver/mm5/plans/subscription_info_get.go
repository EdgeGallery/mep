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
	"mepserver/common/arch/workspace"
	"mepserver/mp1/event"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

type SubscriptionInfoReq struct {
	workspace.TaskBase
	R       *http.Request `json:"r,in"`
	HttpRsp interface{}   `json:"httpRsp,out"`
}

type SubRelation struct {
	SubscribeAppId string   `json:"subscribeAppId"`
	ServiceList    []string `json:"serviceList"`
}

// OnRequest This interface is query numbers of app subscribe other services and services subscribed by other app.
func (t *SubscriptionInfoReq) OnRequest(data string) workspace.TaskCode {
	// query subscription info from DB, all the subscription info stored in DB
	subscriptionInfos := event.GetAllSubscriberInfoFromDB()
	log.Info("New request to query subscription infos.")

	// appInstance set for all the app who subscribe services
	appSubscribeSet := make(map[string]bool)
	// services set for all the services who subscribe by app
	serviceSubscribedSet := make(map[string]bool)
	// subscribe relations
	relations := make(map[string][]string)

	for key, value := range subscriptionInfos {
		log.Debugf("Subscription path: %s.", key)
		//pos := strings.LastIndex(key, "/")
		str := strings.Split(key, "/")
		appInstanceId := str[len(str)-2]
		appSubscribeSet[appInstanceId] = true

		serviceNames := value.FilteringCriteria.SerNames
		for _, name := range serviceNames {
			serviceSubscribedSet[name] = true
		}

		serInstanceIds := value.FilteringCriteria.SerInstanceIds
		if values, ok := relations[appInstanceId]; ok {
			relations[appInstanceId] = append(values, serInstanceIds...)
		} else {
			relations[appInstanceId] = serInstanceIds
		}
	}

	// app numbers who subscribe services
	appSubscribeNum := len(appSubscribeSet)
	// service numbers who subscribed by app
	serviceSubscribedNum := len(serviceSubscribedSet)
	log.Debugf(
		"Subscription query response generated(app subscription count: %d, service subscription count: %d).",
		appSubscribeNum, serviceSubscribedNum)

	result := make(map[string]int)
	result["appSubscribeNum"] = appSubscribeNum
	result["serviceSubscribedNum"] = serviceSubscribedNum

	subscribeRes := make(map[string]interface{})
	subscribeRes["subscribeNum"] = result

	relationsList := make([]SubRelation, 0)
	for k, v := range relations {
		rel := SubRelation{
			SubscribeAppId: k,
			ServiceList:    v,
		}
		relationsList = append(relationsList, rel)
	}
	subscribeRes["subscribeRelations"] = relationsList

	t.HttpRsp = subscribeRes
	return workspace.TaskFinish
}
