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
// @router /health-check/v1/center/tenants/:tenantId/action/start [get]
func (c *RunController) Get() {
	log.Info("Query services request received.")
	clientIp := c.Ctx.Input.IP()
	err := util.ValidateSrcAddress(clientIp)
	if err != nil {
		c.HandleLoggingForError(clientIp, util.BadRequest, util.ClientIpaddressInvalid)
		return
	}
	c.displayReceivedMsg(clientIp)
	tenantId := c.Ctx.Input.Param(":tenantId")
	accessToken := c.Ctx.Input.Header("access_token")
	log.Info("tenantId is: " + tenantId)
	log.Info("accessToken is:" + accessToken)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	//get mecList from mec-m
	client := &http.Client{Transport: tr}
	response, err := c.getMecHostList(tenantId, accessToken, client)
	if err != nil {
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
	log.Info("query mecm inventory to get hosts success")
	var k8sHostList []string
	var osHostList []string
	var requestBody RequestBody

	for _, info := range mecMInfo {
		if info.Vim == "K8S" {
			log.Info("MechostIp has:" + info.MechostIp + "and here is k8s")
			log.Info("MechostIp vim is :" + info.Vim)
			k8sHostList = append(k8sHostList, info.MechostIp)
			requestBody.MechostIpList = append(requestBody.MechostIpList, info.MechostIp)
		} else {
			log.Info("MechostIp has:" + info.MechostIp + "and here is openstack")
			log.Info("MechostIp vim is :" + info.Vim)
			osHostList = append(osHostList, info.MechostIp)
		}
	}

	var VoteMap map[string]map[string]bool
	VoteMap = make(map[string]map[string]bool)

	//after get mec list, tell every edge to get health check result from every edge
	for _, mecIp := range k8sHostList {
		log.Info("tell " + mecIp + " to get health check result from every edge")
		client := &http.Client{Transport: tr}

		requestJson, err := json.Marshal(requestBody)
		if err != nil {
			c.HandleLoggingForError(mecIp, util.StatusInternalServerError, "fail to marshal request body")
			continue
		}
		requestBody := bytes.NewReader(requestJson)
		tmpUrl := "http://" + mecIp + ":" + strconv.Itoa(util.EdgeHealthPort) + util.EdgeHealthCheck
		log.Info("temporary url is " + tmpUrl)
		response, err := client.Post(tmpUrl, "application/json", requestBody)
		if err != nil {
			continue
		}
		defer response.Body.Close()
		body, err := ioutil.ReadAll(response.Body)

		var edgeResult EdgeHealthResult
		err = json.Unmarshal(body, &edgeResult)
		if err != nil {
			c.writeErrorResponse("fail to unmarshall edge result", util.BadRequest)
			continue
		}
		checkerIp := edgeResult.CheckerIp
		if checkerIp != mecIp {
			c.HandleLoggingForError(mecIp, util.StatusInternalServerError, util.ErrCallFromEdge)
			continue
		}

		//Get VoteMap
		tmpMap := make(map[string]bool)
		for _, edgeResult := range edgeResult.EdgeCheckResult {
			tmpMap[edgeResult.CheckedIp] = edgeResult.Condition
		}
		VoteMap[checkerIp] = tmpMap
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

	//add openstack condition
	for _, osIp := range osHostList {
		ResultMap[osIp] = true
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

func (c *RunController) getMecHostList(tenantId string, accessToken string, client *http.Client) (*http.Response, error) {
	url := "https://" + util.GetLocalIp() + ":" + util.GetInventoryPort() + "/inventory/v1/tenants/" + tenantId + "/mechosts"
	log.Info("url is:" + url)
	request, _ := http.NewRequest("GET", url, nil)
	request.Header.Add("access_token", accessToken)
	response, err := client.Do(request)
	if err != nil {
		c.writeErrorResponse(util.ErrCallFromMecM, util.StatusInternalServerError)
		return nil, err
	}
	return response, nil
}
