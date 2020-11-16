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

// Package path implements mep server api plans
package plans

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"github.com/apache/servicecomb-service-center/server/core/proto"

	"mepserver/common/extif/backend"

	"mepserver/common/arch/workspace"
	meputil "mepserver/common/util"
	"mepserver/mm5/models"
	mp1models "mepserver/mp1/models"
)

type DecodeCapabilityQueryReq struct {
	workspace.TaskBase
	R            *http.Request   `json:"r,in"`
	Ctx          context.Context `json:"ctx,out"`
	CapabilityId string          `json:"capabilityId,out"`
	QueryParam   url.Values      `json:"queryParam,out"`
}

func (t *DecodeCapabilityQueryReq) OnRequest(data string) workspace.TaskCode {
	err := t.getParam(t.R)
	if err != nil {
		log.Error("parameters validation failed", nil)
		return workspace.TaskFinish
	}
	return workspace.TaskFinish
}

func (t *DecodeCapabilityQueryReq) getParam(r *http.Request) error {
	queryReq, _ := meputil.GetHTTPTags(r)

	t.CapabilityId = queryReq.Get(":capabilityId")
	t.Ctx = util.SetTargetDomainProject(r.Context(), r.Header.Get("X-Domain-Name"), queryReq.Get(":project"))

	t.QueryParam = queryReq
	return nil
}

type CapabilitiesGet struct {
	workspace.TaskBase
	Ctx                    context.Context `json:"ctx,in"`
	QueryParam             url.Values      `json:"queryParam,in"`
	HttpRsp                interface{}     `json:"httpRsp,out"`
	HttpErrInf             *proto.Response `json:"httpErrInf,out"`
	consumerList           map[string][]models.Consumer
	serviceNameMapping     map[string]string
	serviceCategoryMapping map[mp1models.CategoryRef]string
}

func (t *CapabilitiesGet) OnRequest(dataInput string) workspace.TaskCode {
	log.Debug("query request arrived to fetch all capabilities.")

	capabilities := make([]models.PlatformCapability, 0)

	resp, err := meputil.FindInstanceByKey(t.QueryParam)
	if err != nil {
		if err.Error() == "null" {
			log.Info("the service is empty")
			t.HttpRsp = capabilities
			return workspace.TaskFinish
		}
		log.Error("failed to find instance request", nil)
		t.SetFirstErrorCode(meputil.SerErrServiceNotFound, "failed to find instance request")
		return workspace.TaskFinish
	}

	t.HttpErrInf = resp.Response
	resp.Response = nil

	// Build a complete list of service to its consumers applications
	errCode := t.buildConsumerList()
	if errCode != 0 {
		t.SetFirstErrorCode(workspace.ErrCode(errCode), "get consumer list error")
		return workspace.TaskFinish
	}

	for _, instance := range resp.Instances {

		capabilityId := instance.GetServiceId() + instance.GetInstanceId()
		capabilityState := meputil.ActiveState
		if instance.Status == "DOWN" {
			capabilityState = meputil.InactiveState
		}

		// Build the capability structure
		capability := models.PlatformCapability{CapabilityId: capabilityId,
			CapabilityName: instance.Properties["serName"], Status: capabilityState, Version: instance.GetVersion(),
			Consumers: t.consumerList[capabilityId]}
		if capability.Consumers == nil {
			capability.Consumers = make([]models.Consumer, 0)
		}
		capabilities = append(capabilities, capability)
	}

	t.HttpRsp = capabilities
	return workspace.TaskFinish
}

// Read and build a mapping of service ids to applications it is using
func (t *CapabilitiesGet) buildConsumerList() int {
	t.serviceNameMapping, t.serviceCategoryMapping = getServiceMapping()

	t.consumerList = make(map[string][]models.Consumer)

	subscribeKeyPath := meputil.GetSubscribeKeyPath(meputil.SerAvailabilityNotificationSubscription)

	appServiceList, errCode := backend.GetRecordsWithCompleteKeyPath(subscribeKeyPath[:len(subscribeKeyPath)-1])
	if errCode != 0 {
		log.Errorf(nil, "get entries from data-store failed")
		return errCode
	}

	for keyPath, subscriptionData := range appServiceList {
		paths := strings.Split(keyPath, "/")
		if len(paths) < 2 {
			// Minimum 2 has to be there for appInstanceId and ServiceId
			continue
		}
		appInstanceId := paths[len(paths)-2]

		subscriptionNotify := &mp1models.SerAvailabilityNotificationSubscription{}
		jsonErr := json.Unmarshal(subscriptionData, subscriptionNotify)
		if jsonErr != nil {
			log.Errorf(nil, "failed to parse the subscription entry from data-store")
			return meputil.OperateDataWithEtcdErr
		}
		t.fillConsumerListForSubscription(subscriptionNotify, appInstanceId)

	}
	return 0
}

func (t *CapabilitiesGet) fillConsumerListForSubscription(
	subscriptionNotify *mp1models.SerAvailabilityNotificationSubscription,
	appInstanceId string) {
	if len(subscriptionNotify.FilteringCriteria.SerInstanceIds) > 0 {
		for _, serInstanceId := range subscriptionNotify.FilteringCriteria.SerInstanceIds {
			t.fillConsumerData(serInstanceId, appInstanceId)
		}
	} else if len(subscriptionNotify.FilteringCriteria.SerNames) > 0 {
		for _, serName := range subscriptionNotify.FilteringCriteria.SerNames {
			if serInstanceId, found := t.serviceNameMapping[serName]; found {
				t.fillConsumerData(serInstanceId, appInstanceId)
			}
		}
	} else if len(subscriptionNotify.FilteringCriteria.SerCategories) > 0 {
		for _, serCategory := range subscriptionNotify.FilteringCriteria.SerCategories {
			if serInstanceId, found := t.serviceCategoryMapping[serCategory]; found {
				t.fillConsumerData(serInstanceId, appInstanceId)
			}
		}
	}
}

func (t *CapabilitiesGet) fillConsumerData(serInstanceId string, appInstanceId string) {
	if _, found := t.consumerList[serInstanceId]; !found {
		t.consumerList[serInstanceId] = make([]models.Consumer, 0)
	}
	t.consumerList[serInstanceId] = append(t.consumerList[serInstanceId],
		models.Consumer{AppInstanceId: appInstanceId})
}

// Get service id mapping based on filtering condition
func getServiceMapping() (map[string]string, map[mp1models.CategoryRef]string) {
	serviceNameIdMapping := make(map[string]string, 0)
	serviceCategoryMapping := make(map[mp1models.CategoryRef]string, 0)

	resp, err := meputil.FindInstanceByKey(url.Values{})
	if err != nil {
		return serviceNameIdMapping, nil
	}
	for _, instance := range resp.Instances {
		serviceNameIdMapping[instance.Properties["serName"]] = instance.GetServiceId() + instance.GetInstanceId()
		serviceCategoryMapping[mp1models.CategoryRef{
			Href:    instance.Properties["serCategory/href"],
			ID:      instance.Properties["serCategory/id"],
			Name:    instance.Properties["serCategory/name"],
			Version: instance.Properties["serCategory/version"],
		}] = instance.GetServiceId() + instance.GetInstanceId()
	}

	return serviceNameIdMapping, serviceCategoryMapping
}
