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

// Package path implements mep server api plans
package plans

import (
	"context"
	"fmt"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	svcutil "github.com/apache/servicecomb-service-center/server/service/util"
	"mepserver/common/models"

	"mepserver/common/arch/workspace"
	meputil "mepserver/common/util"
)

type UpdateInstance struct {
	workspace.TaskBase
	HttpErrInf    *proto.Response `json:"httpErrInf,out"`
	Ctx           context.Context `json:"ctx,in"`
	ServiceId     string          `json:"serviceId,in"`
	RestBody      interface{}     `json:"restBody,in"`
	HttpRsp       interface{}     `json:"httpRsp,out"`
	AppInstanceId string          `json:"appInstanceId,in"`
}

// OnRequest
func (t *UpdateInstance) OnRequest(data string) workspace.TaskCode {
	if t.ServiceId == "" {
		log.Error("service id is empty", nil)
		t.SetFirstErrorCode(meputil.RequestParamErr, "service id is empty")
		return workspace.TaskFinish
	}
	mp1Ser, ok := t.RestBody.(*models.ServiceInfo)
	if !ok {
		log.Error("request body invalid", nil)
		t.SetFirstErrorCode(meputil.RequestParamErr, "request body invalid")
		return workspace.TaskFinish
	}
	instance, err := meputil.GetServiceInstance(t.Ctx, t.ServiceId)
	if err != nil {
		log.Error("find service failed", nil)
		t.SetFirstErrorCode(meputil.SerInstanceNotFound, "find service failed")
		return workspace.TaskFinish
	}

	apiGwSerName := meputil.GetApiGwSerName(instance)

	copyInstanceRef := *instance
	req := proto.RegisterInstanceRequest{
		Instance: &copyInstanceRef,
	}
	mp1Ser.ToRegisterInstance(&req, true, apiGwSerName)
	req.Instance.Properties["appInstanceId"] = t.AppInstanceId
	if mp1Ser.LivenessInterval != 0 {
		mp1Ser.Links.Self.Href = fmt.Sprintf(meputil.LivenessPath, t.AppInstanceId,
			instance.ServiceId+instance.InstanceId)
		req.Instance.Properties["liveness"] = fmt.Sprintf(meputil.LivenessPath, t.AppInstanceId,
			instance.ServiceId+instance.InstanceId)
	}
	domainProject := util.ParseDomainProject(t.Ctx)
	centerErr := svcutil.UpdateInstance(t.Ctx, domainProject, &copyInstanceRef)
	if centerErr != nil {
		log.Error("update service failed", nil)
		t.SetFirstErrorCode(meputil.SerErrServiceUpdFailed, "update service failed")
		return workspace.TaskFinish
	}

	err = meputil.Heartbeat(t.Ctx, t.ServiceId)
	if err != nil {
		log.Error("heartbeat failed", nil)
		t.SetFirstErrorCode(meputil.SerErrServiceUpdFailed, "heartbeat failed")
		return workspace.TaskFinish
	}
	mp1Ser.SerInstanceId = instance.ServiceId + instance.InstanceId
	t.HttpRsp = mp1Ser
	return workspace.TaskFinish
}


