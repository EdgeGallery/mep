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
package event

import (
	"mepserver/common/models"
	"testing"

	"github.com/agiledragon/gomonkey"
	"github.com/apache/servicecomb-service-center/server/core"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	"github.com/apache/servicecomb-service-center/server/notify"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/discovery"
	svcutil "github.com/apache/servicecomb-service-center/server/service/util"
	"golang.org/x/net/context"
)

func TestNewInstanceEtsiEventHandler(t *testing.T) {
	h := NewInstanceEtsiEventHandler()

	cases := []discovery.KvEvent{
		{
			Type: proto.EVT_CREATE,
			KV: &discovery.KeyValue{
				Key: []byte(core.GenerateInstanceKey("default", "c936bdb887337c15", "a8612ca7603ad979")),
				Value: &proto.MicroServiceInstance{
					ServiceId:  "c936bdb887337c15",
					InstanceId: "a8612ca7603ad979",
					Properties: map[string]string{
						"serName":     "faceapp01",
						"transportId": "1010",
						"IsLocal":     "true",
						"mecState":    "ACTIVE",
					},
					Version: "1.0",
				},
			},
		},
		{
			Type: proto.EVT_UPDATE,
			KV: &discovery.KeyValue{
				Key: []byte(core.GenerateInstanceKey("default", "c936bdb887337c15", "a8612ca7603ad979")),
				Value: &proto.MicroServiceInstance{
					ServiceId:  "c936bdb887337c15",
					InstanceId: "a8612ca7603ad979",
					Properties: map[string]string{
						"serName":     "faceapp01",
						"transportId": "1010",
						"IsLocal":     "true",
						"mecState":    "ACTIVE",
					},
					Version: "1.0",
				},
			},
		},
		{
			Type: proto.EVT_DELETE,
			KV: &discovery.KeyValue{
				Key: []byte(core.GenerateInstanceKey("default", "c936bdb887337c15", "a8612ca7603ad979")),
				Value: &proto.MicroServiceInstance{
					ServiceId:  "c936bdb887337c15",
					InstanceId: "a8612ca7603ad979",
					Properties: map[string]string{
						"serName":     "faceapp01",
						"transportId": "1010",
						"IsLocal":     "true",
						"mecState":    "ACTIVE",
					},
					Version: "1.0",
				},
			},
		},
		{
			Type: proto.EVT_UPDATE,
			KV: &discovery.KeyValue{
				Key: []byte(core.GenerateInstanceKey("default", "c936bdb887337c15", "a8612ca7603ad979")),
				Value: &proto.MicroServiceInstance{
					ServiceId:  "c936bdb887337c15",
					InstanceId: "a8612ca7603ad979",
					Properties: map[string]string{
						"serName":     "faceapp02",
						"transportId": "1010",
						"IsLocal":     "true",
						"mecState":    "ACTIVE",
					},
					Version: "1.0",
				},
			},
		},
	}
	patch1 := gomonkey.ApplyFunc(svcutil.GetService, func(context.Context, string, string) (*proto.MicroService, error) {
		return &proto.MicroService{ServiceId: "1",
			AppId:       "2",
			ServiceName: "abcd",
		}, nil
	})
	defer patch1.Reset()
	patch2 := gomonkey.ApplyFunc(GetAllSubscriberInfoFromDB, func() map[string]*models.SerAvailabilityNotificationSubscription {
		notifyInfos := map[string]*models.SerAvailabilityNotificationSubscription{
			"/cse-sr/etsi/subscribe/5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f/83b35ec2-0afe-4563-ab25-d36f3709221d": {
				SubscriptionId:    "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f",
				SubscriptionType:  "SerAvailabilityNotification",
				CallbackReference: "http://hello:80/state/notify",
				FilteringCriteria: models.FilteringCriteria{
					SerInstanceIds: []string{
						"1220200222223334",
					},
				},
			},
			"/cse-sr/etsi/subscribe/5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f/83b35ec2-0afe-4563-ab25-d36f3709221a": {
				SubscriptionId:    "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f",
				SubscriptionType:  "SerAvailabilityNotification",
				CallbackReference: "http://hello:80/state/notify",
				FilteringCriteria: models.FilteringCriteria{
					SerNames: []string{
						"ACTIVE",
					},
				},
			},
			"/cse-sr/etsi/subscribe/5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f/83b35ec2-0afe-4563-ab25-d36f3709221b": {
				SubscriptionId:    "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f",
				SubscriptionType:  "SerAvailabilityNotification",
				CallbackReference: "http://hello:80/state/notify",
				FilteringCriteria: models.FilteringCriteria{
					SerNames: []string{
						"faceapp01",
					},
				},
			},
			"/cse-sr/etsi/subscribe/5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f/83b35ec2-0afe-4563-ab25-d36f3709221c": {
				SubscriptionId:    "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f",
				SubscriptionType:  "SerAvailabilityNotification",
				CallbackReference: "http://hello:80/state/notify",
				FilteringCriteria: models.FilteringCriteria{
					SerNames: []string{
						"faceapp02",
					},
				},
			},
		}
		return notifyInfos
	})
	defer patch2.Reset()

	for _, v := range cases {
		notify.NotifyCenter().Start()
		discovery.AddEventHandler(NewInstanceEtsiEventHandler())
		h.OnEvent(v)
		notify.NotifyCenter().Stop()
	}
}
