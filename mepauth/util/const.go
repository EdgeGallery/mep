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

// Package util implements mep auth utility functions and contain constants
package util

// Validation related constants
const (
	portRegex              string = `^([1-9]|[1-9]\d{1,3}|[1-5]\d{4}|6[0-4]\d{3}|65[0-4]\d{2}|655[0-2]\d|6553[0-5])$`
	serverNameRegex        string = `^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*$`
	akRegex                string = `^\w{20}$`
	skRegex                string = `^\w{64}$`
	AuthHeaderRegex        string = `^SDK-HMAC-SHA256 Access=([\w=+/]{20}), SignedHeaders=([^, ]{28}), Signature=([^, ]{64})$`
	ValidationCounter      int64  = 3
	ValidateListClearTimer int64  = 300
	BlockListClearTimer    int64  = 900
	specialCharRegex       string = `^.*['~!@#$%^&*()\-_=+\|[{}\];:'",<.>/?].*$`
	singleDigitRegex       string = `^.*\d.*$`
	lowerCaseRegex         string = `^.*[a-z].*$`
	upperCaseRegex         string = `^.*[A-Z].*$`
	MaxSize                       = 20
	MaxBackups                    = 50
	MaxAge                        = 30
	minPasswordSize               = 8
	maxPasswordSize               = 16
	maxPasswordCount              = 2
	maxHostNameLen                = 253
	BaseVal                       = 10
	MaxMatchVarSize               = 3
	ExpiresVal                    = 3600
)

// End point related constants
const (
	MepserverName               = "mepserver"
	MepServerServiceMgmt        = "/mep/mec_service_mgmt"
	MepServerAppSupport         = "/mep/mec_app_support"
	MepauthName                 = "mepauth"
	ApigwHost            string = "apigw_host"
	ApigwPort            string = "apigw_port"
	UrlApplicationId     string = ":applicationId"
)

// Plugin related constants
const (
	MepserverRateConf               = `{ "minute": 1000, "policy": "local", "hide_client_headers": true }`
	MepserverPreFunctionConf        = `{ "functions": ["ngx.var.upstream_x_forwarded_for=UNKNOWN"] }`
	MepauthRateConf                 = `{ "minute": 100, "policy": "local", "hide_client_headers": true }`
	ResponseTransformerConf         = `{ "name": "response-transformer", "config": { "remove": { "headers": ["server"] } } }`
	AppidPlugin                     = "appid-header"
	PreFunctionPlugin               = "pre-function"
	RateLimitPlugin                 = "rate-limiting"
	IpRestrictPlugin                = "ip-restriction"
	PluginPath               string = "/plugins"
	MepAppJwtName            string = "mepauth.jwt"
	JwtPlugin                       = "jwt"
)

// Other
const componentContent = "j7k0UwOJSsIfi3dzainoBdkcpJJJOJlzd2oBwMQxXdaZ3oCswITWUyLP4eldxdcKGmDvG1qwUEfQjAg71ZeFYyHgXa5OpBlmug3z06bs7ssr2XYTuPydK6y4K34UfsgRKEwMgGP1Ieo8x20lbjXcq0tJG4Q7xgakXs59NwnBeNg2N8R1FgfqD0z9weWgxd7DdJZkDpbJgdANT31y4KDeDCpJXld6XQOxi99mO2xQdMcH6OUyIfgDP7dPaJU57D33"
const PgOkMsg string = "LastInsertId is not supported by this driver"
const ContentType string = "Content-Type"
const JsonUtf8 string = "application/json; charset=utf-8"
const DevMode = "dev"
const ClientIpaddressInvalid = "clientIp address is invalid"

// Failure messages
const (
	AppIDFailMsg = "Application Instance ID validation failed"
	AkFailMsg    = "validate ak failed"
	SkFailMsg    = "validate sk failed"
)
