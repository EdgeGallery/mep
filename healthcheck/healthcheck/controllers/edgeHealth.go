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
	"github.com/prometheus/common/log"
	"healthcheck/util"
	"io/ioutil"
	"net/http"
)

// Edge Controller
type EdgeController struct {
	BaseController
}

func (c *EdgeController) LcmHealthQuery() bool{
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
	response, err := client.Get(util.LcmHealthQuery)
	if err != nil {
		c.HandleLoggingForError(clientIp, util.StatusInternalServerError, util.ErrCallFromLcm)
		return false
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		c.handleLoggingForSuccess(clientIp, "Health Query from lcm is successful")
		body, _ := ioutil.ReadAll(response.Body)
		_, _ = c.Ctx.ResponseWriter.Write(body)
		return true
	} else {
		//error code need to be changed
		c.HandleLoggingForError(clientIp, util.StatusInternalServerError, util.ErrCallFromLcm)
		return false
	}

}

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
		body, _ := ioutil.ReadAll(response.Body)
		_, _ = c.Ctx.ResponseWriter.Write(body)
		return true
	} else {
		//error code need to be changed
		c.HandleLoggingForError(clientIp, util.StatusInternalServerError, util.ErrCallFromMep)
		return false
	}

}

// @Title CheckEdge
// @Description collect mep and lcm health condition and decide this edge health
// @Success 200 ok
// @Failure 400 bad request
// @router /health-check/v1/edge/health [get]
func (c *EdgeController) Get() {
	if c.MepHealthQuery() && c.LcmHealthQuery() {
		_, _ = c.Ctx.ResponseWriter.Write([]byte("ok"))
		return
	}else {
		return
	}
}




