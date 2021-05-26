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

// Package event works for event handling
package event

import (
	"github.com/apache/servicecomb-service-center/pkg/log"
	mgr "github.com/apache/servicecomb-service-center/server/plugin"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/discovery"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/quota"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/quota/buildin"
)

func init() {
	handler := NewInstanceEtsiEventHandler()
	if handler == nil {
		log.Errorf(nil, "Failed to create the event handler for notification.")
		return
	}
	mgr.RegisterPlugin(mgr.Plugin{PName: mgr.QUOTA, Name: "buildin", New: New})
	discovery.AddEventHandler(handler)
}

// New service center plugin
func New() mgr.PluginInstance {
	buildin.InitConfigs()
	log.Infof("Quota init, service: %d, instance: %d, schema: %d/service, tag: %d/service, rule: %d/service.",
		quota.DefaultServiceQuota, quota.DefaultInstanceQuota, quota.DefaultSchemaQuota, quota.DefaultTagQuota,
		quota.DefaultRuleQuota)
	return &buildin.BuildInQuota{}
}
