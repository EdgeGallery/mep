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
	"net"
	"os"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"

	"dns-server/datastore"
	"dns-server/mgmt"
	"dns-server/util"
)

const (
	DefaultTestForwarder = "8.8.8.8"
	TestDomainServer     = "www.edgegallery.org."
	ErrorForwarding      = "Error in forwarding"
	ErrorInResponse      = "Error in response"
	PanicImplement       = "implement me"
	ExampleDomain        = "www.example.com."
)

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
	var forwarder = DefaultTestForwarder
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
		dnsMsg.Question[0] = dns.Question{Name: TestDomainServer, Qtype: dns.TypeA, Qclass: dns.ClassINET}

		rsp, err := dnsServer.forward(dnsMsg)
		assert.Equal(t, nil, err, ErrorForwarding)
		assert.Contains(t, rsp.Answer[0].String(), TestDomainServer, ErrorForwarding)
	})

	t.Run("NonExistingDomainName", func(t *testing.T) {
		dnsMsg := new(dns.Msg)
		dnsMsg.Id = dns.Id()
		dnsMsg.RecursionDesired = true
		dnsMsg.Question = make([]dns.Question, 1)
		dnsMsg.Question[0] = dns.Question{Name: "www.edgegallery0000111.org.", Qtype: dns.TypeA, Qclass: dns.ClassINET}
		_, err := dnsServer.forward(dnsMsg)
		assert.NotEqual(t, nil, err, ErrorForwarding)
		assert.EqualError(t, err, "forward of request \"www.edgegallery0000111.org.\" was not "+
			"accepted", ErrorForwarding)
	})

	t.Run("WrongForwardAddress", func(t *testing.T) {
		config.forwarder = net.ParseIP("0.0.0.0")
		defer func() { config.forwarder = net.ParseIP(DefaultTestForwarder) }()

		dnsMsg := new(dns.Msg)
		dnsMsg.Id = dns.Id()
		dnsMsg.RecursionDesired = true
		dnsMsg.Question = make([]dns.Question, 1)
		dnsMsg.Question[0] = dns.Question{Name: TestDomainServer, Qtype: dns.TypeA, Qclass: dns.ClassINET}

		_, err := dnsServer.forward(dnsMsg)
		assert.NotEqual(t, nil, err, ErrorForwarding)
		assert.EqualError(t, err, "could not resolve the request \"www.edgegallery.org.\" and no forwarder is "+
			"configured", ErrorForwarding)
	})

}

type mockDnsRespWriter struct {
	// mock.Mock
	// dns.ResponseWriter
	rspMsg *dns.Msg
}

func (m *mockDnsRespWriter) LocalAddr() net.Addr {
	panic(PanicImplement)
}

func (m *mockDnsRespWriter) RemoteAddr() net.Addr {
	panic(PanicImplement)
}

func (m *mockDnsRespWriter) WriteMsg(msg *dns.Msg) error {
	// retrieve the configured value we provided at the input and return it back
	m.rspMsg = msg
	return nil
}

func (m *mockDnsRespWriter) Write(bytes []byte) (int, error) {
	panic(PanicImplement)
}

func (m *mockDnsRespWriter) Close() error {
	panic(PanicImplement)
}

func (m *mockDnsRespWriter) TsigStatus() error {
	panic(PanicImplement)
}

func (m *mockDnsRespWriter) TsigTimersOnly(b bool) {
	panic(PanicImplement)
}

func (m *mockDnsRespWriter) Hijack() {
	panic(PanicImplement)
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
	var forwarder = DefaultTestForwarder
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
	rrecord := datastore.ResourceRecord{Name: ExampleDomain, Type: "A", Class: "IN", TTL: 30,
		RData: []string{"179.138.147.240"}}
	err = store.SetResourceRecord(".", &rrecord)
	assert.Equal(t, nil, err, "Error in setting the record")

	t.Run("BasicTest", func(t *testing.T) {
		req := &dns.Msg{Question: []dns.Question{{Name: ExampleDomain,
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET}}}
		mockDnsWriter := &mockDnsRespWriter{}
		dnsServer.handleDNS(mockDnsWriter, req)
		assert.NotEqual(t, nil, mockDnsWriter.rspMsg, ErrorInResponse)
		assert.Equal(t, "www.example.com.\t30\tIN\tA\t179.138.147.240", mockDnsWriter.rspMsg.Answer[0].String(), ErrorInResponse)
	})

	t.Run("QuestionEmpty", func(t *testing.T) {
		req := &dns.Msg{Question: []dns.Question{}}
		mockDnsWriter := &mockDnsRespWriter{}
		dnsServer.handleDNS(mockDnsWriter, req)
		assert.Equal(t, dns.RcodeFormatError, mockDnsWriter.rspMsg.Rcode, ErrorInResponse)
	})

	t.Run("OpCodeError", func(t *testing.T) {
		req := &dns.Msg{Question: []dns.Question{{Name: ExampleDomain,
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET}}}
		req.Opcode = dns.OpcodeStatus
		mockDnsWriter := &mockDnsRespWriter{}
		dnsServer.handleDNS(mockDnsWriter, req)
		assert.Equal(t, dns.RcodeRefused, mockDnsWriter.rspMsg.Rcode, ErrorInResponse)
	})

	t.Run("NonExistingQuery", func(t *testing.T) {
		req := &dns.Msg{Question: []dns.Question{{Name: "www.example12dvfse5652.com.",
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET}}}
		mockDnsWriter := &mockDnsRespWriter{}
		dnsServer.handleDNS(mockDnsWriter, req)
		assert.Equal(t, dns.RcodeServerFailure, mockDnsWriter.rspMsg.Rcode, ErrorInResponse)
	})

	t.Run("ForwardingQuery", func(t *testing.T) {
		dnsMsg := new(dns.Msg)
		dnsMsg.Id = dns.Id()
		dnsMsg.RecursionDesired = true
		dnsMsg.Question = make([]dns.Question, 1)
		dnsMsg.Question[0] = dns.Question{Name: TestDomainServer, Qtype: dns.TypeA, Qclass: dns.ClassINET}

		mockDnsWriter := &mockDnsRespWriter{}
		dnsServer.handleDNS(mockDnsWriter, dnsMsg)
		assert.NotEqual(t, nil, mockDnsWriter.rspMsg, ErrorInResponse)
		// assert.Equal(t, "www.example.com.\t30\tIN\tA\t179.138.147.240", mockDnsWriter.rspMsg.Answer[0].String(), ErrorInResponse)
		assert.Contains(t, mockDnsWriter.rspMsg.Answer[0].String(), TestDomainServer, ErrorInResponse)
	})

}
