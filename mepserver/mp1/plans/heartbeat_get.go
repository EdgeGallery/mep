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

package plans

import (
	"context"
	"encoding/json"
	"fmt"
	"mepserver/common/models"
	"strconv"
	"time"

	"github.com/apache/servicecomb-service-center/server/core"
	"github.com/apache/servicecomb-service-center/server/core/backend"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/registry"

	"mepserver/common/arch/workspace"
	meputil "mepserver/common/util"
	"net/http"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"github.com/apache/servicecomb-service-center/server/core/proto"
)

type GetOneDecodeHeartbeat struct {
	workspace.TaskBase
	R             *http.Request   `json:"r,in"`
	Ctx           context.Context `json:"ctx,out"`
	CoreRequest   interface{}     `json:"coreRequest,out"`
	AppInstanceId string          `json:"appInstanceId,out"`
}

func (t *GetOneDecodeHeartbeat) OnRequest(data string) workspace.TaskCode {
	var err error
	log.Infof("Received message of get heartbeat from ClientIP [%s] AppInstanceId [%s] Operation [%s] Resource [%s].",
		meputil.GetClientIp(t.R), meputil.GetAppInstanceId(t.R), meputil.GetMethod(t.R), meputil.GetHttpResourceInfo(t.R))
	t.Ctx, t.CoreRequest, err = t.getFindParam(t.R)
	if err != nil {
		log.Error("Parameters validation failed on heartbeat.", err)
		return workspace.TaskFinish
	}
	return workspace.TaskFinish

}

func (t *GetOneDecodeHeartbeat) getFindParam(r *http.Request) (context.Context, *proto.GetOneInstanceRequest, error) {
	query, ids := meputil.GetHTTPTags(r)

	var err error
	mp1SrvId := query.Get(":serviceId")
	err = meputil.ValidateServiceID(mp1SrvId)
	if err != nil {
		log.Error("Invalid service id in heart beat request.", err)
		t.SetFirstErrorCode(meputil.RequestParamErr, "Invalid service ID")
		return nil, nil, err
	}

	t.AppInstanceId = query.Get(":appInstanceId")
	err = meputil.ValidateAppInstanceIdWithHeader(t.AppInstanceId, r)
	if err != nil {
		log.Error("Validate X-AppInstanceId in heartbeat failed.", err)
		t.SetFirstErrorCode(meputil.AuthorizationValidateErr, err.Error())
		return nil, nil, err
	}

	serviceId := mp1SrvId[:len(mp1SrvId)/2]
	instanceId := mp1SrvId[len(mp1SrvId)/2:]
	req := &proto.GetOneInstanceRequest{
		ConsumerServiceId:  r.Header.Get("X-ConsumerId"),
		ProviderServiceId:  serviceId,
		ProviderInstanceId: instanceId,
		Tags:               ids,
	}

	ctx := util.SetTargetDomainProject(r.Context(), r.Header.Get("X-Domain-Name"), query.Get(":project"))
	return ctx, req, nil
}

type GetOneInstanceHeartbeat struct {
	workspace.TaskBase
	HttpErrInf    *proto.Response `json:"httpErrInf,out"`
	Ctx           context.Context `json:"ctx,in"`
	CoreRequest   interface{}     `json:"coreRequest,in"`
	HttpRsp       interface{}     `json:"httpRsp,out"`
	AppInstanceId string          `json:"appInstanceId,in"`
}

func (t *GetOneInstanceHeartbeat) OnRequest(data string) workspace.TaskCode {
	req, ok := t.CoreRequest.(*proto.GetOneInstanceRequest)
	if !ok {
		log.Error("Get instance request in get heartbeat error.", nil)
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "get instance request heartbeat error")
		return workspace.TaskFinish
	}
	resp, errGetOneInstance := core.InstanceAPI.GetOneInstance(t.Ctx, req)
	if errGetOneInstance != nil {
		log.Error("Get one instance heartbeat error.", nil)
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "get one instance heartbeat error")
		return workspace.TaskFinish
	}
	t.HttpErrInf = resp.Response
	resp.Response = nil
	mp1Rsp := &models.ServiceLivenessInfo{}
	t.filterAppInstanceId(resp.Instance)
	if resp.Instance != nil {
		if nil != mp1Rsp.FromServiceInstance(resp.Instance) {
			t.SetFirstErrorCode(meputil.SerInstanceNotFound, "heartbeat data parsing failed")
			return workspace.TaskFinish
		}
	} else {
		log.Error("Service instance id in heartbeat not found.", nil)
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "service instance id in heartbeat not found")
		return workspace.TaskFinish
	}
	if mp1Rsp.Interval == 0 {
		log.Error("Service instance is not avail the service of heartbeat.", nil)
		t.SetFirstErrorCode(meputil.HeartbeatServiceNotFound, "Invalid get heartbeat request")
		return workspace.TaskFinish
	}
	t.HttpRsp = mp1Rsp
	log.Debugf("Response for service information in heartbeat with subscriptionId %s.", req.ProviderServiceId)
	return workspace.TaskFinish
}

func (t *GetOneInstanceHeartbeat) filterAppInstanceId(inst *proto.MicroServiceInstance) {
	if inst == nil || inst.Properties == nil {
		return
	}
	if t.AppInstanceId != inst.Properties["appInstanceId"] {
		inst = nil
	}
}

func AvailableServiceForHeartbeat() ([]*proto.MicroServiceInstance, error) {
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

func HeartbeatProcess() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		services, _ := AvailableServiceForHeartbeat()
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
