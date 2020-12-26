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
package datastore

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

const (
	exampleDomain          = "www.example.com."
	example1Domain         = "www.example1.com."
	exampleAbcDomain       = "abc.example.com."
	errorDeleteMessage     = "Error in deleting the record"
	errorSettingMessage    = "Error in setting record"
	maxIPVal               = 255
	ipAddFormatter         = "%d.%d.%d.%d"
	exampleRspFormatter    = "www.example.com.\t30\tIN\tA\t%s"
	abcExampleRspFormatter = "abc.example.com.\t30\tIN\tA\t%s"
)

// Generate test IP, instead of hard coding them
var dnsConfigTestIP1 = fmt.Sprintf(ipAddFormatter, rand.Intn(maxIPVal), rand.Intn(maxIPVal), rand.Intn(maxIPVal),
	rand.Intn(maxIPVal))
var dnsConfigTestIP2 = fmt.Sprintf(ipAddFormatter, rand.Intn(maxIPVal), rand.Intn(maxIPVal), rand.Intn(maxIPVal),
	rand.Intn(maxIPVal))
var dnsConfigTestIP3 = fmt.Sprintf(ipAddFormatter, rand.Intn(maxIPVal), rand.Intn(maxIPVal), rand.Intn(maxIPVal),
	rand.Intn(maxIPVal))
var dnsConfigTestIP4 = fmt.Sprintf(ipAddFormatter, rand.Intn(maxIPVal), rand.Intn(maxIPVal), rand.Intn(maxIPVal),
	rand.Intn(maxIPVal))
var dnsConfigTestIP5 = fmt.Sprintf(ipAddFormatter, rand.Intn(maxIPVal), rand.Intn(maxIPVal), rand.Intn(maxIPVal),
	rand.Intn(maxIPVal))

// Query dns rules request in mp1 interface
func TestBasicDataStoreOperations(t *testing.T) {
	defer func() {
		_ = os.RemoveAll(DBPath)
		if r := recover(); r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	store := &BoltDB{FileName: "testdb", TTL: 30}
	err := store.Open()
	assert.Equal(t, nil, err, "Error in opening the db")

	rrecord := ResourceRecord{Name: exampleDomain, Type: "A", Class: "IN", TTL: 30, RData: []string{dnsConfigTestIP1}}
	err = store.SetResourceRecord(".", &rrecord)
	assert.Equal(t, nil, err, "Error in setting the record")

	question := dns.Question{Name: exampleDomain, Qtype: dns.TypeA, Qclass: dns.ClassINET}
	rrResponse, err := store.GetResourceRecord(&question)
	assert.Equal(t, nil, err, "Error in reading the record")
	assert.NotEqual(t, 0, len(*rrResponse), "Not found")
	assert.Equal(t, fmt.Sprintf(exampleRspFormatter, dnsConfigTestIP1),
		(*rrResponse)[0].String(), "Error")

	err = store.DelResourceRecord(exampleDomain, "A")
	assert.Equal(t, nil, err, errorDeleteMessage)

	t.Run("QueryNonExistingRecord", func(t *testing.T) {
		question := dns.Question{Name: example1Domain, Qtype: dns.TypeA, Qclass: dns.ClassINET}
		_, err = store.GetResourceRecord(&question)
		assert.EqualError(t, err, "could not process/retrieve the query", "Error in reading the db")
	})

	t.Run("QueryNonSupportedRRType", func(t *testing.T) {
		rrecord := ResourceRecord{Name: exampleDomain, Type: "AA", Class: "IN", TTL: 30,
			RData: []string{dnsConfigTestIP1}}
		err = store.SetResourceRecord(".", &rrecord)
		assert.EqualError(t, err, "unsupported rrtype(AA) entry", errorSettingMessage)
	})

	t.Run("QueryNonSupportedRRClass", func(t *testing.T) {
		rrecord := ResourceRecord{Name: exampleDomain, Type: "A", Class: "IN345", TTL: 30,
			RData: []string{dnsConfigTestIP1}}
		err = store.SetResourceRecord(".", &rrecord)
		assert.EqualError(t, err, "unsupported rrclass(IN345) entry", errorSettingMessage)
	})

	t.Run("QueryRRClassAny", func(t *testing.T) {
		rrecord := ResourceRecord{Name: exampleDomain, Type: "A", Class: "*", TTL: 30,
			RData: []string{dnsConfigTestIP1}}
		err = store.SetResourceRecord(".", &rrecord)
		assert.EqualError(t, err, "unsupported rrclass(*) entry", errorSettingMessage)
	})

	t.Run("CheckCaseSensitiveInDomainName", func(t *testing.T) {
		_ = store.SetResourceRecord(".", &ResourceRecord{Name: "WWW.EXAMPLE.COM.", Type: "A",
			Class: "IN", TTL: 30, RData: []string{dnsConfigTestIP1}})
		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: exampleDomain,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, fmt.Sprintf(exampleRspFormatter, dnsConfigTestIP1), (*rrResponse)[0].String(),
			"Error")
		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: "WWW.EXAMPLE.COM.",
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, fmt.Sprintf("WWW.EXAMPLE.COM.\t30\tIN\tA\t%s", dnsConfigTestIP1), (*rrResponse)[0].String(),
			"Error")

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: "WWW.example.COM.",
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, fmt.Sprintf("WWW.example.COM.\t30\tIN\tA\t%s", dnsConfigTestIP1), (*rrResponse)[0].String(),
			"Error")

		err = store.DelResourceRecord(exampleDomain, "A")
		assert.Equal(t, nil, err, errorDeleteMessage)
	})

	t.Run("UpdateARecord", func(t *testing.T) {
		_ = store.SetResourceRecord(".", &ResourceRecord{Name: exampleDomain, Type: "A",
			Class: "IN", TTL: 30, RData: []string{dnsConfigTestIP1}})
		err = store.SetResourceRecord(".", &ResourceRecord{Name: exampleDomain, Type: "A",
			Class: "IN", TTL: 30, RData: []string{dnsConfigTestIP2}})
		assert.Equal(t, nil, err, "Error in setting the db")
		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: exampleDomain,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, fmt.Sprintf(exampleRspFormatter, dnsConfigTestIP2), (*rrResponse)[0].String(),
			"Error")

		err = store.DelResourceRecord(exampleDomain, "A")
		assert.Equal(t, nil, err, errorDeleteMessage)
	})

	t.Run("DeleteNonExistingRecord", func(t *testing.T) {
		err = store.DelResourceRecord(exampleDomain, "A")
		assert.NotEqual(t, nil, err, "Error in deleting the db")
		assert.EqualError(t, err, "not found", errorSettingMessage)
	})

	t.Run("DeleteWithInvalidRRType", func(t *testing.T) {
		_ = store.SetResourceRecord(".", &ResourceRecord{Name: exampleDomain, Type: "A",
			Class: "IN", TTL: 30, RData: []string{dnsConfigTestIP1}})

		err = store.DelResourceRecord(exampleDomain, "None")
		assert.NotEqual(t, nil, err, "Error in deleting the db")
		assert.EqualError(t, err, "unsupported rrtype(None) entry", "Error in deleting record")

		err = store.DelResourceRecord(exampleDomain, "A")
		assert.Equal(t, nil, err, errorDeleteMessage)
	})

	t.Run("NonDefaultZone", func(t *testing.T) {
		_ = store.SetResourceRecord("example.com.", &ResourceRecord{Name: exampleAbcDomain, Type: "A",
			Class: "IN", TTL: 30, RData: []string{dnsConfigTestIP3}})

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: exampleAbcDomain,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, fmt.Sprintf(abcExampleRspFormatter, dnsConfigTestIP3), (*rrResponse)[0].String(),
			"Error")

		err = store.DelResourceRecord(exampleAbcDomain, "A")
		assert.Equal(t, nil, err, errorDeleteMessage)
	})

	t.Run("MultiRecordDomains", func(t *testing.T) {
		_ = store.SetResourceRecord("example.com.", &ResourceRecord{Name: exampleAbcDomain, Type: "A",
			Class: "IN", TTL: 30, RData: []string{dnsConfigTestIP3, dnsConfigTestIP4, dnsConfigTestIP5}})

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: exampleAbcDomain,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, 3, len(*rrResponse), "Not found all records")
		assert.Equal(t, fmt.Sprintf(abcExampleRspFormatter, dnsConfigTestIP3), (*rrResponse)[0].String(),
			"Error")
		assert.Equal(t, fmt.Sprintf(abcExampleRspFormatter, dnsConfigTestIP4), (*rrResponse)[1].String(),
			"Error")
		assert.Equal(t, fmt.Sprintf(abcExampleRspFormatter, dnsConfigTestIP5), (*rrResponse)[2].String(),
			"Error")

		err = store.DelResourceRecord(exampleAbcDomain, "A")
		assert.Equal(t, nil, err, errorDeleteMessage)
	})

	err = store.Close()
	assert.Equal(t, nil, err, "Error in closing the db")
}
