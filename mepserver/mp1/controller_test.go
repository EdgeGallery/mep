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

package mp1

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/beevik/ntp"
	"github.com/ghodss/yaml"
	"io"
	"io/ioutil"
	"math/rand"
	"mepserver/common/config"
	"mepserver/common/extif/dataplane"
	"mepserver/common/extif/dataplane/none"
	"mepserver/common/models"
	"mepserver/mp1/plans"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey"
	"github.com/apache/servicecomb-service-center/pkg/log"
	baseutil "github.com/apache/servicecomb-service-center/pkg/util"
	_ "github.com/apache/servicecomb-service-center/server"
	_ "github.com/apache/servicecomb-service-center/server/bootstrap"
	pb "github.com/apache/servicecomb-service-center/server/core/proto"
	scerr "github.com/apache/servicecomb-service-center/server/error"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/registry"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/registry/buildin"
	srv "github.com/apache/servicecomb-service-center/server/service"
	svcutil "github.com/apache/servicecomb-service-center/server/service/util"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"reflect"

	"mepserver/common/extif/backend"
	"mepserver/common/extif/dns"
	ntpc "mepserver/common/extif/ntp"
	"mepserver/common/util"
)

type mockHttpWriter struct {
	mock.Mock
	response []byte
}

//============================= dns ============================================
const defaultAppInstanceId = "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"
const dnsRuleId = "7d71e54e-81f3-47bb-a2fc-b565a326d794"
const trafficRuleId = "8ft68t22-81f3-47bb-a2fc-56996er4tf37"

const panicFormatString = "Panic: %v"
const getDnsRulesUrlFormat = "/mep/mec_app_support/v1/applications/%s/dns_rules"
const getDnsRuleUrlFormat = "/mep/mec_app_support/v1/applications/%s/dns_rules/%s"
const appInstanceQueryFormat = ":appInstanceId=%s&;"
const appIdAndDnsRuleIdQueryFormat = ":appInstanceId=%s&;:dnsRuleId=%s&;"
const appIdAndTrafficRuleIdQueryFormat = ":appInstanceId=%s&;:trafficRuleId=%s&;"
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
const writeTrafficObjectFormat = "{\"trafficRuleId\":\"" + trafficRuleId + "\",\"filterType\":\"FLOW\",\"priority\":5," +
	"\"trafficFilter\":null,\"action\":\"DROP\",\"dstInterface\":null,\"state\":\"%s\"}\n"
const writeTrafficPutObjectFormat = "{\"trafficRuleId\":\"" + trafficRuleId + "\",\"filterType\":\"FLOW\"," +
	"\"priority\":5," +
	"\"trafficFilter\":[],\"action\":\"DROP\",\"dstInterface\":null,\"state\":\"%s\"}\n"
const writeCurTimetFormat = "{\"seconds\":1623770544,\"nanoSeconds\":468538768," +
	"\"timeSourceStatus\":\"TRACEABLE\"}"
const writeCurTimeNonSynct = "{\"seconds\":1623770544,\"nanoSeconds\":468538768," +
	"\"timeSourceStatus\":\"NONTRACEABLE\"}"
const writeCaps = "{\"timeStamp\":{\"seconds\":1623770544,\"nanoSeconds\":468538768},\"ntpServers\"" +
	":[{\"ntpServerAddrType\":\"DNS_NAME\",\"ntpServerAddr\":\"mep-ntp\",\"minPollingInterval\":4,\"maxPollingInterval\":17," +
	"\"localPriority\":1,\"authenticationOption\":\"NONE\",\"authenticationKeyNum\":0}]}"
const writeTransport = "{\"id\":\"8eb442b7cdfc11eba09314feb5b475da\",\"name\":\"REST\",\"description\":\"REST API\"," +
	"\"type\":\"REST_HTTP\",\"protocol\":\"HTTP\",\"version\":\"2.0\",\"endpoint\":{\"uris\":null,\"addresses\":null," +
	"\"alternative\":null},\"security\":{\"oAuth2Info\":{\"grantTypes\":[\"OAUTH2_CLIENT_CREDENTIALS\"]," +
	"\"tokenEndpoint\":\"/mep/token\"}}}"
const writeCurTimeFormatVal = "{\"seconds\":%s,\"nanoSeconds\":%s," +
	"\"timeSourceStatus\":\"TRACEABLE\"}"
const defCapabilityId = "16384563dca094183778a41ea7701d15"
const svcCatHref = "serCategory/href"
const svcCatResponse = "/example/catalogue1"
const svcCatName = "serCategory/name"
const svcCatId = "serCategory/id"
const svcCatVersion = "serCategory/version"

//===========================Services==============================================
const postSubscribeUrl = "/mec_service_mgmt/v1/applications/%s/services"
const getSubscribeUrl = "/mec_service_mgmt/v1/applications/%s/services"
const getOrDelOneSubscribeOrSveUrl = "/mec_service_mgmt/v1/applications/%s/services/%s"
const responseCheckFor201 = "Response status code must be 201"
const responseCheckFor204 = "Response status code must be 204"
const subtype1 = "SerAvailabilityNotificationSubscription"
const subtype2 = "AppTerminationNotificationSubscription"
const errorSubtypeMissMatch = "Subscription type mismatch"
const postAppTerminologiesUrl = "/mec_app_support/v1/applications/%s/services"
const getAppTerminologiesUrl = "/mec_app_support/v1/applications/%s/services"
const getOneAppTerminologiesUrl = "/mec_app_support/v1/applications/%s/services/%s"
const delOneAppTerminologiesUrl = "/mec_app_support/v1/applications/%s/services/%s"
const appIdAndServiceIdQueryFormat = ":appInstanceId=%s&;:serviceId=%s&;"
const sampleServiceId = "f7e898d1c9ea9edd7496c761ddc92718"
const sampleInstanceId = "f7e898d1c9ea9edd7496c761ddc92718"
const serviceDiscoverUrlFormat = "/mep/mec_service_mgmt/v1/applications/%s/services"
const serNameQueryFormat = ":appInstanceId=%s&;ser_name=%s&;"
const getAllTrafficRuleUrl = "/mec_app_support/v1/applications/%s/traffic_rules"
const getOneTrafficRuleUrl = "/mec_app_support/v1/applications/%s/traffic_rules/%s"
const heartBeatUrl = "/mep/mec_service_mgmt/v1/applications/%s/services/%s/liveness"
const formatIntBase = 10
const secString = "timestamp/seconds"
const nanosecString = "timestamp/nanoseconds"
const getTiming = "/mec_app_support/v1/applications/timing/%s"
const getCurrentTIme = "currentTime"
const getCaps = "timing_caps"
const getTransport = "transports"
const confirm_ready = "/mec_app_support/v1/applications/%s/confirm_ready"
const confirm_termination = "/mec_app_support/v1/applications/%s/confirm_termination"

//=====================================COMMON====================================================================
const restApi = "REST API"
const tokenEndPoint = "/mecSerMgmtApi/security/TokenEndPoint"
const href = "/example/catalogue1"
const callBack = "https://%d.%d.%d.%d:%d/example/catalogue1"
const parseFail = "Parsing configuration file error"

//=======================================END======================================================================

// Generate test IP, instead of hard coding them
var exampleIPAddress = fmt.Sprintf(ipAddFormatter, rand.Intn(maxIPVal), rand.Intn(maxIPVal), rand.Intn(maxIPVal),
	rand.Intn(maxIPVal))
var callBackRef = fmt.Sprintf(callBack, 192, 0, 2, 1, 8080)

func (m *mockHttpWriter) Header() http.Header {
	// Get the argument inputs
	args := m.Called()
	// retrieve the configured value we provided at the input and return it back
	return args.Get(0).(http.Header)
}
func (m *mockHttpWriter) Write(response []byte) (int, error) {
	fmt.Printf("Write: %v", string(response))
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
}

type dnsCreateRule struct {
	DomainName    string `json:"domainName"`
	IpAddressType string `json:"ipAddressType"`
	IpAddress     string `json:"ipAddress"`
	TTL           int    `json:"ttl"`
	State         string `json:"state"`
}

//Query traffic rule gets in mp1 interface
func TestGetsTrafficRules(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getAllTrafficRuleUrl, defaultAppInstanceId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte(fmt.Sprintf("["+writeTrafficObjectFormat[:len(writeTrafficObjectFormat)-1]+"]\n",
		util.InactiveState))).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		trafficRule := dataplane.TrafficRule{TrafficRuleID: trafficRuleId, FilterType: "FLOW", Priority: 5,
			Action: "DROP", State: util.InactiveState}
		var trafficRules []dataplane.TrafficRule
		trafficRules = append(trafficRules, trafficRule)
		entry := models.AppDConfig{AppTrafficRule: trafficRules}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patches.Reset()

	// 21 is the order of the traffic get all handler in the URLPattern
	service.URLPatterns()[21].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)

}

//Query traffic rule gets in mp1 interface
func TestGetTrafficRules(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getOneTrafficRuleUrl, defaultAppInstanceId, trafficRuleId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndTrafficRuleIdQueryFormat, defaultAppInstanceId, trafficRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte(fmt.Sprintf(writeTrafficObjectFormat, util.InactiveState))).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		//trafficRuleFilter := make([]dataplane.TrafficFilter, 1)
		trafficRule := dataplane.TrafficRule{TrafficRuleID: trafficRuleId, FilterType: "FLOW", Priority: 5,
			Action: "DROP", State: util.InactiveState}
		var trafficRules []dataplane.TrafficRule
		trafficRules = append(trafficRules, trafficRule)
		entry := models.AppDConfig{AppTrafficRule: trafficRules}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patches.Reset()

	// 22 is the order of the traffic get one handler in the URLPattern
	service.URLPatterns()[22].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)

}

// Update a dns rule
func TestPutTrafficRule(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	updateRule := dataplane.TrafficRule{
		TrafficRuleID: trafficRuleId,
		FilterType:    "FLOW",
		Priority:      5,
		TrafficFilter: []dataplane.TrafficFilter{},
		Action:        "DROP",
		State:         util.InactiveState,
	}
	updateRuleBytes, _ := json.Marshal(updateRule)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getOneTrafficRuleUrl, defaultAppInstanceId, trafficRuleId),
		bytes.NewReader(updateRuleBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndTrafficRuleIdQueryFormat, defaultAppInstanceId, trafficRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte(fmt.Sprintf(writeTrafficPutObjectFormat, util.InactiveState))).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		TrafficRule := dataplane.TrafficRule{TrafficRuleID: trafficRuleId, FilterType: "FLOW", Priority: 5,
			Action: "DROP", State: util.InactiveState}
		var TrafficRules []dataplane.TrafficRule
		TrafficRules = append(TrafficRules, TrafficRule)
		entry := models.AppDConfig{AppTrafficRule: TrafficRules}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patches.Reset()

	// 23 is the order of the Traffic Rule put handler in the URLPattern
	service.URLPatterns()[23].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
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

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		dnsRule := dataplane.DNSRule{DNSRuleID: dnsRuleId, DomainName: exampleDomainName, IPAddressType: "IP_V4", IPAddress: exampleIPAddress,
			TTL: defaultTTL, State: util.InactiveState}
		var dnsRules []dataplane.DNSRule
		dnsRules = append(dnsRules, dnsRule)
		entry := models.AppDConfig{AppDNSRule: dnsRules}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
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
	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		entry := models.AppDConfig{AppName: "appExample"}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patches.Reset()
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
		dnsRule := dataplane.DNSRule{DNSRuleID: dnsRuleId, DomainName: exampleDomainName, IPAddressType: "IP_V4", IPAddress: exampleIPAddress,
			TTL: 30, State: util.InactiveState}
		var dnsRules []dataplane.DNSRule
		dnsRules = append(dnsRules, dnsRule)
		entry := models.AppDConfig{AppDNSRule: dnsRules}
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

	updateRule := dataplane.DNSRule{
		DNSRuleID:     dnsRuleId,
		DomainName:    exampleDomainName,
		IPAddressType: util.IPv4Type,
		IPAddress:     exampleIPAddress,
		TTL:           defaultTTL,
		State:         util.InactiveState,
	}
	updateRuleBytes, _ := json.Marshal(updateRule)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader(updateRuleBytes))
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
		dnsRule := dataplane.DNSRule{DNSRuleID: dnsRuleId, DomainName: exampleDomainName, IPAddressType: "IP_V4", IPAddress: exampleIPAddress,
			TTL: 30, State: util.InactiveState}
		var dnsRules []dataplane.DNSRule
		dnsRules = append(dnsRules, dnsRule)
		entry := models.AppDConfig{AppDNSRule: dnsRules}
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

	patchInit1 := gomonkey.ApplyFunc(config.LoadMepServerConfig, func() (*config.MepServerConfig, error) {
		configData := `
# dns agent configuration
dnsAgent:
  # values: local, dataplane, all
  type: all
  # local dns server end point
  endPoint:
    address:
      host: localhost
      port: 80


# data plane option to use in Mp2 interface
dataplane:
  # values: none
  type: none

`
		var mepConfig config.MepServerConfig
		err := yaml.Unmarshal([]byte(configData), &mepConfig)
		if err != nil {
			assert.Fail(t, parseFail)
		}
		return &mepConfig, nil
	})
	defer patchInit1.Reset()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err2 := w.Write([]byte(""))
		if err2 != nil {
			t.Error(errorWriteRespErr)
		}
	}))
	defer ts.Close()

	patchInit2 := gomonkey.ApplyFunc(dns.NewRestDNSAgent, func(config *config.MepServerConfig) *dns.RestDNSAgent {
		parse, _ := url.Parse(ts.URL)
		return &dns.RestDNSAgent{ServerEndPoint: parse}
	})
	defer patchInit2.Reset()

	service := Mp1Service{}
	_ = service.Init()

	updateRule := dataplane.DNSRule{
		DNSRuleID:     dnsRuleId,
		DomainName:    exampleDomainName,
		IPAddressType: util.IPv4Type,
		IPAddress:     exampleIPAddress,
		TTL:           defaultTTL,
		State:         util.ActiveState,
	}
	updateRuleBytes, _ := json.Marshal(updateRule)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader(updateRuleBytes))
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

	patch1 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		dnsRule := dataplane.DNSRule{DNSRuleID: dnsRuleId, DomainName: exampleDomainName, IPAddressType: "IP_V4", IPAddress: exampleIPAddress,
			TTL: 30, State: util.InactiveState}
		var dnsRules []dataplane.DNSRule
		dnsRules = append(dnsRules, dnsRule)
		entry := models.AppDConfig{AppDNSRule: dnsRules}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patch1.Reset()

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

	updateRule := dataplane.DNSRule{
		DNSRuleID:     dnsRuleId,
		DomainName:    exampleDomainName,
		IPAddressType: util.IPv4Type,
		IPAddress:     exampleIPAddress,
		TTL:           defaultTTL,
		State:         util.ActiveState,
	}
	updateRuleBytes, _ := json.Marshal(updateRule)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader(updateRuleBytes))
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
		dnsRule := dataplane.DNSRule{DNSRuleID: dnsRuleId, DomainName: exampleDomainName, IPAddressType: "IP_V4", IPAddress: exampleIPAddress,
			TTL: 30, State: util.ActiveState}
		var dnsRules []dataplane.DNSRule
		dnsRules = append(dnsRules, dnsRule)
		entry := models.AppDConfig{AppDNSRule: dnsRules}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	patch2 := gomonkey.ApplyFunc(dns.NewRestDNSAgent, func(config *config.MepServerConfig) *dns.RestDNSAgent {
		parse, _ := url.Parse(ts.URL)
		return &dns.RestDNSAgent{ServerEndPoint: parse}
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

	patchInit1 := gomonkey.ApplyFunc(config.LoadMepServerConfig, func() (*config.MepServerConfig, error) {
		configData := `
# dns agent configuration
dnsAgent:
  # values: local, dataplane, all
  type: all
  # local dns server end point
  endPoint:
    address:
      host: localhost
      port: 80


# data plane option to use in Mp2 interface
dataplane:
  # values: none
  type: none

`
		var mepConfig config.MepServerConfig
		err := yaml.Unmarshal([]byte(configData), &mepConfig)
		if err != nil {
			assert.Fail(t, parseFail)
		}
		return &mepConfig, nil
	})
	defer patchInit1.Reset()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err2 := w.Write([]byte(""))
		if err2 != nil {
			t.Error(errorWriteRespErr)
		}
	}))
	defer ts.Close()

	patchInit2 := gomonkey.ApplyFunc(dns.NewRestDNSAgent, func(config *config.MepServerConfig) *dns.RestDNSAgent {
		parse, _ := url.Parse(ts.URL)
		return &dns.RestDNSAgent{ServerEndPoint: parse}
	})
	defer patchInit2.Reset()

	service := Mp1Service{}
	_ = service.Init()

	updateRule := dataplane.DNSRule{
		DNSRuleID:     dnsRuleId,
		DomainName:    exampleDomainName,
		IPAddressType: util.IPv4Type,
		IPAddress:     exampleIPAddress,
		TTL:           defaultTTL,
		State:         util.InactiveState,
	}
	updateRuleBytes, _ := json.Marshal(updateRule)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader(updateRuleBytes))
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

	patch1 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		dnsRule := dataplane.DNSRule{DNSRuleID: dnsRuleId, DomainName: exampleDomainName, IPAddressType: "IP_V4", IPAddress: exampleIPAddress,
			TTL: 30, State: util.ActiveState}
		var dnsRules []dataplane.DNSRule
		dnsRules = append(dnsRules, dnsRule)
		entry := models.AppDConfig{AppDNSRule: dnsRules}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patch1.Reset()

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

	patchInit1 := gomonkey.ApplyFunc(config.LoadMepServerConfig, func() (*config.MepServerConfig, error) {
		configData := `
# dns agent configuration
dnsAgent:
  # values: local, dataplane, all
  type: all
  # local dns server end point
  endPoint:
    address:
      host: localhost
      port: 80


# data plane option to use in Mp2 interface
dataplane:
  # values: none
  type: none

`
		var mepConfig config.MepServerConfig
		err := yaml.Unmarshal([]byte(configData), &mepConfig)
		if err != nil {
			assert.Fail(t, parseFail)
		}
		return &mepConfig, nil
	})
	defer patchInit1.Reset()

	service := Mp1Service{}
	_ = service.Init()

	updateRule := dataplane.DNSRule{
		DNSRuleID:     dnsRuleId,
		DomainName:    exampleDomainName,
		IPAddressType: util.IPv4Type,
		IPAddress:     exampleIPAddress,
		TTL:           defaultTTL,
		State:         util.ActiveState,
	}
	updateRuleBytes, _ := json.Marshal(updateRule)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader(updateRuleBytes))
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
		dnsRule := dataplane.DNSRule{DNSRuleID: dnsRuleId, DomainName: exampleDomainName, IPAddressType: "IP_V4", IPAddress: exampleIPAddress,
			TTL: 30, State: util.InactiveState}
		var dnsRules []dataplane.DNSRule
		dnsRules = append(dnsRules, dnsRule)
		entry := models.AppDConfig{AppDNSRule: dnsRules}
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

	patchInit1 := gomonkey.ApplyFunc(config.LoadMepServerConfig, func() (*config.MepServerConfig, error) {
		configData := `
# dns agent configuration
dnsAgent:
  # values: local, dataplane, all
  type: all
  # local dns server end point
  endPoint:
    address:
      host: localhost
      port: 80


# data plane option to use in Mp2 interface
dataplane:
  # values: none
  type: none

`
		var mepConfig config.MepServerConfig
		err := yaml.Unmarshal([]byte(configData), &mepConfig)
		if err != nil {
			assert.Fail(t, parseFail)
		}
		return &mepConfig, nil
	})
	defer patchInit1.Reset()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err2 := w.Write([]byte(""))
		if err2 != nil {
			t.Error(errorWriteRespErr)
		}
	}))
	defer ts.Close()

	patchInit2 := gomonkey.ApplyFunc(dns.NewRestDNSAgent, func(config *config.MepServerConfig) *dns.RestDNSAgent {
		parse, _ := url.Parse(ts.URL)
		return &dns.RestDNSAgent{ServerEndPoint: parse}
	})
	defer patchInit2.Reset()

	service := Mp1Service{}
	_ = service.Init()

	updateRule := dataplane.DNSRule{
		DNSRuleID:     dnsRuleId,
		DomainName:    exampleDomainName,
		IPAddressType: util.IPv4Type,
		IPAddress:     exampleIPAddress,
		TTL:           defaultTTL,
		State:         util.ActiveState,
	}
	updateRuleBytes, _ := json.Marshal(updateRule)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getDnsRuleUrlFormat, defaultAppInstanceId, dnsRuleId),
		bytes.NewReader(updateRuleBytes))
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
		dnsRule := dataplane.DNSRule{DNSRuleID: dnsRuleId, DomainName: exampleDomainName, IPAddressType: "IP_V4", IPAddress: exampleIPAddress,
			TTL: 30, State: util.InactiveState}
		var dnsRules []dataplane.DNSRule
		dnsRules = append(dnsRules, dnsRule)
		entry := models.AppDConfig{AppDNSRule: dnsRules}
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

//============================APP SERVICE AVAILABILITY SUBSCRIPTION=========================================
// Post App service availability Notification
func TestAppSubscribePost(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	createSubscription := models.SerAvailabilityNotificationSubscription{
		SubscriptionType:  "SerAvailabilityNotificationSubscription",
		CallbackReference: callBackRef,
		FilteringCriteria: models.FilteringCriteria{
			SerInstanceIds: []string{
				"f7e898d1c9ea9edd8a41295fc55c2373",
			},
			SerNames: []string{
				"FaceRegService5",
			},
			SerCategories: []models.CategoryRef{
				{
					Href:    callBackRef,
					ID:      "id12345",
					Name:    "RNI",
					Version: "1.2.2",
				},
			},
			States: []string{
				"ACTIVE",
			},
			IsLocal: true,
		},
	}
	createSubscriptionBytes, _ := json.Marshal(createSubscription)
	var resp = &pb.GetOneInstanceResponse{
		Response: &pb.Response{Code: pb.Response_SUCCESS},
	}
	n := &srv.InstanceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *pb.GetOneInstanceRequest) (*pb.GetOneInstanceResponse, error) {
		return resp, nil
	})
	defer patch1.Reset()
	// Create http get request
	postRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(postSubscribeUrl, defaultAppInstanceId),
		bytes.NewReader(createSubscriptionBytes))
	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 201)

	service.URLPatterns()[0].Func(mockWriter, postRequest)

	assert.Equal(t, "201", responseHeader.Get(responseStatusHeader),
		responseCheckFor201)
	notification := models.SerAvailabilityNotificationSubscription{}
	_ = json.Unmarshal(mockWriter.response, &notification)
	assert.Equal(t, subtype1, notification.SubscriptionType, errorSubtypeMissMatch)
	mockWriter.AssertExpectations(t)
}

// Post App service availability Notification With invalid json body
func TestAppSubscribePostWrongJsonBody(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	var resp = &pb.GetOneInstanceResponse{}
	n := &srv.InstanceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *pb.GetOneInstanceRequest) (*pb.GetOneInstanceResponse, error) {
		return resp, nil
	})
	defer patch1.Reset()
	// Create http get request
	postRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(postSubscribeUrl, defaultAppInstanceId),
		bytes.NewReader([]byte("Hello")))
	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	service.URLPatterns()[0].Func(mockWriter, postRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor400)
	notification := models.SerAvailabilityNotificationSubscription{}
	_ = json.Unmarshal(mockWriter.response, &notification)
	assert.NotEqual(t, subtype1, notification.SubscriptionType, errorSubtypeMissMatch)
	mockWriter.AssertExpectations(t)
}

// Post App service availability Notification and json marshalling failed
func TestAppSubscribePostJsonMarshallFail(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	createSubscription := models.SerAvailabilityNotificationSubscription{
		SubscriptionType:  "SerAvailabilityNotificationSubscription",
		CallbackReference: callBackRef,
		FilteringCriteria: models.FilteringCriteria{
			SerInstanceIds: []string{
				"f7e898d1c9ea9edd8a41295fc55c2373",
			},
			SerNames: []string{
				"FaceRegService5",
			},
			SerCategories: []models.CategoryRef{
				{
					Href:    callBackRef,
					ID:      "id12345",
					Name:    "RNI",
					Version: "1.2.2",
				},
			},
			States: []string{
				"ACTIVE",
			},
			IsLocal: true,
		},
	}
	createSubscriptionBytes, _ := json.Marshal(createSubscription)
	var resp = &pb.GetOneInstanceResponse{
		Response: &pb.Response{Code: pb.Response_SUCCESS},
	}
	n := &srv.InstanceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *pb.GetOneInstanceRequest) (*pb.GetOneInstanceResponse, error) {
		return resp, nil
	})
	defer patch1.Reset()
	var counter = 0
	patch2 := gomonkey.ApplyFunc(json.Marshal, func(i interface{}) (b []byte, e error) {
		counter++
		if counter == 3 {
			return nil, errors.New("json marshalling failed")
		} else {
			bs := new(bytes.Buffer)
			_ = json.NewEncoder(bs).Encode(i)
			return bs.Bytes(), nil
		}
	})
	defer patch2.Reset()
	// Create http get request
	postRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(postSubscribeUrl, defaultAppInstanceId),
		bytes.NewReader(createSubscriptionBytes))
	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 400)
	service.URLPatterns()[0].Func(mockWriter, postRequest)

	respError := models.ProblemDetails{}

	_ = json.Unmarshal(mockWriter.response, &respError)
	log.Info(respError.String())
	assert.Equal(t, "Bad Request", respError.Title, "Expected error not returned")

}

// Post App service availability Notification and json marshalling failed
func TestAppSubscribePostDbFail(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	createSubscription := models.SerAvailabilityNotificationSubscription{
		SubscriptionType:  "SerAvailabilityNotificationSubscription",
		CallbackReference: callBackRef,
		FilteringCriteria: models.FilteringCriteria{
			SerInstanceIds: []string{
				"f7e898d1c9ea9edd8a41295fc55c2373",
			},
			SerNames: []string{
				"FaceRegService5",
			},
			SerCategories: []models.CategoryRef{
				{
					Href:    callBackRef,
					ID:      "id12345",
					Name:    "RNI",
					Version: "1.2.2",
				},
			},
			States: []string{
				"ACTIVE",
			},
			IsLocal: true,
		},
	}
	createSubscriptionBytes, _ := json.Marshal(createSubscription)
	var resp = &pb.GetOneInstanceResponse{
		Response: &pb.Response{Code: pb.Response_SUCCESS},
	}
	n := &srv.InstanceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *pb.GetOneInstanceRequest) (*pb.GetOneInstanceResponse, error) {
		return resp, nil
	})
	defer patch1.Reset()
	var counter = 0
	patch1.ApplyFunc(json.Marshal, func(i interface{}) (b []byte, e error) {
		counter++
		if counter == 3 {
			return nil, errors.New("json marshalling failed")
		} else {
			bs := new(bytes.Buffer)
			_ = json.NewEncoder(bs).Encode(i)
			return bs.Bytes(), nil
		}
	})

	ec := &buildin.BuildinRegistry{}
	patch1.ApplyMethod(reflect.TypeOf(ec), "TxnWithCmp", func(t *buildin.BuildinRegistry, ctx context.Context, success []registry.PluginOp, cmp []registry.CompareOp, fail []registry.PluginOp) (*registry.PluginResponse, error) {
		return nil, fmt.Errorf("db error")
	})
	// Create http get request
	postRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(postSubscribeUrl, defaultAppInstanceId),
		bytes.NewReader(createSubscriptionBytes))
	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 400)
	service.URLPatterns()[0].Func(mockWriter, postRequest)

	respError := models.ProblemDetails{}

	_ = json.Unmarshal(mockWriter.response, &respError)
	log.Info(respError.String())
	assert.Equal(t, "Bad Request", respError.Title, "Expected error not returned")

}

// Post App service availability Notification and json marshalling failed
func TestAppSubscribePostRspCountExceeds(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	createSubscription := models.SerAvailabilityNotificationSubscription{
		SubscriptionType:  "SerAvailabilityNotificationSubscription",
		CallbackReference: callBackRef,
		FilteringCriteria: models.FilteringCriteria{
			SerInstanceIds: []string{
				"f7e898d1c9ea9edd8a41295fc55c2373",
			},
			SerNames: []string{
				"FaceRegService5",
			},
			SerCategories: []models.CategoryRef{
				{
					Href:    callBackRef,
					ID:      "id12345",
					Name:    "RNI",
					Version: "1.2.2",
				},
			},
			States: []string{
				"ACTIVE",
			},
			IsLocal: true,
		},
	}
	createSubscriptionBytes, _ := json.Marshal(createSubscription)
	var resp = &pb.GetOneInstanceResponse{
		Response: &pb.Response{Code: pb.Response_SUCCESS},
	}
	n := &srv.InstanceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *pb.GetOneInstanceRequest) (*pb.GetOneInstanceResponse, error) {
		return resp, nil
	})
	defer patch1.Reset()
	var counter = 0
	patch1.ApplyFunc(json.Marshal, func(i interface{}) (b []byte, e error) {
		counter++
		if counter == 3 {
			return nil, errors.New("json marshalling failed")
		} else {
			bs := new(bytes.Buffer)
			_ = json.NewEncoder(bs).Encode(i)
			return bs.Bytes(), nil
		}
	})

	ec := &buildin.BuildinRegistry{}
	patch1.ApplyMethod(reflect.TypeOf(ec), "TxnWithCmp", func(t *buildin.BuildinRegistry, ctx context.Context, success []registry.PluginOp, cmp []registry.CompareOp, fail []registry.PluginOp) (*registry.PluginResponse, error) {
		return &registry.PluginResponse{
			Succeeded: true,
			Revision:  1,
			Kvs:       nil,
			Count:     51,
		}, nil
	})
	// Create http get request
	postRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(postSubscribeUrl, defaultAppInstanceId),
		bytes.NewReader(createSubscriptionBytes))
	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 400)
	service.URLPatterns()[0].Func(mockWriter, postRequest)

	respError := models.ProblemDetails{}

	_ = json.Unmarshal(mockWriter.response, &respError)
	log.Info(respError.String())
	assert.Equal(t, "App subscription error", respError.Title, "Expected error not returned")

}

// Query All app service availability Notification subscriptions
func TestAppSubscribeGetAll(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getSubscribeUrl, defaultAppInstanceId),
		nil)
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 404)

	service.URLPatterns()[1].Func(mockWriterGet, getRequest)
}

// Query One app service availability Notification subscriptions
func TestGetOneAppSubscribe(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	subscriptionId := uuid.NewV4().String()
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getOrDelOneSubscribeOrSveUrl, defaultAppInstanceId, subscriptionId),
		nil)
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 404)

	service.URLPatterns()[2].Func(mockWriterGet, getRequest)
}

// Delete app service availability Notification subscription
func TestDelOneAppSubscribe(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	subscriptionId := uuid.NewV4().String()
	getRequest, _ := http.NewRequest("DELETE",
		fmt.Sprintf(getOrDelOneSubscribeOrSveUrl, defaultAppInstanceId, subscriptionId),
		nil)
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 404)

	service.URLPatterns()[3].Func(mockWriterGet, getRequest)
}

//============================APP TERMINATION NOTIFICATION SUBSCRIPTION=====================================

// Post App termination Notification subscription
func TestAppTerminationSubscribePost(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	createSubscription := models.AppTerminationNotificationSubscription{
		SubscriptionType:  "AppTerminationNotificationSubscription",
		CallbackReference: callBackRef,
		AppInstanceId:     "6abe4782-2c70-4e47-9a4e-0ee3a1a0fd1e",
	}
	createSubscriptionBytes, _ := json.Marshal(createSubscription)
	var resp = &pb.GetOneInstanceResponse{
		Response: &pb.Response{Code: pb.Response_SUCCESS},
	}
	n := &srv.InstanceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *pb.GetOneInstanceRequest) (*pb.GetOneInstanceResponse, error) {
		return resp, nil
	})
	defer patch1.Reset()
	// Create http get request
	postRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(postAppTerminologiesUrl, defaultAppInstanceId),
		bytes.NewReader(createSubscriptionBytes))
	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 201)

	service.URLPatterns()[9].Func(mockWriter, postRequest)

	assert.Equal(t, "201", responseHeader.Get(responseStatusHeader),
		responseCheckFor201)
	notification := models.AppTerminationNotificationSubscription{}
	_ = json.Unmarshal(mockWriter.response, &notification)
	assert.Equal(t, subtype2, notification.SubscriptionType, errorSubtypeMissMatch)
	mockWriter.AssertExpectations(t)
}

// Get all App termination Notification subscription
func TestAppTerminationSubscribeGet(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getAppTerminologiesUrl, defaultAppInstanceId),
		nil)
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 404)

	service.URLPatterns()[10].Func(mockWriterGet, getRequest)
}

// Get One App termination Notification subscription
func TestGetOneAppTerminationSubscribe(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	subscriptionId := uuid.NewV4().String()
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getOneAppTerminologiesUrl, defaultAppInstanceId, subscriptionId),
		nil)
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 404)

	service.URLPatterns()[11].Func(mockWriterGet, getRequest)
}

// Delete One App termination Notification subscription
func TestDelOneAppTerminationSubscribe(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	subscriptionId := uuid.NewV4().String()
	getRequest, _ := http.NewRequest("DELETE",
		fmt.Sprintf(delOneAppTerminologiesUrl, defaultAppInstanceId, subscriptionId),
		nil)
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 404)

	service.URLPatterns()[12].Func(mockWriterGet, getRequest)
}

//========================================PRODUCER SERVICE=================================================
type serviceInfo struct {
	//	SerInstanceId     string        `json:"serInstanceId,omitempty"`
	SerName           string               `json:"serName" validate:"required,max=128,validateName"`
	SerCategory       models.CategoryRef   `json:"serCategory" validate:"omitempty"`
	Version           string               `json:"version" validate:"required,max=32"`
	State             string               `json:"state" validate:"required,oneof=ACTIVE INACTIVE"`
	TransportID       string               `json:"transportId" validate:"omitempty,max=64,validateId"`
	TransportInfo     models.TransportInfo `json:"transportInfo" validate:"omitempty"`
	Serializer        string               `json:"serializer" validate:"required,oneof=JSON XML PROTOBUF3"`
	ScopeOfLocality   string               `json:"scopeOfLocality" validate:"omitempty,oneof=MEC_SYSTEM MEC_HOST NFVI_POP ZONE ZONE_GROUP NFVI_NODE"`
	ConsumedLocalOnly bool                 `json:"consumedLocalOnly,omitempty"`
	IsLocal           bool                 `json:"isLocal,omitempty"`
}

// Register a service
func TestPostServiceRegister(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	svcCat := models.CategoryRef{
		Href:    href,
		ID:      "id12345",
		Name:    "RNI",
		Version: "1.2.3",
	}

	var theArray = make([]string, 1)
	theArray[0] = "OAUTH2_CLIENT_CREDENTIALS"

	authInfo := models.SecurityInfoOAuth2Info{
		GrantTypes:    theArray,
		TokenEndpoint: tokenEndPoint,
	}

	sec1 := models.SecurityInfo{
		OAuth2Info: authInfo,
	}
	transInfo := models.TransportInfo{
		ID:               "TransId12345",
		Name:             "REST",
		Description:      restApi,
		TransType:        "REST_HTTP",
		Protocol:         "HTTP",
		Version:          "2.0",
		Endpoint:         models.EndPointInfo{},
		Security:         sec1,
		ImplSpecificInfo: nil,
	}
	serviceInf := serviceInfo{
		SerName:           "FaceRegService5",
		SerCategory:       svcCat,
		Version:           "4.5.8",
		State:             "ACTIVE",
		TransportID:       "Rest1",
		TransportInfo:     transInfo,
		Serializer:        "JSON",
		ScopeOfLocality:   "MEC_SYSTEM",
		ConsumedLocalOnly: false,
		IsLocal:           true,
	}
	serviceInfBytes, _ := json.Marshal(serviceInf)
	//Patching
	var srvresp = &pb.CreateServiceResponse{
		Response:  &pb.Response{Code: pb.Response_SUCCESS},
		ServiceId: sampleServiceId,
	}
	n1 := &srv.MicroServiceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n1), "Create", func(*srv.MicroServiceService, context.Context, *pb.CreateServiceRequest) (*pb.CreateServiceResponse, error) {
		return srvresp, nil
	})
	defer patch1.Reset()

	var instResp = &pb.RegisterInstanceResponse{
		Response:   &pb.Response{Code: pb.Response_SUCCESS},
		InstanceId: sampleServiceId,
	}
	n2 := &srv.InstanceService{}
	patch2 := gomonkey.ApplyMethod(reflect.TypeOf(n2), "Register", func(*srv.InstanceService, context.Context, *pb.RegisterInstanceRequest) (*pb.RegisterInstanceResponse, error) {
		return instResp, nil
	})
	defer patch2.Reset()

	var findInstResp = &pb.FindInstancesResponse{
		Response: &pb.Response{Code: pb.Response_SUCCESS},
		Instances: []*pb.MicroServiceInstance{
			{
				InstanceId: sampleInstanceId,
				ServiceId:  sampleServiceId,
			},
		},
	}
	patch3 := gomonkey.ApplyFunc(util.FindInstanceByKey, func(url.Values) (*pb.FindInstancesResponse, error) {
		return findInstResp, nil
	})
	defer patch3.Reset()

	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(serviceDiscoverUrlFormat, defaultAppInstanceId),
		bytes.NewReader(serviceInfBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 201)

	// 3 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[4].Func(mockWriter, getRequest)
}

// Register a service and json marshalling failed when return response
func TestPostServiceRegisterJsonMarshalFail(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	svcCat := models.CategoryRef{
		Href:    href,
		ID:      "id12345",
		Name:    "RNI",
		Version: "1.2.3",
	}

	var theArray = make([]string, 1)
	theArray[0] = "OAUTH2_CLIENT_CREDENTIALS"

	authInfo := models.SecurityInfoOAuth2Info{
		GrantTypes:    theArray,
		TokenEndpoint: tokenEndPoint,
	}

	sec1 := models.SecurityInfo{
		OAuth2Info: authInfo,
	}
	transInfo := models.TransportInfo{
		ID:               "TransId12345",
		Name:             "REST",
		Description:      restApi,
		TransType:        "REST_HTTP",
		Protocol:         "HTTP",
		Version:          "2.0",
		Endpoint:         models.EndPointInfo{},
		Security:         sec1,
		ImplSpecificInfo: nil,
	}
	serviceInf := serviceInfo{
		SerName:           "FaceRegService5",
		SerCategory:       svcCat,
		Version:           "4.5.8",
		State:             "ACTIVE",
		TransportID:       "Rest1",
		TransportInfo:     transInfo,
		Serializer:        "JSON",
		ScopeOfLocality:   "MEC_SYSTEM",
		ConsumedLocalOnly: false,
		IsLocal:           true,
	}
	serviceInfBytes, _ := json.Marshal(serviceInf)
	//Patching
	var srvResp = &pb.CreateServiceResponse{
		Response:  &pb.Response{Code: pb.Response_SUCCESS},
		ServiceId: sampleServiceId,
	}
	n1 := &srv.MicroServiceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n1), "Create", func(*srv.MicroServiceService, context.Context, *pb.CreateServiceRequest) (*pb.CreateServiceResponse, error) {
		return srvResp, nil
	})
	defer patch1.Reset()

	var instresp = &pb.RegisterInstanceResponse{
		Response:   &pb.Response{Code: pb.Response_SUCCESS},
		InstanceId: sampleServiceId,
	}
	n2 := &srv.InstanceService{}
	patch2 := gomonkey.ApplyMethod(reflect.TypeOf(n2), "Register", func(*srv.InstanceService, context.Context, *pb.RegisterInstanceRequest) (*pb.RegisterInstanceResponse, error) {
		return instresp, nil
	})
	defer patch2.Reset()

	var counter = 0
	patch3 := gomonkey.ApplyFunc(json.Marshal, func(i interface{}) (b []byte, e error) {
		counter++
		if counter == 3 {
			return nil, errors.New("json marshalling failed")
		} else {
			bs := new(bytes.Buffer)
			_ = json.NewEncoder(bs).Encode(i)
			return bs.Bytes(), nil
		}
	})
	defer patch3.Reset()

	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(serviceDiscoverUrlFormat, defaultAppInstanceId),
		bytes.NewReader(serviceInfBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	// 3 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[4].Func(mockWriter, getRequest)
}

// Register a service but find instance failed
func TestPostServiceRegisterFindInstanceByKeyFailed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	svcCat := models.CategoryRef{
		Href:    href,
		ID:      "id12345",
		Name:    "RNI",
		Version: "1.2.3",
	}

	var theArray = make([]string, 1)
	theArray[0] = "OAUTH2_CLIENT_CREDENTIALS"

	authInfo := models.SecurityInfoOAuth2Info{
		GrantTypes:    theArray,
		TokenEndpoint: tokenEndPoint,
	}

	sec1 := models.SecurityInfo{
		OAuth2Info: authInfo,
	}
	transInfo := models.TransportInfo{
		ID:               "TransId12345",
		Name:             "REST",
		Description:      restApi,
		TransType:        "REST_HTTP",
		Protocol:         "HTTP",
		Version:          "2.0",
		Endpoint:         models.EndPointInfo{},
		Security:         sec1,
		ImplSpecificInfo: nil,
	}
	serviceInf := serviceInfo{
		SerName:           "FaceRegService5",
		SerCategory:       svcCat,
		Version:           "4.5.8",
		State:             "ACTIVE",
		TransportID:       "Rest1",
		TransportInfo:     transInfo,
		Serializer:        "JSON",
		ScopeOfLocality:   "MEC_SYSTEM",
		ConsumedLocalOnly: false,
		IsLocal:           true,
	}
	serviceInfBytes, _ := json.Marshal(serviceInf)
	//Patching
	var srvResp = &pb.CreateServiceResponse{
		Response:  &pb.Response{Code: pb.Response_SUCCESS},
		ServiceId: sampleServiceId,
	}
	n1 := &srv.MicroServiceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n1), "Create", func(*srv.MicroServiceService, context.Context, *pb.CreateServiceRequest) (*pb.CreateServiceResponse, error) {
		return srvResp, nil
	})
	defer patch1.Reset()

	var instResp = &pb.RegisterInstanceResponse{
		Response:   &pb.Response{Code: pb.Response_SUCCESS},
		InstanceId: sampleServiceId,
	}
	n2 := &srv.InstanceService{}
	patch2 := gomonkey.ApplyMethod(reflect.TypeOf(n2), "Register", func(*srv.InstanceService, context.Context, *pb.RegisterInstanceRequest) (*pb.RegisterInstanceResponse, error) {
		return instResp, nil
	})
	defer patch2.Reset()

	patch3 := gomonkey.ApplyFunc(util.FindInstanceByKey, func(url.Values) (*pb.FindInstancesResponse, error) {
		return nil, errors.New("instance not found")
	})
	defer patch3.Reset()

	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(serviceDiscoverUrlFormat, defaultAppInstanceId),
		bytes.NewReader(serviceInfBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	// 3 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[4].Func(mockWriter, getRequest)
}

// Discover/Query a service but service name not found
func TestServiceDiscoverServiceNameNotFound(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(serviceDiscoverUrlFormat, defaultAppInstanceId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(serNameQueryFormat, defaultAppInstanceId, "somename")
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	service.URLPatterns()[5].Func(mockWriter, getRequest)

}

// Discover/Query a service and service found
func TestServiceDiscoverServiceFound(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(serviceDiscoverUrlFormat, defaultAppInstanceId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	var findInstResp = &pb.FindInstancesResponse{
		Response: &pb.Response{Code: pb.Response_SUCCESS},
		Instances: []*pb.MicroServiceInstance{
			{
				InstanceId: defaultAppInstanceId,
				ServiceId:  sampleServiceId,
			},
		},
	}
	patch1 := gomonkey.ApplyFunc(util.FindInstanceByKey, func(url.Values) (*pb.FindInstancesResponse, error) {
		return findInstResp, nil
	})
	defer patch1.Reset()

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	service.URLPatterns()[5].Func(mockWriter, getRequest)

}

// Update a service parameter
func TestPutServiceUpdate(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	svcCat := models.CategoryRef{
		Href:    href,
		ID:      "id12345",
		Name:    "RNI",
		Version: "1.2.3",
	}

	var theArray = make([]string, 1)
	theArray[0] = "OAUTH2_CLIENT_CREDENTIALS"

	authInfo := models.SecurityInfoOAuth2Info{
		GrantTypes:    theArray,
		TokenEndpoint: tokenEndPoint,
	}

	sec1 := models.SecurityInfo{
		OAuth2Info: authInfo,
	}
	transInfo := models.TransportInfo{
		ID:               "TransId12345",
		Name:             "REST",
		Description:      restApi,
		TransType:        "REST_HTTP",
		Protocol:         "HTTP",
		Version:          "2.0",
		Endpoint:         models.EndPointInfo{},
		Security:         sec1,
		ImplSpecificInfo: nil,
	}
	serviceInf := serviceInfo{
		SerName:           "FaceRegService5",
		SerCategory:       svcCat,
		Version:           "4.5.8",
		State:             "ACTIVE",
		TransportID:       "Rest1",
		TransportInfo:     transInfo,
		Serializer:        "JSON",
		ScopeOfLocality:   "MEC_SYSTEM",
		ConsumedLocalOnly: false,
		IsLocal:           true,
	}
	serviceInfBytes, _ := json.Marshal(serviceInf)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getOrDelOneSubscribeOrSveUrl, defaultAppInstanceId, sampleServiceId),
		bytes.NewReader(serviceInfBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndServiceIdQueryFormat, defaultAppInstanceId, sampleServiceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	var findInstResp = &pb.MicroServiceInstance{
		InstanceId: sampleInstanceId,
		ServiceId:  sampleServiceId,
	}
	patch1 := gomonkey.ApplyFunc(util.GetServiceInstance, func(ctx context.Context, serviceId string) (*pb.MicroServiceInstance, error) {
		return findInstResp, nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyFunc(svcutil.UpdateInstance, func(context.Context, string, *pb.MicroServiceInstance) *scerr.Error {
		return nil
	})
	defer patch2.Reset()

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	// 3 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[6].Func(mockWriter, getRequest)
}

// Update a service parameter but service not found
func TestPutServiceUpdateFindInstanceFail(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	svcCat := models.CategoryRef{
		Href:    href,
		ID:      "id12345",
		Name:    "RNI",
		Version: "1.2.3",
	}

	var theArray = make([]string, 1)
	theArray[0] = "OAUTH2_CLIENT_CREDENTIALS"

	authInfo := models.SecurityInfoOAuth2Info{
		GrantTypes:    theArray,
		TokenEndpoint: tokenEndPoint,
	}

	sec1 := models.SecurityInfo{
		OAuth2Info: authInfo,
	}
	transInfo := models.TransportInfo{
		ID:               "TransId12345",
		Name:             "REST",
		Description:      restApi,
		TransType:        "REST_HTTP",
		Protocol:         "HTTP",
		Version:          "2.0",
		Endpoint:         models.EndPointInfo{},
		Security:         sec1,
		ImplSpecificInfo: nil,
	}
	serviceInf := serviceInfo{
		SerName:           "FaceRegService5",
		SerCategory:       svcCat,
		Version:           "4.5.8",
		State:             "ACTIVE",
		TransportID:       "Rest1",
		TransportInfo:     transInfo,
		Serializer:        "JSON",
		ScopeOfLocality:   "MEC_SYSTEM",
		ConsumedLocalOnly: false,
		IsLocal:           true,
	}
	serviceInfBytes, _ := json.Marshal(serviceInf)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getOrDelOneSubscribeOrSveUrl, defaultAppInstanceId, sampleServiceId),
		bytes.NewReader(serviceInfBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndServiceIdQueryFormat, defaultAppInstanceId, sampleServiceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 404)

	// 3 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[6].Func(mockWriter, getRequest)
}

// Query a service with invalid service id
func TestGetOneServiceWithInvalidId(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	serviceId := uuid.NewV4().String()
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getOrDelOneSubscribeOrSveUrl, defaultAppInstanceId, serviceId),
		nil)
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 400)

	service.URLPatterns()[7].Func(mockWriterGet, getRequest)
}

// Query a service with valid service id
func TestGetOneServiceWithValidId(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getOrDelOneSubscribeOrSveUrl, defaultAppInstanceId, sampleServiceId),
		nil)
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndServiceIdQueryFormat, defaultAppInstanceId, sampleServiceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	var resp = &pb.GetOneInstanceResponse{
		Response: &pb.Response{Code: pb.Response_SUCCESS},
		Instance: &pb.MicroServiceInstance{
			InstanceId: sampleInstanceId,
			ServiceId:  sampleServiceId,
			Properties: map[string]string{
				"serCategory/href": "b",
				"serCategory/id":   "",
				"serCategory/name": "",
				"serName":          "NewService",
				"endPointType":     "addresses",
			},
			Endpoints: []string{
				"100.1.1.1:8080",
			},
		},
	}
	n := &srv.InstanceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *pb.GetOneInstanceRequest) (*pb.GetOneInstanceResponse, error) {
		return resp, nil
	})
	defer patch1.Reset()

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 200)

	service.URLPatterns()[7].Func(mockWriterGet, getRequest)
}

// Delete a service with valid service id
func TestDelOneServiceWithValidId(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	serviceId := uuid.NewV4().String()
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getOrDelOneSubscribeOrSveUrl, defaultAppInstanceId, serviceId),
		nil)
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndServiceIdQueryFormat, defaultAppInstanceId, sampleServiceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 404)

	service.URLPatterns()[8].Func(mockWriterGet, getRequest)
}

// Delete a service with invalid service id
func TestDelOneServiceWithInValidId(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	serviceId := uuid.NewV4().String()
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getOrDelOneSubscribeOrSveUrl, defaultAppInstanceId, serviceId),
		nil)
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 400)

	service.URLPatterns()[8].Func(mockWriterGet, getRequest)
}

func TestGetHeartbeat(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()
	service := Mp1Service{}
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(heartBeatUrl, defaultAppInstanceId, sampleServiceId),
		nil)
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndServiceIdQueryFormat, defaultAppInstanceId, sampleServiceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)
	seconds := time.Now().UTC().Unix()
	nSeconds := time.Now().UTC().UnixNano() - (seconds * 1000000000)
	var resp = &pb.GetOneInstanceResponse{
		Response: &pb.Response{Code: pb.Response_SUCCESS},
		Instance: &pb.MicroServiceInstance{
			InstanceId: sampleInstanceId,
			ServiceId:  sampleServiceId,
			Properties: map[string]string{
				"mecState":         "ACTIVE",
				"livenessInterval": "60",
				secString:          strconv.FormatInt(seconds, formatIntBase),
				nanosecString:      strconv.FormatInt(nSeconds, formatIntBase),
			},
		},
	}
	n := &srv.InstanceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *pb.GetOneInstanceRequest) (*pb.GetOneInstanceResponse, error) {
		return resp, nil
	})
	defer patch1.Reset()
	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 200)
	service.URLPatterns()[16].Func(mockWriterGet, getRequest)
	assert.Equal(t, "200", responseGetHeader.Get(responseStatusHeader),
		responseCheckFor200)
	mockWriterGet.AssertExpectations(t)
}

// Query a heartbeat data
func TestGetHeartbeatForInvalidServiceId(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()
	service := Mp1Service{}
	serviceId := uuid.NewV4().String()
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(heartBeatUrl, defaultAppInstanceId, serviceId),
		nil)
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndServiceIdQueryFormat, defaultAppInstanceId, serviceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)
	var resp = &pb.GetOneInstanceResponse{
		Response: &pb.Response{Code: pb.Response_SUCCESS},
		Instance: &pb.MicroServiceInstance{
			InstanceId: sampleInstanceId,
			ServiceId:  sampleServiceId,
			Properties: map[string]string{
				"mecState":         "ACTIVE",
				"livenessInterval": "60",
				secString:          strconv.FormatInt(time.Now().UTC().Unix(), formatIntBase),
				nanosecString:      strconv.FormatInt(time.Now().UTC().UnixNano(), formatIntBase),
			},
		},
	}
	n := &srv.InstanceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *pb.GetOneInstanceRequest) (*pb.GetOneInstanceResponse, error) {
		return resp, nil
	})
	defer patch1.Reset()
	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 400)
	service.URLPatterns()[16].Func(mockWriterGet, getRequest)
	assert.Equal(t, "400", responseGetHeader.Get(responseStatusHeader),
		responseCheckFor400)
	mockWriterGet.AssertExpectations(t)
}

// service heartbeat
func TestHeartbeatService(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()
	service := Mp1Service{}
	heartbeatRequest := models.ServiceLivenessUpdate{State: "ACTIVE"}
	heartbeatRequestBytes, _ := json.Marshal(heartbeatRequest)
	//Patching
	var updatePropertiesResp = &pb.UpdateInstancePropsResponse{
		Response: &pb.Response{Code: pb.Response_SUCCESS},
	}
	n1 := &srv.InstanceService{}
	var resp = &pb.GetOneInstanceResponse{
		Response: &pb.Response{Code: pb.Response_SUCCESS},
		Instance: &pb.MicroServiceInstance{
			InstanceId: sampleInstanceId,
			ServiceId:  sampleServiceId,
			Properties: map[string]string{
				"mecState":         "ACTIVE",
				"livenessInterval": "60",
				secString:          strconv.FormatInt(time.Now().UTC().Unix(), formatIntBase),
				nanosecString:      strconv.FormatInt(time.Now().UTC().UnixNano(), formatIntBase),
			},
		},
	}
	n := &srv.InstanceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *pb.GetOneInstanceRequest) (*pb.GetOneInstanceResponse, error) {
		return resp, nil
	})
	defer patch1.Reset()
	patch2 := gomonkey.ApplyMethod(reflect.TypeOf(n1), "UpdateInstanceProperties", func(*srv.InstanceService, context.Context, *pb.UpdateInstancePropsRequest) (*pb.UpdateInstancePropsResponse, error) {
		return updatePropertiesResp, nil
	})
	defer patch2.Reset()
	// Create http put request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(heartBeatUrl, defaultAppInstanceId, sampleServiceId),
		bytes.NewReader(heartbeatRequestBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndServiceIdQueryFormat, defaultAppInstanceId, sampleServiceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)
	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 204)
	// 3 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[17].Func(mockWriter, getRequest)
	assert.Equal(t, "204", responseHeader.Get(responseStatusHeader),
		responseCheckFor204)
	mockWriter.AssertExpectations(t)
}
func TestHeartbeatServiceInvalidServiceId(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()
	service := Mp1Service{}
	heartbeatRequest := models.ServiceLivenessUpdate{State: "ACTIVE"}
	heartbeatRequestBytes, _ := json.Marshal(heartbeatRequest)
	//Patching
	var updatePropertiesResp = &pb.UpdateInstancePropsResponse{
		Response: &pb.Response{Code: pb.Response_SUCCESS},
	}
	n1 := &srv.InstanceService{}
	var resp = &pb.GetOneInstanceResponse{
		Response: &pb.Response{Code: pb.Response_SUCCESS},
		Instance: &pb.MicroServiceInstance{
			InstanceId: sampleInstanceId,
			ServiceId:  sampleServiceId,
			Properties: map[string]string{
				"mecState":         "ACTIVE",
				"livenessInterval": "60",
				secString:          strconv.FormatInt(time.Now().UTC().Unix(), formatIntBase),
				nanosecString:      strconv.FormatInt(time.Now().UTC().UnixNano(), formatIntBase),
			},
		},
	}
	n := &srv.InstanceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *pb.GetOneInstanceRequest) (*pb.GetOneInstanceResponse, error) {
		return resp, nil
	})
	defer patch1.Reset()
	patch2 := gomonkey.ApplyMethod(reflect.TypeOf(n1), "UpdateInstanceProperties", func(*srv.InstanceService, context.Context, *pb.UpdateInstancePropsRequest) (*pb.UpdateInstancePropsResponse, error) {
		return updatePropertiesResp, nil
	})
	defer patch2.Reset()
	// Create http PUT request
	serviceId := uuid.NewV4().String()
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(heartBeatUrl, defaultAppInstanceId, serviceId),
		bytes.NewReader(heartbeatRequestBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndServiceIdQueryFormat, defaultAppInstanceId, serviceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)
	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 400)
	// 3 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[17].Func(mockWriter, getRequest)
	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor400)
	mockWriter.AssertExpectations(t)
}

// get Current time
func TestGetCurrentTime(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getTiming, getCurrentTIme),
		bytes.NewReader([]byte("")))

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte(writeCurTimetFormat+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(ntp.QueryWithOptions, func(host string, opt ntp.QueryOptions) (*ntp.Response, error) {
		var ntpRsp ntp.Response

		ntpRsp.Time = time.Unix(1623770544, 468538768)
		ntpRsp.Stratum = 1
		return &ntpRsp, nil
	})

	defer patches.Reset()

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[24].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// get current time failure
func TestGetCurrentTimeFailed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getTiming, getCurrentTIme),
		bytes.NewReader([]byte("")))

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"title\":\"Bad Request\",\"status\":1,\"detail\":\"Current time get failed\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(ntpc.GetTimeStamp, func() (curTime *ntpc.NtpCurrentTime, errorCode int) {
		var currentTime ntpc.NtpCurrentTime
		return &currentTime, 1
	})

	defer patches.Reset()

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[24].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor400)

	mockWriter.AssertExpectations(t)
}

// Query dns rules request in mp1 interface
func TestGetCurrentTimeNonSyncTime(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getTiming, getCurrentTIme),
		bytes.NewReader([]byte("")))

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte(writeCurTimeNonSynct+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(ntp.QueryWithOptions, func(host string, opt ntp.QueryOptions) (*ntp.Response, error) {
		var ntpRsp ntp.Response

		ntpRsp.Time = time.Unix(1623770544, 468538768)
		ntpRsp.Stratum = 16
		return &ntpRsp, nil
	})

	defer patches.Reset()

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[24].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

//Query current time connect failed
func TestGetCurrentTimeNtpConnFailed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getTiming, getCurrentTIme),
		bytes.NewReader([]byte("")))

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":21,\"detail\":\"Current time get failed\"}"+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(ntp.QueryWithOptions, func(host string, opt ntp.QueryOptions) (*ntp.Response, error) {
		var ntpRsp ntp.Response

		ntpRsp.Time = time.Unix(1623770544, 468538768)
		ntpRsp.Stratum = 16
		return nil, nil
	})

	defer patches.Reset()

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[24].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Get timing capabilities
func TestGetTimeCaps(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getTiming, getCaps),
		bytes.NewReader([]byte("")))

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte(writeCaps+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(ntp.QueryWithOptions, func(host string, opt ntp.QueryOptions) (*ntp.Response, error) {
		var ntpRsp ntp.Response

		ntpRsp.Time = time.Unix(1623770544, 468538768)

		return &ntpRsp, nil
	})

	defer patches.Reset()

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[25].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Get timing capabilities
func TestGetTimeCapsNtpFailed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getTiming, getCaps),
		bytes.NewReader([]byte("")))

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":21,\"detail\":\"Timing caps get failed\"}"+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(ntp.QueryWithOptions, func(host string, opt ntp.QueryOptions) (*ntp.Response, error) {
		return nil, nil
	})

	defer patches.Reset()

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[25].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader), responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query dns rules request in mp1 interface
func TestGetTransportInfo(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getTiming, getTransport),
		bytes.NewReader([]byte("")))

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("["+writeTransport+"]\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})

	defer patches.Reset()

	patches.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		resultList := make(map[string][]byte)
		return resultList, 0
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[26].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query dns rules request in mp1 interface
func TestGetTransportInfoDecodeFailed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getTiming, getTransport),
		bytes.NewReader([]byte("")))

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":4,\"detail\":\"Get transport info failed\"}"+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		resultList := make(map[string][]byte)
		resultList[util.TransportInfoPath] = []byte("[" + writeTransport + "]\n")
		return resultList, 0
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[26].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query dns rules request in mp1 interface
func TestGetTransportInfoList(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getTiming, getTransport),
		bytes.NewReader([]byte("")))

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("["+writeTransport+"]\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		resultList := make(map[string][]byte)
		resultList[util.TransportInfoPath] = []byte(writeTransport)
		return resultList, 0
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[26].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query dns rules request in mp1 interface
func TestGetTransportInfoPutRecordFailed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getTiming, getTransport),
		bytes.NewReader([]byte("")))

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":1,\"detail\":\"Get transport info failed\"}"+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		resultList := make(map[string][]byte)
		return resultList, 0
	})

	patches.ApplyFunc(backend.PutRecord, func(path string, value []byte) int {
		return 1
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[26].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Update a dns rule
func TestPutTrafficRuleStateAddRule(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	service.dataPlane = &none.NoneDataPlane{}
	//service.dataPlane.AddTrafficRule()
	updateRule := dataplane.TrafficRule{
		TrafficRuleID: trafficRuleId,
		FilterType:    "FLOW",
		Priority:      5,
		TrafficFilter: []dataplane.TrafficFilter{},
		Action:        "DROP",
		State:         util.ActiveState,
	}
	updateRuleBytes, _ := json.Marshal(updateRule)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getOneTrafficRuleUrl, defaultAppInstanceId, trafficRuleId),
		bytes.NewReader(updateRuleBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndTrafficRuleIdQueryFormat, defaultAppInstanceId, trafficRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte(fmt.Sprintf(writeTrafficPutObjectFormat, util.ActiveState))).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		TrafficRule := dataplane.TrafficRule{TrafficRuleID: trafficRuleId, FilterType: "FLOW", Priority: 5,
			Action: "DROP", State: util.InactiveState}
		var TrafficRules []dataplane.TrafficRule
		TrafficRules = append(TrafficRules, TrafficRule)
		entry := models.AppDConfig{AppTrafficRule: TrafficRules}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patches.Reset()

	// 23 is the order of the Traffic Rule put handler in the URLPattern
	service.URLPatterns()[23].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Update a dns rule
func TestPutTrafficRuleStateDeleteRule(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	service.dataPlane = &none.NoneDataPlane{}

	updateRule := dataplane.TrafficRule{
		TrafficRuleID: trafficRuleId,
		FilterType:    "FLOW",
		Priority:      5,
		TrafficFilter: []dataplane.TrafficFilter{},
		Action:        "DROP",
		State:         util.InactiveState,
	}
	updateRuleBytes, _ := json.Marshal(updateRule)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getOneTrafficRuleUrl, defaultAppInstanceId, trafficRuleId),
		bytes.NewReader(updateRuleBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndTrafficRuleIdQueryFormat, defaultAppInstanceId, trafficRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte(fmt.Sprintf(writeTrafficPutObjectFormat, util.InactiveState))).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		TrafficRule := dataplane.TrafficRule{TrafficRuleID: trafficRuleId, FilterType: "FLOW", Priority: 5,
			Action: "DROP", State: util.ActiveState}
		var TrafficRules []dataplane.TrafficRule
		TrafficRules = append(TrafficRules, TrafficRule)
		entry := models.AppDConfig{AppTrafficRule: TrafficRules}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patches.Reset()
	patches.ApplyFunc(service.dataPlane.AddTrafficRule, func(appInfo dataplane.ApplicationInfo, trafficRuleId, filterType, action string, priority int,
		filter []dataplane.TrafficFilter) (err error) {
		return errors.New("Error")
	})

	// 23 is the order of the Traffic Rule put handler in the URLPattern
	service.URLPatterns()[23].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Update a dns rule
func TestPutTrafficRuleStateSetRule(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	service.dataPlane = &none.NoneDataPlane{}

	updateRule := dataplane.TrafficRule{
		TrafficRuleID: trafficRuleId,
		FilterType:    "FLOW",
		Priority:      5,
		TrafficFilter: []dataplane.TrafficFilter{},
		Action:        "DROP",
		State:         util.ActiveState,
	}
	updateRuleBytes, _ := json.Marshal(updateRule)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getOneTrafficRuleUrl, defaultAppInstanceId, trafficRuleId),
		bytes.NewReader(updateRuleBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndTrafficRuleIdQueryFormat, defaultAppInstanceId, trafficRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte(fmt.Sprintf(writeTrafficPutObjectFormat, util.ActiveState))).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		TrafficRule := dataplane.TrafficRule{TrafficRuleID: trafficRuleId, FilterType: "FLOW", Priority: 5,
			Action: "DROP", State: util.ActiveState}
		var TrafficRules []dataplane.TrafficRule
		TrafficRules = append(TrafficRules, TrafficRule)
		entry := models.AppDConfig{AppTrafficRule: TrafficRules}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patches.Reset()
	patches.ApplyFunc(service.dataPlane.AddTrafficRule, func(appInfo dataplane.ApplicationInfo, trafficRuleId, filterType, action string, priority int,
		filter []dataplane.TrafficFilter) (err error) {
		return errors.New("Error")
	})

	// 23 is the order of the Traffic Rule put handler in the URLPattern
	service.URLPatterns()[23].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Update a dns rule
func TestPutTrafficRuleStatePutRecordFailed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	service.dataPlane = &none.NoneDataPlane{}

	updateRule := dataplane.TrafficRule{
		TrafficRuleID: trafficRuleId,
		FilterType:    "FLOW",
		Priority:      5,
		TrafficFilter: []dataplane.TrafficFilter{},
		Action:        "DROP",
		State:         util.ActiveState,
	}
	updateRuleBytes, _ := json.Marshal(updateRule)

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getOneTrafficRuleUrl, defaultAppInstanceId, trafficRuleId),
		bytes.NewReader(updateRuleBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndTrafficRuleIdQueryFormat, defaultAppInstanceId, trafficRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"title\":\"Bad Request\",\"status\":6,\"detail\":\"put traffic rule to etcd failed\"}"+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		TrafficRule := dataplane.TrafficRule{TrafficRuleID: trafficRuleId, FilterType: "FLOW", Priority: 5,
			Action: "DROP", State: util.ActiveState}
		var TrafficRules []dataplane.TrafficRule
		TrafficRules = append(TrafficRules, TrafficRule)
		entry := models.AppDConfig{AppTrafficRule: TrafficRules}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.PutRecord, func(path string, value []byte) int {
		return 1
	})

	// 23 is the order of the Traffic Rule put handler in the URLPattern
	service.URLPatterns()[23].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Test ServerAuthen discover
func TestMp1CvtSrvAuthenDiscover(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	patches := gomonkey.ApplyFunc(util.SendRequest, func(url string, method string, jsonStr []byte, tlsCfg *tls.Config) (string, error) {
		reqServ := make([]string, 0)
		reqServ = append(reqServ, "service1", "service2")
		bytes, err := json.Marshal(&reqServ)
		log.Infof("err %v", err)
		authInfo := util.AuthInfoRecord{}
		authInfo.Ak = "ak"
		authInfo.Sk = "sk"
		authInfo.AppInsId = defaultAppInstanceId
		authInfo.Nonce = "nonce"
		authInfo.RequiredServices = string(bytes)
		bytesOut, errOut := json.Marshal(&authInfo)
		return string(bytesOut), errOut
	})
	defer patches.Reset()
	response := pb.FindInstancesResponse{}
	response.Instances = make([]*pb.MicroServiceInstance, 0)
	response.Instances = append(response.Instances, &pb.MicroServiceInstance{
		InstanceId: defCapabilityId[len(defCapabilityId)/2:],
		ServiceId:  defCapabilityId[:len(defCapabilityId)/2],
		Status:     "UP",
		Version:    "3.2.1",
		Properties: map[string]string{
			"serName":     "FaceRegService6",
			svcCatHref:    svcCatResponse,
			svcCatId:      "id12345",
			svcCatName:    "RNI",
			svcCatVersion: "v1.1",
		},
	})

	resp, serviceInfosErr := Mp1CvtSrvAuthenDiscover(&response, defaultAppInstanceId)
	for _, serviceInfo := range serviceInfosErr {
		log.Infof("%v %v", resp.GetCode(), serviceInfo.SerName)
	}
	os.Setenv(util.EnvMepAuthPort, "3000")
	resp, serviceInfosErr2 := Mp1CvtSrvAuthenDiscover(&response, defaultAppInstanceId)
	for _, serviceInfo := range serviceInfosErr2 {
		log.Infof("%v %v", resp.GetCode(), serviceInfo.SerName)
	}
	os.Setenv(util.EnvMepAuthPort, "3000")
	os.Setenv(util.EnvMepAuthHost, exampleIPAddress)
	os.Setenv("MEPSERVER_CERT_DOMAIN_NAME", exampleDomainName)
	resp, serviceInfos := Mp1CvtSrvAuthenDiscover(&response, defaultAppInstanceId)
	for _, serviceInfo := range serviceInfos {
		log.Infof("%v %v", resp.GetCode(), serviceInfo.SerName)
	}
	assert.Equal(t, []*models.ServiceInfo([]*models.ServiceInfo{}), serviceInfos)
}

// Query dns rules request in mp1 interface
func TestGetTransportInfoGetRecordFailed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	var apihook APIGwHook
	SetAPIHook(apihook)
	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getTiming, getTransport),
		bytes.NewReader([]byte("")))

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":1,\"detail\":\"Get transport info failed\"}"+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		resultList := make(map[string][]byte)
		return resultList, 1
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[26].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query dns rules request in mp1 interface
func TestGetTransportInfoMarshalFailed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getTiming, getTransport),
		bytes.NewReader([]byte("")))

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":1,\"detail\":\"Get transport info failed\"}")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		resultList := make(map[string][]byte)
		return resultList, 0
	})

	patches.ApplyFunc(json.Marshal, func(v interface{}) ([]byte, error) {
		return []byte("test"), fmt.Errorf("error")
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[26].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)
}

// Update a dns rule
func TestPutTrafficRuleInvalidRestBody(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getOneTrafficRuleUrl, defaultAppInstanceId, trafficRuleId),
		bytes.NewReader([]byte("test")))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndTrafficRuleIdQueryFormat, defaultAppInstanceId, trafficRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"title\":\"Bad Request\",\"status\":1,\"detail\":\"invalid character 'e' in literal true (expecting 'r')\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	// 23 is the order of the Traffic Rule put handler in the URLPattern
	service.URLPatterns()[23].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Update a dns rule
func TestPutTrafficRuleReadAllFailed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	updateRule := dataplane.TrafficRule{
		TrafficRuleID: trafficRuleId,
		FilterType:    "FLOW",
		Priority:      5,
		TrafficFilter: []dataplane.TrafficFilter{},
		Action:        "DROP",
		State:         util.InactiveState,
	}
	updateRuleBytes, _ := json.Marshal(updateRule)
	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf(getOneTrafficRuleUrl, defaultAppInstanceId, trafficRuleId),
		bytes.NewReader(updateRuleBytes))
	getRequest.URL.RawQuery = fmt.Sprintf(appIdAndTrafficRuleIdQueryFormat, defaultAppInstanceId, trafficRuleId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"title\":\"Can not found resource\",\"status\":5,\"detail\":\"update rule retrieval failed\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 404)

	patches := gomonkey.ApplyFunc(ioutil.ReadAll, func(r io.Reader) ([]byte, error) {
		return []byte("8eb442b7cdfc11eba09314feb5b475da"), fmt.Errorf("error")
	})
	defer patches.Reset()
	// 23 is the order of the Traffic Rule put handler in the URLPattern
	service.URLPatterns()[23].Func(mockWriter, getRequest)

	assert.Equal(t, "404", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

//Query traffic rule gets in mp1 interface
func TestGetsTrafficRulesInvalidAppInstace(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getAllTrafficRuleUrl, ""),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, "")
	getRequest.Header.Set(appInstanceIdHeader, "")

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"UnAuthorization\",\"status\":11,\"detail\":\"invalid app instance id\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 401)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		trafficRule := dataplane.TrafficRule{TrafficRuleID: trafficRuleId, FilterType: "FLOW", Priority: 5,
			Action: "DROP", State: util.InactiveState}
		var trafficRules []dataplane.TrafficRule
		trafficRules = append(trafficRules, trafficRule)
		entry := models.AppDConfig{AppTrafficRule: trafficRules}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patches.Reset()

	// 21 is the order of the traffic get all handler in the URLPattern
	service.URLPatterns()[21].Func(mockWriter, getRequest)

	assert.Equal(t, "401", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)

}

// Check confirm ready app instances
func TestConfirmReady(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmReady := models.ConfirmReady{"READY"}
	body, _ := json.Marshal(confirmReady)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_ready, defaultAppInstanceId),
		bytes.NewReader(body))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("\"\""+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 204)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		resultList := make(map[string][]byte)
		resultList[util.AppDLCMJobsPath] = []byte(writeTransport)
		return nil, 0
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[27].Func(mockWriter, getRequest)

	assert.Equal(t, "204", responseHeader.Get(responseStatusHeader),
		responseCheckFor204)

	mockWriter.AssertExpectations(t)
}

// // Check confirm ready app instances
func TestConfirmReadyInvalidBody(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmReady := models.ConfirmReady{"NREADY"}
	body, _ := json.Marshal(confirmReady)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_ready, defaultAppInstanceId),
		bytes.NewReader(body))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":4,\"detail\":\"validate param failed\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		resultList := make(map[string][]byte)
		resultList[util.AppDLCMJobsPath] = []byte(writeTransport)
		return nil, 0
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[27].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor400)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmReadyOngoingExist(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmReady := models.ConfirmReady{"READY"}
	body, _ := json.Marshal(confirmReady)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_ready, defaultAppInstanceId),
		bytes.NewReader(body))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Service is in INACTIVE state\",\"status\":18,\"detail\":\"configuration sync for application is in progress on confirm ready\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 409)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()

	patches.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		resultList := make(map[string][]byte)
		resultList[util.AppDLCMJobsPath] = []byte(writeTransport)
		return resultList, 0
	})
	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[27].Func(mockWriter, getRequest)

	assert.Equal(t, "409", responseHeader.Get(responseStatusHeader),
		responseCheckFor400)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmIndiactionWrong(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	//confirmReady := models.ConfirmReady{"READY"}
	//body, _ := json.Marshal(confirmReady)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_ready, defaultAppInstanceId),
		bytes.NewReader([]byte("indicatio: READY")))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":4,\"detail\":\"validate param failed\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()

	patches.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		resultList := make(map[string][]byte)
		resultList[util.AppDLCMJobsPath] = []byte(writeTransport)
		return resultList, 0
	})
	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[27].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor400)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmReadBodyFailed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmReady := models.ConfirmReady{"READY"}
	body, _ := json.Marshal(confirmReady)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_ready, defaultAppInstanceId),
		bytes.NewReader([]byte(body)))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":1,\"detail\":\"read request body error\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()

	patches.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		resultList := make(map[string][]byte)
		resultList[util.AppDLCMJobsPath] = []byte(writeTransport)
		return resultList, 0
	})

	patches.ApplyFunc((*bytes.Buffer).ReadFrom, func(b *bytes.Buffer, r io.Reader) (n int64, err error) {
		log.Infof("reading from buffer")
		return 5000 * 8, fmt.Errorf("error")
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[27].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor400)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmReadUnMarshalErr(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmReady := models.ConfirmReady{"READY"}
	body, _ := json.Marshal(confirmReady)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_ready, defaultAppInstanceId),
		bytes.NewReader([]byte(body)))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":4,\"detail\":\"unmarshal request body error\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()

	patches.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		resultList := make(map[string][]byte)
		resultList[util.AppDLCMJobsPath] = []byte(writeTransport)
		return resultList, 0
	})
	count := 0
	patches.ApplyFunc(json.Unmarshal, func(data []byte, v interface{}) error {
		count++
		if count == 1 {
			v.(*models.ConfirmReady).Indication = "READY"
		} else {
			v = nil
			return fmt.Errorf("invalid")
		}
		return nil
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[27].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor400)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmReadyRestBodyNil(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmReady := models.ConfirmReady{"READY"}
	body, _ := json.Marshal(confirmReady)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_ready, defaultAppInstanceId),
		bytes.NewReader(body))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	req := plans.DecodeConfirmReadyReq{}
	r := &http.Request{}
	req.RestBody = nil
	req.ParseBody(r)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":4,\"detail\":\"unmarshal request body error\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()

	patches.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		resultList := make(map[string][]byte)
		resultList[util.AppDLCMJobsPath] = []byte(writeTransport)
		return resultList, 0
	})
	count := 0
	patches.ApplyFunc(json.Unmarshal, func(data []byte, v interface{}) error {
		count++
		if count == 1 {
			v.(*models.ConfirmReady).Indication = "READY"
		} else {
			v = nil
			return fmt.Errorf("invalid")
		}
		return nil
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[27].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor400)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmReadBodyLenExceed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmReady := models.ConfirmReady{"READY"}
	body, _ := json.Marshal(confirmReady)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_ready, defaultAppInstanceId),
		bytes.NewReader([]byte(body)))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Request parameter error\",\"status\":14,\"detail\":\"request body too large\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()

	patches.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		resultList := make(map[string][]byte)
		resultList[util.AppDLCMJobsPath] = []byte(writeTransport)
		return nil, 0
	})

	patches.ApplyFunc((*bytes.Buffer).ReadFrom, func(b *bytes.Buffer, r io.Reader) (n int64, err error) {
		log.Infof("reading from buffer")
		b.WriteByte('A')
		out := make([]byte, 5000*8)
		b.WriteString(string(out))
		return 5000 * 8, nil
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[27].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor400)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmReadyInvalidAppInstaces(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmReady := models.ConfirmReady{"READY"}
	body, _ := json.Marshal(confirmReady)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_ready, defaultAppInstanceId),
		bytes.NewReader(body))
	//getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"UnAuthorization\",\"status\":11,\"detail\":\"invalid app instance id\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 401)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		resultList := make(map[string][]byte)
		resultList[util.AppDLCMJobsPath] = []byte(writeTransport)
		return nil, 0
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[27].Func(mockWriter, getRequest)

	assert.Equal(t, "401", responseHeader.Get(responseStatusHeader),
		responseCheckFor204)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmTermination(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmTermination := models.ConfirmTermination{OperationAction: "TERMINATING"}
	body, _ := json.Marshal(confirmTermination)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_termination, defaultAppInstanceId),
		bytes.NewReader(body))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("\"\""+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 204)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		rec := models.ConfirmTerminationRecord{}
		rec.TerminationStatus = util.TerminationInProgress
		rec.OperationAction = util.TERMINATING
		bytes, _ := json.Marshal(rec)
		return bytes, 0
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[28].Func(mockWriter, getRequest)

	assert.Equal(t, "204", responseHeader.Get(responseStatusHeader),
		responseCheckFor204)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmTerminationParamErr(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmTermination := models.ConfirmTermination{OperationAction: "TERMINATING"}
	body, _ := json.Marshal(confirmTermination)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_termination, ""),
		bytes.NewReader(body))
	//getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	//getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"UnAuthorization\",\"status\":11,\"detail\":\"invalid app instance id\"}"+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 401)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		rec := models.ConfirmTerminationRecord{}
		rec.TerminationStatus = util.TerminationInProgress
		rec.OperationAction = util.TERMINATING
		bytes, _ := json.Marshal(rec)
		return bytes, 0
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[28].Func(mockWriter, getRequest)

	assert.Equal(t, "401", responseHeader.Get(responseStatusHeader),
		responseCheckFor204)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmTerminationParseErr(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmTermination := models.ConfirmTermination{OperationAction: "TERMINATING"}
	body, _ := json.Marshal(confirmTermination)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_termination, defaultAppInstanceId),
		bytes.NewReader(body))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":1,\"detail\":\"read request body error\"}"+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		rec := models.ConfirmTerminationRecord{}
		rec.TerminationStatus = util.TerminationInProgress
		rec.OperationAction = util.TERMINATING
		bytes, _ := json.Marshal(rec)
		return bytes, 0
	})

	patches.ApplyFunc((*bytes.Buffer).ReadFrom, func(b *bytes.Buffer, r io.Reader) (n int64, err error) {
		log.Infof("reading from buffer")
		return 0, fmt.Errorf("mock error")
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[28].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor204)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmTerminationParseInvalidLen(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmTermination := models.ConfirmTermination{OperationAction: "TERMINATING"}
	body, _ := json.Marshal(confirmTermination)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_termination, defaultAppInstanceId),
		bytes.NewReader(body))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Request parameter error\",\"status\":14,\"detail\":\"request body too large\"}"+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		rec := models.ConfirmTerminationRecord{}
		rec.TerminationStatus = util.TerminationInProgress
		rec.OperationAction = util.TERMINATING
		bytes, _ := json.Marshal(rec)
		return bytes, 0
	})

	patches.ApplyFunc((*bytes.Buffer).ReadFrom, func(b *bytes.Buffer, r io.Reader) (n int64, err error) {
		log.Infof("reading from buffer")
		b.WriteByte('A')
		out := make([]byte, 5000*8)
		b.WriteString(string(out))
		return 5000 * 8, nil
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[28].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor204)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmTerminationValidateFailed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmTermination := models.ConfirmTermination{OperationAction: "TERMINATewewe"}
	body, _ := json.Marshal(confirmTermination)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_termination, defaultAppInstanceId),
		bytes.NewReader(body))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Request parameter error\",\"status\":14,\"detail\":\"operation action is not matching\"}"+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		rec := models.ConfirmTerminationRecord{}
		rec.TerminationStatus = util.TerminationInProgress
		rec.OperationAction = util.TERMINATING
		bytes, _ := json.Marshal(rec)
		return bytes, 0
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[28].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor204)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmTerminationValidateUnMarshalFailed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_termination, defaultAppInstanceId),
		bytes.NewReader([]byte("test")))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("\"\""+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 204)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		rec := models.ConfirmTerminationRecord{}
		rec.TerminationStatus = util.TerminationInProgress
		rec.OperationAction = util.TERMINATING
		bytes, _ := json.Marshal(rec)
		return bytes, 0
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[28].Func(mockWriter, getRequest)

	assert.Equal(t, "204", responseHeader.Get(responseStatusHeader),
		responseCheckFor204)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmTerminationIsNotInProgressErr(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmTermination := models.ConfirmTermination{OperationAction: "TERMINATING"}
	body, _ := json.Marshal(confirmTermination)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_termination, defaultAppInstanceId),
		bytes.NewReader(body))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Service is in INACTIVE state\",\"status\":18,\"detail\":\"no termination is going on\"}"+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 409)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		rec := models.ConfirmTerminationRecord{}
		rec.TerminationStatus = util.TerminationInProgress
		rec.OperationAction = util.TERMINATING
		bytes, _ := json.Marshal(rec)
		return bytes, -1
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[28].Func(mockWriter, getRequest)

	assert.Equal(t, "409", responseHeader.Get(responseStatusHeader),
		responseCheckFor204)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmTerminationIsNotInProgressReErr(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmTermination := models.ConfirmTermination{OperationAction: "TERMINATING"}
	body, _ := json.Marshal(confirmTermination)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_termination, defaultAppInstanceId),
		bytes.NewReader(body))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Request parameter error\",\"status\":14,\"detail\":\"subscription parsed failed\"}"+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		return []byte("test"), 0
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[28].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor204)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmTerminationRestBodyNil(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmTermination := models.ConfirmTermination{OperationAction: "TERMINATING"}
	body, _ := json.Marshal(confirmTermination)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_termination, defaultAppInstanceId),
		bytes.NewReader(body))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":4,\"detail\":\"unmarshal request body error\"}"+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		rec := models.ConfirmTerminationRecord{}
		rec.TerminationStatus = util.TerminationInProgress
		rec.OperationAction = util.TERMINATING
		bytes, _ := json.Marshal(rec)
		return bytes, 0
	})
	count := 0
	patches.ApplyFunc(json.Unmarshal, func(data []byte, v interface{}) error {
		count++
		if count == 1 || count == 2 {
			rec := models.ConfirmTermination{}
			rec.OperationAction = util.TERMINATING
			v = &rec
		} else {
			v = nil
			return fmt.Errorf("invalid")
		}

		return nil
	})
	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[28].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor204)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmTerminationGetRecordErr(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmTermination := models.ConfirmTermination{OperationAction: "TERMINATING"}
	body, _ := json.Marshal(confirmTermination)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_termination, defaultAppInstanceId),
		bytes.NewReader(body))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Service is in INACTIVE state\",\"status\":18,\"detail\":\"no termination is going on\"}"+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 409)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	count := 0
	patches.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		count++
		if count == 1 {
			rec := models.ConfirmTerminationRecord{}
			rec.TerminationStatus = util.TerminationInProgress
			rec.OperationAction = util.TERMINATING
			bytes, _ := json.Marshal(rec)
			return bytes, 0
		} else {
			return nil, 1
		}
	})

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[28].Func(mockWriter, getRequest)

	assert.Equal(t, "409", responseHeader.Get(responseStatusHeader),
		responseCheckFor204)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmTerminationRecordUnMarshalErr(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmTermination := models.ConfirmTermination{OperationAction: "TERMINATING"}
	body, _ := json.Marshal(confirmTermination)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_termination, defaultAppInstanceId),
		bytes.NewReader(body))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":4,\"detail\":\"unmarshal request body error\"}"+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 417)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		rec := models.ConfirmTerminationRecord{}
		rec.TerminationStatus = util.TerminationInProgress
		rec.OperationAction = util.TERMINATING
		bytes, _ := json.Marshal(rec)
		return bytes, 0
	})
	count := 0
	patches.ApplyFunc(json.Unmarshal, func(data []byte, v interface{}) error {
		count++
		if count == 1 || count == 2 || count == 3 {
			rec := models.ConfirmTermination{}
			rec.OperationAction = util.TERMINATING
			v = &rec
		} else {
			v = nil
			return fmt.Errorf("invalid")
		}

		return nil
	})
	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[28].Func(mockWriter, getRequest)

	assert.Equal(t, "417", responseHeader.Get(responseStatusHeader),
		responseCheckFor204)
}

// Check confirm ready app instances
func TestConfirmTerminationPutRecERr(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmTermination := models.ConfirmTermination{OperationAction: "TERMINATING"}
	body, _ := json.Marshal(confirmTermination)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_termination, defaultAppInstanceId),
		bytes.NewReader(body))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":6,\"detail\":\"put record failed for confirm termination\"}"+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		rec := models.ConfirmTerminationRecord{}
		rec.TerminationStatus = util.TerminationInProgress
		rec.OperationAction = util.TERMINATING
		bytes, _ := json.Marshal(rec)
		return bytes, 0
	})
	patches.ApplyFunc(backend.PutRecord, func(path string, value []byte) int {
		return 1
	})
	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[28].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor204)

	mockWriter.AssertExpectations(t)
}

// Check confirm ready app instances
func TestConfirmTerminationMarshalErr(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	confirmTermination := models.ConfirmTermination{OperationAction: "TERMINATING"}
	body, _ := json.Marshal(confirmTermination)
	// Create http get request
	getRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(confirm_termination, defaultAppInstanceId),
		bytes.NewReader(body))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	req := plans.DecodeConfirmTerminateReq{}
	r := http.Request{}
	req.RestBody = nil
	req.ParseBody(&r)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":4,\"detail\":\"unmarshal request body error\"}"+"\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 417)

	patches := gomonkey.ApplyFunc(baseutil.GenerateUuid, func() string {
		return "8eb442b7cdfc11eba09314feb5b475da"
	})
	defer patches.Reset()
	patches.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		rec := models.ConfirmTerminationRecord{}
		rec.TerminationStatus = util.TerminationInProgress
		rec.OperationAction = util.TERMINATING
		bytes := []byte("{\"operationAction\":\"TERMINATING\",\"TerminationStatus\":1}")
		return bytes, 0
	})
	count := 0
	patches.ApplyFunc(json.Marshal, func(v interface{}) ([]byte, error) {
		count++
		return []byte(""), fmt.Errorf("invalid")
	})
	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[28].Func(mockWriter, getRequest)

	assert.Equal(t, "417", responseHeader.Get(responseStatusHeader),
		responseCheckFor204)
}

//Query traffic rule gets in mp1 interface
func TestGetsTrafficRulesAppInstIdNil(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getAllTrafficRuleUrl, ""),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("[]\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		trafficRule := dataplane.TrafficRule{TrafficRuleID: trafficRuleId, FilterType: "FLOW", Priority: 5,
			Action: "DROP", State: util.InactiveState}
		var trafficRules []dataplane.TrafficRule
		trafficRules = append(trafficRules, trafficRule)
		entry := models.AppDConfig{AppTrafficRule: trafficRules}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, util.SubscriptionNotFound
	})
	defer patches.Reset()

	// 21 is the order of the traffic get all handler in the URLPattern
	service.URLPatterns()[21].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

//Query traffic rule gets in mp1 interface
func TestGetsTrafficRulesAppInstIdErr(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getAllTrafficRuleUrl, ""),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":6,\"detail\":\"traffic rules not found\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		trafficRule := dataplane.TrafficRule{TrafficRuleID: trafficRuleId, FilterType: "FLOW", Priority: 5,
			Action: "DROP", State: util.InactiveState}
		var trafficRules []dataplane.TrafficRule
		trafficRules = append(trafficRules, trafficRule)
		entry := models.AppDConfig{AppTrafficRule: trafficRules}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, util.SubscriptionNotFound + 1
	})
	defer patches.Reset()

	// 21 is the order of the traffic get all handler in the URLPattern
	service.URLPatterns()[21].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

//Query traffic rule gets in mp1 interface
func TestGetsTrafficRulesTrafficRuleDbNil(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getAllTrafficRuleUrl, ""),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Can not found resource\",\"status\":3,\"detail\":\"traffic rules not found\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 404)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		return nil, 0
	})
	defer patches.Reset()

	// 21 is the order of the traffic get all handler in the URLPattern
	service.URLPatterns()[21].Func(mockWriter, getRequest)

	assert.Equal(t, "404", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

//Query traffic rule gets in mp1 interface
func TestGetsTrafficRulesUnmarshalErr(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getAllTrafficRuleUrl, ""),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Bad Request\",\"status\":6,\"detail\":\"parse dns rules from data-store failed\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		trafficRule := dataplane.TrafficRule{TrafficRuleID: trafficRuleId, FilterType: "FLOW", Priority: 5,
			Action: "DROP", State: util.InactiveState}
		var trafficRules []dataplane.TrafficRule
		trafficRules = append(trafficRules, trafficRule)
		entry := models.AppDConfig{AppTrafficRule: trafficRules}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patches.Reset()

	patches.ApplyFunc(json.Unmarshal, func(data []byte, v interface{}) error {
		return fmt.Errorf("invalid")
	})

	// 21 is the order of the traffic get all handler in the URLPattern
	service.URLPatterns()[21].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Post App service availability Notification
func TestAppSubscribeInsertDbError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	createSubscription := models.AppTerminationNotificationSubscription{
		SubscriptionType:  "AppTerminationNotificationSubscription",
		CallbackReference: callBackRef,
	}
	createSubscriptionBytes, _ := json.Marshal(createSubscription)
	var resp = &pb.GetOneInstanceResponse{
		Response: &pb.Response{Code: pb.Response_SUCCESS},
	}
	n := &srv.InstanceService{}
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *pb.GetOneInstanceRequest) (*pb.GetOneInstanceResponse, error) {
		return resp, nil
	})
	defer patch1.Reset()
	ec := &buildin.BuildinRegistry{}
	var response registry.PluginResponse
	patch1.ApplyMethod(reflect.TypeOf(ec), "TxnWithCmp", func(t *buildin.BuildinRegistry, ctx context.Context, success []registry.PluginOp, cmp []registry.CompareOp, fail []registry.PluginOp) (*registry.PluginResponse, error) {
		if success[0].Action == 1 {
			return nil, fmt.Errorf("Error")
		}
		return &registry.PluginResponse{
			Succeeded: true,
			Revision:  1,
			Kvs:       response.Kvs,
			Count:     response.Count,
		}, nil
	})
	// Create http get request
	postRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(postSubscribeUrl, defaultAppInstanceId),
		bytes.NewReader(createSubscriptionBytes))
	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	service.URLPatterns()[0].Func(mockWriter, postRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor201)
	mockWriter.AssertExpectations(t)
}

// Post App service availability Notification and json marshalling failed
func TestAppSubscribePostJsonMarshallFailDbErr(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	createSubscription := models.SerAvailabilityNotificationSubscription{
		SubscriptionType:  "SerAvailabilityNotificationSubscription",
		CallbackReference: callBackRef,
		FilteringCriteria: models.FilteringCriteria{
			SerInstanceIds: []string{
				"f7e898d1c9ea9edd8a41295fc55c2373",
			},
			SerNames: []string{
				"FaceRegService5",
			},
			SerCategories: []models.CategoryRef{
				{
					Href:    callBackRef,
					ID:      "id12345",
					Name:    "RNI",
					Version: "1.2.2",
				},
			},
			States: []string{
				"ACTIVE",
			},
			IsLocal: true,
		},
	}
	createSubscriptionBytes, _ := json.Marshal(createSubscription)
	var resp = &pb.GetOneInstanceResponse{
		Response: &pb.Response{Code: pb.Response_SUCCESS},
	}
	n := &srv.InstanceService{}
	patches := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *pb.GetOneInstanceRequest) (*pb.GetOneInstanceResponse, error) {
		return resp, nil
	})
	defer patches.Reset()
	var counter = 0
	patches.ApplyFunc(json.Marshal, func(i interface{}) (b []byte, e error) {
		counter++
		if counter == 3 {
			return nil, errors.New("json marshalling failed")
		} else {
			bs := new(bytes.Buffer)
			_ = json.NewEncoder(bs).Encode(i)
			return bs.Bytes(), nil
		}
	})
	ec := &buildin.BuildinRegistry{}
	var response registry.PluginResponse
	patches.ApplyMethod(reflect.TypeOf(ec), "TxnWithCmp", func(t *buildin.BuildinRegistry, ctx context.Context, success []registry.PluginOp, cmp []registry.CompareOp, fail []registry.PluginOp) (*registry.PluginResponse, error) {
		if success[0].Action == 2 {
			return nil, fmt.Errorf("Error")
		}
		return &registry.PluginResponse{
			Succeeded: true,
			Revision:  1,
			Kvs:       response.Kvs,
			Count:     response.Count,
		}, nil
	})
	// Create http get request
	postRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(postSubscribeUrl, defaultAppInstanceId),
		bytes.NewReader(createSubscriptionBytes))
	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 400)
	service.URLPatterns()[0].Func(mockWriter, postRequest)

	respError := models.ProblemDetails{}

	_ = json.Unmarshal(mockWriter.response, &respError)
	log.Info(respError.String())
	assert.Equal(t, "Bad Request", respError.Title, "Expected error not returned")

}

// Post App service availability Notification and json marshalling failed
func TestAppSubscribePostJsonMarshallFailValidationFail(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mp1Service{}
	createSubscription := models.SerAvailabilityNotificationSubscription{
		SubscriptionType:  "SerAvailabilityNotificationSubscription",
		CallbackReference: callBackRef,
		FilteringCriteria: models.FilteringCriteria{
			SerInstanceIds: []string{
				"f7e898d1c9ea9edd8a41295fc55c2373",
			},
			SerNames: []string{
				"FaceRegService5",
			},
			SerCategories: []models.CategoryRef{
				{
					Href:    callBackRef,
					ID:      "id12345",
					Name:    "RNI",
					Version: "1.2.2",
				},
			},
			States: []string{
				"ACTIVE",
			},
			IsLocal: true,
		},
	}
	createSubscriptionBytes, _ := json.Marshal(createSubscription)
	var resp = &pb.GetOneInstanceResponse{
		Response: &pb.Response{Code: pb.Response_SUCCESS},
	}
	n := &srv.InstanceService{}
	patches := gomonkey.ApplyMethod(reflect.TypeOf(n), "GetOneInstance", func(*srv.InstanceService, context.Context, *pb.GetOneInstanceRequest) (*pb.GetOneInstanceResponse, error) {
		return resp, nil
	})
	defer patches.Reset()
	var counter = 0
	patches.ApplyFunc(json.Marshal, func(i interface{}) (b []byte, e error) {
		counter++
		if counter == 3 {
			return nil, errors.New("json marshalling failed")
		} else {
			bs := new(bytes.Buffer)
			_ = json.NewEncoder(bs).Encode(i)
			return bs.Bytes(), nil
		}
	})
	count := 0
	patches.ApplyFunc(url.ParseRequestURI, func(rawurl string) (*url.URL, error) {
		count++
		if count == 2 {
			return nil, fmt.Errorf("error")
		}
		return nil, nil
	})

	// Create http get request
	postRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(postSubscribeUrl, defaultAppInstanceId),
		bytes.NewReader(createSubscriptionBytes))
	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 400)
	service.URLPatterns()[0].Func(mockWriter, postRequest)

	respError := models.ProblemDetails{}

	_ = json.Unmarshal(mockWriter.response, &respError)
	log.Info(respError.String())
	assert.Equal(t, "", respError.Title, "Expected error not returned")
}
