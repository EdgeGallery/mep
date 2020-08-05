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

// Package works for event handling
package event

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/util"
	apt "github.com/apache/servicecomb-service-center/server/core"
	"github.com/apache/servicecomb-service-center/server/core/backend"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	"github.com/apache/servicecomb-service-center/server/notify"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/discovery"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/registry"
	"github.com/apache/servicecomb-service-center/server/service/cache"
	"github.com/apache/servicecomb-service-center/server/service/metrics"
	svcutil "github.com/apache/servicecomb-service-center/server/service/util"
	"golang.org/x/net/context"

	"mepserver/mp1/models"
	util2 "mepserver/mp1/util"
)

const ConsumeIdLength = 16

type InstanceEtsiEventHandler struct {
}

// event handler type
func (h *InstanceEtsiEventHandler) Type() discovery.Type {
	return backend.INSTANCE
}

// event handling
func (h *InstanceEtsiEventHandler) OnEvent(evt discovery.KvEvent) {
	action := evt.Type
	instance, ok := evt.KV.Value.(*proto.MicroServiceInstance)
	if !ok {
		log.Error("cast to instance failed", nil)
		return
	}
	providerId, providerInstanceId, domainProject := apt.GetInfoFromInstKV(evt.KV.Key)
	if len(domainProject) == 0 {
		log.Warnf("caught [%s] instance [%s/%s] event, endpoints %v, but empty domain project string",
			action, providerId, providerInstanceId, instance.Endpoints)
		return
	}

	idx := strings.Index(domainProject, "/")
	domainName := domainProject[:idx]
	switch action {
	case proto.EVT_INIT:
		metrics.ReportInstances(domainName, 1)
		return
	case proto.EVT_CREATE:
		metrics.ReportInstances(domainName, 1)
	case proto.EVT_DELETE:
		metrics.ReportInstances(domainName, -1)
		if !apt.IsDefaultDomainProject(domainProject) {
			projectName := domainProject[idx+1:]
			svcutil.RemandInstanceQuota(util.SetDomainProject(context.Background(), domainName, projectName))
		}
	}

	if notify.NotifyCenter().Closed() {
		log.Warnf("caught [%s] instance [%s/%s] event, endpoints %v, but notify service is closed",
			action, providerId, providerInstanceId, instance.Endpoints)
		return
	}

	ctx := context.WithValue(context.WithValue(context.Background(),
		svcutil.CTX_CACHEONLY, "1"),
		svcutil.CTX_GLOBAL, "1")
	ms, err := svcutil.GetService(ctx, domainProject, providerId)
	if ms == nil || err != nil {
		log.Errorf(errors.New("failed to find instance"),
			"caught [%s] instance [%s/%s] event, endpoints %v, get cached provider's file failed",
			action, providerId, providerInstanceId, instance.Endpoints)
		return
	}

	log.Infof("caught [%s] service[%s][%s/%s/%s/%s] instance[%s] event, endpoints %v",
		action, providerId, ms.Environment, ms.AppId, ms.ServiceName, ms.Version, providerInstanceId,
		instance.Endpoints)

	consumerIds := getConsumerIds()

	log.Infof("there are %d consumerIDs, %s", len(consumerIds), consumerIds)
	PublishInstanceEvent(evt, domainProject, proto.MicroServiceToKey(domainProject, ms), consumerIds)
}

func getConsumerIds() []string {
	var consumerIds []string
	opts := []registry.PluginOp{
		registry.OpGet(registry.WithStrKey("/cse-sr/inst/files"), registry.WithPrefix()),
	}
	resp, err := backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if err != nil {
		log.Errorf(nil, "get subscription from etcd failed")
		return consumerIds
	}

	for _, kvs := range resp.Kvs {
		key := kvs.Key
		keystring := string(key)
		value := kvs.Value

		var mp1Req models.ServiceInfo
		err = json.Unmarshal(value, &mp1Req)
		if err != nil {
			log.Errorf(nil, "parse serviceInfo failed")
		}
		consumerKeys := strings.Split(keystring, "/")
		if len(consumerKeys) < 2 {
			log.Errorf(nil, "parse instance key error")
			return consumerIds
		}
		consumerId := consumerKeys[len(consumerKeys)-2]
		if len(consumerId) != ConsumeIdLength {
			log.Errorf(nil, "get consumer id failed")
			return consumerIds
		}
		if util2.StringContains(consumerIds, consumerId) == -1 {
			consumerIds = append(consumerIds, consumerId)
		}
	}
	return consumerIds
}

// new instance event handler
func NewInstanceEtsiEventHandler() *InstanceEtsiEventHandler {
	return &InstanceEtsiEventHandler{}
}

// publish instance event
func PublishInstanceEvent(evt discovery.KvEvent, domainProject string, serviceKey *proto.MicroServiceKey,
	subscribers []string) {
	defer cache.FindInstances.Remove(serviceKey)
	if len(subscribers) == 0 {
		log.Warn("the subscribers size is 0")
		return
	}
	value, ok := evt.KV.Value.(*proto.MicroServiceInstance)
	if !ok {
		log.Error("interface cast is failed", nil)
		return
	}

	response := &proto.WatchInstanceResponse{
		Response: proto.CreateResponse(proto.Response_SUCCESS, "Watch instance successfully."),
		Action:   string(evt.Type),
		Key:      serviceKey,
		Instance: value,
	}
	for _, consumerId := range subscribers {
		job := notify.NewInstanceEventWithTime(consumerId, domainProject, evt.Revision, evt.CreateAt, response)
		err := notify.NotifyCenter().Publish(job)
		if err != nil {
			log.Errorf(nil, "instance notification published failed")
		}
	}
}
