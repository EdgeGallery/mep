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
	"bytes"
	"crypto/tls"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"healthcheck-m/util"
	"io/ioutil"
	"net/http"
	"strconv"
)

type RunController struct {
	BaseController
}

type EdgeHealthResult struct {
	CheckerIp       string            `json:"checkerIp"`
	EdgeCheckResult []CheckedEdgeInfo `json:"edgeCheckInfo"`
}

type CheckedEdgeInfo struct {
	CheckedIp string `json:"checkedIp"`
	Condition bool   `json:"condition"`
}

type RequestBody struct {
	MechostIpList []string `json:"mechostIp"`
}

// @Title Get
// @Description start total health check for this mec-m
// @Success 200 ok
// @Failure 400 bad request
// @router /health-check/v1/center/action/start [get]
func (c *RunController) Get() {
	log.Info("Query services request received.")
	clientIp := c.Ctx.Input.IP()
	err := util.ValidateSrcAddress(clientIp)
	if err != nil {
		c.HandleLoggingForError(clientIp, util.BadRequest, util.ClientIpaddressInvalid)
		return
	}
	c.displayReceivedMsg(clientIp)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	//get mecList from mec-m
	client := &http.Client{Transport: tr}
	response, err := client.Get(util.MecMServiceQuery)

	if err != nil {
		c.HandleLoggingForError(clientIp, util.StatusInternalServerError, util.ErrCallFromMecM)
		return
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)

	var mecMInfo []MecHostInfo
	err = json.Unmarshal(body, &mecMInfo)

	if err != nil {
		c.writeErrorResponse(util.FailedToUnmarshal, util.BadRequest)
		return
	}

	var hostList []string
	var requestBody RequestBody

	for _, info := range mecMInfo {
		hostList = append(hostList, info.MechostIp)
		requestBody.MechostIpList = append(requestBody.MechostIpList, info.MechostIp)
	}

	var VoteMap map[string]map[string]bool
	VoteMap = make(map[string]map[string]bool)

	//after get mec list, tell every edge to get health check result from every edge
	for _, mecIp := range hostList {
		client := &http.Client{Transport: tr}

		requestJson, err := json.Marshal(requestBody)
		requestBody := bytes.NewReader(requestJson)
		tmpUrl := "http://" + mecIp + ":" + strconv.Itoa(util.EdgeHealthPort) + util.EdgeHealthCheck

		//	response, err := client.Get(tmpUrl )
		response, err := client.Post(tmpUrl, "application/json", requestBody)
		if err != nil {
			//	c.HandleLoggingForError(clientIp, util.StatusInternalServerError, util.ErrCallFromMecM)
			continue
		}
		defer response.Body.Close()
		body, err := ioutil.ReadAll(response.Body)

		var edgeResult EdgeHealthResult
		err = json.Unmarshal(body, &edgeResult)
		if err != nil {
			c.writeErrorResponse(util.FailedToUnmarshal, util.BadRequest)
			continue
		}
		checkerIp := edgeResult.CheckerIp
		if checkerIp != mecIp {
			c.HandleLoggingForError(mecIp, util.StatusInternalServerError, util.ErrCallFromEdge)
			continue
		}

		//TODO: determine if it needs to check checkerIp map already have or not

		//Get VoteMap
		_, ok := VoteMap[checkerIp]
		if !ok {
			tmpMap := make(map[string]bool)
			for _, edgeResult := range edgeResult.EdgeCheckResult {
				tmpMap[edgeResult.CheckedIp] = edgeResult.Condition
			}
			VoteMap[checkerIp] = tmpMap
		} else {
			tmpMap := make(map[string]bool)
			for _, edgeResult := range edgeResult.EdgeCheckResult {
				tmpMap[edgeResult.CheckedIp] = edgeResult.Condition
			}
			VoteMap[checkerIp] = tmpMap
		}
	}

	//totalNum := len(hostList)
	VoteCountMap := make(map[string]int)
	ResultMap := make(map[string]bool)

	for _, edgeMap := range VoteMap {
		for checkedIp, condition := range edgeMap {
			if condition { //if checked ip edge is true, then vote count++
				_, ok := VoteCountMap[checkedIp] //getOrDefault in java
				if !ok {
					VoteCountMap[checkedIp] = 1
				} else {
					VoteCountMap[checkedIp]++
				}
			} else { //checked edge is false
				_, ok := VoteCountMap[checkedIp]
				if !ok {
					VoteCountMap[checkedIp] = 0
				} else {
					continue
				}
			}
		}
	}

	//here is how we vote
	for checkedIp, voteNum := range VoteCountMap {
		if voteNum >= 1 {
			ResultMap[checkedIp] = true
		} else {
			ResultMap[checkedIp] = false
		}
	}

	var result []CheckedEdgeInfo
	for checkedIp, condition := range ResultMap {
		edgeInfo := CheckedEdgeInfo{
			CheckedIp: checkedIp,
			Condition: condition,
		}
		result = append(result, edgeInfo)
	}

	resp, err := json.Marshal(result)

	if err != nil {
		c.HandleLoggingForError(clientIp, util.StatusInternalServerError, "fail to marshal details")
		return
	}

	_, _ = c.Ctx.ResponseWriter.Write(resp)
}
