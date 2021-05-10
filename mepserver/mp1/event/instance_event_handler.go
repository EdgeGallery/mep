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

// Package event handling function
package event

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"mepserver/common/models"
	"strconv"
	"strings"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/util"
	apt "github.com/apache/servicecomb-service-center/server/core"
	"github.com/apache/servicecomb-service-center/server/core/backend"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	"github.com/apache/servicecomb-service-center/server/notify"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/discovery"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/registry"
	"github.com/apache/servicecomb-service-center/server/service/metrics"
	svcutil "github.com/apache/servicecomb-service-center/server/service/util"
	"golang.org/x/net/context"

	util2 "mepserver/common/util"
)

//ConsumeIDLength Consumer Id Length
const ConsumeIDLength = 16

//InstanceEtsiEventHandler notification handler
type InstanceEtsiEventHandler struct {
	tlsCfg *tls.Config
}

//Type event handler type
func (h *InstanceEtsiEventHandler) Type() discovery.Type {
	return backend.INSTANCE
}

//OnEvent event handling
func (h *InstanceEtsiEventHandler) OnEvent(evt discovery.KvEvent) {
	action := evt.Type
	instance, ok := evt.KV.Value.(*proto.MicroServiceInstance)
	if !ok {
		log.Error("cast to instance failed", nil)
		return
	}
	log.Infof("receive event %s", action)
	providerID, providerInstanceID, domainProject := apt.GetInfoFromInstKV(evt.KV.Key)
	if len(domainProject) == 0 {
		log.Warnf("caught [%s] instance [%s/%s] event, endpoints %v, but empty domain project string",
			action, providerID, providerInstanceID, instance.Endpoints)
		return
	}

	idx := strings.Index(domainProject, "/")
	if idx == -1 {
		log.Error("get domain name failed", nil)
		return
	}
	domainName := domainProject[:idx]
	switch action {
	case proto.EVT_INIT:
		metrics.ReportInstances(domainName, 1)
		return
	case proto.EVT_CREATE:
		metrics.ReportInstances(domainName, 1)
	case proto.EVT_UPDATE:
		metrics.ReportInstances(domainName, 0)
	case proto.EVT_DELETE:
		metrics.ReportInstances(domainName, -1)
		if !apt.IsDefaultDomainProject(domainProject) {
			projectName := domainProject[idx+1:]
			svcutil.RemandInstanceQuota(util.SetDomainProject(context.Background(), domainName, projectName))
		}
	}

	if notify.NotifyCenter().Closed() {
		log.Warnf("caught [%s] instance [%s/%s] event, endpoints %v, but notify service is closed",
			action, providerID, providerInstanceID, instance.Endpoints)
		return
	}

	ctx := context.WithValue(context.WithValue(context.Background(),
		svcutil.CTX_CACHEONLY, "1"),
		svcutil.CTX_GLOBAL, "1")
	ms, err := svcutil.GetService(ctx, domainProject, providerID)
	if ms == nil || err != nil {
		log.Errorf(errors.New("failed to find instance"),
			"caught [%s] instance [%s/%s] event, endpoints %v, get cached provider's file failed",
			action, providerID, providerInstanceID, instance.Endpoints)
		return
	}

	log.Infof("caught [%s] service[%s][%s/%s/%s/%s] instance[%s] event, endpoints %v, domainproject %s",
		action, providerID, ms.Environment, ms.AppId, ms.ServiceName, ms.Version, providerInstanceID,
		instance.Endpoints, domainProject)

	h.sendRestMessageToApp(instance, string(action))
}

// SendRestMessageToApp sendRestMessageToApp
func (h *InstanceEtsiEventHandler) sendRestMessageToApp(instance *proto.MicroServiceInstance, action string) {
	instanceID := instance.ServiceId + instance.InstanceId
	serName := instance.Properties["serName"]
	isLocal := instance.Properties["isLocal"]
	state := instance.Properties["mecState"]
	serCategory := models.CategoryRef{
		Href:    instance.Properties["serCategory/href"],
		ID:      instance.Properties["serCategory/id"],
		Name:    instance.Properties["serCategory/name"],
		Version: instance.Properties["serCategory/version"],
	}
	callBackUris := getCallBackUris(instanceID, serName, isLocal, state, serCategory)
	if len(callBackUris) == 0 {
		log.Info("callback uris is empty")
		return
	}

	h.doSend(action, instance, callBackUris)
}

func (h *InstanceEtsiEventHandler) doSend(action string, instance *proto.MicroServiceInstance, callbackUris map[string]string) {
	var notificationInfo models.ServiceAvailabilityNotification
	notificationInfo.ServiceReferences = make([]models.ServiceReferences, 1, 1)
	notificationInfo.NotificationType = "SerAvailabilityNotification"
	notificationInfo.ServiceReferences[0].SerName = instance.Properties["serName"]
	notificationInfo.ServiceReferences[0].SerInstanceID = instance.ServiceId + instance.InstanceId
	notificationInfo.ServiceReferences[0].State = instance.Properties["mecState"]
	href := "/mec_service_mgmt/v1/services/" + instance.ServiceId + instance.InstanceId

	//Currently we cant identify only state changed. So Only attributes change will be supported
	if action == "CREATE" {
		notificationInfo.ServiceReferences[0].ChangeType = "ADDED"
		notificationInfo.ServiceReferences[0].Link.Href = href
	} else if action == "DELETE" {
		notificationInfo.ServiceReferences[0].ChangeType = "REMOVED"
	} else if action == "UPDATE" {
		notificationInfo.ServiceReferences[0].ChangeType = "ATTRIBUTES_CHANGED"
		notificationInfo.ServiceReferences[0].Link.Href = href
	}
	for subscription, callBackURI := range callbackUris {
		h.sendMsg(notificationInfo, callBackURI, subscription)
	}
}

//SendMsg send message
func (h *InstanceEtsiEventHandler) sendMsg(notificationInfo models.ServiceAvailabilityNotification,
	callBackURI string, susbcription string) {
	log.Infof("subscription key %s uri %s", susbcription, callBackURI)
	app := strings.Split(susbcription, "/")
	subscriptionID := app[len(app)-1]
	appInstID := app[len(app)-2]
	location := fmt.Sprintf("%s/applications/%s/subscriptions/%s", util2.MecServicePath, appInstID,
		subscriptionID)
	notificationInfo.Links.Susbcription.Href = location
	notificationInfoJSON, err := json.Marshal(notificationInfo)
	if err != nil {
		return
	}

	err = util2.SendPostRequest(callBackURI, notificationInfoJSON, h.tlsCfg)
	if err != nil {
		log.Error("failed to send notification", nil)
	}
}

func getCallBackUris(instanceID string, serName string, isLocal string, state string, serCategory models.CategoryRef) map[string]string {
	notifyInfos := GetAllSubscriberInfoFromDB()
	callBackUris := make(map[string]string, len(notifyInfos))

	for subKey, notifyInfo := range notifyInfos {
		callBackURI := notifyInfo.CallbackReference
		filter := notifyInfo.FilteringCriteria
		if isInFilter(filter, instanceID, serName, isLocal, state, serCategory) {
			callBackUris[subKey] = callBackURI
		}
	}

	return callBackUris
}
func isInFilter(filter models.FilteringCriteria, instanceID string, serName string, isLocal string, state string, serCategory models.CategoryRef) bool {
	localFilter := false
	stateFilter := false
	if strconv.FormatBool(filter.IsLocal) == isLocal {
		localFilter = true
	}
	state = stateConvert(state)
	if filter.States == nil || len(filter.States) == 0 {
		stateFilter = true
	} else if util2.StringContains(filter.States, state) != -1 {
		stateFilter = true
	}

	localFilter = true
	if !localFilter || !stateFilter {
		return false
	}
	if isAllFilterEmpty(filter) {
		return true
	}
	return isFilterContain(filter, instanceID, serName, serCategory)
}

func stateConvert(state string) string {
	log.Infof("state %s", state)
	return state
}
func isAllFilterEmpty(filter models.FilteringCriteria) bool {
	if len(filter.SerNames) == 0 && len(filter.SerInstanceIds) == 0 && len(filter.SerCategories) == 0 && len(filter.States) == 0 && !filter.IsLocal {
		return true
	}
	return false
}
func isFilterContain(filter models.FilteringCriteria, instanceID string, serName string, serCategory models.CategoryRef) bool {
	if (len(filter.SerInstanceIds) != 0 && util2.StringContains(filter.SerInstanceIds, instanceID) != -1) ||
		(len(filter.SerNames) != 0 && util2.StringContains(filter.SerNames, serName) != -1) ||
		(len(filter.SerCategories) != 0 && isServiceCategoryMatched(filter.SerCategories, serCategory)) {
		return true
	}
	return false
}

func isServiceCategoryMatched(serCategories []models.CategoryRef, serCategory models.CategoryRef) bool {
	for _, tempSerCat := range serCategories {
		if tempSerCat.Href == serCategory.Href &&
			tempSerCat.ID == serCategory.ID &&
			tempSerCat.Name == serCategory.Name &&
			tempSerCat.Version == serCategory.Version {
			return true
		}
	}
	return false
}

//GetAllSubscriberInfoFromDB GetAllSubscriberInfoFromDB
func GetAllSubscriberInfoFromDB() map[string]*models.SerAvailabilityNotificationSubscription {
	subscribeKeyPath := util2.GetSubscribeKeyPath(util2.SerAvailabilityNotificationSubscription)
	notifyInfos := make(map[string]*models.SerAvailabilityNotificationSubscription, 1000)
	opts := []registry.PluginOp{
		registry.OpGet(registry.WithStrKey(subscribeKeyPath), registry.WithPrefix()),
	}
	resp, err := backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if err != nil {
		log.Errorf(nil, "get subscription from etcd failed")
		return nil
	}
	for _, kvs := range resp.Kvs {
		notifyInfo := &models.SerAvailabilityNotificationSubscription{}
		if err := json.Unmarshal(kvs.Value, notifyInfo); err != nil {
			continue
		}
		notifyInfos[string(kvs.Key)] = notifyInfo
	}
	return notifyInfos
}

//NewInstanceEtsiEventHandler new instance event handler
func NewInstanceEtsiEventHandler() *InstanceEtsiEventHandler {
	config, err := util2.TLSConfig(util2.ApiGwCaCertName, true)
	if err != nil {
		return nil
	}
	return &InstanceEtsiEventHandler{config}
}
