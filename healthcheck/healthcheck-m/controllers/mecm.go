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

import "github.com/prometheus/common/log"

// Mec host information
//TODO: check here if it needs full information
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

//var HostList []string
/*
func (c *MecMController) GetNodeIpList() ([]string, error) {
	log.Info("Query services request received.")
	clientIp := c.Ctx.Input.IP()
	err := util.ValidateSrcAddress(clientIp)
	if err != nil {
		c.HandleLoggingForError(clientIp, util.BadRequest, util.ClientIpaddressInvalid)
		return nil, err
	}
	c.displayReceivedMsg(clientIp)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	response, err := client.Get(util.MecMServiceQuery)
	if err != nil {
		c.HandleLoggingForError(clientIp, util.StatusInternalServerError, util.ErrCallFromMecM)
		return nil, err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)

	var mecMInfo []MecHostInfo
	err = json.Unmarshal(body, &mecMInfo)

	if err != nil {
		c.writeErrorResponse(util.FailedToUnmarshal, util.BadRequest)
		return nil, err
	}

	for _, info := range mecMInfo {
		HostList = append(HostList, info.MechostIp)
	}

	iplistJson, _ := json.Marshal(HostList)
	_, _ = c.Ctx.ResponseWriter.Write(iplistJson)

	c.handleLoggingForSuccess(clientIp, "Query Service from mecm is successful")
	return nil, nil
}

*/
