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

// Package util implements mep server utility functions and constants
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
	HeartbeatServiceNotFound                      = 17
	ServiceInactive                               = 18
	DuplicateOperation                            = 19
	ForbiddenOperation                            = 20
	NtpConnectionErr                              = 21
)

// Mep server api paths
const (
	RootPath              = "/mep"
	Mm5RootPath           = "/mepcfg"
	MecServicePath        = "/mec_service_mgmt/v1"
	MecAppSupportPath     = "/mec_app_support/v1"
	MecPlatformConfigPath = "/mec_platform_config/v1"
	MecAppDConfigPath     = "/app_lcm/v1"
	MecServiceGovernPath  = "/service_govern/v1"

	AppServicesPath     = RootPath + MecServicePath + "/applications/:appInstanceId" + "/services"
	AppSubscribePath    = RootPath + MecServicePath + "/applications/:appInstanceId/subscriptions"
	ServicesPath        = RootPath + MecServicePath + "/services"
	EndAppSubscribePath = RootPath + MecAppSupportPath + "/applications/:appInstanceId/subscriptions"
	DNSRulesPath        = RootPath + MecAppSupportPath + "/applications/:appInstanceId/dns_rules"
	TrafficRulesPath    = RootPath + MecAppSupportPath + "/applications/:appInstanceId/traffic_rules"
	TimingPath          = RootPath + MecAppSupportPath + "/timing"
	TransportPath       = RootPath + MecAppSupportPath + "/transports"

	CapabilityPath        = Mm5RootPath + MecPlatformConfigPath + "/capabilities"
	AppDConfigPath        = Mm5RootPath + MecAppDConfigPath + "/applications/:appInstanceId/appd_configuration"
	AppDQueryResPath      = Mm5RootPath + MecAppDConfigPath + "/tasks/:taskId/appd_configuration"
	AppInsTerminationPath = RootPath + MecAppSupportPath + "/applications/:appInstanceId/AppInstanceTermination"

	KongHttpLogPath        = RootPath + MecServiceGovernPath + "/kong_log"
	SubscribeStatisticPath = RootPath + MecServiceGovernPath + "/subscribe_statistic"

	DNSRuleIdPath      = "/:dnsRuleId"
	TrafficRuleIdPath  = "/:trafficRuleId"
	SubscriptionIdPath = "/:subscriptionId"
	ServiceIdPath      = "/:serviceId"
	CapabilityIdPath   = "/:capabilityId"
	Liveness           = "/liveness"
	CurrentTIme        = "/current_time"
	TimingCaps         = "/timing_caps"
)

// Resource state
const (
	ActiveState    = "ACTIVE"
	InactiveState  = "INACTIVE"
	SuspendedState = "SUSPENDED"
)

// Address type
const (
	IPv4Type = "IP_V4"
	IPv6Type = "IP_V6"
)

const DBRootPath = "/cse-sr/etsi/"
const (
	EndAppSubKeyPath      = DBRootPath + "app-end-subscribe/"
	AvailAppSubKeyPath    = DBRootPath + "subscribe/"
	AppDConfigKeyPath     = DBRootPath + "appd/"
	AppDLCMJobsPath       = DBRootPath + "mep/applcm/jobs/"
	AppDLCMTasksPath      = DBRootPath + "mep/applcm/tasks/"
	AppDLCMTaskStatusPath = DBRootPath + "mep/applcm/taskstatus/"
)

const (
	GetMethod    = "GET"
	PostMethod   = "POST"
	PutMethod    = "PUT"
	DeleteMethod = "DELETE"
)

const (
	Uris         = "uris"
	Addresses    = "addresses"
	Alternatives = "alternative"
)

const DefaultHeartbeatInterval = 60
const BitSize = 32
const FormatIntBase = 10

const SerAvailabilityNotificationSubscription string = "SerAvailabilityNotificationSubscription"
const AppTerminationNotificationSubscription string = "AppTerminationNotificationSubscription"

const RequestBodyLength = 4096
const ServicesMaxCount = 50
const AppSubscriptionCount = 50
const ServerHeader = "Server"
const JwtPlugin = "jwt"

const specialCharRegex string = `^.*['~!@#$%^&*()-_=+\|[{}\];:'",<.>/?].*$`
const singleDigitRegex string = `^.*\d.*$`
const lowerCaseRegex string = `^.*[a-z].*$`
const upperCaseRegex string = `^.*[A-Z].*$`
const ServerNameRegex string = `^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*$`
const DomainPattern string = `^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*$`

const pwdLengthMin int = 8
const pwdLengthMax int = 16
const pwdCount int = 2
const HookTimerLimit time.Duration = 5

const ComponentContent = "j7k0UwOJSsIfi3dzainoBdkcpJJJOJlzd2oBwMQxXdaZ3oCswITWUyLP4eldxdcKGmDvG1qwUEfQjAg71ZeFYyHgXa5OpBlmug3z06bs7ssr2XYTuPydK6y4K34UfsgRKEwMgGP1Ieo8x20lbjXcq0tJG4Q7xgakXs59NwnBeNg2N8R1FgfqD0z9weWgxd7DdJZkDpbJgdANT31y4KDeDCpJXld6XQOxi99mO2xQdMcH6OUyIfgDP7dPaJU57D33"

const ErrorRequestBodyMessage = "request body invalid"

// MaxFQDNLength As per RFC-1035 section-2.3.4, the maximum length of full FQDN name is 255 octets including
// one length and one null terminating character. Hence it is limited as 253.
const MaxFQDNLength = 253

// MaxIPLength Considering IPV4(15), IPV6(39) and IPV4-mapped IPV6(45
const MaxIPLength = 45

const MaxDNSRuleIdLength = 36
const MaxTrafficRuleIdLength = 36

const MaxAppDAppInstId = 32

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

const MepServerConfigPath = "/usr/mep/conf/mep/config.yaml"

// DataPlaneNone Data plane options
const (
	DataPlaneNone = "none"
)

// Dns agent options
const (
	DnsAgentTypeLocal     = "local"
	DnsAgentTypeDataPlane = "dataplane"
	DnsAgentTypeAll       = "all"
)

type AppDRuleType int

const (
	RuleTypeDns AppDRuleType = iota
	RuleTypeTraffic
)

// OperType Operation list
type OperType int

const (
	OperCreate OperType = iota // Operation type create
	OperModify                 // Operation type modify
	OperDelete                 // Operation type delete
)

// AppDRuleStatus AppD rule state machine
type AppDRuleStatus int

const (
	WaitMp2           AppDRuleStatus = iota // wait to be process
	WaitLocal                               // Local handling pending(for DNS)
	WaitConfigDBWrite                       // Wait for Config DB write
)

// FuncType Function table index
type FuncType int

const (
	ApplyFunc  FuncType = iota // Normal operation of apply rule
	RevertFunc                 // Revert handler index

)

const TaskProgressFailure = -1

const (
	TaskStateSuccess    = "SUCCESS"
	TaskStateProcessing = "PROCESSING"
	TaskStateFailure    = "FAILURE"
)

const (
	IpTypeIpv4 = "IP_V4"
	IpTypeIpv6 = "IP_V6"
)

const KongHttpLogIndex = "http-log"
const WeekDay = 7

const ApiGwCaCertName = "apigw_cacert"
const ConfigFilePath = "/usr/mep/conf/app.conf"

const (
	ServiceInfoDataCenter = "datacenterinfo"
)

const LivenessPath = "/mep/mec_service_mgmt/v1/applications/%s/services/%s/liveness"

const (
	EnvMepAuthPort = "MEPAUTH_SERVICE_PORT"
	EnvMepAuthHost = "MEPAUTH_PORT_10443_TCP_ADDR"
)
const MepAuthBaseUrlFormat = "https://%s:%s/mepauth/v1/applications"
