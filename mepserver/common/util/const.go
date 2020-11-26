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

// Package path implements mep server utility functions and constants
package util

import (
	"time"

	"mepserver/common/arch/workspace"
)

const (
	SerErrFailBase              workspace.ErrCode = workspace.TaskFail
	SerErrServiceNotFound                         = 2
	SerInstanceNotFound                           = 3
	ParseInfoErr                                  = 4
	SubscriptionNotFound                          = 5
	OperateDataWithEtcdErr                        = 6
	SerErrServiceDelFailed                        = 7
	SerErrServiceUpdFailed                        = 8
	RemoteServerErr                               = 9
	EtagMissMatchErr                              = 10
	AuthorizationValidateErr                      = 11
	SerErrServiceRegFailed                        = 12
	SerErrServiceInstanceFailed                   = 13
	RequestParamErr                               = 14
	SubscriptionErr                               = 15
	ResourceExists                                = 16
	HeartbeatServiceNotFound    		      = 17
	ServiceInactive             		      = 18
)

const (
	RootPath              = "/mep"
	Mm5RootPath           = "/mepcfg"
	MecServicePath        = "/mec_service_mgmt/v1"
	MecAppSupportPath     = "/mec_app_support/v1"
	MecRuleConfigPath     = "/mec_app_config/v1"
	MecPlatformConfigPath = "/mec_platform_config/v1"
	AppServicesPath       = RootPath + MecServicePath + "/applications/:appInstanceId" + "/services"
	AppSubscribePath      = RootPath + MecServicePath + "/applications/:appInstanceId/subscriptions"
	EndAppSubscribePath   = RootPath + MecAppSupportPath + "/applications/:appInstanceId/subscriptions"
	DNSRulesPath          = RootPath + MecAppSupportPath + "/applications/:appInstanceId/dns_rules"
	DNSConfigRulesPath    = Mm5RootPath + MecRuleConfigPath + "/rules/:appInstanceId/dns_rules"
	CapabilityPath        = Mm5RootPath + MecPlatformConfigPath + "/capabilities"

	DNSRuleIdPath      = "/:dnsRuleId"
	SubscriptionIdPath = "/:subscriptionId"
	ServiceIdPath      = "/:serviceId"
	CapabilityIdPath   = "/:capabilityId"
	Liveness           = "/liveness"
)

const (
	ActiveState   = "ACTIVE"
	InactiveState = "INACTIVE"
	SuspendedState = "SUSPENDED"
)

const (
	IPv4Type = "IP_V4"
	IPv6Type = "IP_V6"
)

const Uris string = "uris"

const DefaultHeartbeatInterval = 60
const BitSize = 32
const FormatIntBase = 10

const SerAvailabilityNotificationSubscription string = "SerAvailabilityNotificationSubscription"
const AppTerminationNotificationSubscription string = "AppTerminationNotificationSubscription"
const EndAppSubKeyPath string = "/cse-sr/etsi/app-end-subscribe/"
const AvailAppSubKeyPath string = "/cse-sr/etsi/subscribe/"
const RequestBodyLength = 4096
const ServicesMaxCount = 50
const AppSubscriptionCount = 50
const ServerHeader = "Server"

const specialCharRegex string = `^.*['~!@#$%^&*()-_=+\|[{}\];:'",<.>/?].*$`
const singleDigitRegex string = `^.*\d.*$`
const lowerCaseRegex string = `^.*[a-z].*$`
const upperCaseRegex string = `^.*[A-Z].*$`
const ServerNameRegex string = `^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*$`

const pwdLengthMin int = 8
const pwdLengthMax int = 16
const pwdCount int = 2
const HookTimerLimit time.Duration = 5

const ComponentContent = "j7k0UwOJSsIfi3dzainoBdkcpJJJOJlzd2oBwMQxXdaZ3oCswITWUyLP4eldxdcKGmDvG1qwUEfQjAg71ZeFYyHgXa5OpBlmug3z06bs7ssr2XYTuPydK6y4K34UfsgRKEwMgGP1Ieo8x20lbjXcq0tJG4Q7xgakXs59NwnBeNg2N8R1FgfqD0z9weWgxd7DdJZkDpbJgdANT31y4KDeDCpJXld6XQOxi99mO2xQdMcH6OUyIfgDP7dPaJU57D33"

const EndDNSRuleKeyPath string = "/cse-sr/etsi/dns-rule/"

const ErrorRequestBodyMessage = "request body invalid"

// As per RFC-1035 section-2.3.4, the maximum length of full FQDN name is 255 octets including
// one length and one null terminating character. Hence it is limited as 253.
const MaxFQDNLength = 253

// Considering IPV4(15), IPV6(39) and IPV4-mapped IPV6(45
const MaxIPLength = 45

const MaxDNSRuleId = 36

const MaxPortNumber = 65535
const MaxPortLength = 5
const maxHostNameLen = 253

const (
	DefaultDnsHost           = "localhost"
	DefaultDnsManagementPort = 8080
)

const (
	RRTypeA    = "A"
	RRTypeAAAA = "AAAA"
)

const (
	RRClassIN = "IN"
)
