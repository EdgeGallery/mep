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
)

type mockHttpWriter struct {
	mock.Mock
	response []byte
}

const DefaultAppInstanceId = "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"
const DnsRuleId = "7d71e54e-81f3-47bb-a2fc-b565a326d794"

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

// Query dns rules request in mp1 interface
func TestGetDnsRules(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf("/mepcfg/mec_app_config/v1/rules/%s/dns_rules", DefaultAppInstanceId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(":appInstanceId=%s&;", DefaultAppInstanceId)
	getRequest.Header.Set("X-AppinstanceID", DefaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("[{\"dnsRuleId\":\"7d71e54e-81f3-47bb-a2fc-b565a326d794\","+
		"\"domainName\":\"www.example.com\",\"ipAddressType\":\"IP_V4\",\"ipAddress\":\"179.138.147.240\","+
		"\"ttl\":30,\"state\":\"INACTIVE\"}]\n")).Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := dns.RuleEntry{DomainName: "www.example.com", IpAddressType: "IP_V4", IpAddress: "179.138.147.240",
			TTL: 30, State: "INACTIVE"}
		outBytes, _ := json.Marshal(&entry)
		records[DnsRuleId] = outBytes
		return records, 0
	})
	defer patches.Reset()

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[13].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get("X-Response-Status"),
		"Response status code must be 200")

	mockWriter.AssertExpectations(t)

}

// Query an empty dns rules request in mp1 interface
func TestGetEmptyDnsRules(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf("/mepcfg/mec_app_config/v1/rules/%s/dns_rules", DefaultAppInstanceId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(":appInstanceId=%s&;", DefaultAppInstanceId)
	getRequest.Header.Set("X-AppinstanceID", DefaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("null\n")).Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	// 13 is the order of the DNS get all handler in the URLPattern
	service.URLPatterns()[13].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get("X-Response-Status"),
		"Response status code must be 200")

	mockWriter.AssertExpectations(t)

}

// Query empty dns rules with unmatched application instance id
func TestGetEmptyDnsRulesAppInstanceIdUnMatched(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf("/mepcfg/mec_app_config/v1/rules/%s/dns_rules", DefaultAppInstanceId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(":appInstanceId=%s&;", DefaultAppInstanceId)
	getRequest.Header.Set("X-AppinstanceID", "wrong-app-instance-id")

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

	assert.Equal(t, "401", responseHeader.Get("X-Response-Status"),
		"Response status code must be 401 Unauthorized")

	mockWriter.AssertExpectations(t)
}

// Query empty dns rules with invalid application instance id
func TestGetEmptyDnsRulesAppInstanceIdInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	invalidAppInstanceId := "invalid-app-instance-id"

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf("/mepcfg/mec_app_config/v1/rules/%s/dns_rules", invalidAppInstanceId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(":appInstanceId=%s&;", invalidAppInstanceId)
	getRequest.Header.Set("X-AppinstanceID", invalidAppInstanceId)

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

	assert.Equal(t, "400", responseHeader.Get("X-Response-Status"),
		"Response status code must be 400 Unauthorized")

	mockWriter.AssertExpectations(t)

}

// Query single dns rule request in mp1 interface
func TestGetSingleDnsRule(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf("/mepcfg/mec_app_config/v1/rules/%s/dns_rules/%s", DefaultAppInstanceId, DnsRuleId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(":appInstanceId=%s&;:dnsRuleId=%s&;", DefaultAppInstanceId, DnsRuleId)
	getRequest.Header.Set("X-AppinstanceID", DefaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"dnsRuleId\":\"7d71e54e-81f3-47bb-a2fc-b565a326d794\","+
		"\"domainName\":\"www.example.com\",\"ipAddressType\":\"IP_V4\",\"ipAddress\":\"179.138.147.240\","+
		"\"ttl\":30,\"state\":\"INACTIVE\"}\n")).Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		entry := dns.RuleEntry{DomainName: "www.example.com", IpAddressType: "IP_V4", IpAddress: "179.138.147.240",
			TTL: 30, State: "INACTIVE"}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patches.Reset()

	// 14 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[14].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get("X-Response-Status"),
		"Response status code must be 200")

	mockWriter.AssertExpectations(t)

}

// Query single rule
func TestGetSingleDnsRuleNotFound(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf("/mepcfg/mec_app_config/v1/rules/%s/dns_rules/%s", DefaultAppInstanceId, DnsRuleId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(":appInstanceId=%s&;:dnsRuleId=%s&;", DefaultAppInstanceId, DnsRuleId)
	getRequest.Header.Set("X-AppinstanceID", DefaultAppInstanceId)

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

	assert.Equal(t, "404", responseHeader.Get("X-Response-Status"),
		"Response status code must be 404")

	mockWriter.AssertExpectations(t)
}

// Query single rule with empty rule id
func TestGetSingleDnsRuleNoId(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf("/mepcfg/mec_app_config/v1/rules/%s/dns_rules/%s", DefaultAppInstanceId, ""),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(":appInstanceId=%s&;:dnsRuleId=%s&;", DefaultAppInstanceId, "")
	getRequest.Header.Set("X-AppinstanceID", "")

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

	assert.Equal(t, "401", responseHeader.Get("X-Response-Status"),
		"Response status code must be 404")

	mockWriter.AssertExpectations(t)
}

// Update a dns rule
func TestPutSingleDnsRule(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf("/mepcfg/mec_app_config/v1/rules/%s/dns_rules/%s", DefaultAppInstanceId, DnsRuleId),
		bytes.NewReader([]byte("{\"domainName\": \"www.example.com\",\"ipAddressType\": \"IP_V4\",\"ipAddress\":"+
			" \"179.138.147.240\",\"ttl\": 30,\"state\": \"INACTIVE\"}")))
	getRequest.URL.RawQuery = fmt.Sprintf(":appInstanceId=%s&;:dnsRuleId=%s&;", DefaultAppInstanceId, DnsRuleId)
	getRequest.Header.Set("X-AppinstanceID", DefaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"domainName\":\"www.example.com\",\"ipAddressType\":\"IP_V4\",\"ipAddress\":\"179.138.147.240\","+
			"\"ttl\":30,\"state\":\"INACTIVE\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		entry := dns.RuleEntry{DomainName: "www.example.com", IpAddressType: "IP_V4", IpAddress: "179.138.147.240",
			TTL: 30, State: "INACTIVE"}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patches.Reset()

	// 15 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[15].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get("X-Response-Status"),
		"Response status code must be 200")

	mockWriter.AssertExpectations(t)
}

// Update a dns rule from INACTIVE to ACTIVE
func TestPutSingleDnsRuleActive(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf("/mepcfg/mec_app_config/v1/rules/%s/dns_rules/%s", DefaultAppInstanceId, DnsRuleId),
		bytes.NewReader([]byte("{\"domainName\": \"www.example.com\",\"ipAddressType\": \"IP_V4\",\"ipAddress\":"+
			" \"179.138.147.240\",\"ttl\": 30,\"state\": \"ACTIVE\"}")))
	getRequest.URL.RawQuery = fmt.Sprintf(":appInstanceId=%s&;:dnsRuleId=%s&;", DefaultAppInstanceId, DnsRuleId)
	getRequest.Header.Set("X-AppinstanceID", DefaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"dnsRuleId\":\"7d71e54e-81f3-47bb-a2fc-b565a326d794\",\"domainName\":\"www.example.com\","+
			"\"ipAddressType\":\"IP_V4\",\"ipAddress\":\"179.138.147.240\",\"ttl\":30,\"state\":\"ACTIVE\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err2 := w.Write([]byte(""))
		if err2 != nil {
			t.Error("Write Response Error")
		}
	}))
	defer ts.Close()
	patch1 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		entry := dns.RuleEntry{DomainName: "www.example.com", IpAddressType: "IP_V4", IpAddress: "179.138.147.240",
			TTL: 30, State: "INACTIVE"}
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

	assert.Equal(t, "200", responseHeader.Get("X-Response-Status"),
		"Response status code must be 200")

	mockWriter.AssertExpectations(t)
}

// Update a dns rule from ACTIVE to INACTIVE
func TestPutSingleDnsRuleInactive(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf("/mepcfg/mec_app_config/v1/rules/%s/dns_rules/%s", DefaultAppInstanceId, DnsRuleId),
		bytes.NewReader([]byte("{\"domainName\": \"www.example.com\",\"ipAddressType\": \"IP_V4\",\"ipAddress\":"+
			" \"179.138.147.240\",\"ttl\": 30,\"state\": \"INACTIVE\"}")))
	getRequest.URL.RawQuery = fmt.Sprintf(":appInstanceId=%s&;:dnsRuleId=%s&;", DefaultAppInstanceId, DnsRuleId)
	getRequest.Header.Set("X-AppinstanceID", DefaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"dnsRuleId\":\"7d71e54e-81f3-47bb-a2fc-b565a326d794\",\"domainName\":\"www.example.com\","+
			"\"ipAddressType\":\"IP_V4\",\"ipAddress\":\"179.138.147.240\",\"ttl\":30,\"state\":\"INACTIVE\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err2 := w.Write([]byte(""))
		if err2 != nil {
			t.Error("Write Response Error")
		}
	}))
	defer ts.Close()
	patch1 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		entry := dns.RuleEntry{DomainName: "www.example.com", IpAddressType: "IP_V4", IpAddress: "179.138.147.240",
			TTL: 30, State: "ACTIVE"}
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

	assert.Equal(t, "200", responseHeader.Get("X-Response-Status"),
		"Response status code must be 200")

	mockWriter.AssertExpectations(t)
}

// Update a dns rule from INACTIVE to ACTIVE when the server is not reachable
func TestPutSingleDnsRuleActiveWithServerNotReachable(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf("/mepcfg/mec_app_config/v1/rules/%s/dns_rules/%s", DefaultAppInstanceId, DnsRuleId),
		bytes.NewReader([]byte("{\"domainName\": \"www.example.com\",\"ipAddressType\": \"IP_V4\",\"ipAddress\":"+
			" \"179.138.147.240\",\"ttl\": 30,\"state\": \"ACTIVE\"}")))
	getRequest.URL.RawQuery = fmt.Sprintf(":appInstanceId=%s&;:dnsRuleId=%s&;", DefaultAppInstanceId, DnsRuleId)
	getRequest.Header.Set("X-AppinstanceID", DefaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"title\":\"Remote server error\",\"status\":9,\"detail\":\"failed to apply the dns modification\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 503)

	patch1 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		entry := dns.RuleEntry{DomainName: "www.example.com", IpAddressType: "IP_V4", IpAddress: "179.138.147.240",
			TTL: 30, State: "INACTIVE"}
		outBytes, _ := json.Marshal(&entry)
		return outBytes, 0
	})
	defer patch1.Reset()

	// 15 is the order of the DNS put handler in the URLPattern
	service.URLPatterns()[15].Func(mockWriter, getRequest)

	assert.Equal(t, "503", responseHeader.Get("X-Response-Status"),
		"Response status code miss-match")

	mockWriter.AssertExpectations(t)
}

// Update a dns rule from INACTIVE to ACTIVE with error in the dns server
func TestPutSingleDnsRuleActiveWithServerError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf("/mepcfg/mec_app_config/v1/rules/%s/dns_rules/%s", DefaultAppInstanceId, DnsRuleId),
		bytes.NewReader([]byte("{\"domainName\": \"www.example.com\",\"ipAddressType\": \"IP_V4\",\"ipAddress\":"+
			" \"179.138.147.240\",\"ttl\": 30,\"state\": \"ACTIVE\"}")))
	getRequest.URL.RawQuery = fmt.Sprintf(":appInstanceId=%s&;:dnsRuleId=%s&;", DefaultAppInstanceId, DnsRuleId)
	getRequest.Header.Set("X-AppinstanceID", DefaultAppInstanceId)

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
			t.Error("Write Response Error")
		}
	}))
	defer ts.Close()
	patch1 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		entry := dns.RuleEntry{DomainName: "www.example.com", IpAddressType: "IP_V4", IpAddress: "179.138.147.240",
			TTL: 30, State: "INACTIVE"}
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

	assert.Equal(t, "503", responseHeader.Get("X-Response-Status"),
		"Response status code miss-match")

	mockWriter.AssertExpectations(t)
}

// Put a dns rule which doesn't exists
func TestPutSingleDnsRuleNotFound(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf("/mepcfg/mec_app_config/v1/rules/%s/dns_rules/%s", DefaultAppInstanceId, DnsRuleId),
		bytes.NewReader([]byte("{}")))
	getRequest.URL.RawQuery = fmt.Sprintf(":appInstanceId=%s&;:dnsRuleId=%s&;", DefaultAppInstanceId, DnsRuleId)
	getRequest.Header.Set("X-AppinstanceID", DefaultAppInstanceId)

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

	assert.Equal(t, "404", responseHeader.Get("X-Response-Status"),
		"Response status code must be 404")

	mockWriter.AssertExpectations(t)
}

// Put a dns rule with invalid body
func TestPutSingleDnsRuleBodyParseError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf("/mepcfg/mec_app_config/v1/rules/%s/dns_rules/%s", DefaultAppInstanceId, DnsRuleId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(":appInstanceId=%s&;:dnsRuleId=%s&;", DefaultAppInstanceId, DnsRuleId)
	getRequest.Header.Set("X-AppinstanceID", DefaultAppInstanceId)

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

	assert.Equal(t, "400", responseHeader.Get("X-Response-Status"),
		"Response status code must be 404")

	mockWriter.AssertExpectations(t)
}

// Put a dns rule with large body
func TestPutSingleDnsRuleOverLengthBody(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	messageBody := ""
	for i := 0; i <= 64; i++ {
		messageBody += "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz123456789011"
	}

	service := Mp1Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("PUT",
		fmt.Sprintf("/mepcfg/mec_app_config/v1/rules/%s/dns_rules/%s", DefaultAppInstanceId, DnsRuleId),
		bytes.NewReader([]byte(messageBody)))
	getRequest.URL.RawQuery = fmt.Sprintf(":appInstanceId=%s&;:dnsRuleId=%s&;", DefaultAppInstanceId, DnsRuleId)
	getRequest.Header.Set("X-AppinstanceID", DefaultAppInstanceId)

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

	assert.Equal(t, "400", responseHeader.Get("X-Response-Status"),
		"Response status code must be 400")

	mockWriter.AssertExpectations(t)
}
