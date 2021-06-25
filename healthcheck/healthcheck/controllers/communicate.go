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
	"encoding/json"
	"github.com/prometheus/common/log"
	"healthcheck/data"
	"healthcheck/util"
	"io/ioutil"
	"net/http"
	"strconv"
)

type ComController struct {
	BaseController
}

type MecHostInfo struct {
	MechostIp string `json:"mechostIp"`
}

type EdgeHealthResult struct {
	CheckerIp       string            `json:"checkerIp"`
	EdgeCheckResult []CheckedEdgeInfo `json:"edgeCheckInfo"`
}

type CheckedEdgeInfo struct {
	CheckedIp string `json:"checkedIp"`
	Condition bool   `json:"condition"`
}

var MecList []string

// @Title Post
// @Description start edge side health check
// @Success 200 ok
// @Failure 400 bad request
// @router /health-check/v1/edge/action/start [post]
func (c *ComController) Post() {
	log.Info("Query other edge nodes health situation request received.")

	clientIp := c.Ctx.Input.IP()
	err := util.ValidateSrcAddress(clientIp)
	if err != nil {
		c.HandleLoggingForError(clientIp, util.BadRequest, util.ClientIpaddressInvalid)
		return
	}
	c.displayReceivedMsg(clientIp)

	var mecInfo []MecHostInfo

	err = json.Unmarshal(c.Ctx.Input.RequestBody, &mecInfo)
	if err != nil {
		c.writeErrorResponse(util.FailedToUnmarshal, util.BadRequest)
		return
	}

	//we can use HostList in mecm.go, think it twice

	for _, info := range mecInfo {
		MecList = append(MecList, info.MechostIp)
	}

	data.EdgeList = data.EdgeList.NewNodeList(MecList)

	//TODO: can use go routine to check every edge at same time
	for _, ip := range MecList {
		if ip == util.LocalIp {
			err = data.EdgeList.SetResult(ip)
			if err != nil {
				c.HandleLoggingForError(ip, util.StatusInternalServerError, util.ErrSetResult)
			}
			continue
		}
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		client := &http.Client{Transport: tr}
		response, err := client.Get(ip + ":" + strconv.Itoa(util.EdgeHealthPort) + "/health-check/v1/edge/health") // 192.168.1.1:33666/health
		if err != nil {
			c.HandleLoggingForError(ip, util.StatusNotFound, util.ErrCallForEdge)
			continue
		}
		if response.StatusCode == http.StatusOK {
			c.handleLoggingForSuccess(ip, "Querying this edge is successful")

			err = data.EdgeList.SetResult(ip)
			if err != nil {
				c.HandleLoggingForError(ip, util.StatusInternalServerError, util.ErrSetResult)
			}
			body, _ := ioutil.ReadAll(response.Body)
			_, _ = c.Ctx.ResponseWriter.Write(body)
		} else {
			//TODO:check here if it should return error code when the checked edge is unhealthy
			c.HandleLoggingForError(ip, util.StatusInternalServerError, "this edge is unhealthy")
			err = data.EdgeList.SetBadResult(ip)
			if err != nil {
				c.HandleLoggingForError(ip, util.StatusInternalServerError, util.ErrSetResult)
			}
			body, _ := ioutil.ReadAll(response.Body)
			_, _ = c.Ctx.ResponseWriter.Write(body)
		}
		response.Body.Close()
	}

	edgeResultMap := make(map[string]map[string]bool)

	edgeResultMap[util.LocalIp] = data.EdgeList.NodeList

	var edgeResult EdgeHealthResult

	edgeResult.CheckerIp = util.LocalIp
	for checkedIp, condition := range data.EdgeList.NodeList {
		edgeHealthResult := CheckedEdgeInfo{
			CheckedIp: checkedIp,
			Condition: condition,
		}
		edgeResult.EdgeCheckResult = append(edgeResult.EdgeCheckResult, edgeHealthResult)
	}

	jsonResp, err := json.Marshal(edgeResult)
	if err != nil {
		c.HandleLoggingForError(clientIp, util.StatusInternalServerError, "fail to return upload details")
		return
	}

	_, _ = c.Ctx.ResponseWriter.Write(jsonResp)

}
