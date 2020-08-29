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
	var ipAddString = "0.0.0.0"
	var ipMgmtAddString = "0.0.0.0"
	var forwarder = "8.8.8.8"
	var loadBalance = false
	parameters := &InputParameters{&dbName, &port, &mgmtPort, &connTimeOut,
		&ipAddString, &ipMgmtAddString, &forwarder, &loadBalance}
	config := validateInputAndGenerateConfig(parameters)

	store := &datastore.BoltDB{FileName: config.dbName, TTL: util.DefaultTTL}
	mgmtCtl := &mgmt.EchoController{}
	dnsServer := NewServer(config, store, mgmtCtl)

	t.Run("BasicTest", func(t *testing.T) {
		rsp, err := dnsServer.forward(&dns.Msg{Question: []dns.Question{{Name: "www.google.com.", Qtype: dns.TypeA,
			Qclass: dns.ClassINET}}})
		assert.Equal(t, nil, err, "Error in forwarding")
		assert.Contains(t, rsp.Answer[0].String(), "www.google.com.", "Error in forwarding")
	})

	t.Run("NonExistingDomainName", func(t *testing.T) {
		_, err := dnsServer.forward(&dns.Msg{Question: []dns.Question{{Name: "www.goooooo3232o234gle.com.",
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET}}})
		assert.NotEqual(t, nil, err, "Error in forwarding")
		assert.EqualError(t, err, "forward of request \"www.goooooo3232o234gle.com.\" was not accepted "+
			"by \"8.8.8.8\", return code: SERVFAIL", "Error in forwarding")
	})

	t.Run("WrongForwardAddress", func(t *testing.T) {
		config.forwarder = net.ParseIP("0.0.0.0")
		defer func() { config.forwarder = net.ParseIP("8.8.8.8") }()
		_, err := dnsServer.forward(&dns.Msg{Question: []dns.Question{{Name: "www.google.com.",
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET}}})
		assert.NotEqual(t, nil, err, "Error in forwarding")
		assert.EqualError(t, err, "forward of request \"www.google.com.\" was not accepted "+
			"by \"0.0.0.0\"", "Error in forwarding")
	})

}

type mockDnsRespWriter struct {
	// mock.Mock
	// dns.ResponseWriter
	rspMsg *dns.Msg
}

func (m *mockDnsRespWriter) LocalAddr() net.Addr {
	panic("implement me")
}

func (m *mockDnsRespWriter) RemoteAddr() net.Addr {
	panic("implement me")
}

func (m *mockDnsRespWriter) WriteMsg(msg *dns.Msg) error {
	// retrieve the configured value we provided at the input and return it back
	m.rspMsg = msg
	return nil
}

func (m *mockDnsRespWriter) Write(bytes []byte) (int, error) {
	panic("implement me")
}

func (m *mockDnsRespWriter) Close() error {
	panic("implement me")
}

func (m *mockDnsRespWriter) TsigStatus() error {
	panic("implement me")
}

func (m *mockDnsRespWriter) TsigTimersOnly(b bool) {
	panic("implement me")
}

func (m *mockDnsRespWriter) Hijack() {
	panic("implement me")
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
	var ipAddString = "0.0.0.0"
	var ipMgmtAddString = "0.0.0.0"
	var forwarder = "8.8.8.8"
	var loadBalance = false
	parameters := &InputParameters{&dbName, &port, &mgmtPort, &connTimeOut,
		&ipAddString, &ipMgmtAddString, &forwarder, &loadBalance}
	config := validateInputAndGenerateConfig(parameters)

	store := &datastore.BoltDB{FileName: config.dbName, TTL: util.DefaultTTL}
	mgmtCtl := &mgmt.EchoController{}
	dnsServer := NewServer(config, store, mgmtCtl)

	err := store.Open()
	assert.Equal(t, nil, err, "Error in opening the db")
	defer store.Close()
	rrecord := datastore.ResourceRecord{Name: "www.example.com.", Type: "A", Class: "IN", TTL: 30,
		RData: []string{"179.138.147.240"}}
	err = store.SetResourceRecord(".", &rrecord)
	assert.Equal(t, nil, err, "Error in setting the record")

	t.Run("BasicTest", func(t *testing.T) {
		req := &dns.Msg{Question: []dns.Question{{Name: "www.example.com.",
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET}}}
		mockDnsWriter := &mockDnsRespWriter{}
		dnsServer.handleDNS(mockDnsWriter, req)
		assert.NotEqual(t, nil, mockDnsWriter.rspMsg, "Error in response")
		assert.Equal(t, "www.example.com.\t30\tIN\tA\t179.138.147.240", mockDnsWriter.rspMsg.Answer[0].String(), "Error in response")
	})

	t.Run("QuestionEmpty", func(t *testing.T) {
		req := &dns.Msg{Question: []dns.Question{}}
		mockDnsWriter := &mockDnsRespWriter{}
		dnsServer.handleDNS(mockDnsWriter, req)
		assert.Equal(t, dns.RcodeFormatError, mockDnsWriter.rspMsg.Rcode, "Error in response")
	})

	t.Run("OpCodeError", func(t *testing.T) {
		req := &dns.Msg{Question: []dns.Question{{Name: "www.example.com.",
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET}}}
		req.Opcode = dns.OpcodeStatus
		mockDnsWriter := &mockDnsRespWriter{}
		dnsServer.handleDNS(mockDnsWriter, req)
		assert.Equal(t, dns.RcodeRefused, mockDnsWriter.rspMsg.Rcode, "Error in response")
	})

	t.Run("NonExistingQuery", func(t *testing.T) {
		req := &dns.Msg{Question: []dns.Question{{Name: "www.example12dvfse5652.com.",
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET}}}
		mockDnsWriter := &mockDnsRespWriter{}
		dnsServer.handleDNS(mockDnsWriter, req)
		assert.Equal(t, dns.RcodeServerFailure, mockDnsWriter.rspMsg.Rcode, "Error in response")
	})

	t.Run("ForwardingQuery", func(t *testing.T) {
		req := &dns.Msg{Question: []dns.Question{{Name: "www.google.com.",
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET}}}
		mockDnsWriter := &mockDnsRespWriter{}
		dnsServer.handleDNS(mockDnsWriter, req)
		assert.NotEqual(t, nil, mockDnsWriter.rspMsg, "Error in response")
		// assert.Equal(t, "www.example.com.\t30\tIN\tA\t179.138.147.240", mockDnsWriter.rspMsg.Answer[0].String(), "Error in response")
		assert.Contains(t, mockDnsWriter.rspMsg.Answer[0].String(), "www.google.com.", "Error in response")
	})

}
