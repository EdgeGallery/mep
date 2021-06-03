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
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"time"
)

/* ntp external interface functions*/
// DS which used to communicate with NTP server.

// CurrentTimeIf if struct
type CurrentTime struct {
	seconds          uint64 `json:"seconds"`
	nanoSeconds      uint64 `json:"nanoSeconds"`
	timeSourceStatus string `json:"timeSourceStatus"`
}

// TimingCaps ntp platform capabilities
//type TimingCaps struct {
//	timeStamp    byte `json:"timeStamp"`
//	seconds      uint64 `json:"seconds"`
//	nanoSeconds  uint64 `json:"nanoSeconds"`
//	ntpServers   string `json:"ntpServers"`
//	ntpServerAddrType    string `json:"ntpServerAddrType"`
//	ntpServerAddr	     string `json:"ntpServerAddr"`
//	minPollingInterval   int `json:"minPollingInterval"`
//	maxPollingInterval   int `json:"maxPollingInterval"`
//	localPriority        int `json:"localPriority"`
//	authenticationOption string `json:"authenticationOption"`
//	authenticationKeyNum int `json:"authenticationKeyNum"`
//	ptpMasters           string `json:"ptpMasters"`
//	ptpMasterIpAddress   string `json:"ptpMasterIpAddress"`
//	ptpMasterLocalPriority int `json:"authenticationKeyNum"`
//	delayReqMaxRate        int `json:"delayReqMaxRate"`
//}

const ntpEpochOffset = 2208988800

/* NTP packet format (v3 with optional v4 fields removed)

0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|LI | VN  |Mode |    Stratum     |     Poll      |  Precision   |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         Root Delay                            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         Root Dispersion                       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                          Reference ID                         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                     Reference Timestamp (64)                  +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                      Origin Timestamp (64)                    +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                      Receive Timestamp (64)                   +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                      Transmit Timestamp (64)                  +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/

type packet struct {
	Settings       uint8  // leap yr indicator, ver number, and mode
	Stratum        uint8  // stratum of local clock
	Poll           int8   // poll exponent
	Precision      int8   // precision exponent
	RootDelay      uint32 // root delay
	RootDispersion uint32 // root dispersion
	ReferenceID    uint32 // reference id
	RefTimeSec     uint32 // reference timestamp sec
	RefTimeFrac    uint32 // reference timestamp fractional
	OrigTimeSec    uint32 // origin time secs
	OrigTimeFrac   uint32 // origin time fractional
	RxTimeSec      uint32 // receive time secs
	RxTimeFrac     uint32 // receive time frac
	TxTimeSec      uint32 // transmit time secs
	TxTimeFrac     uint32 // transmit time frac
}

// GetCurrentTime to get current time form NTP server
func GetCurrentTime() (record []byte, seconds uint64, nanosecondsß uint64, errorCode int) {
	var data []byte
	data[0] = 1
	var host string
	flag.StringVar(&host, "e", "us.pool.ntp.org:123", "NTP host")
	flag.Parse()

	// Setup a UDP connection
	conn, err := net.Dial("udp", host)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
		log.Fatalf("failed to set deadline: %v", err)
	}

	/* configure request settings by specifying the first byte as
	00 011 011 (or 0x1B)
	|  |   +-- client mode (3)
	|  + ----- version (3)
	+ -------- leap year indicator, 0 no warning
	*/
	req := &packet{Settings: 0x1B}

	// send time request
	if err := binary.Write(conn, binary.BigEndian, req); err != nil {
		log.Fatalf("failed to send request: %v", err)
	}

	// block to receive server response
	rsp := &packet{}
	if err := binary.Read(conn, binary.BigEndian, rsp); err != nil {
		log.Fatalf("failed to read server response: %v", err)
	}

	/*  On POSIX-compliant OS, time is expressed
	using the Unix time epoch (or secs since year 1970).
	NTP seconds are counted since 1900 and therefore must
	be corrected with an epoch offset to convert NTP seconds
	to Unix time by removing 70 yrs of seconds (1970-1900)
	or 2208988800 seconds.
	*/

	secs := float64(rsp.TxTimeSec) - ntpEpochOffset
	nanos := (int64(rsp.TxTimeFrac) * 1e9) >> 32 // convert fractional to nanos
	fmt.Printf("%v\n", time.Unix(int64(secs), nanos))
	return data, uint64(secs), uint64(nanos), 0
}

// API connect to NTP server and get the response here
// called from mp1
