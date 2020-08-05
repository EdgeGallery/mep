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

package controllers

import (
	log "github.com/sirupsen/logrus"
	"mepauth/models"
	"mepauth/util"
	"strconv"
	"time"
)

var authInfoList map[string]*models.AkSessionInfo

// Initialize auth info list
func InitAuthInfoList() {
	authInfoList = make(map[string]*models.AkSessionInfo)
}

// Verify that Ak is in block list or not
func IsAkInBlockList(ak string) bool {
	akInfo, ok := authInfoList[ak]
	if ok && akInfo.State == "UnderBlockList" {
		return true
	}
	return false
}

// Verify that Ak is in validation list or not
func IsAkInValidationList(ak string) bool {
	akInfo, ok := authInfoList[ak]
	if ok && akInfo.State == "ValidationInProgress" {
		return true
	}
	return false
}

// Start Ak validation
func StartValidatingAk(ak string) {
	akInfo := new(models.AkSessionInfo)
	akInfo.Ak = ak
	akInfo.State = "ValidationInProgress"
	akInfo.ValidateCounter++
	akInfo.ClearTimer = time.NewTimer(time.Duration(util.ValidateListClearTimer) * time.Second)
	go func() {
		_, ok := <-akInfo.ClearTimer.C
		if !ok {
			log.Error("Timer C channel closed")
		}
		delete(authInfoList, ak)
	}()
	authInfoList[ak] = akInfo
}

// Stop Ak validation
func StopValidatingAk(ak string) {
	defer func() {
		if err1 := recover(); err1 != nil {
			log.Error("panic handled:", err1)
		}
	}()

	akInfo, ok := authInfoList[ak]
	if ok {
		akInfo.State = "None"
		akInfo.ValidateCounter = 0
		ok := akInfo.ClearTimer.Stop()
		if ok {
			log.Info("Validating Timer stopped")
		}
	}
}

// Start AK block listing
func StartBlockListingAk(ak string) {
	akInfo, ok := authInfoList[ak]
	if ok {
		akInfo.State = "UnderBlockList"
		akInfo.ClearTimer = time.NewTimer(time.Duration(util.BlockListClearTimer) * time.Second)
		go func() {
			_, ok := <-akInfo.ClearTimer.C
			if !ok {
				log.Error("Timer C channel closed")
			}
			// Timer expired, so ak is safe now
			delete(authInfoList, ak)
			log.Info("BlockList timer expired. Ak " + ak + " is moving out of blockList")
		}()
	}
}

// Process Ak for block listing
func ProcessAkForBlockListing(ak string) {
	if IsAkInValidationList(ak) {
		akInfo, ok := authInfoList[ak]
		if ok {
			akInfo.ValidateCounter++
			// If received invalid Ak for 3 times move to blockList
			if akInfo.ValidateCounter >= util.ValidationCounter {
				log.Info("Received invalid signature " + strconv.FormatInt(akInfo.ValidateCounter, util.BaseVal) +
					" times, Ak " + ak + " is now under blockList")
				StopValidatingAk(ak)
				StartBlockListingAk(ak)
				return
			}
		}
	} else {
		StartValidatingAk(ak)
	}
}

// Clear Ak from block listing
func ClearAkFromBlockListing(ak string) {
	if IsAkInValidationList(ak) {
		StopValidatingAk(ak)
		delete(authInfoList, ak)
	}
}
