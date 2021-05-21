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

// Package util utility package
package util

const (
	// DBStringExceptions .
	DBStringExceptions    = "/.\\"
	// DefaultDNSPort  Default DNS port.
	DefaultDNSPort        = 53
	// DefaultManagementPort Default Management port.
	DefaultManagementPort = 8080
	// DefaultConnTimeout  Default connection timeout.
	DefaultConnTimeout    = 2
	// MinConnTimeout  Minimum connection timeout.
	MinConnTimeout        = 2
	// MaxConnTimeout  Maximum connection timeout.
	MaxConnTimeout        = 50
	// MaxDBNameLength  Maximum Database name length.
	MaxDBNameLength       = 256
	// MaxPortNumber  Maximum port number.
	MaxPortNumber         = 65535
	// DefaultTTL  Default TTL value.
	DefaultTTL            = 30
	// DNSUDPPacketSize  DNS UDP packet size.
	DNSUDPPacketSize      = 65535
	// ForwardRetryCount  Max Forward retry count.
	ForwardRetryCount     = 3
	// DefaultIP  default ip.
	DefaultIP             = "0.0.0.0"
	// MaxPacketSize  Maximum packet size.
	MaxPacketSize         = "4K"
)

const MaxDNSFQDNLength = 253
const MaxDNSQuestionLength = MaxDNSFQDNLength + 1

// MaxIPLength Considering IPV4(15), IPV6(39) and IPV4-mapped IPV6(45).
const MaxIPLength = 45
