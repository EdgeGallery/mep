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
	"github.com/apache/servicecomb-service-center/server/core"
	"github.com/apache/servicecomb-service-center/server/core/proto"

	"mepserver/common/arch/workspace"
	"mepserver/common/extif/backend"
	meputil "mepserver/common/util"
	"mepserver/mm5/models"
	mp1models "mepserver/mp1/models"
)

type CapabilityGet struct {
	workspace.TaskBase
	R                      *http.Request   `json:"r,in"`
	Ctx                    context.Context `json:"ctx,in"`
	QueryParam             url.Values      `json:"queryParam,in"`
	CapabilityId           string          `json:"capabilityId,in"`
	HttpRsp                interface{}     `json:"httpRsp,out"`
	HttpErrInf             *proto.Response `json:"httpErrInf,out"`
	consumerList           []models.Consumer
	serviceNameMapping     map[string]string
	serviceCategoryMapping map[mp1models.CategoryRef]string
}

func (t *CapabilityGet) OnRequest(dataInput string) workspace.TaskCode {
	log.Debug("query request arrived to fetch a capability.")

	_, ids := meputil.GetHTTPTags(t.R)

	var err = meputil.ValidateServiceID(t.CapabilityId)
	if err != nil {
		log.Error("Invalid service ID", err)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "Invalid service ID")
		return workspace.TaskFinish
	}

	serviceId := t.CapabilityId[:len(t.CapabilityId)/2]
	instanceId := t.CapabilityId[len(t.CapabilityId)/2:]
	req := &proto.GetOneInstanceRequest{
		ConsumerServiceId:  t.R.Header.Get("X-ConsumerId"),
		ProviderServiceId:  serviceId,
		ProviderInstanceId: instanceId,
		Tags:               ids,
	}

	resp, errGetOneInstance := core.InstanceAPI.GetOneInstance(t.Ctx, req)
	if errGetOneInstance != nil || resp.Instance == nil {
		log.Error("get one instance error", nil)
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "get one instance error")
		return workspace.TaskFinish
	}
	t.HttpErrInf = resp.Response
	resp.Response = nil

	capabilityId := resp.Instance.GetServiceId() + resp.Instance.GetInstanceId()
	if capabilityId != t.CapabilityId {
		log.Error("capability id miss-match", nil)
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "capability id miss-match")
		return workspace.TaskFinish
	}
	capabilityState := meputil.ActiveState
	if resp.Instance.Status == "DOWN" {
		capabilityState = meputil.InactiveState
	}

	// Build a complete list of service to its consumers applications
	errCode := t.buildConsumerList()
	if errCode != 0 {
		t.SetFirstErrorCode(workspace.ErrCode(errCode), "get consumer list error")
		return workspace.TaskFinish
	}

	// Build the capability structure
	capability := models.PlatformCapability{CapabilityId: capabilityId,
		CapabilityName: resp.Instance.Properties["serName"], Status: capabilityState, Version: resp.Instance.GetVersion(),
		Consumers: t.consumerList}
	if capability.Consumers == nil {
		capability.Consumers = make([]models.Consumer, 0)
	}

	t.HttpRsp = capability
	return workspace.TaskFinish
}

// Read and build a mapping of service ids to applications it is using
func (t *CapabilityGet) buildConsumerList() int {
	t.serviceNameMapping, t.serviceCategoryMapping = getServiceMapping()
	t.consumerList = make([]models.Consumer, 0)

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

func (t *CapabilityGet) fillConsumerListForSubscription(
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

func (t *CapabilityGet) fillConsumerData(serInstanceId string, appInstanceId string) {
	if serInstanceId == t.CapabilityId {
		t.consumerList = append(t.consumerList, models.Consumer{AppInstanceId: appInstanceId})
	}
}
