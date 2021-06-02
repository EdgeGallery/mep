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

package mm5

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	uuid "github.com/satori/go.uuid"
	"math/rand"
	"mepserver/common/arch/workspace"
	"mepserver/common/config"
	"mepserver/common/extif/dns"
	"mepserver/common/models"
	"mepserver/mm5/task"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey"
	"github.com/apache/servicecomb-service-center/pkg/log"
	_ "github.com/apache/servicecomb-service-center/server"
	_ "github.com/apache/servicecomb-service-center/server/bootstrap"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	srv "github.com/apache/servicecomb-service-center/server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"mepserver/common/extif/backend"
	"mepserver/common/util"
	"mepserver/mm5/plans"
)

const defaultAppInstanceId = "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"

const panicFormatString = "Panic: %v"
const getTaskStatusFormat = "/mepcfg/app_lcm/v1/tasks/%s/appd_configuration"
const appConfigUrlFormat = "/mepcfg/app_lcm/v1/applications/%s/appd_configuration"
const delAppInstFormat = "/mep/mec_app_support/v1/applications/%s/AppInstanceTermination"

const defaultTaskId = "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"
const taskQueryFormat = ":taskId=%s&;"

const appInstanceQueryFormat = ":appInstanceId=%s&;"
const appInstanceIdHeader = "X-AppinstanceID"
const responseStatusHeader = "X-Response-Status"
const responseCheckFor200 = "Response status code must be 200"
const responseCheckFor400 = "Response status code must be 404"
const maxIPVal = 255
const ipAddFormatter = "%d.%d.%d.%d"

const getCapabilitiesUrl = "/mepcfg/mec_platform_config/v1/capabilities"

const defCapabilityId = "16384563dca094183778a41ea7701d15"
const defCapabilityId2 = "f7e898d1c9ea9edd05e1181bc09afc5e"
const subscriberId1 = "05ddef81-dd83-4a37-b0fe-85999585b929"
const subscriberId2 = "09022fec-a63c-49fc-857a-dcd7ecaa40a2"
const appInstanceId2 = "3abe4278-9c70-2e47-3a4e-7ee3a1a0fd1e"

const capabilityQueryFormat = ":capabilityId=%s&;"

const recordDB = "csr/etcd/%s/%s"
const svcCatHref = "serCategory/href"
const svcCatResponse = "/example/catalogue1"
const svcCatName = "serCategory/name"
const svcCatId = "serCategory/id"
const svcCatVersion = "serCategory/version"

const respMsg = "[{\"capabilityId\":\"16384563dca094183778a41ea7701d15\",\"capabilityName\":\"FaceRegService6\",\"status\":\"ACTIVE\",\"version\":\"3.2.1\",\"consumers\":[{\"applicationInstanceId\":\"5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f\"}]}]\n"
const respMsg1 = "{\"capabilityId\":\"16384563dca094183778a41ea7701d15\",\"capabilityName\":\"FaceRegService6\",\"status\":\"ACTIVE\",\"version\":\"3.2.1\",\"consumers\":[{\"applicationInstanceId\":\"5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f\"}]}\n"

const writeObjectStatusFormat = "{\"taskId\":\"%s\",\"appInstanceId\":\"5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f\"," +
	"\"configResult\":\"PROCESSING\",\"configPhase\":\"%d\",\"Detailed\":\"%s\"}\n"

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

// DB implementation with concurrency protector
type safeDB struct {
	mu sync.Mutex
	db map[string][]byte
}

func (c *safeDB) Put(key string, value []byte) {
	c.mu.Lock()
	if c.db == nil {
		c.db = make(map[string][]byte)
	}
	c.db[key] = value
	c.mu.Unlock()
}

func (c *safeDB) String() {
	c.mu.Lock()
	for k, v := range c.db {
		log.Infof("DB: Key-> %v; Value-> %v.", k, string(v))
	}
	c.mu.Unlock()
}

func (c *safeDB) Get(key string) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.db[key]
}

func (c *safeDB) GetMultiple(path string) map[string][]byte {
	c.mu.Lock()
	resultList := make(map[string][]byte)
	for k, v := range c.db {
		if strings.HasPrefix(k, path) {
			resultList[filepath.Base(k)] = v
		}
	}
	defer c.mu.Unlock()
	return resultList
}

func (c *safeDB) Delete(key string) {
	c.mu.Lock()
	delete(c.db, key)
	c.mu.Unlock()
}

// Query Task Status with valid values
func TestGetTaskStatus(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	//taskId
	//taskId := uuid.NewV4()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getTaskStatusFormat, defaultTaskId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(taskQueryFormat, defaultTaskId)
	//getRequest.Header.Set(appInstanceIdHeader, "wrong-app-instance-id")

	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte(fmt.Sprintf(writeObjectStatusFormat, defaultTaskId, 50, "Status"))).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patch1 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {

		// To handle two db call for Task Status or taskID-->AppID database.
		if strings.Contains(path, "taskstatus") {

			TrfSts := models.RuleStatus{Id: "r123", State: 0, Method: 0}
			DnsSts := models.RuleStatus{Id: "r144", State: 0, Method: 0}

			status := models.TaskStatus{}
			status.Progress = 1
			status.Details = "Status"
			status.DNSRuleStatusLst = append(status.DNSRuleStatusLst, DnsSts)
			status.TrafficRuleStatusLst = append(status.TrafficRuleStatusLst, TrfSts)

			outBytes, _ := json.Marshal(&status)
			return outBytes, 0
		} else {
			return []byte(defaultAppInstanceId), 0
		}
	})

	defer patch1.Reset()

	// 1
	service.URLPatterns()[4].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader), responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query Task Status with non existing Task ID
func TestGetTaskStatusInvalidTaskID(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	//taskId

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(getTaskStatusFormat, defaultTaskId),
		bytes.NewReader([]byte("")))
	getRequest.URL.RawQuery = fmt.Sprintf(taskQueryFormat, defaultTaskId)
	//getRequest.Header.Set(appInstanceIdHeader, "wrong-app-instance-id")

	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Can not found resource\",\"status\":5,\"detail\":\"task rule retrieval failed\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 404)

	patch1 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {

		return nil, util.SubscriptionNotFound
	})

	defer patch1.Reset()

	// 1
	service.URLPatterns()[4].Func(mockWriter, getRequest)

	assert.Equal(t, "404", responseHeader.Get(responseStatusHeader), responseCheckFor400)

	mockWriter.AssertExpectations(t)
}

// Delete ConfigRules - Success case
func TestDeleteConfigRules(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	recordInDb := []byte(`
	{
	  "appTrafficRule": [
		{
		  "trafficRuleId": "TrafficRule1",
		  "filterType": "FLOW",
		  "priority": 1,
		  "trafficFilter": [
			{
			  "srcAddress": [
				"192.168.1.1"
			  ],
			  "dstAddress": [
				"192.168.1.1"
			  ],
			  "srcPort": [
				"8080"
			  ],
			  "dstPort": [
				"8080"
			  ],
			  "protocol": [
				"TCP"
			  ],
			  "qCI": 1,
			  "dSCP": 0,
			  "tC": 1
			}
		  ],
		  "action": "DROP",
		  "state": "ACTIVE"
		}
	  ],
	  "appDNSRule": [
		{
		  "dnsRuleId": "dnsRule1",
		  "domainName": "www.example.com",
		  "ipAddressType": "IP_V6",
		  "ipAddress": "192.0.2.0",
		  "ttl": 30,
		  "state": "ACTIVE"
		}
	  ],
	  "appSupportMp1": true,
	  "appName": "abc"
	}`)

	//taskId
	taskId := uuid.NewV4()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("DELETE",
		fmt.Sprintf(appConfigUrlFormat, defaultAppInstanceId),
		nil)

	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte(fmt.Sprintf(writeObjectStatusFormat, taskId.String(), 0, "Operation In progress"))).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {

		// To handle two db call for Task Status or taskID-->AppID database.
		if strings.Contains(path, "jobs") {
			//Ongoing operation exist or not. Not exist is return error from DB.
			return nil, 1111
		} else {
			//Is instance id exist or not. To delete it must exist so return DB query True with some value.
			//return []byte("Hello"), 0
			return recordInDb, 0
		}
	})
	defer patches.Reset()

	patches.ApplyFunc(backend.PutRecord, func(path string, value []byte) int {
		// Return Success.
		return 0
	})

	patches.ApplyFunc(backend.DeletePaths, func(paths []string, continueOnFailure bool) int {
		// Return Success.
		return 0
	})

	var appDComm *plans.AppDCommon
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *plans.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})

	patches.ApplyFunc(util.GenerateUniqueId, func() string {
		// Return Success.
		return taskId.String()
	})

	// 1
	service.URLPatterns()[3].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader), responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Delete ConfigRules - Application Instance not exist
func TestDeleteConfigRulesAppNotExist(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	//taskId
	//taskId := uuid.NewV4()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("DELETE",
		fmt.Sprintf(appConfigUrlFormat, defaultAppInstanceId),
		nil)

	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 404)

	patch1 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {

		return nil, -1
	})

	defer patch1.Reset()

	// 1
	service.URLPatterns()[3].Func(mockWriter, getRequest)

	assert.Equal(t, "404", responseHeader.Get(responseStatusHeader), responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Delete ConfigRules - Some other Operation in progress
func TestDeleteConfigRulesOperationInProgress(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	//taskId
	//taskId := uuid.NewV4()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("DELETE",
		fmt.Sprintf(appConfigUrlFormat, defaultAppInstanceId),
		nil)

	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 403)

	var appDComm *plans.AppDCommon
	patches := gomonkey.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *plans.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})
	defer patches.Reset()
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAnyOngoingOperationExist", func(a *plans.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})

	// 1
	service.URLPatterns()[3].Func(mockWriter, getRequest)

	assert.Equal(t, "403", responseHeader.Get(responseStatusHeader), responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query ConfigRules - Success case
func TestGetConfigRules(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	recordInDb := []byte(`
	{
	  "appTrafficRule": [
		{
		  "trafficRuleId": "TrafficRule1",
		  "filterType": "FLOW",
		  "priority": 1,
		  "trafficFilter": [
			{
			  "srcAddress": [
				"192.168.1.1"
			  ],
			  "dstAddress": [
				"192.168.1.1"
			  ],
			  "srcPort": [
				"8080"
			  ],
			  "dstPort": [
				"8080"
			  ],
			  "protocol": [
				"TCP"
			  ],
			  "qCI": 1,
			  "dSCP": 0,
			  "tC": 1
			}
		  ],
		  "action": "DROP",
		  "state": "ACTIVE"
		}
	  ],
	  "appDNSRule": [
		{
		  "dnsRuleId": "dnsRule1",
		  "domainName": "www.example.com",
		  "ipAddressType": "IP_V6",
		  "ipAddress": "192.0.2.0",
		  "ttl": 30,
		  "state": "ACTIVE"
		}
	  ],
	  "appSupportMp1": true,
	  "appName": "abc"
	}`)

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(appConfigUrlFormat, defaultAppInstanceId),
		nil)

	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patch1 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {

		return recordInDb, 0
	})

	defer patch1.Reset()

	// 1
	service.URLPatterns()[2].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader), responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query ConfigRules - No record Exists
func TestGetConfigRulesFailure(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	getRequest, _ := http.NewRequest("GET",
		fmt.Sprintf(appConfigUrlFormat, defaultAppInstanceId),
		nil)

	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	//mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 417)

	patch1 := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {

		return nil, -1
	})

	defer patch1.Reset()

	// 1
	service.URLPatterns()[2].Func(mockWriter, getRequest)

	assert.Equal(t, "417", responseHeader.Get(responseStatusHeader), responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Create ConfigRules - Validate Input Parameters
func TestCreateAppDConfigInValidFilterType(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	postRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(appConfigUrlFormat, defaultAppInstanceId),
		bytes.NewReader([]byte(`
{
  "appTrafficRule": [
    {
      "trafficRuleId": "TrafficRule1",
      "filterType": "2FLOW22",
      "priority": 1,
      "trafficFilter": [
        {
          "srcAddress": [
            "192.168.1.1"
          ],
          "dstAddress": [
            "192.168.1.1"
          ],
          "srcPort": [
            "8080"
          ],
          "dstPort": [
            "8080"
          ],
          "protocol": [
            "TCP"
          ],
          "qCI": 1,
          "dSCP": 0,
          "tC": 1
        }
      ],
      "action": "DROP",
      "state": "ACTIVE"
    }
  ],
  "appDNSRule": [
    {
      "dnsRuleId": "dnsRule1",
      "domainName": "www.example.com",
      "ipAddressType": "IP_V6",
      "ipAddress": "192.0.2.0",
      "ttl": 30,
      "state": "ACTIVE"
    }
  ],
  "appSupportMp1": true,
  "appName": "abc"
}`)))

	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	// 1
	service.URLPatterns()[0].Func(mockWriter, postRequest)
	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader), responseCheckFor400)
	mockWriter.AssertExpectations(t)
}

// Create ConfigRules - Filter type missing
func TestCreateAppDConfigInValid1(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	postRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(appConfigUrlFormat, defaultAppInstanceId),
		bytes.NewReader([]byte(`
{
  "appTrafficRule": [
    {
      "trafficRuleId": "TrafficRule1",
      "priority": 1,
      "trafficFilter": [
        {
          "srcAddress": [
            "192.168.1.1"
          ],
          "dstAddress": [
            "192.168.1.1"
          ],
          "srcPort": [
            "8080"
          ],
          "dstPort": [
            "8080"
          ],
          "protocol": [
            "TCP"
          ],
          "qCI": 1,
          "dSCP": 0,
          "tC": 1
        }
      ],
      "action": "DROP",
      "state": "ACTIVE"
    }
  ],
  "appDNSRule": [
    {
      "dnsRuleId": "dnsRule1",
      "domainName": "www.example.com",
      "ipAddressType": "IP_V6",
      "ipAddress": "192.0.2.0",
      "ttl": 30,
      "state": "ACTIVE"
    }
  ],
  "appSupportMp1": true,
  "appName": "abc"
}`)))

	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	// 1
	service.URLPatterns()[0].Func(mockWriter, postRequest)
	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader), responseCheckFor400)
	mockWriter.AssertExpectations(t)
}

// Create ConfigRules - Priority Invalid value
func TestCreateAppDConfigInValid2(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	postRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(appConfigUrlFormat, defaultAppInstanceId),
		bytes.NewReader([]byte(`
{
  "appTrafficRule": [
    {
      "trafficRuleId": "TrafficRule1",
      "filterType": "FLOW",
      "priority": 555,
      "trafficFilter": [
        {
          "srcAddress": [
            "192.168.1.1"
          ],
          "dstAddress": [
            "192.168.1.1"
          ],
          "srcPort": [
            "8080"
          ],
          "dstPort": [
            "8080"
          ],
          "protocol": [
            "TCP"
          ],
          "qCI": 1,
          "dSCP": 0,
          "tC": 1
        }
      ],
      "action": "DROP",
      "state": "ACTIVE"
    }
  ],
  "appDNSRule": [
    {
      "dnsRuleId": "dnsRule1",
      "domainName": "www.example.com",
      "ipAddressType": "IP_V6",
      "ipAddress": "192.0.2.0",
      "ttl": 30,
      "state": "ACTIVE"
    }
  ],
  "appSupportMp1": true,
  "appName": "abc"
}`)))

	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	// 1
	service.URLPatterns()[0].Func(mockWriter, postRequest)
	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader), responseCheckFor400)
	mockWriter.AssertExpectations(t)
}

// Create ConfigRules - Empty Traffic Filters
func TestCreateAppDConfigInValid3(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}

	// Create http get request
	postRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(appConfigUrlFormat, defaultAppInstanceId),
		bytes.NewReader([]byte(`
{
  "appTrafficRule": [
    {
      "trafficRuleId": "TrafficRule1",
      "filterType": "FLOW",
      "priority": 1,
      "action": "DROP",
      "state": "ACTIVE"
    }
  ],
  "appDNSRule": [
    {
      "dnsRuleId": "dnsRule1",
      "domainName": "www.example.com",
      "ipAddressType": "IP_V6",
      "ipAddress": "192.0.2.0",
      "ttl": 30,
      "state": "ACTIVE"
    }
  ],
  "appSupportMp1": true,
  "appName": "abc"
}`)))

	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 400)

	// 1
	service.URLPatterns()[0].Func(mockWriter, postRequest)
	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader), responseCheckFor400)
	mockWriter.AssertExpectations(t)
}

// Create ConfigRules - Success case for none dataplane
func TestCreateAppDConfigRuleNoneDataPlane(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	//taskId

	patches := gomonkey.ApplyFunc(config.LoadMepServerConfig, func() (*config.MepServerConfig, error) {
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
			assert.Fail(t, "Parsing configuration file error")
		}
		return &mepConfig, nil
	})
	defer patches.Reset()
	patches.ApplyFunc(util.ReadMepAuthEndpoint, func() (string, error) {
		return "", nil
	})

	service := Mm5Service{}
	err := service.Init()
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}

	// Create http get request
	postRequest, _ := http.NewRequest("POST",
		fmt.Sprintf(appConfigUrlFormat, defaultAppInstanceId),
		bytes.NewReader([]byte(`
{
  "appTrafficRule": [
    {
      "trafficRuleId": "TrafficRule1",
      "filterType": "FLOW",
      "priority": 1,
      "trafficFilter": [
        {
          "srcAddress": [
            "192.168.1.1"
          ],
          "dstAddress": [
            "192.168.1.1"
          ],
          "srcPort": [
            "8080"
          ],
          "dstPort": [
            "8080"
          ],
          "protocol": [
            "TCP"
          ],
          "qCI": 1,
          "dSCP": 0,
          "tC": 1
        }
      ],
      "action": "DROP",
      "state": "ACTIVE"
    }
  ],
  "appDNSRule": [
    {
      "dnsRuleId": "dnsRule1",
      "domainName": "www.example.com",
      "ipAddressType": "IPv4",
      "ipAddress": "192.0.2.0",
      "ttl": 30,
      "state": "ACTIVE"
    }
  ],
  "appSupportMp1": true,
  "appName": "abc"
}`)))

	taskId := uuid.NewV4()

	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte(fmt.Sprintf(writeObjectStatusFormat, taskId.String(), 0,
		"Operation In progress"))).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	db := safeDB{}
	patches.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		log.Infof("Get path: %v.", path)
		db.String()
		if db.Get(path) != nil {
			log.Infof("Found the path in db: %s.", string(db.Get(path)))
			return db.Get(path), 0
		}

		return nil, util.SubscriptionNotFound
	})

	patches.ApplyFunc(backend.PutRecord, func(path string, value []byte) int {
		log.Infof("Put path: %v.", path)
		log.Infof("Put value: %v.", string(value))
		db.String()
		db.Put(path, value)
		// Return Success.
		return 0
	})

	patches.ApplyFunc(backend.DeletePaths, func(paths []string, continueOnFailure bool) int {
		log.Infof("Delete path: %v", paths)
		db.String()
		for _, path := range paths {
			db.Delete(path)
		}
		return 0
	})

	patches.ApplyFunc(util.GenerateUniqueId, func() string {
		return taskId.String()
	})

	patches.ApplyFunc((*http.Client).Do, func(client *http.Client, req *http.Request) (*http.Response,
		error) {
		response := http.Response{Status: "200 OK", StatusCode: 200}
		return &response, nil
	})

	dnsTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err2 := w.Write([]byte(""))
		if err2 != nil {
			t.Error("Error: Write Response Error")
		}
	}))

	defer dnsTestServer.Close()

	patches.ApplyFunc((*dns.RestDNSAgent).BuildDNSEndpoint, func(d *dns.RestDNSAgent, paths ...string) string {
		log.Infof("DNS Agent End Point: %v", dnsTestServer.URL)
		return dnsTestServer.URL
	})

	// 1
	service.URLPatterns()[0].Func(mockWriter, postRequest)
	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader), responseCheckFor200)
	mockWriter.AssertExpectations(t)

	// Getting the task status

	for true {
		time.Sleep(100 * time.Millisecond)

		mockWriterGet := &mockHttpWriterWithoutWrite{}
		responseHeaderGet := http.Header{} // Create http response header
		mockWriterGet.On("Header").Return(responseHeaderGet)
		mockWriterGet.On("Write").Return(0, nil)
		mockWriterGet.On("WriteHeader", 200)
		getRequest, _ := http.NewRequest("GET",
			fmt.Sprintf(getTaskStatusFormat, taskId.String()),
			bytes.NewReader([]byte("")))
		getRequest.URL.RawQuery = fmt.Sprintf(taskQueryFormat, taskId.String())

		mockWriterGet.response = []byte{}
		service.URLPatterns()[4].Func(mockWriterGet, getRequest)

		getResp := models.TaskProgress{}
		err := json.Unmarshal(mockWriterGet.response, &getResp)
		if err != nil {
			assert.Fail(t, err.Error(), string(mockWriterGet.response))
			return
		}
		if getResp.ConfigResult == util.TaskStateFailure {
			assert.Fail(t, "Operation failed", getResp, getRequest, postRequest)
			return
		} else if getResp.ConfigResult == util.TaskStateSuccess {
			log.Info("Create finished successfully.")
			break
		}
	}

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
		[]byte(respMsg)).
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
			SubscriptionId:    subscriberId1,
			FilteringCriteria: models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerInstanceIds = append(entry.FilteringCriteria.SerInstanceIds, defCapabilityId)
		outBytes, _ := json.Marshal(&entry)
		records[fmt.Sprintf(recordDB, defaultAppInstanceId, subscriberId1)] = outBytes
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
		[]byte(respMsg)).
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
			SubscriptionId:    subscriberId1,
			FilteringCriteria: models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerInstanceIds = append(entry.FilteringCriteria.SerInstanceIds, defCapabilityId)
		outBytes, _ := json.Marshal(&entry)
		records[fmt.Sprintf(recordDB, defaultAppInstanceId, subscriberId1)] = outBytes

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
			SubscriptionId:    subscriberId1,
			FilteringCriteria: models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerInstanceIds = append(entry.FilteringCriteria.SerInstanceIds, defCapabilityId)
		outBytes, _ := json.Marshal(&entry)
		records[fmt.Sprintf(recordDB, defaultAppInstanceId, subscriberId1)] = outBytes

		entry2 := models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    "09022fec-a63c-49fc-857a-dcd7ecaa40a2",
			FilteringCriteria: models.FilteringCriteria{},
		}
		entry2.FilteringCriteria.SerInstanceIds = append(entry2.FilteringCriteria.SerInstanceIds, defCapabilityId2)
		outBytes2, _ := json.Marshal(&entry2)
		records[fmt.Sprintf(recordDB, appInstanceId2, subscriberId2)] = outBytes2

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
		[]byte(respMsg)).
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
			SubscriptionId:    subscriberId1,
			FilteringCriteria: models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerNames = append(entry.FilteringCriteria.SerNames, "FaceRegService6")
		outBytes, _ := json.Marshal(&entry)
		records[fmt.Sprintf(recordDB, defaultAppInstanceId, subscriberId1)] = outBytes

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
		[]byte(respMsg)).
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
				"serName":     "FaceRegService6",
				"mecState":    "ACTIVE",
				svcCatHref:    svcCatResponse,
				svcCatId:      "id12345",
				svcCatName:    "RNI",
				svcCatVersion: "v1.1",
			},
		})
		return &response, nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyFunc(backend.GetRecordsWithCompleteKeyPath, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    subscriberId1,
			FilteringCriteria: models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerCategories = append(entry.FilteringCriteria.SerCategories, models.CategoryRef{
			Href:    svcCatResponse,
			ID:      "id12345",
			Name:    "RNI",
			Version: "v1.1",
		})
		outBytes, _ := json.Marshal(&entry)
		records[fmt.Sprintf(recordDB, defaultAppInstanceId, subscriberId1)] = outBytes

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
		[]byte(respMsg1)).
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
			SubscriptionId:    subscriberId1,
			FilteringCriteria: models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerInstanceIds = append(entry.FilteringCriteria.SerInstanceIds, defCapabilityId)
		outBytes, _ := json.Marshal(&entry)
		records[fmt.Sprintf(recordDB, defaultAppInstanceId, subscriberId1)] = outBytes

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
		[]byte(respMsg1)).
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
			SubscriptionId:    subscriberId1,
			FilteringCriteria: models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerNames = append(entry.FilteringCriteria.SerNames, "FaceRegService6")
		outBytes, _ := json.Marshal(&entry)
		records[fmt.Sprintf(recordDB, defaultAppInstanceId, subscriberId1)] = outBytes

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
				"serName":     "FaceRegService6",
				svcCatHref:    svcCatResponse,
				svcCatId:      "id12345",
				svcCatName:    "RNI",
				svcCatVersion: "v1.1",
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
		[]byte(respMsg1)).
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
				"serName":     "FaceRegService6",
				"mecState":    "ACTIVE",
				svcCatHref:    svcCatResponse,
				svcCatId:      "id12345",
				svcCatName:    "RNI",
				svcCatVersion: "v1.1",
			},
		}
		return &response, nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyFunc(backend.GetRecordsWithCompleteKeyPath, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		entry := models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    subscriberId1,
			FilteringCriteria: models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerCategories = append(entry.FilteringCriteria.SerCategories, models.CategoryRef{
			Href:    svcCatResponse,
			ID:      "id12345",
			Name:    "RNI",
			Version: "v1.1",
		})
		outBytes, _ := json.Marshal(&entry)
		records[fmt.Sprintf(recordDB, defaultAppInstanceId, subscriberId1)] = outBytes

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
				"serName":     "FaceRegService6",
				svcCatHref:    svcCatResponse,
				svcCatId:      "id12345",
				svcCatName:    "RNI",
				svcCatVersion: "v1.1",
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

func TestAppInstanceTermination3(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}
	getRequest, _ := http.NewRequest("DELETE",
		fmt.Sprintf(delAppInstFormat, defaultAppInstanceId),
		nil)
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 200)

	service.URLPatterns()[7].Func(mockWriterGet, getRequest)
}

func TestAppInstanceTermination(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}
	getRequest, _ := http.NewRequest("DELETE",
		fmt.Sprintf(delAppInstFormat, defaultAppInstanceId),
		nil)
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	patches := gomonkey.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)

		ins1 := &proto.MicroServiceInstance{
			InstanceId:     defCapabilityId[len(defCapabilityId)/2:],
			ServiceId:      defCapabilityId[:len(defCapabilityId)/2],
			Status:         "UP",
			Version:        "3.2.1",
			DataCenterInfo: &proto.DataCenterInfo{Name: "", Region: "", AvailableZone: ""},
			Properties: map[string]string{
				"appInstanceId": defaultAppInstanceId,
				"serName":       "FaceRegService6",
				"mecState":      "ACTIVE",
			},
		}

		json1, _ := json.Marshal(ins1)
		records[fmt.Sprintf(util.ServiceInfoDataCenter)] = json1

		return records, 0
	})
	defer patches.Reset()

	n := &srv.InstanceService{}
	patches.ApplyMethod(reflect.TypeOf(n), "Unregister", func(*srv.InstanceService, context.Context,
		*proto.UnregisterInstanceRequest) (*proto.UnregisterInstanceResponse, error) {
		return nil, nil
	})

	patches.ApplyFunc(os.Getenv, func(key string) string {
		if key == "MEPAUTH_SERVICE_PORT" {
			return "10443"
		}
		if key == "MEPAUTH_PORT_10443_TCP_ADDR" {
			return "1"
		}
		return "edgegallery"
	})

	var appDComm *plans.AppDCommon
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *plans.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAnyOngoingOperationExist", func(a *plans.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return false
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "StageNewTask", func(*plans.AppDCommon, string, string,
		*models.AppDConfig) (workspace.ErrCode, string) {
		return 0, ""
	})

	n1 := &task.Worker{}
	patches.ApplyMethod(reflect.TypeOf(n1), "ProcessDataPlaneSync", func(*task.Worker, string, string, string) {
		return
	})

	patches.ApplyFunc(task.CheckForStatusDBError, func(string, string) error {
		return nil
	})

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 200)

	service.URLPatterns()[7].Func(mockWriterGet, getRequest)
}

func TestAppInstanceTermination1(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}
	getRequest, _ := http.NewRequest("DELETE",
		fmt.Sprintf(delAppInstFormat, defaultAppInstanceId),
		nil)
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	patches := gomonkey.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)

		ins1 := &proto.MicroServiceInstance{
			InstanceId:     defCapabilityId[len(defCapabilityId)/2:],
			ServiceId:      defCapabilityId[:len(defCapabilityId)/2],
			Status:         "UP",
			Version:        "3.2.1",
			DataCenterInfo: &proto.DataCenterInfo{Name: "", Region: "", AvailableZone: ""},
			Properties: map[string]string{
				"appInstanceId": defaultAppInstanceId,
				"serName":       "FaceRegService6",
				"mecState":      "ACTIVE",
			},
		}

		json1, _ := json.Marshal(ins1)
		records[fmt.Sprintf(util.ServiceInfoDataCenter)] = json1

		return records, 0
	})
	defer patches.Reset()

	n := &srv.InstanceService{}
	patches.ApplyMethod(reflect.TypeOf(n), "Unregister", func(*srv.InstanceService, context.Context,
		*proto.UnregisterInstanceRequest) (*proto.UnregisterInstanceResponse, error) {
		return nil, nil
	})

	patches.ApplyFunc(os.Getenv, func(key string) string {
		if key == "MEPAUTH_SERVICE_PORT" {
			return "10443"
		}
		if key == "MEPAUTH_PORT_10443_TCP_ADDR" {
			return ""
		}
		return "edgegallery"
	})

	var appDComm *plans.AppDCommon
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *plans.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAnyOngoingOperationExist", func(a *plans.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return false
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "StageNewTask", func(*plans.AppDCommon, string, string,
		*models.AppDConfig) (workspace.ErrCode, string) {
		return 0, ""
	})

	n1 := &task.Worker{}
	patches.ApplyMethod(reflect.TypeOf(n1), "ProcessDataPlaneSync", func(*task.Worker, string, string, string) {
		return
	})

	patches.ApplyFunc(task.CheckForStatusDBError, func(string, string) error {
		return nil
	})

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 200)

	service.URLPatterns()[7].Func(mockWriterGet, getRequest)
}

func TestAppInstanceTermination2(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}
	getRequest, _ := http.NewRequest("DELETE",
		fmt.Sprintf(delAppInstFormat, defaultAppInstanceId),
		nil)
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	patches := gomonkey.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)

		ins1 := &proto.MicroServiceInstance{
			InstanceId:     defCapabilityId[len(defCapabilityId)/2:],
			ServiceId:      defCapabilityId[:len(defCapabilityId)/2],
			Status:         "UP",
			Version:        "3.2.1",
			DataCenterInfo: &proto.DataCenterInfo{Name: "", Region: "", AvailableZone: ""},
			Properties: map[string]string{
				"appInstanceId": defaultAppInstanceId,
				"serName":       "FaceRegService6",
				"mecState":      "ACTIVE",
			},
		}

		json1, _ := json.Marshal(ins1)
		records[fmt.Sprintf(util.ServiceInfoDataCenter)] = json1

		return records, 0
	})
	defer patches.Reset()

	n := &srv.InstanceService{}
	patches.ApplyMethod(reflect.TypeOf(n), "Unregister", func(*srv.InstanceService, context.Context,
		*proto.UnregisterInstanceRequest) (*proto.UnregisterInstanceResponse, error) {
		return nil, nil
	})

	patches.ApplyFunc(os.Getenv, func(key string) string {
		if key == "MEPAUTH_SERVICE_PORT" {
			return ""
		}
		if key == "MEPAUTH_PORT_10443_TCP_ADDR" {
			return "1"
		}
		return "edgegallery"
	})

	var appDComm *plans.AppDCommon
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *plans.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAnyOngoingOperationExist", func(a *plans.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return false
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "StageNewTask", func(*plans.AppDCommon, string, string,
		*models.AppDConfig) (workspace.ErrCode, string) {
		return 0, ""
	})

	n1 := &task.Worker{}
	patches.ApplyMethod(reflect.TypeOf(n1), "ProcessDataPlaneSync", func(*task.Worker, string, string, string) {
		return
	})

	patches.ApplyFunc(task.CheckForStatusDBError, func(string, string) error {
		return nil
	})

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 200)

	service.URLPatterns()[7].Func(mockWriterGet, getRequest)
}
