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

// Package uuid util package
package uuid

import (
	"crypto/sha256"
	"fmt"

	"github.com/apache/servicecomb-service-center/pkg/util"
	mgr "github.com/apache/servicecomb-service-center/server/plugin"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/uuid"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/uuid/buildin"
	"golang.org/x/net/context"
)

func init() {
	mgr.RegisterPlugin(mgr.Plugin{PName: mgr.UUID, Name: "mp1context", New: New})

}

// New plugin instance
func New() mgr.PluginInstance {
	return &Mp1ContextUUID{}
}

type Mp1ContextUUID struct {
	buildin.BuildinUUID
}

func (cu *Mp1ContextUUID) fromContext(ctx context.Context) string {
	key, ok := ctx.Value(uuid.ContextKey).(string)
	if !ok {
		return ""
	}
	return key
}

// GetServiceId to get service id
func (cu *Mp1ContextUUID) GetServiceId(ctx context.Context) string {
	content := cu.fromContext(ctx)
	if len(content) == 0 {
		return cu.BuildinUUID.GetServiceId(ctx)
	}

	shaSum := sha256.Sum256([]byte(content))
	shaHalf := shaSum[0:8]
	return fmt.Sprintf("%x", shaHalf)
}

// GetInstanceId to get instanceId
func (cu *Mp1ContextUUID) GetInstanceId(_ context.Context) string {
	shaSum := sha256.Sum256([]byte(util.GenerateUuid()))
	shaHalf := shaSum[0:8]
	return fmt.Sprintf("%x", shaHalf)
}
