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

package mm5

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"mepserver/common/models"
	"net/http"
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
	"mepserver/common/util"
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
				"serName":  "FaceRegService6",
				"mecState": "ACTIVE",
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
				"serName":  "FaceRegService6",
				"mecState": "ACTIVE",
			},
		})
		return &response, nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyFunc(backend.GetRecordsWithCompleteKeyPath, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "05ddef81-dd83-4a37-b0fe-85999585b929",
			FilteringCriteria: models.FilteringCriteria{},
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
				"serName":  "FaceRegService6",
				"mecState": "ACTIVE",
			},
		})
		return &response, nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyFunc(backend.GetRecordsWithCompleteKeyPath, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "05ddef81-dd83-4a37-b0fe-85999585b929",
			FilteringCriteria: models.FilteringCriteria{},
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
				"serName":  "FaceRegService6",
				"mecState": "ACTIVE",
			},
		})
		response.Instances = append(response.Instances, &proto.MicroServiceInstance{
			InstanceId: defCapabilityId2[len(defCapabilityId2)/2:],
			ServiceId:  defCapabilityId2[:len(defCapabilityId2)/2],
			Status:     "UP",
			Version:    "3.2.1",
			Properties: map[string]string{
				"serName":  "FaceRegService5",
				"mecState": "ACTIVE",
			},
		})
		return &response, nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyFunc(backend.GetRecordsWithCompleteKeyPath, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "05ddef81-dd83-4a37-b0fe-85999585b929",
			FilteringCriteria: models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerInstanceIds = append(entry.FilteringCriteria.SerInstanceIds, defCapabilityId)
		outBytes, _ := json.Marshal(&entry)
		records[fmt.Sprintf("csr/etcd/%s/%s", defaultAppInstanceId, subscriberId1)] = outBytes

		entry2 := models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "09022fec-a63c-49fc-857a-dcd7ecaa40a2",
			FilteringCriteria: models.FilteringCriteria{},
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
				"serName":  "FaceRegService6",
				"mecState": "ACTIVE",
			},
		})
		return &response, nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyFunc(backend.GetRecordsWithCompleteKeyPath, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "05ddef81-dd83-4a37-b0fe-85999585b929",
			FilteringCriteria: models.FilteringCriteria{},
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
				"mecState":            "ACTIVE",
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
		entry := models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "05ddef81-dd83-4a37-b0fe-85999585b929",
			FilteringCriteria: models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerCategories = append(entry.FilteringCriteria.SerCategories, models.CategoryRef{
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
				"serName":  "FaceRegService6",
				"mecState": "ACTIVE",
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
				"serName":  "FaceRegService6",
				"mecState": "ACTIVE",
			},
		}
		return &response, nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyFunc(backend.GetRecordsWithCompleteKeyPath, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "05ddef81-dd83-4a37-b0fe-85999585b929",
			FilteringCriteria: models.FilteringCriteria{},
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
				"serName":  "FaceRegService6",
				"mecState": "ACTIVE",
			},
		}
		return &response, nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyFunc(backend.GetRecordsWithCompleteKeyPath, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "05ddef81-dd83-4a37-b0fe-85999585b929",
			FilteringCriteria: models.FilteringCriteria{},
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
				"mecState":            "ACTIVE",
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
		entry := models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "05ddef81-dd83-4a37-b0fe-85999585b929",
			FilteringCriteria: models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerCategories = append(entry.FilteringCriteria.SerCategories, models.CategoryRef{
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
