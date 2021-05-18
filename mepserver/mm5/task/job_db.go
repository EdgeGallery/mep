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

package task

import (
	"encoding/json"
	"fmt"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"mepserver/common/extif/backend"
	"mepserver/common/models"
	"mepserver/common/util"
)

type appDJobDB struct {
	appInstanceId string
	appDConfig    *models.AppDConfig
}

func newAppDJobDB(appInstanceId string) *appDJobDB {
	jobsEntry, errCode := backend.GetRecord(util.AppDLCMJobsPath + appInstanceId)
	if errCode != 0 {
		log.Errorf(nil, "Retrieve jobs from temp-cache on data-store failed.")
		return nil
	}
	appDConfig := &models.AppDConfig{}
	err := json.Unmarshal(jobsEntry, appDConfig)
	if err != nil {
		log.Errorf(nil, "Failed to parse the appd config from data-store.")
		return nil
	}
	return &appDJobDB{appInstanceId, appDConfig}
}

func (a *appDJobDB) deleteEntry() error {
	errCode := backend.DeleteRecord(util.AppDLCMJobsPath + a.appInstanceId)
	if errCode != 0 {
		log.Errorf(nil, "Delete jobs from temp-cache on data-store failed.")
		return fmt.Errorf("error: delete entry failed")
	}
	return nil
}
