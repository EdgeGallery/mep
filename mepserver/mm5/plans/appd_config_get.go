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

package plans

import (
	"encoding/json"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"mepserver/common/arch/workspace"
	"mepserver/common/extif/backend"
	"mepserver/common/models"
	"mepserver/common/util"
)

type AppDConfigGet struct {
	workspace.TaskBase
	AppInstanceId string      `json:"appInstanceId,in"`
	HttpRsp       interface{} `json:"httpRsp,out"`
}

func (t *AppDConfigGet) OnRequest(inputData string) workspace.TaskCode {
	log.Debugf("query request arrived to fetch appD config for appId %s.", t.AppInstanceId)

	appDConfigEntry, err := backend.GetRecord(util.AppDConfigKeyPath + t.AppInstanceId)
	if err != 0 {
		log.Errorf(nil, "Get appD config from data-store failed.")
		t.SetFirstErrorCode(workspace.ErrCode(err), "appD config retrieval failed")
		return workspace.TaskFinish
	}

	appDInStore := &models.AppDConfig{}
	jsonErr := json.Unmarshal(appDConfigEntry, appDInStore)
	if jsonErr != nil {
		log.Errorf(nil, "Failed to parse the appd config from data-store.")
		t.SetFirstErrorCode(util.OperateDataWithEtcdErr, "parse appd config  from data-store failed")
		return workspace.TaskFinish
	}
	t.HttpRsp = appDInStore
	return workspace.TaskFinish
}
