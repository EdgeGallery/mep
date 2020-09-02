/*
 *  Copyright 2020 Huawei Technologies Co., Ltd.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */
package datastore

import (
	"os"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

const (
	ExampleDomain       = "www.example.com."
	Example1Domain      = "www.example1.com."
	ExampleAbcDomain    = "abc.example.com."
	DNSConfigTestIP1    = "179.138.147.240"
	ErrorDeleteMessage  = "Error in deleting the record"
	ErrorSettingMessage = "Error in setting record"
)

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

	rrecord := ResourceRecord{Name: ExampleDomain, Type: "A", Class: "IN", TTL: 30, RData: []string{DNSConfigTestIP1}}
	err = store.SetResourceRecord(".", &rrecord)
	assert.Equal(t, nil, err, "Error in setting the record")

	question := dns.Question{Name: ExampleDomain, Qtype: dns.TypeA, Qclass: dns.ClassINET}
	rrResponse, err := store.GetResourceRecord(&question)
	assert.Equal(t, nil, err, "Error in reading the record")
	assert.NotEqual(t, 0, len(*rrResponse), "Not found")
	assert.Equal(t, "www.example.com.\t30\tIN\tA\t179.138.147.240", (*rrResponse)[0].String(), "Error")

	err = store.DelResourceRecord(ExampleDomain, "A")
	assert.Equal(t, nil, err, ErrorDeleteMessage)

	t.Run("QueryNonExistingRecord", func(t *testing.T) {
		question := dns.Question{Name: Example1Domain, Qtype: dns.TypeA, Qclass: dns.ClassINET}
		_, err = store.GetResourceRecord(&question)
		assert.EqualError(t, err, "could not process/retrieve the query", "Error in reading the db")
	})

	t.Run("QueryNonSupportedRRType", func(t *testing.T) {
		rrecord := ResourceRecord{Name: ExampleDomain, Type: "AA", Class: "IN", TTL: 30,
			RData: []string{DNSConfigTestIP1}}
		err = store.SetResourceRecord(".", &rrecord)
		assert.EqualError(t, err, "unsupported rrtype(AA) entry", ErrorSettingMessage)
	})

	t.Run("QueryNonSupportedRRClass", func(t *testing.T) {
		rrecord := ResourceRecord{Name: ExampleDomain, Type: "A", Class: "IN345", TTL: 30,
			RData: []string{DNSConfigTestIP1}}
		err = store.SetResourceRecord(".", &rrecord)
		assert.EqualError(t, err, "unsupported rrclass(IN345) entry", ErrorSettingMessage)
	})

	t.Run("QueryRRClassAny", func(t *testing.T) {
		rrecord := ResourceRecord{Name: ExampleDomain, Type: "A", Class: "*", TTL: 30,
			RData: []string{DNSConfigTestIP1}}
		err = store.SetResourceRecord(".", &rrecord)
		assert.EqualError(t, err, "unsupported rrclass(*) entry", ErrorSettingMessage)
	})

	t.Run("CheckCaseSensitiveInDomainName", func(t *testing.T) {
		_ = store.SetResourceRecord(".", &ResourceRecord{Name: "WWW.EXAMPLE.COM.", Type: "A",
			Class: "IN", TTL: 30, RData: []string{DNSConfigTestIP1}})
		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: ExampleDomain,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example.com.\t30\tIN\tA\t179.138.147.240", (*rrResponse)[0].String(),
			"Error")
		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: "WWW.EXAMPLE.COM.",
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "WWW.EXAMPLE.COM.\t30\tIN\tA\t179.138.147.240", (*rrResponse)[0].String(),
			"Error")

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: "WWW.example.COM.",
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "WWW.example.COM.\t30\tIN\tA\t179.138.147.240", (*rrResponse)[0].String(),
			"Error")

		err = store.DelResourceRecord(ExampleDomain, "A")
		assert.Equal(t, nil, err, ErrorDeleteMessage)
	})

	t.Run("UpdateARecord", func(t *testing.T) {
		_ = store.SetResourceRecord(".", &ResourceRecord{Name: ExampleDomain, Type: "A",
			Class: "IN", TTL: 30, RData: []string{DNSConfigTestIP1}})
		err = store.SetResourceRecord(".", &ResourceRecord{Name: ExampleDomain, Type: "A",
			Class: "IN", TTL: 30, RData: []string{"179.138.147.241"}})
		assert.Equal(t, nil, err, "Error in setting the db")
		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: ExampleDomain,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example.com.\t30\tIN\tA\t179.138.147.241", (*rrResponse)[0].String(),
			"Error")

		err = store.DelResourceRecord(ExampleDomain, "A")
		assert.Equal(t, nil, err, ErrorDeleteMessage)
	})

	t.Run("DeleteNonExistingRecord", func(t *testing.T) {
		err = store.DelResourceRecord(ExampleDomain, "A")
		assert.NotEqual(t, nil, err, "Error in deleting the db")
		assert.EqualError(t, err, "not found", ErrorSettingMessage)
	})

	t.Run("DeleteWithInvalidRRType", func(t *testing.T) {
		_ = store.SetResourceRecord(".", &ResourceRecord{Name: ExampleDomain, Type: "A",
			Class: "IN", TTL: 30, RData: []string{DNSConfigTestIP1}})

		err = store.DelResourceRecord(ExampleDomain, "None")
		assert.NotEqual(t, nil, err, "Error in deleting the db")
		assert.EqualError(t, err, "unsupported rrtype(None) entry", "Error in deleting record")

		err = store.DelResourceRecord(ExampleDomain, "A")
		assert.Equal(t, nil, err, ErrorDeleteMessage)
	})

	t.Run("NonDefaultZone", func(t *testing.T) {
		_ = store.SetResourceRecord("example.com.", &ResourceRecord{Name: ExampleAbcDomain, Type: "A",
			Class: "IN", TTL: 30, RData: []string{"179.138.147.242"}})

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: ExampleAbcDomain,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "abc.example.com.\t30\tIN\tA\t179.138.147.242", (*rrResponse)[0].String(),
			"Error")

		err = store.DelResourceRecord(ExampleAbcDomain, "A")
		assert.Equal(t, nil, err, ErrorDeleteMessage)
	})

	t.Run("MultiRecordDomains", func(t *testing.T) {
		_ = store.SetResourceRecord("example.com.", &ResourceRecord{Name: ExampleAbcDomain, Type: "A",
			Class: "IN", TTL: 30, RData: []string{"179.138.147.242", "179.138.147.243", "179.138.147.244"}})

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: ExampleAbcDomain,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, 3, len(*rrResponse), "Not found all records")
		assert.Equal(t, "abc.example.com.\t30\tIN\tA\t179.138.147.242", (*rrResponse)[0].String(),
			"Error")
		assert.Equal(t, "abc.example.com.\t30\tIN\tA\t179.138.147.243", (*rrResponse)[1].String(),
			"Error")
		assert.Equal(t, "abc.example.com.\t30\tIN\tA\t179.138.147.244", (*rrResponse)[2].String(),
			"Error")

		err = store.DelResourceRecord(ExampleAbcDomain, "A")
		assert.Equal(t, nil, err, ErrorDeleteMessage)
	})

	err = store.Close()
	assert.Equal(t, nil, err, "Error in closing the db")
}
