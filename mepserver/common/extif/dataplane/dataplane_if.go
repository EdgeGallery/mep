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

// Package dataplane defines the data-plane interfaces
package dataplane

import (
	"mepserver/common/config"
)

// TunnelInfo represents the traffic tunnel configurations
type TunnelInfo struct {
	TunnelType       string `json:"tunnelType" validate:"omitempty,oneof=GTP_U GRE"`
	TunnelDstAddress string `json:"tunnelDstAddress" validate:"omitempty,ip"`
	TunnelSrcAddress string `json:"tunnelSrcAddress" validate:"omitempty,ip"`
}

// DstInterface holds the traffic destination interface configurations
type DstInterface struct {
	InterfaceType string     `json:"interfaceType" validate:"omitempty,oneof=TUNNEL MAC IP"`
	TunnelInfo    TunnelInfo `json:"tunnelInfo"`
	SrcMacAddress string     `json:"srcMacAddress" validate:"omitempty,mac"`
	DstMacAddress string     `json:"dstMacAddress" validate:"omitempty,mac"`
	DstIPAddress  string     `json:"dstIpAddress" validate:"omitempty,ip"`
}

// TrafficFilter Keeps traffic filtering configurations
type TrafficFilter struct {
	SrcAddress       []string `json:"srcAddress" validate:"omitempty,dive,max=64"`
	DstAddress       []string `json:"dstAddress" validate:"omitempty,dive,max=64"`
	SrcPort          []string `json:"srcPort" validate:"omitempty,dive,number"`
	DstPort          []string `json:"dstPort" validate:"omitempty,dive,number"`
	Protocol         []string `json:"protocol" validate:"omitempty,dive,min=1,max=8"`
	Tag              []string `json:"tag" validate:"omitempty,dive,min=1,max=8"`
	SrcTunnelAddress []string `json:"srcTunnelAddress" validate:"omitempty,dive,ip"`
	TgtTunnelAddress []string `json:"tgtTunnelAddress" validate:"omitempty,dive,ip"`
	SrcTunnelPort    []string `json:"srcTunnelPort" validate:"omitempty,dive,number"`
	DstTunnelPort    []string `json:"dstTunnelPort" validate:"omitempty,dive,number"`
	QCI              int      `json:"qCI" validate:"omitempty"`
	DSCP             int      `json:"dSCP" validate:"omitempty"`
	TC               int      `json:"tC" validate:"omitempty"`
}

// TrafficRule Keeps traffic rule related configurations
type TrafficRule struct {
	TrafficRuleID string          `json:"trafficRuleId" validate:"required,min=1,max=63"`
	FilterType    string          `json:"filterType" validate:"required,oneof=FLOW PACKET"`
	Priority      int             `json:"priority" validate:"required,min=1,max=255"`
	TrafficFilter []TrafficFilter `json:"trafficFilter" validate:"required,dive,max=16"` //TBD to decide max number.
	Action        string          `json:"action" validate:"required,oneof=DROP FORWARD_DECAPSULATED FORWARD_AS_IS PASSTHROUGH DUPLICATE_DECAPSULATED DUPLICATE_AS_IS"`
	DstInterface  []DstInterface  `json:"dstInterface" validate:"omitempty,dive,max=2"`
	State         string          `json:"state" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

// DNSRule Keeps all configurations related to dns
type DNSRule struct {
	DNSRuleID     string `json:"dnsRuleId" validate:"required,min=1,max=63"`
	DomainName    string `json:"domainName" validate:"required,min=1,max=255"`
	IPAddressType string `json:"ipAddressType" validate:"required,oneof=IP_V4 IP_V6 IPv4 IPv6"`
	IPAddress     string `json:"ipAddress" validate:"required,ip"`
	TTL           uint32 `json:"ttl" validate:"omitempty,min=0,max=4294967295"`
	State         string `json:"state" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

// ApplicationInfo Application info struct
type ApplicationInfo struct {
	Id   string
	Name string
}

// DataPlane interface functions
type DataPlane interface {

	// InitDataPlane Initialize the data-plane
	InitDataPlane(config *config.MepServerConfig) (err error)

	// AddTrafficRule Add new Traffic Rule
	AddTrafficRule(appInfo ApplicationInfo, trafficRuleId, filterType, action string, priority int,
		filter []TrafficFilter) (err error)

	// SetTrafficRule Set rule
	SetTrafficRule(appInfo ApplicationInfo, trafficRuleId, filterType, action string, priority int, filter []TrafficFilter) (err error)

	// DeleteTrafficRule Delete Traffic rule
	DeleteTrafficRule(appInfo ApplicationInfo, trafficRuleId string) (err error)

	// AddDNSRule Add new DNS redirect rule
	AddDNSRule(appInfo ApplicationInfo, dnsRuleId, domainName, ipAddressType, ipAddress string, ttl uint32) (err error)

	// SetDNSRule Set DNS redirect rule
	SetDNSRule(appInfo ApplicationInfo, dnsRuleId, domainName, ipAddressType, ipAddress string, ttl uint32) (err error)

	// DeleteDNSRule Delete DNS rule from data-plane
	DeleteDNSRule(appInfo ApplicationInfo, dnsRuleId string) (err error)
}
