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

package models

// CurrentTime ntp current time record
type CurrentTime struct {
	Seconds          int64  `json:"seconds"`
	NanoSeconds      int    `json:"nanoSeconds"`
	TimeSourceStatus string `json:"timeSourceStatus"`
}

// Timestamp record
type Timestamp struct {
	Seconds     int64 `json:"seconds"`
	NanoSeconds int   `json:"nanoSeconds"`
}

// NtpServers record
type NtpServers struct {
	NtpServerAddrType string `json:"ntpServerAddrType"`
	NtpServerAddr     string `json:"ntpServerAddr"`
	MinPolInterval    int    `json:"minPollingInterval"`
	MaxPolInterval    int    `json:"maxPollingInterval"`
	LocalPriority     int    `json:"localPriority"`
	AuthOption        string `json:"authenticationOption"`
	AuthKeyNum        int    `json:"authenticationKeyNum"`
}

// TimingCaps ntp timing capabilities record
type TimingCaps struct {
	TimeStamp  Timestamp  `json:"timeStamp"`
	NtpServers NtpServers `json:"ntpServers"`
}
