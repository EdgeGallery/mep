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

// Package path implements mep server object models
package models

// Represents a dns configuration message from MECM to MEP
type DnsConfigRule struct {
	DnsRuleId     string `json:"dnsRuleId"`
	DomainName    string `json:"domainName"`
	IpAddressType string `json:"ipAddressType"`
	IpAddress     string `json:"ipAddress"`
	TTL           int    `json:"ttl"`
	State         string `json:"state"`
}

func NewDnsConfigRule(
	dnsRuleId string, domainName string, ipAddressType string, ipAddress string, TTL int, state string) *DnsConfigRule {
	return &DnsConfigRule{DnsRuleId: dnsRuleId,
		DomainName:    domainName,
		IpAddressType: ipAddressType,
		IpAddress:     ipAddress,
		TTL:           TTL,
		State:         state}
}
