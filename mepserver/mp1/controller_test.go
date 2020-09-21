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

package mp1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/agiledragon/gomonkey"
	_ "github.com/apache/servicecomb-service-center/server"
	_ "github.com/apache/servicecomb-service-center/server/bootstrap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"mepserver/common/extif/backend"
	"mepserver/common/extif/dns"
	"mepserver/common/util"
)

type mockHttpWriter struct {
	mock.Mock
	response []byte
}

const defaultAppInstanceId = "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"
const dnsRuleId = "7d71e54e-81f3-47bb-a2fc-b565a326d794"

const panicFormatString = "Panic: %v"
const getDnsRulesUrlFormat = "/mep/mec_app_support/v1/applications/%s/dns_rules"
const getDnsRuleUrlFormat = "/mep/mec_app_support/v1/applications/%s/dns_rules/%s"
const appInstanceQueryFormat = ":appInstanceId=%s&;"
const appIdAndDnsRuleIdQueryFormat = ":appInstanceId=%s&;:dnsRuleId=%s&;"
const appInstanceIdHeader = "X-AppinstanceID"
const responseStatusHeader = "X-Response-Status"
const responseCheckFor200 = "Response status code must be 200"
const responseCheckFor400 = "Response status code must be 404"
const errorWriteRespErr = "Write Response Error"
const exampleDomainName = "www.example.com"
const defaultTTL = 30
const maxIPVal = 255
const ipAddFormatter = "%d.%d.%d.%d"
const writeObjectFormat = "{\"dnsRuleId\":\"7d71e54e-81f3-47bb-a2fc-b565a326d794\",\"domainName\":\"www.example.com\"," +
	"\"ipAddressType\":\"IP_V4\",\"ipAddress\":\"%s\",\"ttl\":30,\"state\":\"%s\"}\n"

// Generate test IP, instead of hard coding them
var exampleIPAddress = fmt.Sprintf(ipAddFormatter, rand.Intn(maxIPVal), rand.Intn(maxIPVal), rand.Intn(maxIPVal),
	rand.Intn(maxIPVal))

func (m *mockHttpWriter) Header() http.Header {
	// Get the argument inputs
	args := m.Called()
	// retrieve the configured value we provided at the input and return it back
	return args.Get(0).(http.Header)
}
func (m *mockHttpWriter) Write(response []byte) (int, error) {
	fmt.Printf("Write: %v", response)
	// Get the argument inputs and marking the function is called with correct input
	args := m.Called(response)

	if response != nil {
		m.response = bytes.Join([][]byte{m.response, response}, []byte(""))
	}
	// retrieve the configured value we provided at the input and return it back
	// return args.Get(0).(http.Header)
	return args.Int(0), args.Error(1)
}
func (m *mockHttpWriter) WriteHeader(statusCode int) {
	// Get the argument inputs and marking the function is called with correct input
	m.Called(statusCode)
	return
}

type dnsCreateRule struct {
	DomainName    string `json:"domainName"`
	IpAddressType string `json:"ipAddressType"`
	IpAddress     string `json:"ipAddress"`
	TTL           int    `json:"ttl"`
	State         string `json:"state"`
}

// Query dns rules request in mp1 interface
func TestGetDnsRules(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getDnsRulesUrlFormat, defaultAppInstanceId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte(fmt.Sprintf("["+writeObjectFormat[:len(writeObjectFormat)-1]+"]\n", exampleIPAddress,
		util.InactiveState))).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := dns.RuleEntry{DomainName: exampleDomainName, IpAddressType: "IP_V4", IpAddress: exampleIPAddress,
			TTL: 30, State: util.InactiveState}
		outBytes, _ := json.Marshal(&entry)
		records[dnsRuleId] = outBytes
		return records, 0
	})
	defer patches.Reset()

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[13].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)

}

// Query an empty dns rules request in mp1 interface
func TestGetEmptyDnsRules(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getDnsRulesUrlFormat, defaultAppInstanceId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("null\n")).Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[13].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)

}

// Query empty dns rules with unmatched application instance id
func TestGetEmptyDnsRulesAppInstanceIdUnMatched(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getDnsRulesUrlFormat, defaultAppInstanceId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, "wrong-app-instance-id")

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte(
		"{\"title\":\"UnAuthorization\",\"status\":11,\"detail\":\"UnAuthorization to access the resource\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 401)

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[13].Func(mockWriter, getRequest)

	assert.Equal(t, "401", responseHeader.Get(responseStatusHeader),
		"Response status code must be 401 Unauthorized")

	mockWriter.AssertExpectations(t)
}

// Query empty dns rules with invalid application instance id
func TestGetEmptyDnsRulesAppInstanceIdInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	invalidAppInstanceId := "invalid-app-instance-id"

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getDnsRulesUrlFormat, invalidAppInstanceId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, invalidAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, invalidAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte(
		"{\"title\":\"Request parameter error\",\"status\":14,\"detail\":\"app Instance ID validation failed, invalid uuid\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[13].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		"Response status code must be 400 Unauthorized")

	mockWriter.AssertExpectations(t)

}

// Query single dns rule request in mp1 interface
func TestGetSingleDnsRule(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte(fmt.Sprintf(writeObjectFormat, exampleIPAddress, util.InactiveState))).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		entry := dns.RuleEntry{DomainName: exampleDomainName, IpAddressType: "IP_V4", IpAddress: exampleIPAddress,
			TTL: 30, State: util.InactiveState}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patches.Reset()

	// 14 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[14].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)

}

// Query single rule
func TestGetSingleDnsRuleNotFound(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"title\":\"Can not found resource\",\"status\":5,\"detail\":\"dns rule retrieval failed\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 404)

	// 14 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[14].Func(mockWriter, getRequest)

	assert.Equal(t, "404", responseHeader.Get(responseStatusHeader),
		responseCheckFor400)

	mockWriter.AssertExpectations(t)
}

// Query single rule with empty rule id
func TestGetSingleDnsRuleNoId(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, ""),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, "")
	getRequest.Header.Set(appInstanceIdHeader, "")

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"title\":\"UnAuthorization\",\"status\":11,\"detail\":\"UnAuthorization to access the resource\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 401)

	// 14 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[14].Func(mockWriter, getRequest)

	assert.Equal(t, "401", responseHeader.Get(responseStatusHeader),
		responseCheckFor400)

	mockWriter.AssertExpectations(t)
}

// Update a dns rule
func TestPutSingleDnsRule(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	createRule := dnsCreateRule{
		DomainName:    exampleDomainName,
		IpAddressType: util.IPv4Type,
		IpAddress:     exampleIPAddress,
		TTL:           defaultTTL,
		State:         util.InactiveState,
	}
	createRuleBytes, _ := json.Marshal(createRule)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader(createRuleBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte(fmt.Sprintf(writeObjectFormat, exampleIPAddress, util.InactiveState))).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		entry := dns.RuleEntry{DomainName: exampleDomainName, IpAddressType: "IP_V4", IpAddress: exampleIPAddress,
			TTL: 30, State: util.InactiveState}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patches.Reset()

	// 15 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[15].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Update a dns rule from INACTIVE to ACTIVE
func TestPutSingleDnsRuleActive(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	createRule := dnsCreateRule{
		DomainName:    exampleDomainName,
		IpAddressType: util.IPv4Type,
		IpAddress:     exampleIPAddress,
		TTL:           defaultTTL,
		State:         util.ActiveState,
	}
	createRuleBytes, _ := json.Marshal(createRule)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader(createRuleBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte(fmt.Sprintf(writeObjectFormat, exampleIPAddress, util.ActiveState))).
		Return(0, nil)

	mockWriter.On("WriteHeader", 200)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err2 := w.Write([]byte(""))
		if err2 != nil {
			t.Error(errorWriteRespErr)
		}
	}))
	defer ts.Close()
	patch1 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		entry := dns.RuleEntry{DomainName: exampleDomainName, IpAddressType: "IP_V4", IpAddress: exampleIPAddress,
			TTL: 30, State: util.InactiveState}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	patch2 := gomonkey.ApplyFunc(dns.NewRestClient, func() *dns.RestClient {
		parse, _ := url.Parse(ts.URL)
		return &dns.RestClient{ServerEndPoint: parse}
	})

	defer patch1.Reset()
	defer patch2.Reset()

	// 15 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[15].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Update a dns rule from ACTIVE to ACTIVE
func TestPutSingleDnsRuleReActive(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	createRule := dnsCreateRule{
		DomainName:    exampleDomainName,
		IpAddressType: util.IPv4Type,
		IpAddress:     exampleIPAddress,
		TTL:           defaultTTL,
		State:         util.ActiveState,
	}
	createRuleBytes, _ := json.Marshal(createRule)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader(createRuleBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte(fmt.Sprintf(writeObjectFormat, exampleIPAddress, util.ActiveState))).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err2 := w.Write([]byte(""))
		if err2 != nil {
			t.Error(errorWriteRespErr)
		}
	}))
	defer ts.Close()
	patch1 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		entry := dns.RuleEntry{DomainName: exampleDomainName, IpAddressType: "IP_V4", IpAddress: exampleIPAddress,
			TTL: 30, State: util.ActiveState}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	patch2 := gomonkey.ApplyFunc(dns.NewRestClient, func() *dns.RestClient {
		parse, _ := url.Parse(ts.URL)
		return &dns.RestClient{ServerEndPoint: parse}
	})

	defer patch1.Reset()
	defer patch2.Reset()

	// 15 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[15].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Update a dns rule from ACTIVE to INACTIVE
func TestPutSingleDnsRuleInactive(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	createRule := dnsCreateRule{
		DomainName:    exampleDomainName,
		IpAddressType: util.IPv4Type,
		IpAddress:     exampleIPAddress,
		TTL:           defaultTTL,
		State:         util.InactiveState,
	}
	createRuleBytes, _ := json.Marshal(createRule)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader(createRuleBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte(fmt.Sprintf(writeObjectFormat, exampleIPAddress, util.InactiveState))).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err2 := w.Write([]byte(""))
		if err2 != nil {
			t.Error(errorWriteRespErr)
		}
	}))
	defer ts.Close()
	patch1 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		entry := dns.RuleEntry{DomainName: exampleDomainName, IpAddressType: "IP_V4", IpAddress: exampleIPAddress,
			TTL: 30, State: util.ActiveState}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	patch2 := gomonkey.ApplyFunc(dns.NewRestClient, func() *dns.RestClient {
		parse, _ := url.Parse(ts.URL)
		return &dns.RestClient{ServerEndPoint: parse}
	})

	defer patch1.Reset()
	defer patch2.Reset()

	// 15 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[15].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Update a dns rule from INACTIVE to ACTIVE when the server is not reachable
func TestPutSingleDnsRuleActiveWithServerNotReachable(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	createRule := dnsCreateRule{
		DomainName:    exampleDomainName,
		IpAddressType: util.IPv4Type,
		IpAddress:     exampleIPAddress,
		TTL:           defaultTTL,
		State:         util.ActiveState,
	}
	createRuleBytes, _ := json.Marshal(createRule)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader(createRuleBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"title\":\"Remote server error\",\"status\":9,\"detail\":\"failed to apply the dns modification\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 503)

	patch1 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		entry := dns.RuleEntry{DomainName: exampleDomainName, IpAddressType: "IP_V4", IpAddress: exampleIPAddress,
			TTL: 30, State: util.InactiveState}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patch1.Reset()

	// 15 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[15].Func(mockWriter, getRequest)

	assert.Equal(t, "503", responseHeader.Get(responseStatusHeader),
		"Response status code miss-match")

	mockWriter.AssertExpectations(t)
}

// Update a dns rule from INACTIVE to ACTIVE with error in the dns server
func TestPutSingleDnsRuleActiveWithServerError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	createRule := dnsCreateRule{
		DomainName:    exampleDomainName,
		IpAddressType: util.IPv4Type,
		IpAddress:     exampleIPAddress,
		TTL:           defaultTTL,
		State:         util.ActiveState,
	}
	createRuleBytes, _ := json.Marshal(createRule)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader(createRuleBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"title\":\"Remote server error\",\"status\":9,\"detail\":\"failed to apply the dns modification\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 503)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err2 := w.Write([]byte(""))
		if err2 != nil {
			t.Error(errorWriteRespErr)
		}
	}))
	defer ts.Close()
	patch1 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		entry := dns.RuleEntry{DomainName: exampleDomainName, IpAddressType: "IP_V4", IpAddress: exampleIPAddress,
			TTL: 30, State: util.InactiveState}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	patch2 := gomonkey.ApplyFunc(dns.NewRestClient, func() *dns.RestClient {
		parse, _ := url.Parse(ts.URL)
		return &dns.RestClient{ServerEndPoint: parse}
	})

	defer patch1.Reset()
	defer patch2.Reset()

	// 15 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[15].Func(mockWriter, getRequest)

	assert.Equal(t, "503", responseHeader.Get(responseStatusHeader),
		"Response status code miss-match")

	mockWriter.AssertExpectations(t)
}

// Put a dns rule which doesn't exists
func TestPutSingleDnsRuleNotFound(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader([]byte("{}")))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"title\":\"Can not found resource\",\"status\":5,\"detail\":\"dns rule retrieval failed\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 404)

	// 15 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[15].Func(mockWriter, getRequest)

	assert.Equal(t, "404", responseHeader.Get(responseStatusHeader),
		responseCheckFor400)

	mockWriter.AssertExpectations(t)
}

// Put a dns rule with invalid body
func TestPutSingleDnsRuleBodyParseError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"title\":\"Bad Request\",\"status\":1,\"detail\":\"check Param failed\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	// 15 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[15].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor400)

	mockWriter.AssertExpectations(t)
}

// Put a dns rule with large body
func TestPutSingleDnsRuleOverLengthBody(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	messageBody := ""
	for i := 0; i <= 64; i++ {
		messageBody += "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz123456789011"
	}

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader([]byte(messageBody)))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"title\":\"Request parameter error\",\"status\":14,\"detail\":\"request body too large\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	// 15 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[15].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		"Response status code must be 400")

	mockWriter.AssertExpectations(t)
}
