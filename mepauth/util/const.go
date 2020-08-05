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

// util package
package util

const MepAppJwtName string = "mepauth.jwt"
const PortRegex string = `^([1-9]|[1-9]\d{1,3}|[1-5]\d{4}|6[0-4]\d{3}|65[0-4]\d{2}|655[0-2]\d|6553[0-5])$`
const ServerNameRegex string = `^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*$`
const AkRegex string = `^\w{20}$`
const SkRegex string = `^\w{64}$`
const AuthHeaderRegex string = `^SDK-HMAC-SHA256 Access=(\w{20}), SignedHeaders=([^, ]{28}), Signature=([^, ]{64})$`
const MepserverName = "mepserver"
const MepserverRootPath = "mep"
const MepauthName = "mepauth"
const MepserverRateConf = `{ "minute": 1000, "policy": "local", "hide_client_headers": true }`
const MepserverPreFunctionConf = `{ "functions": ["ngx.var.upstream_x_forwarded_for=UNKNOWN"] }`
const MepauthRateConf = `{ "minute": 100, "policy": "local", "hide_client_headers": true }`
const ResponseTransformerConf = `{ "name": "response-transformer", "config": { "remove": { "headers": ["server"] } } }`
const JwtPlugin = "jwt"
const AppidPlugin = "appid-header"
const PreFunctionPlugin = "pre-function"
const RateLimitPlugin = "rate-limiting"
const IpRestrictPlugin = "ip-restriction"
const ComponentContent = "j7k0UwOJSsIfi3dzainoBdkcpJJJOJlzd2oBwMQxXdaZ3oCswITWUyLP4eldxdcKGmDvG1qwUEfQjAg71ZeFYyHgXa5OpBlmug3z06bs7ssr2XYTuPydK6y4K34UfsgRKEwMgGP1Ieo8x20lbjXcq0tJG4Q7xgakXs59NwnBeNg2N8R1FgfqD0z9weWgxd7DdJZkDpbJgdANT31y4KDeDCpJXld6XQOxi99mO2xQdMcH6OUyIfgDP7dPaJU57D33"
const ValidationCounter int64 = 3
const ValidateListClearTimer int64 = 300
const BlockListClearTimer int64 = 900
const specialCharRegex string = `^.*['~!@#$%^&*()-_=+\|[{}\];:'",<.>/?].*$`
const singleDigitRegex string = `^.*\d.*$`
const lowerCaseRegex string = `^.*[a-z].*$`
const upperCaseRegex string = `^.*[A-Z].*$`
const MaxSize = 20
const MaxBackups = 50
const MaxAge = 30
const BaseVal = 10
const BadRequest = 400
const Unauthorized = 401
const Forbidden = 403
const IntSerErr = 500
const MaxMatchVarSize = 3
const ExpiresVal = 3600
const minPasswordSize = 8
const maxPasswordSize = 16
const maxPasswordCount = 2
const maxHostNameLen = 253
