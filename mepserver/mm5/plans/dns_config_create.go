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
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/util"
	uuid "github.com/satori/go.uuid"

	"mepserver/common/arch/workspace"
	"mepserver/common/extif/backend"
	"mepserver/common/extif/dns"
	meputil "mepserver/common/util"
	"mepserver/mm5/models"
)

type DecodeDnsConfigRestReq struct {
	workspace.TaskBase
	R             *http.Request   `json:"r,in"`
	Ctx           context.Context `json:"ctx,out"`
	AppInstanceId string          `json:"appInstanceId,out"`
	DNSRuleId     string          `json:"dnsRuleId,out"`
	RestBody      interface{}     `json:"restBody,out"`
}

func (t *DecodeDnsConfigRestReq) OnRequest(data string) workspace.TaskCode {
	err := t.getParam(t.R)
	if err != nil {
		log.Error("parameters validation failed", nil)
		return workspace.TaskFinish
	}
	err = t.parseBody(t.R)
	if err != nil {
		log.Error("parse rest body failed", nil)
	}
	return workspace.TaskFinish
}

func (t *DecodeDnsConfigRestReq) parseBody(r *http.Request) error {
	if t.RestBody == nil {
		return nil
	}
	msg, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("read failed", nil)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "read request body error")
		return err
	}
	if len(msg) > meputil.RequestBodyLength {
		err = errors.New("request body too large")
		log.Errorf(nil, "request body too large %d", len(msg))
		t.SetFirstErrorCode(meputil.RequestParamErr, "request body too large")
		return err
	}

	newMsg, err := t.checkParam(msg)
	if err != nil {
		log.Error("check param failed", nil)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "check Param failed")
		return err
	}

	err = json.Unmarshal(newMsg, t.RestBody)
	if err != nil {
		log.Errorf(nil, "json unmarshalling failed")
		t.SetFirstErrorCode(meputil.ParseInfoErr, "unmarshal request body error")
		return errors.New("json unmarshalling failed")
	}
	return nil
}

func (t *DecodeDnsConfigRestReq) checkParam(msg []byte) ([]byte, error) {

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

func (t *DecodeDnsConfigRestReq) WithBody(body interface{}) *DecodeDnsConfigRestReq {
	t.RestBody = body
	return t
}

func (t *DecodeDnsConfigRestReq) getParam(r *http.Request) error {
	queryReq, _ := meputil.GetHTTPTags(r)

	var errRes error

	t.AppInstanceId = queryReq.Get(":appInstanceId")
	if errRes = meputil.ValidateAppInstanceIdWithHeader(t.AppInstanceId, r); errRes != nil {
		log.Error("validate X-AppinstanceId failed", errRes)
		t.SetFirstErrorCode(meputil.AuthorizationValidateErr, errRes.Error())
		return errRes
	}
	errRes = meputil.ValidateUUID(t.AppInstanceId)
	if errRes != nil {
		log.Error("app Instance ID validation failed", errRes)
		t.SetFirstErrorCode(meputil.RequestParamErr, "app Instance ID validation failed, invalid uuid")
		return errRes
	}

	t.DNSRuleId = queryReq.Get(":dnsRuleId")
	if len(t.DNSRuleId) > meputil.MaxDNSRuleId {
		log.Error("dns rule ID validation failed", nil)
		t.SetFirstErrorCode(meputil.RequestParamErr, "dns rule ID validation failed, invalid length")
		return errRes
	}
	t.Ctx = util.SetTargetDomainProject(r.Context(), r.Header.Get("X-Domain-Name"), queryReq.Get(":project"))
	return nil
}

type CreateDNSRule struct {
	workspace.TaskBase
	Ctx           context.Context     `json:"ctx,in"`
	W             http.ResponseWriter `json:"w,in"`
	AppInstanceId string              `json:"appInstanceId,in"`
	RestBody      interface{}         `json:"restBody,in"`
	DNSRuleId     string              `json:"dnsRuleId,out"`
	HttpRsp       interface{}         `json:"httpRsp,out"`
}

func (t *CreateDNSRule) OnRequest(data string) workspace.TaskCode {

	dnsConfigInput, ok := t.RestBody.(*models.DnsConfigRule)
	if !ok {
		t.SetFirstErrorCode(1, "input body parse failed")
		t.SetSerErrInfo(&workspace.SerErrInfo{ErrCode: http.StatusBadRequest, Message: "Parse body error!"})
		return workspace.TaskFinish
	}

	errorString, errorCode := t.validateInputs(dnsConfigInput)
	if errorCode != 0 {
		t.SetFirstErrorCode(workspace.ErrCode(errorCode), errorString)
		return workspace.TaskFinish
	}

	// Generate dns rule id
	t.DNSRuleId = uuid.NewV4().String()

	dnsConfigInput.DnsRuleId = t.DNSRuleId
	dnsConfigBytes, err := json.Marshal(
		dns.NewRuleRecord(
			dnsConfigInput.DomainName,
			dnsConfigInput.IpAddressType,
			dnsConfigInput.IpAddress,
			dnsConfigInput.TTL,
			dnsConfigInput.State))
	if err != nil {
		log.Errorf(nil, "can not marshal subscribe info")
		t.SetFirstErrorCode(meputil.ParseInfoErr, "can not marshal subscribe info")
		return workspace.TaskFinish
	}

	errCode := backend.PutRecord(meputil.EndDNSRuleKeyPath+t.AppInstanceId+"/"+t.DNSRuleId, dnsConfigBytes)
	if errCode != 0 {
		log.Errorf(nil, "dns rule(appId: %s, dnsRuleId: %s) insertion on data-store failed!",
			t.AppInstanceId, t.DNSRuleId)
		t.SetFirstErrorCode(workspace.ErrCode(errCode), "put dns rule to data-store failed")
		return workspace.TaskFinish
	}

	if dnsConfigInput.State == meputil.InactiveState {
		t.W.Header().Set("ETag", meputil.GenerateStrongETag(dnsConfigBytes))
		t.HttpRsp = dnsConfigInput
		return workspace.TaskFinish
	}

	dnsAgent := dns.NewRestClient()

	rrType := meputil.RRTypeA
	if dnsConfigInput.IpAddressType == meputil.IPv6Type {
		rrType = meputil.RRTypeAAAA
	}

	httpResp, err := dnsAgent.SetResourceRecordTypeA(
		dnsConfigInput.DomainName, rrType, meputil.RRClassIN, []string{dnsConfigInput.IpAddress},
		uint32(dnsConfigInput.TTL))
	if err != nil || !meputil.IsHttpStatusOK(httpResp.StatusCode) {
		if err != nil {
			log.Errorf(err, "DNS rule(appId: %s, dnsRuleId: %s) create fail on server!",
				t.AppInstanceId, t.DNSRuleId)
			t.SetFirstErrorCode(meputil.RemoteServerErr, "failed to apply changes on remote server")
		} else {
			log.Errorf(err, "DNS rule create failed on server(%d: %s)!", httpResp.StatusCode, httpResp.Status)
			t.SetFirstErrorCode(meputil.RemoteServerErr, "could not apply rule on dns server")
		}

		errCode := backend.DeleteRecord(meputil.EndDNSRuleKeyPath + t.AppInstanceId + "/" + t.DNSRuleId)
		if errCode != 0 {
			log.Errorf(err, "DNS rule(appId: %s, dnsRuleId: %s) delete from etcd failed, "+
				"this might lead to data inconsistency!", t.AppInstanceId, t.DNSRuleId)
			t.SetFirstErrorCode(workspace.ErrCode(errCode), "delete dns rule from etcd failed on server error")
			return workspace.TaskFinish
		}
		return workspace.TaskFinish
	}

	t.W.Header().Set("ETag", meputil.GenerateStrongETag(dnsConfigBytes))
	t.HttpRsp = dnsConfigInput
	return workspace.TaskFinish
}

func (t *CreateDNSRule) validateInputs(dnsConfigInput *models.DnsConfigRule) (errorString string, errorCode int) {
	if (dnsConfigInput.State != meputil.ActiveState && dnsConfigInput.State != meputil.InactiveState) ||
		len(dnsConfigInput.DomainName) == 0 || len(dnsConfigInput.DomainName) > meputil.MaxFQDNLength ||
		len(dnsConfigInput.IpAddress) == 0 || len(dnsConfigInput.IpAddress) > meputil.MaxIPLength ||
		(dnsConfigInput.IpAddressType != meputil.IPv4Type && dnsConfigInput.IpAddressType != meputil.IPv6Type) ||
		dnsConfigInput.TTL == 0 {
		return "dns input error", meputil.ParseInfoErr
	}

	ip := net.ParseIP(dnsConfigInput.IpAddress)
	if ip == nil ||
		(strings.Contains(dnsConfigInput.IpAddress, ".") && dnsConfigInput.IpAddressType != meputil.IPv4Type) ||
		(strings.Contains(dnsConfigInput.IpAddress, ":") && dnsConfigInput.IpAddressType != meputil.IPv6Type) {
		return "dns ip error", meputil.ParseInfoErr
	}

	isExists, errorCode := t.isDomainAlreadyExists(dnsConfigInput.DomainName)
	if errorCode != 0 {
		return "validation failure", errorCode
	}
	if isExists {
		return "domain already exists", meputil.ResourceExists
	}

	return "", 0
}

func (t *CreateDNSRule) isDomainAlreadyExists(domainName string) (isExists bool, errorCode int) {
	records, errCode := backend.GetRecords(meputil.EndDNSRuleKeyPath + t.AppInstanceId)
	if errCode != 0 {
		return false, errCode
	}
	for _, value := range records {
		dnsRuleInStore := &dns.RuleEntry{}
		jsonErr := json.Unmarshal(value, dnsRuleInStore)
		if jsonErr != nil {
			log.Errorf(nil, "failed to parse the dns entries from data-store")
			return false, meputil.OperateDataWithEtcdErr
		}
		if dnsRuleInStore.DomainName == domainName {
			return true, 0
		}
	}
	return false, 0
}
