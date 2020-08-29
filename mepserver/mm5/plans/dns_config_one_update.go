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

// Package path implements mep server api plans
package plans

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/apache/servicecomb-service-center/pkg/log"

	"mepserver/common/extif/backend"
	"mepserver/common/extif/dns"

	"mepserver/common/arch/workspace"
	meputil "mepserver/common/util"
	"mepserver/mm5/models"
)

type DNSRuleUpdate struct {
	workspace.TaskBase
	R             *http.Request       `json:"r,in"`
	W             http.ResponseWriter `json:"w,in"`
	RestBody      interface{}         `json:"restBody,in"`
	AppInstanceId string              `json:"appInstanceId,in"`
	DNSRuleId     string              `json:"dnsRuleId,in"`
	HttpRsp       interface{}         `json:"httpRsp,out"`
}

func (t *DNSRuleUpdate) OnRequest(data string) workspace.TaskCode {

	log.Debugf("update request arrived for dns rule %s and appId %s.", t.DNSRuleId, t.AppInstanceId)

	if len(t.DNSRuleId) == 0 {
		log.Errorf(nil, "invalid dns id on query request")
		t.SetFirstErrorCode(meputil.ParseInfoErr, "invalid update request")
		return workspace.TaskFinish
	}

	dnsRuleEntry, errCode := backend.GetRecord(meputil.EndDNSRuleKeyPath + t.AppInstanceId + "/" + t.DNSRuleId)
	if errCode != 0 {
		log.Errorf(errors.New("get operation failed"),
			"dns rule retrieval from data-store failed on update request")
		t.SetFirstErrorCode(workspace.ErrCode(errCode), "dns rule retrieval failed")
		return workspace.TaskFinish
	}

	dnsRuleOnDataStore := &dns.RuleEntry{}
	jsonErr := json.Unmarshal(dnsRuleEntry, dnsRuleOnDataStore)
	if jsonErr != nil {
		log.Errorf(errors.New("json parse failed"),
			"failed to parse the dns entry from data-store on update request")
		t.SetFirstErrorCode(meputil.OperateDataWithEtcdErr, "parse dns rules failed")
		return workspace.TaskFinish
	}

	// Check for E-Tags precondition. More details could be found here: https://tools.ietf.org/html/rfc7232#section-2.3
	ifMatchTag := t.R.Header.Get("If-Match")
	if len(ifMatchTag) != 0 && ifMatchTag != meputil.GenerateStrongETag(dnsRuleEntry) {
		t.SetFirstErrorCode(meputil.EtagMissMatchErr, "e-tag miss-match")
		return workspace.TaskFinish
	}

	// E-Tag check need to be done before parsing, hence added parsing here
	dnsConfigInput, ok := t.RestBody.(*models.DnsConfigRule)
	if !ok {
		t.SetFirstErrorCode(meputil.ParseInfoErr, "input parsing failed")
		return workspace.TaskFinish
	}

	if len(dnsConfigInput.DnsRuleId) != 0 && t.DNSRuleId != dnsConfigInput.DnsRuleId {
		t.SetFirstErrorCode(meputil.ParseInfoErr, "dns identifier miss-match")
		return workspace.TaskFinish
	}

	if dnsRuleOnDataStore.State == dnsConfigInput.State {
		t.W.Header().Set("ETag", meputil.GenerateStrongETag(dnsRuleEntry))
		t.HttpRsp = dnsRuleOnDataStore
		return workspace.TaskFinish
	}
	if dnsConfigInput.State != "ACTIVE" && dnsConfigInput.State != "INACTIVE" {
		t.SetFirstErrorCode(meputil.ParseInfoErr, "invalid dns state input")
		return workspace.TaskFinish
	}

	// Backing up old state to revert in case of dns server error
	oldState := dnsRuleOnDataStore.State
	dnsRuleOnDataStore.State = dnsConfigInput.State

	errCode, errString, err := t.updateDnsRecordOnDataStore(dnsRuleOnDataStore)
	if err != nil {
		t.SetFirstErrorCode(workspace.ErrCode(errCode), errString)
		return workspace.TaskFinish
	}

	// Update the DNS server as per the new configurations
	dnsAgent := dns.NewRestClient()
	var httpResp *http.Response
	if dnsConfigInput.State == "ACTIVE" {
		httpResp, err = dnsAgent.SetResourceRecordTypeA(dnsRuleOnDataStore.DomainName, "A", "IN",
			[]string{dnsRuleOnDataStore.IpAddress}, uint32(dnsRuleOnDataStore.TTL))
	} else {
		httpResp, err = dnsAgent.DeleteResourceRecordTypeA(dnsRuleOnDataStore.DomainName, "A")
	}
	if err != nil {
		log.Errorf(nil, "dns rule(app-id: %s, dns-rule-id: %s) update fail on dns server!",
			t.AppInstanceId, t.DNSRuleId)

		// Revert the update in the data store in failure case
		dnsRuleOnDataStore.State = oldState
		_, _, err := t.updateDnsRecordOnDataStore(dnsRuleOnDataStore)
		if err != nil {
			log.Errorf(nil, "failed to revert dns rule(app-id: %s, dns-rule-id: %s) update on data-store, "+
				"this might lead to inconsistency!", t.AppInstanceId, t.DNSRuleId)
		}

		t.SetFirstErrorCode(meputil.RemoteServerErr, "failed to apply the dns modification")
		return workspace.TaskFinish
	}
	if !meputil.IsHttpStatusOK(httpResp.StatusCode) {
		log.Errorf(nil, "dns rule(app-id: %s, dns-rule-id: %s) update failed on server(%d: %s).",
			t.AppInstanceId, t.DNSRuleId, httpResp.StatusCode, httpResp.Status)

		// Revert the update in the data store in failure case
		dnsRuleOnDataStore.State = oldState
		_, _, err := t.updateDnsRecordOnDataStore(dnsRuleOnDataStore)
		if err != nil {
			log.Errorf(nil, "failed to revert dns rule(app-id: %s, dns-rule-id: %s) update on data-store, "+
				"this might lead to inconsistency!", t.AppInstanceId, t.DNSRuleId)
		}

		t.SetFirstErrorCode(meputil.RemoteServerErr, "could not apply rule on dns server")
		return workspace.TaskFinish
	}

	dataStoreEntryBytes, err := json.Marshal(dnsRuleOnDataStore)
	if err == nil {
		t.W.Header().Set("ETag", meputil.GenerateStrongETag(dataStoreEntryBytes))
	}

	t.HttpRsp = models.NewDnsConfigRule(
		t.DNSRuleId,
		dnsRuleOnDataStore.DomainName,
		dnsRuleOnDataStore.IpAddressType,
		dnsRuleOnDataStore.IpAddress,
		dnsRuleOnDataStore.TTL,
		dnsRuleOnDataStore.State)
	return workspace.TaskFinish
}

// Update the dns record to the data-store
func (t *DNSRuleUpdate) updateDnsRecordOnDataStore(dnsRecord *dns.RuleEntry) (int, string, error) {
	updateJSON, err := json.Marshal(dnsRecord)
	if err != nil {
		log.Errorf(nil, "marshal dns rule failed")
		return meputil.ParseInfoErr, "output dns rule parse failed", err
	}

	errCode := backend.PutRecord(meputil.EndDNSRuleKeyPath+t.AppInstanceId+"/"+t.DNSRuleId, updateJSON)
	if errCode != 0 {
		log.Errorf(nil, "dns rule(app-id: %s, dns-rule-id: %s) insertion on data-store failed!",
			t.AppInstanceId, t.DNSRuleId)
		return errCode, "dns rule insertion failed", fmt.Errorf("dns rule insertion failed")
	}

	return 0, "", nil
}
