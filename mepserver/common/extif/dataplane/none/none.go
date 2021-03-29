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

// Package interface for data-plane none
package none

import (
	"github.com/apache/servicecomb-service-center/pkg/log"
	"mepserver/common/config"
	"mepserver/common/extif/dataplane"
)

type NoneDataPlane struct {
	dataplane.DataPlane
}

func (n *NoneDataPlane) InitDataPlane(config *config.MepServerConfig) (err error) {
	return nil
}

func (n *NoneDataPlane) AddTrafficRule(appInfo dataplane.ApplicationInfo, trafficRuleId, filterType, action string, priority int,
	filter []dataplane.TrafficFilter) (err error) {
	log.Infof("Added traffic rule(%s) successfully to data-plane for app %v.", trafficRuleId, appInfo)
	return nil
}

func (n *NoneDataPlane) SetTrafficRule(appInfo dataplane.ApplicationInfo, trafficRuleId, filterType, action string, priority int, filter []dataplane.TrafficFilter) (err error) {
	log.Infof("Updated traffic rule(%s) successfully on data-plane for app %v.", trafficRuleId, appInfo)
	return nil
}

func (n *NoneDataPlane) DeleteTrafficRule(appInfo dataplane.ApplicationInfo, trafficRuleId string) (err error) {
	log.Infof("Deleted traffic rule(%s) successfully from data-plane for app %v.", trafficRuleId, appInfo)
	return nil
}

func (n *NoneDataPlane) AddDNSRule(appInfo dataplane.ApplicationInfo, dnsRuleId, domainName, ipAddressType, ipAddress string, ttl uint32) (err error) {
	log.Infof("Added dns rule(%s) successfully to data-plane for app %v.", dnsRuleId, appInfo)
	return nil
}

func (n *NoneDataPlane) SetDNSRule(appInfo dataplane.ApplicationInfo, dnsRuleId, domainName, ipAddressType, ipAddress string,
	ttl uint32) (err error) {
	log.Infof("Updated dns rule(%s) successfully on data-plane for app %v.", dnsRuleId, appInfo)
	return nil
}

func (n *NoneDataPlane) DeleteDNSRule(appInfo dataplane.ApplicationInfo, dnsRuleId string) (err error) {
	log.Infof("Deleted dns rule(%s) successfully from data-plane for app %v.", dnsRuleId, appInfo)
	return nil
}
