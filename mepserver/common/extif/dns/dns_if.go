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

// Package path implements dns client
package dns

import "net/http"

// DNS rule record
type RuleEntry struct {
	DomainName    string `json:"domainName"`
	IpAddressType string `json:"ipAddressType"`
	IpAddress     string `json:"ipAddress"`
	TTL           int    `json:"ttl"`
	State         string `json:"state"`
}

// Add a new DNs rule record
func NewRuleRecord(domainName string, ipAddressType string, ipAddress string, TTL int, state string) *RuleEntry {
	return &RuleEntry{
		DomainName:    domainName,
		IpAddressType: ipAddressType,
		IpAddress:     ipAddress,
		TTL:           TTL,
		State:         state}
}

// DNS agent interface
type DNSAgent interface {
	// Set/Add DNS entry
	SetResourceRecordTypeA(host, rrtype, class string, pointTo []string, ttl uint32) (resp *http.Response, err error)
	// Delete DNS entry
	DeleteResourceRecordTypeA(host, rrtype string) (resp *http.Response, err error)
}
