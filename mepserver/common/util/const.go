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
)

const (
	RootPath            = "/mep"
	Mm5RootPath         = "/mepcfg"
	MecServicePath      = "/mec_service_mgmt/v1"
	MecAppSupportPath   = "/mec_app_support/v1"
	MecRuleConfigPath   = "/mec_app_config/v1"
	AppServicesPath     = RootPath + MecServicePath + "/applications/:appInstanceId" + "/services"
	AppSubscribePath    = RootPath + MecServicePath + "/applications/:appInstanceId/subscriptions"
	EndAppSubscribePath = RootPath + MecAppSupportPath + "/applications/:appInstanceId/subscriptions"
	DNSRulesPath        = RootPath + MecAppSupportPath + "/applications/:appInstanceId/dns_rules"
	DNSConfigRulesPath  = Mm5RootPath + MecRuleConfigPath + "/rules/:appInstanceId/dns_rules"

	DNSRuleIdPath      = "/:dnsRuleId"
	SubscriptionIdPath = "/:subscriptionId"
	ServiceIdPath      = "/:serviceId"
)

const (
	ActiveState   = "ACTIVE"
	InactiveState = "INACTIVE"
)

const Uris string = "uris"

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

const pwdLengthMin int = 8
const pwdLengthMax int = 16
const pwdCount int = 2
const HookTimerLimit time.Duration = 5

const ComponentContent = "j7k0UwOJSsIfi3dzainoBdkcpJJJOJlzd2oBwMQxXdaZ3oCswITWUyLP4eldxdcKGmDvG1qwUEfQjAg71ZeFYyHgXa5OpBlmug3z06bs7ssr2XYTuPydK6y4K34UfsgRKEwMgGP1Ieo8x20lbjXcq0tJG4Q7xgakXs59NwnBeNg2N8R1FgfqD0z9weWgxd7DdJZkDpbJgdANT31y4KDeDCpJXld6XQOxi99mO2xQdMcH6OUyIfgDP7dPaJU57D33"

const EndDNSRuleKeyPath string = "/cse-sr/etsi/dns-rule/"

const ErrorRequestBodyMessage = "request body invalid"
