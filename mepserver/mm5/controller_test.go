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

package mm5

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey"
	_ "github.com/apache/servicecomb-service-center/server"
	_ "github.com/apache/servicecomb-service-center/server/bootstrap"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	srv "github.com/apache/servicecomb-service-center/server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"mepserver/common/extif/backend"
	"mepserver/common/extif/dns"
	"mepserver/common/util"
	"mepserver/mm5/models"
	mp1models "mepserver/mp1/models"
)

const defaultAppInstanceId = "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"
const dnsRuleId = "7d71e54e-81f3-47bb-a2fc-b565a326d794"

const panicFormatString = "Panic: %v"
const getDnsRulesUrlFormat = "/mepcfg/mec_app_config/v1/rules/%s/dns_rules"
const getDnsRuleUrlFormat = "/mepcfg/mec_app_config/v1/rules/%s/dns_rules/%s"
const appInstanceQueryFormat = ":appInstanceId=%s&;"
const appIdAndDnsRuleIdQueryFormat = ":appInstanceId=%s&;:dnsRuleId=%s&;"
const appInstanceIdHeader = "X-AppinstanceID"
const responseStatusHeader = "X-Response-Status"
const responseCheckFor200 = "Response status code must be 200"
const responseCheckFor400 = "Response status code must be 404"
const errorDomainMissMatch = "Domain name miss-match in the response"
const errorIPTypeMissMatch = "IP type miss-match in the response"
const errorIPAddrMissMatch = "IP address miss-match in the response"
const errorTTLMissMatch = "TTL miss-match in the response"
const errorStateMissMatch = "State miss-match in the response"
const errorWriteRespErr = "Write Response Error"
const errorRspMissMatch = "Miss-match in the response"
const exampleDomainName = "www.example.com"
const defaultTTL = 30
const TTL35 = 35
const maxIPVal = 255
const ipAddFormatter = "%d.%d.%d.%d"
const writeObjectFormat = "{\"dnsRuleId\":\"7d71e54e-81f3-47bb-a2fc-b565a326d794\",\"domainName\":\"www.example.com\"," +
	"\"ipAddressType\":\"IP_V4\",\"ipAddress\":\"%s\",\"ttl\":30,\"state\":\"%s\"}\n"

const getCapabilitiesUrl = "/mepcfg/mec_platform_config/v1/capabilities"

const defCapabilityId = "16384563dca094183778a41ea7701d15"
const defCapabilityId2 = "f7e898d1c9ea9edd05e1181bc09afc5e"
const subscriberId1 = "05ddef81-dd83-4a37-b0fe-85999585b929"
const subscriberId2 = "09022fec-a63c-49fc-857a-dcd7ecaa40a2"
const appInstanceId2 = "3abe4278-9c70-2e47-3a4e-7ee3a1a0fd1e"

const capabilityQueryFormat = ":capabilityId=%s&;"

// Generate test IP, instead of hard coding them
var exampleIPAddress = fmt.Sprintf(ipAddFormatter, rand.Intn(maxIPVal), rand.Intn(maxIPVal), rand.Intn(maxIPVal),
	rand.Intn(maxIPVal))

type mockHttpWriter struct {
	mock.Mock
	response []byte
}

func (m *mockHttpWriter) Header() http.Header {
	// Get the argument inputs
	args := m.Called()
	// retrieve the configured value we provided at the input and return it back
	return args.Get(0).(http.Header)
}
func (m *mockHttpWriter) Write(response []byte) (int, error) {
	// fmt.Printf("Write: %v", response)
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

type mockHttpWriterWithoutWrite struct {
	mock.Mock
	response []byte
}

func (m *mockHttpWriterWithoutWrite) Header() http.Header {
	// Get the argument inputs
	args := m.Called()
	// retrieve the configured value we provided at the input and return it back
	return args.Get(0).(http.Header)
}
func (m *mockHttpWriterWithoutWrite) Write(response []byte) (int, error) {
	// fmt.Printf("Write: %v", response)
	// Get the argument inputs and marking the function is called with correct input
	args := m.Called()

	if response != nil {
		m.response = bytes.Join([][]byte{m.response, response}, []byte(""))
	}
	// retrieve the configured value we provided at the input and return it back
	// return args.Get(0).(http.Header)
	return args.Int(0), args.Error(1)
}
func (m *mockHttpWriterWithoutWrite) WriteHeader(statusCode int) {
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

// Query an empty dns rules request in mm5 interface
func TestGetEmptyDnsRules(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

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

	// 1 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[1].Func(mockWriter, getRequest)

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

	service := Mm5Service{}

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

	// 1 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[1].Func(mockWriter, getRequest)

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

	service := Mm5Service{}

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

	// 1 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[1].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		"Response status code must be 400 Unauthorized")

	mockWriter.AssertExpectations(t)

}

// Query single rule
func TestGetSingleDnsRuleNotFound(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

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

	// 2 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[2].Func(mockWriter, getRequest)

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

	service := Mm5Service{}

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

	// 2 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[2].Func(mockWriter, getRequest)

	assert.Equal(t, "401", responseHeader.Get(responseStatusHeader),
		responseCheckFor400)

	mockWriter.AssertExpectations(t)
}

// Update a dns rule from INACTIVE to ACTIVE
func TestPutSingleDnsRuleActive(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

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
	service.URLPatterns()[3].Func(mockWriter, getRequest)

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

	service := Mm5Service{}

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
	service.URLPatterns()[3].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Put a dns rule which doesn't exists
func TestPutSingleDnsRuleNotFound(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

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

	// 3 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[3].Func(mockWriter, getRequest)

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

	service := Mm5Service{}

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

	// 3 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[3].Func(mockWriter, getRequest)

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

	service := Mm5Service{}

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

	// 3 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[3].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		"Response status code must be 400")

	mockWriter.AssertExpectations(t)
}

// Create a dns rule
func TestPostDnsRule(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	createRule := dnsCreateRule{
		DomainName:    exampleDomainName,
		IpAddressType: util.IPv4Type,
		IpAddress:     exampleIPAddress,
		TTL:           defaultTTL,
		State:         util.InactiveState,
	}
	createRuleBytes, _ := json.Marshal(createRule)

	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(getDnsRulesUrlFormat, defaultAppInstanceId),
		bytes.NewReader(createRuleBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	// 3 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[0].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)
	rule := models.DnsConfigRule{}
	_ = json.Unmarshal(mockWriter.response, &rule)
	assert.Equal(t, exampleDomainName, rule.DomainName, errorDomainMissMatch)
	assert.Equal(t, util.IPv4Type, rule.IpAddressType, errorIPTypeMissMatch)
	assert.Equal(t, exampleIPAddress, rule.IpAddress, errorIPAddrMissMatch)
	assert.Equal(t, defaultTTL, rule.TTL, errorTTLMissMatch)
	assert.Equal(t, util.InactiveState, rule.State, errorStateMissMatch)

	mockWriter.AssertExpectations(t)
}

// Create a dns rule with ACTIVE rule
func TestPostDnsRuleWithActiveRule(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}
	createRule := dnsCreateRule{
		DomainName:    exampleDomainName,
		IpAddressType: util.IPv4Type,
		IpAddress:     exampleIPAddress,
		TTL:           TTL35,
		State:         util.ActiveState,
	}
	createRuleBytes, _ := json.Marshal(createRule)

	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(getDnsRulesUrlFormat, defaultAppInstanceId),
		bytes.NewReader([]byte(createRuleBytes)))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inputData, err := ioutil.ReadAll(r.Body)
		assert.Equal(t, nil, err, "Read http response failed")
		defer r.Body.Close()
		fmt.Printf("Input Data: %s \n", inputData)
		assert.Equal(t, string(inputData), fmt.Sprintf("[{\"zone\":\".\",\"rr\":[{\"name\":\"www.example.com.\","+
			"\"type\":\"A\",\"class\":\"IN\",\"ttl\":35,\"rData\":[\"%s\"]}]}]", exampleIPAddress),
			"DNS request issues")

		w.WriteHeader(http.StatusOK)
		_, err2 := w.Write([]byte(""))
		if err2 != nil {
			t.Error(errorWriteRespErr)
		}
	}))
	defer ts.Close()

	patches := gomonkey.ApplyFunc(dns.NewRestClient, func() *dns.RestClient {
		parse, _ := url.Parse(ts.URL)
		return &dns.RestClient{ServerEndPoint: parse}
	})
	defer patches.Reset()

	// 3 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[0].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)
	rule := models.DnsConfigRule{}
	_ = json.Unmarshal(mockWriter.response, &rule)
	assert.Equal(t, exampleDomainName, rule.DomainName, errorDomainMissMatch)
	assert.Equal(t, util.IPv4Type, rule.IpAddressType, errorIPTypeMissMatch)
	assert.Equal(t, exampleIPAddress, rule.IpAddress, errorIPAddrMissMatch)
	assert.Equal(t, 35, rule.TTL, errorTTLMissMatch)
	assert.Equal(t, util.ActiveState, rule.State, errorStateMissMatch)

	mockWriter.AssertExpectations(t)
}

// Create a dns rule with ACTIVE rule with unreachable server
func TestPostDnsRuleWithActiveRuleUnReachableServer(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	createRule := dnsCreateRule{
		DomainName:    exampleDomainName,
		IpAddressType: util.IPv4Type,
		IpAddress:     exampleIPAddress,
		TTL:           defaultTTL,
		State:         util.ActiveState,
	}
	createRuleBytes, _ := json.Marshal(createRule)

	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(getDnsRulesUrlFormat, defaultAppInstanceId),
		bytes.NewReader(createRuleBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 503)

	// 3 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[0].Func(mockWriter, getRequest)

	assert.Equal(t, "503", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)
	respError := models.ProblemDetails{}

	_ = json.Unmarshal(mockWriter.response, &respError)
	assert.Equal(t, "Remote server error", respError.Title, errorRspMissMatch)
	assert.Equal(t, uint32(9), respError.Status, errorRspMissMatch)
	assert.Equal(t, "failed to apply changes on remote server", respError.Detail, errorRspMissMatch)

	mockWriter.AssertExpectations(t)
}

// Create a dns rule with ACTIVE rule and server return StatusBadRequest failure
func TestPostDnsRuleWithActiveRuleAndStatusBadRequestInServer(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	createRule := dnsCreateRule{
		DomainName:    exampleDomainName,
		IpAddressType: util.IPv4Type,
		IpAddress:     exampleIPAddress,
		TTL:           defaultTTL,
		State:         util.ActiveState,
	}
	createRuleBytes, _ := json.Marshal(createRule)

	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(getDnsRulesUrlFormat, defaultAppInstanceId),
		bytes.NewReader(createRuleBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 503)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err2 := w.Write([]byte(""))
		if err2 != nil {
			t.Error(errorWriteRespErr)
		}
	}))
	defer ts.Close()

	patches := gomonkey.ApplyFunc(dns.NewRestClient, func() *dns.RestClient {
		parse, _ := url.Parse(ts.URL)
		return &dns.RestClient{ServerEndPoint: parse}
	})
	defer patches.Reset()

	// 3 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[0].Func(mockWriter, getRequest)

	assert.Equal(t, "503", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)
	respError := models.ProblemDetails{}

	_ = json.Unmarshal(mockWriter.response, &respError)
	assert.Equal(t, "Remote server error", respError.Title, errorRspMissMatch)
	assert.Equal(t, uint32(9), respError.Status, errorRspMissMatch)
	assert.Equal(t, "failed to apply changes on remote server", respError.Detail, errorRspMissMatch)

	mockWriter.AssertExpectations(t)
}

// Delete a dns rule
func TestDeleteDnsRule(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 204)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		entry := dns.RuleEntry{DomainName: exampleDomainName, IpAddressType: util.IPv4Type, IpAddress: exampleIPAddress,
			TTL: defaultTTL, State: util.InactiveState}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patches.Reset()

	// 3 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[4].Func(mockWriter, getRequest)

	assert.Equal(t, "204", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)
	rule := models.DnsConfigRule{}
	_ = json.Unmarshal(mockWriter.response, &rule)
	assert.Equal(t, exampleDomainName, rule.DomainName, errorDomainMissMatch)
	assert.Equal(t, util.IPv4Type, rule.IpAddressType, errorIPTypeMissMatch)
	assert.Equal(t, exampleIPAddress, rule.IpAddress, errorIPAddrMissMatch)
	assert.Equal(t, defaultTTL, rule.TTL, errorTTLMissMatch)
	assert.Equal(t, util.InactiveState, rule.State, errorStateMissMatch)

	mockWriter.AssertExpectations(t)
}

// Delete a dns rule which does not exists
func TestDeleteDnsRuleNotFound(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 404)

	// 3 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[4].Func(mockWriter, getRequest)

	assert.Equal(t, "404", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)
	respError := models.ProblemDetails{}

	_ = json.Unmarshal(mockWriter.response, &respError)
	assert.Equal(t, "Can not found resource", respError.Title, errorRspMissMatch)
	assert.Equal(t, uint32(5), respError.Status, errorRspMissMatch)
	assert.Equal(t, "dns rule retrieval failed", respError.Detail, errorRspMissMatch)

	mockWriter.AssertExpectations(t)
}

// Delete a dns rule
func TestDeleteActiveDnsRule(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 204)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		entry := dns.RuleEntry{DomainName: exampleDomainName, IpAddressType: util.IPv4Type, IpAddress: exampleIPAddress,
			TTL: defaultTTL, State: util.InactiveState}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patches.Reset()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err2 := w.Write([]byte(""))
		if err2 != nil {
			t.Error(errorWriteRespErr)
		}
	}))
	defer ts.Close()

	// 3 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[4].Func(mockWriter, getRequest)

	assert.Equal(t, "204", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)
	rule := models.DnsConfigRule{}
	_ = json.Unmarshal(mockWriter.response, &rule)
	assert.Equal(t, exampleDomainName, rule.DomainName, errorDomainMissMatch)
	assert.Equal(t, util.IPv4Type, rule.IpAddressType, errorIPTypeMissMatch)
	assert.Equal(t, exampleIPAddress, rule.IpAddress, errorIPAddrMissMatch)
	assert.Equal(t, defaultTTL, rule.TTL, errorTTLMissMatch)
	assert.Equal(t, util.InactiveState, rule.State, errorStateMissMatch)

	mockWriter.AssertExpectations(t)
}

// Query capability
func TestGetCapabilitiesEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET", getCapabilitiesUrl, bytes.NewReader([]byte("")))
	// getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("[]\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	// 2 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[5].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query capability
func TestGetCapabilitiesSuccessCase(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET", getCapabilitiesUrl, bytes.NewReader([]byte("")))
	// getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("[{\"capabilityId\":\"16384563dca094183778a41ea7701d15\",\"capabilityName\":\"FaceRegService6\",\"status\":\"ACTIVE\",\"version\":\"3.2.1\",\"consumers\":[]}]\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patch1 := gomonkey.ApplyFunc(util.FindInstanceByKey, func(result url.Values) (*proto.FindInstancesResponse, error) {
		response := proto.FindInstancesResponse{}
		response.Instances = make([]*proto.MicroServiceInstance, 0)
		response.Instances = append(response.Instances, &proto.MicroServiceInstance{
			InstanceId: defCapabilityId[len(defCapabilityId)/2:],
			ServiceId:  defCapabilityId[:len(defCapabilityId)/2],
			Status:     "UP",
			Version:    "3.2.1",
			Properties: map[string]string{
				"serName": "FaceRegService6",
			},
		})
		return &response, nil
	})
	defer patch1.Reset()

	// 2 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[5].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query capability
func TestGetCapabilitiesWithConsumers(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET", getCapabilitiesUrl, bytes.NewReader([]byte("")))
	// getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("[{\"capabilityId\":\"16384563dca094183778a41ea7701d15\",\"capabilityName\":\"FaceRegService6\",\"status\":\"ACTIVE\",\"version\":\"3.2.1\",\"consumers\":[{\"applicationInstanceId\":\"5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f\"}]}]\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patch1 := gomonkey.ApplyFunc(util.FindInstanceByKey, func(result url.Values) (*proto.FindInstancesResponse, error) {
		response := proto.FindInstancesResponse{}
		response.Instances = make([]*proto.MicroServiceInstance, 0)
		response.Instances = append(response.Instances, &proto.MicroServiceInstance{
			InstanceId: defCapabilityId[len(defCapabilityId)/2:],
			ServiceId:  defCapabilityId[:len(defCapabilityId)/2],
			Status:     "UP",
			Version:    "3.2.1",
			Properties: map[string]string{
				"serName": "FaceRegService6",
			},
		})
		return &response, nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyFunc(backend.GetRecordsWithCompleteKeyPath, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := mp1models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "05ddef81-dd83-4a37-b0fe-85999585b929",
			FilteringCriteria: mp1models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerInstanceIds = append(entry.FilteringCriteria.SerInstanceIds, defCapabilityId)
		outBytes, _ := json.Marshal(&entry)
		records[fmt.Sprintf("csr/etcd/%s/%s", defaultAppInstanceId, subscriberId1)] = outBytes
		return records, 0
	})
	defer patch2.Reset()

	// 2 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[5].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query capability
func TestGetCapabilitiesWithMultiConsumers(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET", getCapabilitiesUrl, bytes.NewReader([]byte("")))
	// getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("[{\"capabilityId\":\"16384563dca094183778a41ea7701d15\",\"capabilityName\":\"FaceRegService6\",\"status\":\"ACTIVE\",\"version\":\"3.2.1\",\"consumers\":[{\"applicationInstanceId\":\"5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f\"}]}]\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patch1 := gomonkey.ApplyFunc(util.FindInstanceByKey, func(result url.Values) (*proto.FindInstancesResponse, error) {
		response := proto.FindInstancesResponse{}
		response.Instances = make([]*proto.MicroServiceInstance, 0)
		response.Instances = append(response.Instances, &proto.MicroServiceInstance{
			InstanceId: defCapabilityId[len(defCapabilityId)/2:],
			ServiceId:  defCapabilityId[:len(defCapabilityId)/2],
			Status:     "UP",
			Version:    "3.2.1",
			Properties: map[string]string{
				"serName": "FaceRegService6",
			},
		})
		return &response, nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyFunc(backend.GetRecordsWithCompleteKeyPath, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := mp1models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "05ddef81-dd83-4a37-b0fe-85999585b929",
			FilteringCriteria: mp1models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerInstanceIds = append(entry.FilteringCriteria.SerInstanceIds, defCapabilityId)
		outBytes, _ := json.Marshal(&entry)
		records[fmt.Sprintf("csr/etcd/%s/%s", defaultAppInstanceId, subscriberId1)] = outBytes

		return records, 0
	})
	defer patch2.Reset()

	// 2 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[5].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query capability
func TestGetCapabilitiesWithMultiCapabilityMultiConsumers(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET", getCapabilitiesUrl, bytes.NewReader([]byte("")))
	// getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("[{\"capabilityId\":\"16384563dca094183778a41ea7701d15\",\"capabilityName\":\"FaceRegService6\",\"status\":\"ACTIVE\",\"version\":\"3.2.1\",\"consumers\":[{\"applicationInstanceId\":\"5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f\"}]},{\"capabilityId\":\"f7e898d1c9ea9edd05e1181bc09afc5e\",\"capabilityName\":\"FaceRegService5\",\"status\":\"ACTIVE\",\"version\":\"3.2.1\",\"consumers\":[{\"applicationInstanceId\":\"3abe4278-9c70-2e47-3a4e-7ee3a1a0fd1e\"}]}]\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patch1 := gomonkey.ApplyFunc(util.FindInstanceByKey, func(result url.Values) (*proto.FindInstancesResponse, error) {
		response := proto.FindInstancesResponse{}
		response.Instances = make([]*proto.MicroServiceInstance, 0)
		response.Instances = append(response.Instances, &proto.MicroServiceInstance{
			InstanceId: defCapabilityId[len(defCapabilityId)/2:],
			ServiceId:  defCapabilityId[:len(defCapabilityId)/2],
			Status:     "UP",
			Version:    "3.2.1",
			Properties: map[string]string{
				"serName": "FaceRegService6",
			},
		})
		response.Instances = append(response.Instances, &proto.MicroServiceInstance{
			InstanceId: defCapabilityId2[len(defCapabilityId2)/2:],
			ServiceId:  defCapabilityId2[:len(defCapabilityId2)/2],
			Status:     "UP",
			Version:    "3.2.1",
			Properties: map[string]string{
				"serName": "FaceRegService5",
			},
		})
		return &response, nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyFunc(backend.GetRecordsWithCompleteKeyPath, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := mp1models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "05ddef81-dd83-4a37-b0fe-85999585b929",
			FilteringCriteria: mp1models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerInstanceIds = append(entry.FilteringCriteria.SerInstanceIds, defCapabilityId)
		outBytes, _ := json.Marshal(&entry)
		records[fmt.Sprintf("csr/etcd/%s/%s", defaultAppInstanceId, subscriberId1)] = outBytes

		entry2 := mp1models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "09022fec-a63c-49fc-857a-dcd7ecaa40a2",
			FilteringCriteria: mp1models.FilteringCriteria{},
		}
		entry2.FilteringCriteria.SerInstanceIds = append(entry2.FilteringCriteria.SerInstanceIds, defCapabilityId2)
		outBytes2, _ := json.Marshal(&entry2)
		records[fmt.Sprintf("csr/etcd/%s/%s", appInstanceId2, subscriberId2)] = outBytes2

		return records, 0
	})
	defer patch2.Reset()

	// 2 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[5].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

func TestGetCapabilitiesWithMultiConsumersAndServiceNameFilter(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET", getCapabilitiesUrl, bytes.NewReader([]byte("")))
	// getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("[{\"capabilityId\":\"16384563dca094183778a41ea7701d15\",\"capabilityName\":\"FaceRegService6\",\"status\":\"ACTIVE\",\"version\":\"3.2.1\",\"consumers\":[{\"applicationInstanceId\":\"5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f\"}]}]\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patch1 := gomonkey.ApplyFunc(util.FindInstanceByKey, func(result url.Values) (*proto.FindInstancesResponse, error) {
		response := proto.FindInstancesResponse{}
		response.Instances = make([]*proto.MicroServiceInstance, 0)
		response.Instances = append(response.Instances, &proto.MicroServiceInstance{
			InstanceId: defCapabilityId[len(defCapabilityId)/2:],
			ServiceId:  defCapabilityId[:len(defCapabilityId)/2],
			Status:     "UP",
			Version:    "3.2.1",
			Properties: map[string]string{
				"serName": "FaceRegService6",
			},
		})
		return &response, nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyFunc(backend.GetRecordsWithCompleteKeyPath, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := mp1models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "05ddef81-dd83-4a37-b0fe-85999585b929",
			FilteringCriteria: mp1models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerNames = append(entry.FilteringCriteria.SerNames, "FaceRegService6")
		outBytes, _ := json.Marshal(&entry)
		records[fmt.Sprintf("csr/etcd/%s/%s", defaultAppInstanceId, subscriberId1)] = outBytes

		return records, 0
	})
	defer patch2.Reset()

	// 2 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[5].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

func TestGetCapabilitiesWithMultiConsumersAndCategoryFilter(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET", getCapabilitiesUrl, bytes.NewReader([]byte("")))
	// getRequest.URL.RawQuery = fmt.Sprintf(appIdAndDnsRuleIdQueryFormat, defaultAppInstanceId, dnsRuleId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("[{\"capabilityId\":\"16384563dca094183778a41ea7701d15\",\"capabilityName\":\"FaceRegService6\",\"status\":\"ACTIVE\",\"version\":\"3.2.1\",\"consumers\":[{\"applicationInstanceId\":\"5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f\"}]}]\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patch1 := gomonkey.ApplyFunc(util.FindInstanceByKey, func(result url.Values) (*proto.FindInstancesResponse, error) {
		response := proto.FindInstancesResponse{}
		response.Instances = make([]*proto.MicroServiceInstance, 0)
		response.Instances = append(response.Instances, &proto.MicroServiceInstance{
			InstanceId: defCapabilityId[len(defCapabilityId)/2:],
			ServiceId:  defCapabilityId[:len(defCapabilityId)/2],
			Status:     "UP",
			Version:    "3.2.1",
			Properties: map[string]string{
				"serName":             "FaceRegService6",
				"serCategory/href":    "/example/catalogue1",
				"serCategory/id":      "id12345",
				"serCategory/name":    "RNI",
				"serCategory/version": "v1.1",
			},
		})
		return &response, nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyFunc(backend.GetRecordsWithCompleteKeyPath, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := mp1models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "05ddef81-dd83-4a37-b0fe-85999585b929",
			FilteringCriteria: mp1models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerCategories = append(entry.FilteringCriteria.SerCategories, mp1models.CategoryRef{
			Href:    "/example/catalogue1",
			ID:      "id12345",
			Name:    "RNI",
			Version: "v1.1",
		})
		outBytes, _ := json.Marshal(&entry)
		records[fmt.Sprintf("csr/etcd/%s/%s", defaultAppInstanceId, subscriberId1)] = outBytes

		return records, 0
	})
	defer patch2.Reset()

	// 2 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[5].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query capability
func TestGetCapabilitySuccessCase(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET", getCapabilitiesUrl, bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(capabilityQueryFormat, defCapabilityId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"capabilityId\":\"16384563dca094183778a41ea7701d15\",\"capabilityName\":\"FaceRegService6\",\"status\":\"ACTIVE\",\"version\":\"3.2.1\",\"consumers\":[]}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	n := &srv.InstanceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *proto.GetOneInstanceRequest) (*proto.GetOneInstanceResponse, error) {
		response := proto.GetOneInstanceResponse{}

		response.Instance = &proto.MicroServiceInstance{
			InstanceId: defCapabilityId[len(defCapabilityId)/2:],
			ServiceId:  defCapabilityId[:len(defCapabilityId)/2],
			Status:     "UP",
			Version:    "3.2.1",
			Properties: map[string]string{
				"serName": "FaceRegService6",
			},
		}
		return &response, nil
	})
	defer patch1.Reset()

	// 2 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[6].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query capability
func TestGetCapabilitySuccessCaseWithConsumers(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET", getCapabilitiesUrl, bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(capabilityQueryFormat, defCapabilityId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"capabilityId\":\"16384563dca094183778a41ea7701d15\",\"capabilityName\":\"FaceRegService6\",\"status\":\"ACTIVE\",\"version\":\"3.2.1\",\"consumers\":[{\"applicationInstanceId\":\"5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f\"}]}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	n := &srv.InstanceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *proto.GetOneInstanceRequest) (*proto.GetOneInstanceResponse, error) {
		response := proto.GetOneInstanceResponse{}

		response.Instance = &proto.MicroServiceInstance{
			InstanceId: defCapabilityId[len(defCapabilityId)/2:],
			ServiceId:  defCapabilityId[:len(defCapabilityId)/2],
			Status:     "UP",
			Version:    "3.2.1",
			Properties: map[string]string{
				"serName": "FaceRegService6",
			},
		}
		return &response, nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyFunc(backend.GetRecordsWithCompleteKeyPath, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := mp1models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "05ddef81-dd83-4a37-b0fe-85999585b929",
			FilteringCriteria: mp1models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerInstanceIds = append(entry.FilteringCriteria.SerInstanceIds, defCapabilityId)
		outBytes, _ := json.Marshal(&entry)
		records[fmt.Sprintf("csr/etcd/%s/%s", defaultAppInstanceId, subscriberId1)] = outBytes

		return records, 0
	})
	defer patch2.Reset()

	// 2 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[6].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query capability
func TestGetCapabilitySuccessCaseWithConsumersAndSerNameFilter(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET", getCapabilitiesUrl, bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(capabilityQueryFormat, defCapabilityId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"capabilityId\":\"16384563dca094183778a41ea7701d15\",\"capabilityName\":\"FaceRegService6\",\"status\":\"ACTIVE\",\"version\":\"3.2.1\",\"consumers\":[{\"applicationInstanceId\":\"5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f\"}]}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	n := &srv.InstanceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *proto.GetOneInstanceRequest) (*proto.GetOneInstanceResponse, error) {
		response := proto.GetOneInstanceResponse{}

		response.Instance = &proto.MicroServiceInstance{
			InstanceId: defCapabilityId[len(defCapabilityId)/2:],
			ServiceId:  defCapabilityId[:len(defCapabilityId)/2],
			Status:     "UP",
			Version:    "3.2.1",
			Properties: map[string]string{
				"serName": "FaceRegService6",
			},
		}
		return &response, nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyFunc(backend.GetRecordsWithCompleteKeyPath, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := mp1models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "05ddef81-dd83-4a37-b0fe-85999585b929",
			FilteringCriteria: mp1models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerNames = append(entry.FilteringCriteria.SerNames, "FaceRegService6")
		outBytes, _ := json.Marshal(&entry)
		records[fmt.Sprintf("csr/etcd/%s/%s", defaultAppInstanceId, subscriberId1)] = outBytes

		return records, 0
	})
	defer patch2.Reset()

	patch3 := gomonkey.ApplyFunc(util.FindInstanceByKey, func(result url.Values) (*proto.FindInstancesResponse, error) {
		response := proto.FindInstancesResponse{}
		response.Instances = make([]*proto.MicroServiceInstance, 0)
		response.Instances = append(response.Instances, &proto.MicroServiceInstance{
			InstanceId: defCapabilityId[len(defCapabilityId)/2:],
			ServiceId:  defCapabilityId[:len(defCapabilityId)/2],
			Status:     "UP",
			Version:    "3.2.1",
			Properties: map[string]string{
				"serName":             "FaceRegService6",
				"serCategory/href":    "/example/catalogue1",
				"serCategory/id":      "id12345",
				"serCategory/name":    "RNI",
				"serCategory/version": "v1.1",
			},
		})
		return &response, nil
	})
	defer patch3.Reset()

	// 2 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[6].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query capability
func TestGetCapabilitySuccessCaseWithConsumersAndCategoryFilter(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET", getCapabilitiesUrl, bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(capabilityQueryFormat, defCapabilityId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"capabilityId\":\"16384563dca094183778a41ea7701d15\",\"capabilityName\":\"FaceRegService6\",\"status\":\"ACTIVE\",\"version\":\"3.2.1\",\"consumers\":[{\"applicationInstanceId\":\"5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f\"}]}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	n := &srv.InstanceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *proto.GetOneInstanceRequest) (*proto.GetOneInstanceResponse, error) {
		response := proto.GetOneInstanceResponse{}

		response.Instance = &proto.MicroServiceInstance{
			InstanceId: defCapabilityId[len(defCapabilityId)/2:],
			ServiceId:  defCapabilityId[:len(defCapabilityId)/2],
			Status:     "UP",
			Version:    "3.2.1",
			Properties: map[string]string{
				"serName":             "FaceRegService6",
				"serCategory/href":    "/example/catalogue1",
				"serCategory/id":      "id12345",
				"serCategory/name":    "RNI",
				"serCategory/version": "v1.1",
			},
		}
		return &response, nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyFunc(backend.GetRecordsWithCompleteKeyPath, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := mp1models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "05ddef81-dd83-4a37-b0fe-85999585b929",
			FilteringCriteria: mp1models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerCategories = append(entry.FilteringCriteria.SerCategories, mp1models.CategoryRef{
			Href:    "/example/catalogue1",
			ID:      "id12345",
			Name:    "RNI",
			Version: "v1.1",
		})
		outBytes, _ := json.Marshal(&entry)
		records[fmt.Sprintf("csr/etcd/%s/%s", defaultAppInstanceId, subscriberId1)] = outBytes

		return records, 0
	})
	defer patch2.Reset()

	patch3 := gomonkey.ApplyFunc(util.FindInstanceByKey, func(result url.Values) (*proto.FindInstancesResponse, error) {
		response := proto.FindInstancesResponse{}
		response.Instances = make([]*proto.MicroServiceInstance, 0)
		response.Instances = append(response.Instances, &proto.MicroServiceInstance{
			InstanceId: defCapabilityId[len(defCapabilityId)/2:],
			ServiceId:  defCapabilityId[:len(defCapabilityId)/2],
			Status:     "UP",
			Version:    "3.2.1",
			Properties: map[string]string{
				"serName":             "FaceRegService6",
				"serCategory/href":    "/example/catalogue1",
				"serCategory/id":      "id12345",
				"serCategory/name":    "RNI",
				"serCategory/version": "v1.1",
			},
		})
		return &response, nil
	})
	defer patch3.Reset()

	// 2 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[6].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}
