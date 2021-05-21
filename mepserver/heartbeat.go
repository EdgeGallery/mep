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

// Package plans implements entry point and heart beat functionalities
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/server/core"
	"github.com/apache/servicecomb-service-center/server/core/backend"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/registry"
	meputil "mepserver/common/util"
	"strconv"
	"time"
)

func availableServiceForHeartbeat() ([]*proto.MicroServiceInstance, error) {
	opts := []registry.PluginOp{
		registry.OpGet(registry.WithStrKey("/cse-sr/inst/files///"), registry.WithPrefix()),
	}
	resp, err := backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if err != nil {
		log.Errorf(nil, "Query error from etcd.")
		return nil, fmt.Errorf("query from etcd error")
	}
	var findResp []*proto.MicroServiceInstance
	for _, value := range resp.Kvs {
		var instances map[string]interface{}
		err = json.Unmarshal(value.Value, &instances)
		if err != nil {
			return nil, fmt.Errorf("string convert to instance get failed in heartbeat process")
		}
		dci := &proto.DataCenterInfo{Name: "", Region: "", AvailableZone: ""}
		instances[meputil.ServiceInfoDataCenter] = dci
		message, err := json.Marshal(&instances)
		if err != nil {
			log.Errorf(nil, "Instance convert to string failed in heartbeat process.")
			return nil, err
		}
		var ins *proto.MicroServiceInstance
		err = json.Unmarshal(message, &ins)
		if err != nil {
			log.Errorf(nil, "String convert to MicroServiceInstance failed in heartbeat process.")
			return nil, err
		}
		property := ins.Properties
		liveInterval, err := strconv.Atoi(property["livenessInterval"])
		if err != nil {
			log.Errorf(nil, "Failed to parse liveness interval.")
			return nil, err
		}
		mecState := property["mecState"]
		if liveInterval > 0 && mecState == meputil.ActiveState {
			findResp = append(findResp, ins)
		}
	}
	if len(findResp) == 0 {
		return nil, fmt.Errorf("null")
	}
	return findResp, nil
}

// periodically checks for any heart beat changes
func heartbeatProcess() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		services, _ := availableServiceForHeartbeat()
		var seconds int64
		var timeInterval int
		var err1, err2 error
		for _, svc := range services {
			seconds, err1 = strconv.ParseInt(svc.Properties["timestamp/seconds"], meputil.FormatIntBase, meputil.BitSize)
			timeInterval, err2 = strconv.Atoi(svc.Properties["livenessInterval"])
			if err1 != nil && err2 != nil {
				log.Warn("Time Interval or timestamp parse failed.")
			}
			sec := time.Now().UTC().Unix() - seconds
			if sec > int64(meputil.BufferHeartbeatInterval(timeInterval)) {
				property := svc.Properties
				property["mecState"] = meputil.SuspendedState
				req := &proto.UpdateInstancePropsRequest{
					ServiceId:  svc.ServiceId,
					InstanceId: svc.InstanceId,
					Properties: property,
				}
				_, err := core.InstanceAPI.UpdateInstanceProperties(context.Background(), req)
				log.Infof("Service(%s) send to suspended state.", svc.ServiceId)
				if err != nil {
					log.Error("Updating service properties for heartbeat failed.", nil)
				}
			}
		}
	}
}
