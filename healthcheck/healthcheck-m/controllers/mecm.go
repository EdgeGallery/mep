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

package controllers

import log "github.com/sirupsen/logrus"

// Mec host information
type MecHostInfo struct {
	MechostIp          string              `json:"mechostIp"`
	MechostName        string              `json:"mechostName"`
	ZipCode            string              `json:"zipCode"`
	City               string              `json:"city"`
	Address            string              `json:"address"`
	Affinity           string              `json:"affinity"`
	UserName           string              `json:"userName"`
	MepMIp             string              `json:"mepmIp"`
	Coordinates        string              `json:"coordinates"`
	Hwcapabilities     []MecHwCapabilities `json:"hwcapabilities"`
	Vim                string              `json:"vim"`
	ConfigUploadStatus string              `json:"configUploadStatus"`
}

// Mec hardware capabilities
type MecHwCapabilities struct {
	HwType   string `json:"hwType"`
	HwVendor string `json:"hwVendor"`
	HwModel  string `json:"hwModel"`
}

type MecMController struct {
	BaseController
}

// @Title Get
// @Description test connection is ok or not
// @Success 200 ok
// @Failure 400 bad request
// @router /health-check/v1/center/health [get]
func (c *MecMController) Get() {
	log.Info("Health Check center side connection is ok.")
	c.Ctx.WriteString("Health Check center side connection is ok.")
}
