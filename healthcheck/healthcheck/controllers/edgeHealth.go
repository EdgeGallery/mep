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

import (
	"crypto/tls"
	log "github.com/sirupsen/logrus"
	"healthcheck/util"
	"net/http"
	"strconv"
)

// Edge Controller
type EdgeController struct {
	BaseController
}

// @Title LcmHealthQuery
// @Description collect lcm health condition
// @Success true
// @Failure false
func (c *EdgeController) LcmHealthQuery() bool {
	log.Info("Lcm Health Query request received.")
	clientIp := c.Ctx.Input.IP()
	err := util.ValidateSrcAddress(clientIp)
	if err != nil {
		c.HandleLoggingForError(clientIp, util.BadRequest, util.ClientIpaddressInvalid)
		return false
	}
	c.displayReceivedMsg(clientIp)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	LcmHealthQueryUrl := "https://" + util.GetLocalIp() + ":" + strconv.Itoa(util.LcmPort) + util.LcmHealthUri

	response, err := client.Get(LcmHealthQueryUrl)
	if err != nil {
		c.HandleLoggingForError(clientIp, util.StatusInternalServerError, util.ErrCallFromLcm)
		return false
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		c.handleLoggingForSuccess(clientIp, "Health Query from lcm is successful")
		return true
	} else {
		//error code need to be changed
		c.HandleLoggingForError(clientIp, util.StatusInternalServerError, util.ErrCallFromLcm)
		return false
	}

}

// @Title MepHealthQuery
// @Description collect mep health condition
// @Success true
// @Failure false
func (c *EdgeController) MepHealthQuery() bool{
	log.Info("Mep Health Query request received.")
	clientIp := c.Ctx.Input.IP()
	err := util.ValidateSrcAddress(clientIp)
	if err != nil {
		c.HandleLoggingForError(clientIp, util.BadRequest, util.ClientIpaddressInvalid)
		return false
	}
	c.displayReceivedMsg(clientIp)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	response, err := client.Get(util.MepHealthQuery)
	if err != nil {
		c.HandleLoggingForError(clientIp, util.StatusInternalServerError, util.ErrCallFromMep)
		return false
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		c.handleLoggingForSuccess(clientIp, "Health Query from mep is successful")
		return true
	} else {
		//error code need to be changed
		c.HandleLoggingForError(clientIp, util.StatusInternalServerError, util.ErrCallFromMep)
		return false
	}

}

// @Title Get
// @Description collect mep and lcm health condition and decide this edge health
// @Success 200 ok
// @Failure 500 StatusInternalServerError
// @router /health-check/v1/edge/health [get]
func (c *EdgeController) Get() {
	if   c.LcmHealthQuery() || c.MepHealthQuery(){
		return
	} else {
		c.writeErrorResponse(util.ErrCallForEdge,util.StatusInternalServerError)
		return
	}
}
