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

// Package dns implements dns client functionalities
package dns

// RuleEntry DNS rule record
type RuleEntry struct {
	DomainName    string `json:"domainName"`
	IpAddressType string `json:"ipAddressType"`
	IpAddress     string `json:"ipAddress"`
	TTL           int    `json:"ttl"`
	State         string `json:"state"`
}

// NewRuleRecord Add a new DNS rule record
func NewRuleRecord(domainName string, ipAddressType string, ipAddress string, TTL int, state string) *RuleEntry {
	return &RuleEntry{
		DomainName:    domainName,
		IpAddressType: ipAddressType,
		IpAddress:     ipAddress,
		TTL:           TTL,
		State:         state}
}

// DNSAgent interface
type DNSAgent interface {
	// SetResourceRecordTypeA Set/Add DNS entry
	SetResourceRecordTypeA(host, rrType, class string, pointTo []string, ttl uint32) error
	// DeleteResourceRecordTypeA Delete DNS entry
	DeleteResourceRecordTypeA(host, rrType string) error
}
