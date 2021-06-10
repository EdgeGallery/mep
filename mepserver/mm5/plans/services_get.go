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
 *
 */

package plans

import (
	"github.com/apache/servicecomb-service-center/pkg/log"
	"mepserver/common/arch/workspace"
	"mepserver/common/models"
	meputil "mepserver/common/util"
	"mepserver/mp1"
	"net/http"
	"net/url"
)

// AllServicesReq steps to query all services registered in mep
type AllServicesReq struct {
	workspace.TaskBase
	R       *http.Request `json:"r,in"`
	HttpRsp interface{}   `json:"httpRsp,out"`
}

// OnRequest This interface is query all services registered in mep.
func (t *AllServicesReq) OnRequest(data string) workspace.TaskCode {
	services := getAllServices()

	responseInfo := models.ResponseInfo{
		Data:    services,
		RetCode: meputil.SuccessRetCode,
	}
	t.HttpRsp = responseInfo
	return workspace.TaskFinish
}

func getAllServices() []*models.ServiceInfo {
	serviceInfos := make([]*models.ServiceInfo, 0)
	findInstancesResponse, err := meputil.FindInstanceByKey(url.Values{})
	if err != nil {
		log.Errorf(nil, "Find service instance failed for retrieving the service names.")
		return serviceInfos
	}

	_, serviceInfos = mp1.Mp1CvtSrvDiscoverAll(findInstancesResponse)
	if serviceInfos == nil {
		log.Errorf(nil, "Service discovery failed.")
		return serviceInfos
	}

	return serviceInfos
}
