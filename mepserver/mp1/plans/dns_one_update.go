/*
 * Copyright 2020-2021 Huawei Technologies Co., Ltd.
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

// Package path implements mep server api plans
package plans

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"mepserver/common/extif/dataplane"
	"mepserver/common/models"
	"net/http"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/util"

	"mepserver/common/extif/backend"
	"mepserver/common/extif/dns"

	"mepserver/common/arch/workspace"
	meputil "mepserver/common/util"
)

type DecodeDnsRestReq struct {
	workspace.TaskBase
	R             *http.Request   `json:"r,in"`
	Ctx           context.Context `json:"ctx,out"`
	AppInstanceId string          `json:"appInstanceId,out"`
	DNSRuleId     string          `json:"dnsRuleId,out"`
	RestBody      interface{}     `json:"restBody,out"`
}

func (t *DecodeDnsRestReq) OnRequest(data string) workspace.TaskCode {
	err := t.getParam(t.R)
	if err != nil {
		log.Error("Parameters validation failed.", err)
		return workspace.TaskFinish
	}
	err = t.parseBody(t.R)
	if err != nil {
		log.Error("Parse rest body failed.", nil)
	}
	return workspace.TaskFinish
}

func (t *DecodeDnsRestReq) parseBody(r *http.Request) error {
	if t.RestBody == nil {
		return nil
	}
	msg, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("Dns request body read failed.", nil)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "read request body error")
		return err
	}
	if len(msg) > meputil.RequestBodyLength {
		err = errors.New("request body too large")
		log.Errorf(err, "Request body too large %d.", len(msg))
		t.SetFirstErrorCode(meputil.RequestParamErr, "request body too large")
		return err
	}

	newMsg, err := t.checkParam(msg)
	if err != nil {
		log.Error("Check param failed.", nil)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "check Param failed")
		return err
	}

	err = json.Unmarshal(newMsg, t.RestBody)
	if err != nil {
		log.Errorf(nil, "Dns request body unmarshalling failed.")
		t.SetFirstErrorCode(meputil.ParseInfoErr, "unmarshal request body error")
		return errors.New("json unmarshalling failed")
	}
	return nil
}

func (t *DecodeDnsRestReq) checkParam(msg []byte) ([]byte, error) {

	var temp map[string]interface{}
	err := json.Unmarshal(msg, &temp)
	if err != nil {
		return nil, errors.New("unmarshal msg error")
	}

	meputil.SetMapValue(temp, "consumedLocalOnly", true)
	meputil.SetMapValue(temp, "isLocal", true)
	meputil.SetMapValue(temp, "scopeOfLocality", "MEC_HOST")

	msg, err = json.Marshal(&temp)
	if err != nil {
		return nil, errors.New("marshal map to json error")
	}

	return msg, nil
}

func (t *DecodeDnsRestReq) WithBody(body interface{}) *DecodeDnsRestReq {
	t.RestBody = body
	return t
}

func (t *DecodeDnsRestReq) getParam(r *http.Request) error {
	query, _ := meputil.GetHTTPTags(r)

	var err error

	t.AppInstanceId = query.Get(":appInstanceId")
	if len(t.AppInstanceId) == 0 {
		err = fmt.Errorf("invalid app instance id")
		t.SetFirstErrorCode(meputil.AuthorizationValidateErr, err.Error())
		return err
	}
	if err = meputil.ValidateAppInstanceIdWithHeader(t.AppInstanceId, r); err != nil {
		log.Error("Validate X-AppinstanceId failed.", err)
		t.SetFirstErrorCode(meputil.AuthorizationValidateErr, err.Error())
		return err
	}
	err = meputil.ValidateUUID(t.AppInstanceId)
	if err != nil {
		t.SetFirstErrorCode(meputil.RequestParamErr, "app Instance ID validation failed, invalid uuid")
		return err
	}

	t.DNSRuleId = query.Get(":dnsRuleId")
	if len(t.DNSRuleId) > meputil.MaxDNSRuleIdLength {
		err = fmt.Errorf("dns rule id validation failed, invalid length")
		t.SetFirstErrorCode(meputil.RequestParamErr, err.Error())
		return err
	}
	t.Ctx = util.SetTargetDomainProject(r.Context(), r.Header.Get("X-Domain-Name"), query.Get(":project"))
	return nil
}

type DNSRuleUpdate struct {
	workspace.TaskBase
	R             *http.Request       `json:"r,in"`
	W             http.ResponseWriter `json:"w,in"`
	RestBody      interface{}         `json:"restBody,in"`
	AppInstanceId string              `json:"appInstanceId,in"`
	DNSRuleId     string              `json:"dnsRuleId,in"`
	HttpRsp       interface{}         `json:"httpRsp,out"`
	dnsAgent      dns.DNSAgent
	dataPlane     dataplane.DataPlane
	AppName       string
}

func (t *DNSRuleUpdate) WithDNSAgent(dnsAgent dns.DNSAgent) *DNSRuleUpdate {
	t.dnsAgent = dnsAgent
	return t
}

func (t *DNSRuleUpdate) WithDataPlane(dataPlane dataplane.DataPlane) *DNSRuleUpdate {
	t.dataPlane = dataPlane
	return t
}

func (t *DNSRuleUpdate) OnRequest(data string) workspace.TaskCode {

	log.Debugf("update request arrived for dns rule %s and appId %s.", t.DNSRuleId, t.AppInstanceId)

	// Read dns entry from data-store
	appDConfigEntry, errCode := backend.GetRecord(meputil.AppDConfigKeyPath + t.AppInstanceId)
	if errCode != 0 {
		log.Errorf(errors.New("get operation failed"),
			"Dns rule retrieval from data-store failed on update request.")
		t.SetFirstErrorCode(workspace.ErrCode(errCode), "dns rule retrieval failed")
		return workspace.TaskFinish
	}

	appDInStore := models.AppDConfig{}
	var dnsOnStore *dataplane.DNSRule
	var ruleIndex int
	if appDConfigEntry != nil {
		jsonErr := json.Unmarshal(appDConfigEntry, &appDInStore)
		if jsonErr != nil {
			log.Errorf(errors.New("json parse failed"),
				"Failed to parse the dns entry from data-store on update request.")
			t.SetFirstErrorCode(meputil.OperateDataWithEtcdErr, "parse dns rules failed")
			return workspace.TaskFinish
		}
		for i, rule := range appDInStore.AppDNSRule {
			if rule.DNSRuleID == t.DNSRuleId {
				dnsOnStore = &rule
				ruleIndex = i
				break
			}
		}
	}
	if dnsOnStore == nil {
		log.Error("Requested dns rule id doesn't exists.", nil)
		t.SetFirstErrorCode(meputil.SubscriptionNotFound, "dns rule retrieval failed")
		return workspace.TaskFinish
	}

	t.AppName = appDInStore.AppName

	dataOnStoreBytes, err := json.Marshal(dnsOnStore)
	if err != nil {
		log.Errorf(err, "Failed to parse the dns entry from data-store on update request.")
		t.SetFirstErrorCode(meputil.OperateDataWithEtcdErr, "parse dns rules failed")
		return workspace.TaskFinish
	}

	// Check for E-Tags precondition. More details could be found here: https://tools.ietf.org/html/rfc7232#section-2.3
	ifMatchTag := t.R.Header.Get("If-Match")
	if len(ifMatchTag) != 0 && ifMatchTag != meputil.GenerateStrongETag(dataOnStoreBytes) {
		t.SetFirstErrorCode(meputil.EtagMissMatchErr, "e-tag miss-match")
		return workspace.TaskFinish
	}

	errCode, errString := t.updateDnsRecordToRemoteServer(appDInStore, ruleIndex, dnsOnStore, dataOnStoreBytes)
	if errCode > 0 {
		t.SetFirstErrorCode(workspace.ErrCode(errCode), errString)
		return workspace.TaskFinish
	}

	//if errCode == -1 {
	//	return workspace.TaskFinish
	//}
	return workspace.TaskFinish
}

func (t *DNSRuleUpdate) validateInputs(dnsConfigInput *dataplane.DNSRule,
	dnsOnStore *dataplane.DNSRule) (errorString string, errorCode int) {

	if len(dnsConfigInput.DNSRuleID) != 0 && t.DNSRuleId != dnsConfigInput.DNSRuleID {
		return "dns identifier miss-match", meputil.ParseInfoErr
	}

	if dnsConfigInput.DomainName != dnsOnStore.DomainName ||
		dnsConfigInput.IPAddress != dnsOnStore.IPAddress ||
		dnsConfigInput.IPAddressType != dnsOnStore.IPAddressType ||
		dnsConfigInput.TTL != dnsOnStore.TTL {
		return "update supported only for state", meputil.ParseInfoErr
	}

	if dnsConfigInput.State != meputil.ActiveState && dnsConfigInput.State != meputil.InactiveState {
		return "invalid dns state input", meputil.ParseInfoErr
	}

	return "", 0
}

// Update the dns modification request to the remote dns server
func (t *DNSRuleUpdate) updateDnsRecordToRemoteServer(appDConfig models.AppDConfig, ruleIndex int,
	dnsOnStore *dataplane.DNSRule, dataOnStoreBytes []byte) (int, string) {

	// E-Tag check need to be done before parsing, hence added parsing here
	dnsConfigInPut, ok := t.RestBody.(*dataplane.DNSRule)
	if !ok {
		return meputil.ParseInfoErr, "input parsing failed"
	}

	errorString, errorCode := t.validateInputs(dnsConfigInPut, dnsOnStore)
	if errorCode != 0 {
		return 1, errorString
	}

	if dnsOnStore.State == dnsConfigInPut.State {
		t.W.Header().Set("ETag", meputil.GenerateStrongETag(dataOnStoreBytes))
		t.HttpRsp = dnsOnStore
		return -1, ""
	}

	var err error
	// Backing up state data for reconfigure in case of failure
	oldState := dnsOnStore.State

	dnsOnStore.State = dnsConfigInPut.State
	appDConfig.AppDNSRule[ruleIndex].State = dnsConfigInPut.State
	errCode, errString := t.updateDnsRecordOnDataStore(appDConfig)
	if errCode != 0 {
		return errCode, errString
	}

	rrType := meputil.RRTypeA
	if dnsOnStore.IPAddressType == meputil.IPv6Type {
		rrType = meputil.RRTypeAAAA
	}

	// Update the DNS server as per the new configurations
	if dnsConfigInPut.State == meputil.ActiveState {
		err = t.dnsAgent.SetResourceRecordTypeA(dnsOnStore.DomainName, rrType, meputil.RRClassIN,
			[]string{dnsOnStore.IPAddress}, dnsOnStore.TTL)
	} else {
		err = t.dnsAgent.DeleteResourceRecordTypeA(dnsOnStore.DomainName, rrType)
	}
	if err != nil {
		log.Errorf(err, "Dns rule(app-id: %s, dns-rule-id: %s) update fail on dns server.",
			t.AppInstanceId, t.DNSRuleId)

		// Revert the update in the data store in failure case
		appDConfig.AppDNSRule[ruleIndex].State = oldState
		t.revertEntryFromDB(&appDConfig)

		return meputil.RemoteServerErr, "failed to apply the dns modification"
	}

	appInfo := dataplane.ApplicationInfo{
		Id:   t.AppInstanceId,
		Name: t.AppName,
	}

	err = t.updateDNSToDataPlane(dnsConfigInPut, dnsOnStore, appInfo, rrType)

	if err != nil {
		log.Errorf(err, "Dns rule(app-id: %s, dns-rule-id: %s) update fail on data-plane.",
			t.AppInstanceId, t.DNSRuleId)
		// Revert the entry in dns server
		t.revertEntryFromDNSServer(dnsConfigInPut.State, dnsOnStore.DomainName, rrType, dnsOnStore.IPAddress,
			dnsOnStore.TTL)
		// Revert the update in the data store in failure case
		appDConfig.AppDNSRule[ruleIndex].State = oldState
		t.revertEntryFromDB(&appDConfig)

		return meputil.RemoteServerErr, "failed to apply the dns modification"
	}

	// State updated on dnsOnStore, so regenerate the byte array
	dataOnStoreBytes, err = json.Marshal(dnsOnStore)
	if err == nil {
		t.W.Header().Set("ETag", meputil.GenerateStrongETag(dataOnStoreBytes))
	}

	t.HttpRsp = dnsOnStore

	return 0, ""
}

func (t *DNSRuleUpdate) revertEntryFromDB(appDConfig *models.AppDConfig) {
	errCode, _ := t.updateDnsRecordOnDataStore(*appDConfig)
	if errCode != 0 {
		log.Errorf(nil, "Failed to revert dns rule(app-id: %s, dns-rule-id: %s) update on data-store, "+
			"this might lead to inconsistency.", t.AppInstanceId, t.DNSRuleId)
	}
}

func (t *DNSRuleUpdate) revertEntryFromDNSServer(state, domainName, rrType, ipAddress string, ttl uint32) {
	var err error
	if state == meputil.ActiveState {
		err = t.dnsAgent.DeleteResourceRecordTypeA(domainName, rrType)
	} else {
		err = t.dnsAgent.SetResourceRecordTypeA(domainName, rrType, meputil.RRClassIN,
			[]string{ipAddress}, ttl)
	}
	if err != nil {
		log.Errorf(nil, "Failed to revert dns rule(app-id: %s, dns-rule-id: %s) update on dns-server, "+
			"this might lead to inconsistency.", t.AppInstanceId, t.DNSRuleId)
	}
}

// Update the dns record to the data-store
func (t *DNSRuleUpdate) updateDnsRecordOnDataStore(appDConfig models.AppDConfig) (int, string) {
	updateJSON, err := json.Marshal(appDConfig)
	if err != nil {
		return meputil.ParseInfoErr, "output rule parse failed"
	}
	errCode := backend.PutRecord(meputil.AppDConfigKeyPath+t.AppInstanceId, updateJSON)
	if errCode != 0 {
		return errCode, "rule insertion failed"
	}

	return 0, ""
}

func (t *DNSRuleUpdate) updateDNSToDataPlane(dnsConfigInput *dataplane.DNSRule, dnsOnStore *dataplane.DNSRule,
	appInfo dataplane.ApplicationInfo, rrType string) error {
	var err error
	if dnsConfigInput.State == meputil.ActiveState {
		err = t.dataPlane.AddDNSRule(appInfo, t.DNSRuleId, dnsOnStore.DomainName,
			dnsOnStore.IPAddressType, dnsOnStore.IPAddress, dnsOnStore.TTL)
		if err != nil {
			if err1 := t.dnsAgent.DeleteResourceRecordTypeA(dnsOnStore.DomainName, rrType); err1 != nil {
				log.Errorf(err1, "Failed to revert the configuration(oper: delete, app-id: %s, "+
					"dns-rule-id: %s) from dns-server, this might lead to data inconsistency.",
					t.AppInstanceId, t.DNSRuleId)
			}
		}
	} else {
		err = t.dataPlane.DeleteDNSRule(appInfo, t.DNSRuleId)
		if err != nil {
			if err1 := t.dnsAgent.SetResourceRecordTypeA(dnsOnStore.DomainName, rrType, meputil.RRClassIN,
				[]string{dnsOnStore.IPAddress}, dnsOnStore.TTL); err1 != nil {
				log.Errorf(err1, "Failed to revert the configuration(oper: create, app-id: %s, "+
					"dns-rule-id: %s) from dns-server, this might lead to data inconsistency.",
					t.AppInstanceId, t.DNSRuleId)
			}
		}
	}
	return err
}
