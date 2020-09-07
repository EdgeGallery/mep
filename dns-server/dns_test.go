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
package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"

	"dns-server/datastore"
	"dns-server/mgmt"
	"dns-server/util"
)

const (
	testDomainServer  = "www.edgegallery.org."
	testInvalidDomain = "www.example12dvfse5652.com."
	errorForwarding   = "Error in forwarding"
	errorInResponse   = "Error in response"
	panicImplement    = "implement me"
	exampleDomain     = "www.example.com."
	maxIPVal          = 255
	ipAddFormatter    = "%d.%d.%d.%d"
)

// Generate test IP, instead of hard coding them
var dnsConfigTestIP1 = fmt.Sprintf(ipAddFormatter, rand.Intn(maxIPVal), rand.Intn(maxIPVal), rand.Intn(maxIPVal),
	rand.Intn(maxIPVal))

var defaultTestForwarder = fmt.Sprintf(ipAddFormatter, rand.Intn(maxIPVal), rand.Intn(maxIPVal), rand.Intn(maxIPVal),
	rand.Intn(maxIPVal))

func TestForward(t *testing.T) {
	defer func() {
		_ = os.RemoveAll(datastore.DBPath)
		r := recover()
		if r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	var dbName = "test_db"
	var port uint = util.DefaultDnsPort
	var mgmtPort uint = util.DefaultManagementPort
	var connTimeOut uint = util.DefaultConnTimeout
	var ipAddString = util.DefaultIP
	var ipMgmtAddString = util.DefaultIP
	var forwarder = defaultTestForwarder
	var loadBalance = false
	parameters := &InputParameters{&dbName, &port, &mgmtPort, &connTimeOut,
		&ipAddString, &ipMgmtAddString, &forwarder, &loadBalance}
	config := validateInputAndGenerateConfig(parameters)

	store := &datastore.BoltDB{FileName: config.dbName, TTL: util.DefaultTTL}
	mgmtCtl := &mgmt.Controller{}
	dnsServer := NewServer(config, store, mgmtCtl)

	t.Run("BasicTest", func(t *testing.T) {
		dnsMsg := new(dns.Msg)
		dnsMsg.Id = dns.Id()
		dnsMsg.RecursionDesired = true
		dnsMsg.Question = make([]dns.Question, 1)
		dnsMsg.Question[0] = dns.Question{Name: testDomainServer, Qtype: dns.TypeA, Qclass: dns.ClassINET}

		var c *dns.Client
		patch1 := gomonkey.ApplyMethod(reflect.TypeOf(c), "Exchange", func(client *dns.Client, m *dns.Msg,
			address string) (r *dns.Msg, rtt time.Duration, err error) {
			m.Rcode = dns.RcodeSuccess
			m.Answer = make([]dns.RR, 1)
			m.Answer[0] = &dns.A{Hdr: dns.RR_Header{Name: m.Question[0].Name, Rrtype: m.Question[0].Qtype,
				Class: dns.ClassINET, Ttl: 30}, A: net.ParseIP(dnsConfigTestIP1)}
			return m, 10, nil
		})
		defer patch1.Reset()

		rsp, err := dnsServer.forward(dnsMsg)
		assert.Equal(t, nil, err, errorForwarding)
		assert.Contains(t, rsp.Answer[0].String(), testDomainServer, errorForwarding)
	})

	t.Run("NonExistingDomainName", func(t *testing.T) {
		dnsMsg := new(dns.Msg)
		dnsMsg.Id = dns.Id()
		dnsMsg.RecursionDesired = true
		dnsMsg.Question = make([]dns.Question, 1)
		dnsMsg.Question[0] = dns.Question{Name: "www.edgegallery0000111.org.", Qtype: dns.TypeA, Qclass: dns.ClassINET}
		var c *dns.Client
		patch1 := gomonkey.ApplyMethod(reflect.TypeOf(c), "Exchange", func(client *dns.Client, m *dns.Msg,
			address string) (r *dns.Msg, rtt time.Duration, err error) {
			m.Rcode = dns.RcodeServerFailure
			m.Answer = make([]dns.RR, 1)
			m.Answer[0] = &dns.A{Hdr: dns.RR_Header{Name: m.Question[0].Name, Rrtype: m.Question[0].Qtype,
				Class: dns.ClassINET, Ttl: 30}, A: net.ParseIP(dnsConfigTestIP1)}
			return m, 10, nil
		})
		defer patch1.Reset()

		_, err := dnsServer.forward(dnsMsg)
		assert.NotEqual(t, nil, err, errorForwarding)
		assert.EqualError(t, err, "forward of request \"www.edgegallery0000111.org.\" was not "+
			"accepted", errorForwarding)
	})

	t.Run("WrongForwardAddress", func(t *testing.T) {
		config.forwarder = net.ParseIP("0.0.0.0")
		defer func() { config.forwarder = net.ParseIP(defaultTestForwarder) }()

		dnsMsg := new(dns.Msg)
		dnsMsg.Id = dns.Id()
		dnsMsg.RecursionDesired = true
		dnsMsg.Question = make([]dns.Question, 1)
		dnsMsg.Question[0] = dns.Question{Name: testDomainServer, Qtype: dns.TypeA, Qclass: dns.ClassINET}

		_, err := dnsServer.forward(dnsMsg)
		assert.NotEqual(t, nil, err, errorForwarding)
		assert.EqualError(t, err, "could not resolve the request \"www.edgegallery.org.\" and no forwarder is "+
			"configured", errorForwarding)
	})

}

type mockDnsRespWriter struct {
	// mock.Mock
	// dns.ResponseWriter
	rspMsg *dns.Msg
}

func (m *mockDnsRespWriter) LocalAddr() net.Addr {
	panic(panicImplement)
}

func (m *mockDnsRespWriter) RemoteAddr() net.Addr {
	panic(panicImplement)
}

func (m *mockDnsRespWriter) WriteMsg(msg *dns.Msg) error {
	// retrieve the configured value we provided at the input and return it back
	m.rspMsg = msg
	return nil
}

func (m *mockDnsRespWriter) Write(bytes []byte) (int, error) {
	panic(panicImplement)
}

func (m *mockDnsRespWriter) Close() error {
	panic(panicImplement)
}

func (m *mockDnsRespWriter) TsigStatus() error {
	panic(panicImplement)
}

func (m *mockDnsRespWriter) TsigTimersOnly(b bool) {
	panic(panicImplement)
}

func (m *mockDnsRespWriter) Hijack() {
	panic(panicImplement)
}

func TestHandleDNS(t *testing.T) {
	defer func() {
		_ = os.RemoveAll(datastore.DBPath)
		r := recover()
		if r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	var dbName = "test_db"
	var port uint = util.DefaultDnsPort
	var mgmtPort uint = util.DefaultManagementPort
	var connTimeOut uint = util.DefaultConnTimeout
	var ipAddString = util.DefaultIP
	var ipMgmtAddString = util.DefaultIP
	var forwarder = defaultTestForwarder
	var loadBalance = false
	parameters := &InputParameters{&dbName, &port, &mgmtPort, &connTimeOut,
		&ipAddString, &ipMgmtAddString, &forwarder, &loadBalance}
	config := validateInputAndGenerateConfig(parameters)

	store := &datastore.BoltDB{FileName: config.dbName, TTL: util.DefaultTTL}
	mgmtCtl := &mgmt.Controller{}
	dnsServer := NewServer(config, store, mgmtCtl)

	err := store.Open()
	assert.Equal(t, nil, err, "Error in opening the db")
	defer store.Close()
	rrecord := datastore.ResourceRecord{Name: exampleDomain, Type: "A", Class: "IN", TTL: 30,
		RData: []string{dnsConfigTestIP1}}
	err = store.SetResourceRecord(".", &rrecord)
	assert.Equal(t, nil, err, "Error in setting the record")

	var c *dns.Client
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(c), "Exchange", func(client *dns.Client, m *dns.Msg,
		address string) (r *dns.Msg, rtt time.Duration, err error) {
		m.Rcode = dns.RcodeSuccess
		if m.Question[0].Name == testInvalidDomain {
			m.Rcode = dns.RcodeServerFailure
		}

		m.Answer = make([]dns.RR, 1)
		m.Answer[0] = &dns.A{Hdr: dns.RR_Header{Name: m.Question[0].Name, Rrtype: m.Question[0].Qtype,
			Class: dns.ClassINET, Ttl: 30}, A: net.ParseIP(dnsConfigTestIP1)}
		return m, 10, nil
	})
	defer patch1.Reset()

	t.Run("BasicTest", func(t *testing.T) {
		req := &dns.Msg{Question: []dns.Question{{Name: exampleDomain,
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET}}}
		mockDnsWriter := &mockDnsRespWriter{}
		dnsServer.handleDNS(mockDnsWriter, req)
		assert.NotEqual(t, nil, mockDnsWriter.rspMsg, errorInResponse)
		assert.Equal(t, fmt.Sprintf("www.example.com.\t30\tIN\tA\t%s", dnsConfigTestIP1),
			mockDnsWriter.rspMsg.Answer[0].String(),
			errorInResponse)
	})

	t.Run("QuestionEmpty", func(t *testing.T) {
		req := &dns.Msg{Question: []dns.Question{}}
		mockDnsWriter := &mockDnsRespWriter{}
		dnsServer.handleDNS(mockDnsWriter, req)
		assert.Equal(t, dns.RcodeFormatError, mockDnsWriter.rspMsg.Rcode, errorInResponse)
	})

	t.Run("OpCodeError", func(t *testing.T) {
		req := &dns.Msg{Question: []dns.Question{{Name: exampleDomain,
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET}}}
		req.Opcode = dns.OpcodeStatus
		mockDnsWriter := &mockDnsRespWriter{}
		dnsServer.handleDNS(mockDnsWriter, req)
		assert.Equal(t, dns.RcodeRefused, mockDnsWriter.rspMsg.Rcode, errorInResponse)
	})

	t.Run("NonExistingQuery", func(t *testing.T) {
		req := &dns.Msg{Question: []dns.Question{{Name: "www.example12dvfse5652.com.",
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET}}}
		mockDnsWriter := &mockDnsRespWriter{}
		dnsServer.handleDNS(mockDnsWriter, req)
		assert.Equal(t, dns.RcodeServerFailure, mockDnsWriter.rspMsg.Rcode, errorInResponse)
	})

	t.Run("ForwardingQuery", func(t *testing.T) {
		dnsMsg := new(dns.Msg)
		dnsMsg.Id = dns.Id()
		dnsMsg.RecursionDesired = true
		dnsMsg.Question = make([]dns.Question, 1)
		dnsMsg.Question[0] = dns.Question{Name: testDomainServer, Qtype: dns.TypeA, Qclass: dns.ClassINET}

		mockDnsWriter := &mockDnsRespWriter{}
		dnsServer.handleDNS(mockDnsWriter, dnsMsg)
		assert.NotEqual(t, nil, mockDnsWriter.rspMsg, errorInResponse)
		// assert.Equal(t, "www.example.com.\t30\tIN\tA\t179.138.147.240", mockDnsWriter.rspMsg.Answer[0].String(), errorInResponse)
		assert.Contains(t, mockDnsWriter.rspMsg.Answer[0].String(), testDomainServer, errorInResponse)
	})

}
