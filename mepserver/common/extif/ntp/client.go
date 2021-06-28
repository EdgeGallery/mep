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

package ntp

import (
	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/beevik/ntp"
	"mepserver/common/util"
)

// Package ntp provides an implementation of a Simple NTP (SNTP) client
// capable of querying the current time from a remote NTP server.  See
// RFC5905 (https://tools.ietf.org/html/rfc5905) for more details.

// NtpCurrentTime protocol message storing data structure
type NtpCurrentTime struct {
	Seconds          int
	NanoSeconds      int
	TimeSourceStatus string
}

func GetTimeStamp() (timeStamp *NtpCurrentTime, errorCode int) {
	var currentTime NtpCurrentTime
	ntpRsp, err := ntp.QueryWithOptions(util.NtpHost, ntp.QueryOptions{Version: 4})
	if ntpRsp == nil {
		log.Errorf(err, "Failed to read server response")
		return nil, util.NtpConnectionErr
	}

	// The number of seconds elapsed since January 1, 1970 UTC
	currentTime.Seconds = int(ntpRsp.Time.Unix())
	currentTime.NanoSeconds = ntpRsp.Time.Nanosecond() // Nanosecond part within the second

	if ntpRsp.Stratum >= 1 && ntpRsp.Stratum <= 15 {
		currentTime.TimeSourceStatus = util.Traceable
	} else {
		currentTime.TimeSourceStatus = util.NonTraceable
	}

	return &currentTime, 0
}
