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

	"github.com/apache/servicecomb-service-center/pkg/log"

	"mepserver/common/arch/workspace"
	"mepserver/common/extif/backend"
	"mepserver/common/extif/dns"
	"mepserver/common/util"
	"mepserver/mm5/models"
)

type DNSRuleDelete struct {
	RestBody interface{} `json:"restBody,in"`
	workspace.TaskBase
	AppInstanceId string      `json:"appInstanceId,in"`
	DNSRuleId     string      `json:"dnsRuleId,in"`
	HttpRsp       interface{} `json:"httpRsp,out"`
}

func (t *DNSRuleDelete) OnRequest(data string) workspace.TaskCode {

	if len(t.DNSRuleId) == 0 {
		log.Errorf(nil, "invalid dns id on query request")
		t.SetFirstErrorCode(util.ParseInfoErr, "invalid delete request")
		return workspace.TaskFinish
	}

	dnsRuleEntry, errCode := backend.GetRecord(util.EndDNSRuleKeyPath + t.AppInstanceId + "/" + t.DNSRuleId)
	if errCode != 0 {
		log.Errorf(nil, "get dns rules from data-store failed")
		t.SetFirstErrorCode(workspace.ErrCode(errCode), "dns rule retrieval failed")
		return workspace.TaskFinish
	}

	dnsRule := &dns.RuleEntry{}
	jsonErr := json.Unmarshal(dnsRuleEntry, dnsRule)
	if jsonErr != nil {
		log.Errorf(nil, "failed to parse the dns entry from data-store on delete request")
		t.SetFirstErrorCode(util.OperateDataWithEtcdErr, "parse dns rules failed")
		return workspace.TaskFinish
	}

	rrType := util.RRTypeA
	if dnsRule.IpAddressType == util.IPv6Type {
		rrType = util.RRTypeAAAA
	}

	// Delete the dns entry on remote dns server only if it was active
	if dnsRule.State == util.ActiveState {
		dnsAgent := dns.NewRestClient()
		httpResp, err := dnsAgent.DeleteResourceRecordTypeA(dnsRule.DomainName, rrType)
		if err != nil {
			log.Errorf(nil, "dns rule(app-id: %s, dns-rule-id: %s) delete fail on dns server!",
				t.AppInstanceId, t.DNSRuleId)
			t.SetFirstErrorCode(util.RemoteServerErr, "failed to delete the dns record")
			return workspace.TaskFinish
		}
		if !util.IsHttpStatusOK(httpResp.StatusCode) {
			log.Errorf(err, "dns rule delete failed on server(%d: %s)",
				httpResp.StatusCode, httpResp.Status)
			t.SetFirstErrorCode(util.RemoteServerErr, "could not delete rule on dns server")
			return workspace.TaskFinish
		}
	}

	errCode = backend.DeleteRecord(util.EndDNSRuleKeyPath + t.AppInstanceId + "/" + t.DNSRuleId)
	if errCode != 0 {
		log.Errorf(nil, "delete dns rules from data-store failed, this will lead to inconsistency.")
		t.SetFirstErrorCode(workspace.ErrCode(errCode), "delete dns rules from data-store failed")
		return workspace.TaskFinish
	}

	t.HttpRsp = models.NewDnsConfigRule(
		t.DNSRuleId,
		dnsRule.DomainName,
		dnsRule.IpAddressType,
		dnsRule.IpAddress,
		dnsRule.TTL,
		dnsRule.State)
	return workspace.TaskFinish
}
