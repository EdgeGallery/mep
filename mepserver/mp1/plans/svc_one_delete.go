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

// Package plans implements mep server api
package plans

import (
	"context"
	"strings"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/server/core"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	scerr "github.com/apache/servicecomb-service-center/server/error"

	"mepserver/common/arch/workspace"
	"mepserver/common/util"
)

// DeleteService step to delete a service registration
type DeleteService struct {
	HttpErrInf *proto.Response `json:"httpErrInf,out"`
	workspace.TaskBase
	Ctx       context.Context `json:"ctx,in"`
	ServiceId string          `json:"serviceId,in"`
	HttpRsp   interface{}     `json:"httpRsp,out"`
}

// OnRequest handles service delete request
func (t *DeleteService) OnRequest(data string) workspace.TaskCode {
	if t.ServiceId == "" {
		log.Error("Service id empty in service delete request.", nil)
		t.SetFirstErrorCode(util.SerErrServiceDelFailed, "param is empty")
		return workspace.TaskFinish
	}
	instance, err := util.GetServiceInstance(t.Ctx, t.ServiceId)
	if err != nil {
		log.Error("Find service on update failed.", nil)
		t.SetFirstErrorCode(util.SerInstanceNotFound, "find service failed")
		return workspace.TaskFinish
	}
	for k, v := range instance.Properties {
		if !strings.HasPrefix(k, util.EndPointPropPrefix) {
			continue
		}
		delete(instance.Properties, k)
		util.ApiGWInterface.CleanUpApiGwEntry(v)
	}

	serviceID := t.ServiceId[:len(t.ServiceId)/2]
	log.Debugf("Delete request arrived for service with serviceId %s.", serviceID)
	instanceID := t.ServiceId[len(t.ServiceId)/2:]
	req := &proto.UnregisterInstanceRequest{
		ServiceId:  serviceID,
		InstanceId: instanceID,
	}
	resp, err := core.InstanceAPI.Unregister(t.Ctx, req)
	if err != nil {
		log.Errorf(nil, "Service(id: %s) delete failed.", req.ServiceId)
		t.SetFirstErrorCode(util.SerErrServiceInstanceFailed, "service delete failed")
		return workspace.TaskFinish
	}
	if resp != nil && resp.Response.Code == scerr.ErrInstanceNotExists {
		log.Error("Instance not found on service delete request.", nil)
		t.SetFirstErrorCode(util.SerInstanceNotFound, "instance not found")
		return workspace.TaskFinish
	}
	t.HttpErrInf = resp.Response
	t.HttpRsp = ""
	log.Debugf("Service with serviceId %s is deleted successfully.", serviceID)
	return workspace.TaskFinish
}
