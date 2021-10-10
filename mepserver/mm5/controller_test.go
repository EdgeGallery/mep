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
	es "github.com/olivere/elastic/v7"
	uuid "github.com/satori/go.uuid"
	"io/ioutil"
	"math/rand"
	"mepserver/common/appd"
	"mepserver/common/arch/workspace"
	"mepserver/common/config"
	"mepserver/common/extif/dataplane"
	"mepserver/common/extif/dns"
	"mepserver/common/models"
	"mepserver/mm5/plans"
	"mepserver/mm5/task"
	"mepserver/mp1/event"
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
)

const defaultAppInstanceId = "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"

const panicFormatString = "Panic: %v"
const getTaskStatusFormat = "/mepcfg/app_lcm/v1/tasks/%s/appd_configuration"
const appConfigUrlFormat = "/mepcfg/app_lcm/v1/applications/%s/appd_configuration"
const delAppInstFormat = "/mep/mec_app_support/v1/applications/%s/AppInstanceTermination"
const kongLogFormat = "/service_govern/v1/kong_log"

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

const respMsg = "[{\"capabilityId\":\"16384563dca094183778a41ea7701d15\",\"capabilityName\":\"FaceRegService6\",\"statu" +
	"s\":\"ACTIVE\",\"version\":\"3.2.1\",\"consumers\":[{\"applicationInstanceId\":\"5abe4782-2c70-4e47-9a4e-0ee3a1a0f" +
	"d1f\"}]}]\n"
const respMsg1 = "{\"capabilityId\":\"16384563dca094183778a41ea7701d15\",\"capabilityName\":\"FaceRegService6\",\"statu" +
	"s\":\"ACTIVE\",\"version\":\"3.2.1\",\"consumers\":[{\"applicationInstanceId\":\"5abe4782-2c70-4e47-9a4e-0ee3a1a0" +
	"fd1f\"}]}\n"
const writeAllServices = "{\"data\":[{\"serInstanceId\":\"16384563dca094183778a41ea7701d15\",\"serName\":\"FaceRegServ" +
	"ice6\",\"serCategory\":{\"href\":\"/example/catalogue1\",\"id\":\"id12345\",\"name\":\"RNI\",\"version\":\"v1.1\"" +
	"},\"version\":\"3.2.1\",\"state\":\"\",\"transportId\":\"\",\"transportInfo\":{\"id\":\"\",\"name\":\"\",\"descrip" +
	"tion\":\"\",\"type\":\"\",\"protocol\":\"\",\"version\":\"\",\"endpoint\":{\"uris\":null,\"addresses\":null,\"alte" +
	"rnative\":null},\"security\":{\"oAuth2Info\":{\"grantTypes\":[\"\"],\"tokenEndpoint\":\"\"}}},\"serializer\":\"\"," +
	"\"scopeOfLocality\":\"\",\"livenessInterval\":0,\"_links\":{\"self\":{},\"appInstanceId\":\"\"}}],\"retCode\":0,\"m" +
	"essage\":\"\",\"params\":\"\"}\n"
const writeObjectStatusFormat = "{\"taskId\":\"%s\",\"appInstanceId\":\"5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f\"," +
	"\"configResult\":\"PROCESSING\",\"configPhase\":\"%d\",\"Detailed\":\"%s\"}\n"

const dnsRuleId = "7d71e54e-81f3-47bb-a2fc-b565a326d794"
const trafficRuleId = "8ft68t22-81f3-47bb-a2fc-56996er4tf37"
const exampleDomainName = "www.example.com"
const appUpdateRsp = "{\"taskId\":\"703e0f3b-b993-4d35-8d93-a469a4909ca3\",\"appInstanceId\":\"\",\"configResult\":\"PR" +
	"OCESSING\",\"configPhase\":\"0\",\"Detailed\":\"Operation In progress\"}\n"
const subscribeInfoRsp = "{\"data\":{\"subscribeNum\":{\"appSubscribeNum\":0,\"serviceSubscribedNum\":0},\"subscribeRe" +
	"lations\":[]},\"retCode\":0,\"message\":\"\",\"params\":\"\"}\n"
const subscribeRecords = "{\"data\":{\"subscribeNum\":{\"appSubscribeNum\":1,\"serviceSubscribedNum\":1},\"subscribe" +
	"Relations\":[{\"subscribeAppId\":\"5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f\",\"serviceList\":[\"16384563dca094183778a" +
	"41ea7701d15\",\"16384563dca094183778a41ea7701d15\",\"16384563dca094183778a41ea7701d15\"]}]},\"retCode\":0,\"mes" +
	"sage\":\"\",\"params\":\"\"}\n"

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

	var appDComm *appd.AppDCommon
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *appd.AppDCommon,
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

	var appDComm *appd.AppDCommon
	patches := gomonkey.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})
	defer patches.Reset()
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAnyOngoingOperationExist", func(a *appd.AppDCommon,
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

// Create ConfigRules - Success case for none dataplane

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

// Create ConfigRules - Success case for none dataplane
func TestCreateAppDConfigNoInstance(t *testing.T) {
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

	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Duplicate request error\",\"status\":19,\"detail\":\"duplicate app instance\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

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
	a := &appd.AppDCommon{}
	patches.ApplyMethod(reflect.TypeOf(a), "IsAppInstanceAlreadyCreated", func(t *appd.AppDCommon, appInstanceId string) bool {
		return true
	})

	// 1
	service.URLPatterns()[0].Func(mockWriter, postRequest)
	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader), responseCheckFor200)
	mockWriter.AssertExpectations(t)
}

// Create ConfigRules - Success case for none dataplane
func TestCreateAppDConfigDuplicate(t *testing.T) {
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

	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Duplicate request error\",\"status\":19,\"detail\":\"duplicate app name\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)

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
	a := &appd.AppDCommon{}
	patches.ApplyMethod(reflect.TypeOf(a), "IsDuplicateAppNameExists", func(t *appd.AppDCommon, appName string) bool {
		return true
	})

	// 1
	service.URLPatterns()[0].Func(mockWriter, postRequest)
	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader), responseCheckFor200)
	mockWriter.AssertExpectations(t)
}

// Create ConfigRules - Success case for none dataplane
func TestCreateAppDConfigOperationInProgress(t *testing.T) {
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

	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write", []byte("{\"title\":\"Operation Not Allowed\",\"status\":20,\"detail\":\"app instance has other operation in progress\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 403)

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

	a := &appd.AppDCommon{}
	patches.ApplyMethod(reflect.TypeOf(a), "IsAnyOngoingOperationExist", func(t *appd.AppDCommon, appName string) bool {
		return true
	})

	// 1
	service.URLPatterns()[0].Func(mockWriter, postRequest)
	assert.Equal(t, "403", responseHeader.Get(responseStatusHeader), responseCheckFor200)
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

// Query capability

// Query capability

// Query capability

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

	var appDComm *appd.AppDCommon
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAnyOngoingOperationExist", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return false
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "StageNewTask", func(*appd.AppDCommon, string, string,
		*models.AppDConfig, bool) (workspace.ErrCode, string) {
		return 0, ""
	})

	n1 := &task.Worker{}
	patches.ApplyMethod(reflect.TypeOf(n1), "ProcessDataPlaneSync", func(*task.Worker, string, string, string) {
		return
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

	var appDComm *appd.AppDCommon
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAnyOngoingOperationExist", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return false
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "StageNewTask", func(*appd.AppDCommon, string, string,
		*models.AppDConfig, bool) (workspace.ErrCode, string) {
		return 0, ""
	})

	n1 := &task.Worker{}
	patches.ApplyMethod(reflect.TypeOf(n1), "ProcessDataPlaneSync", func(*task.Worker, string, string, string) {
		return
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

	var appDComm *appd.AppDCommon
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAnyOngoingOperationExist", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return false
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "StageNewTask", func(*appd.AppDCommon, string, string,
		*models.AppDConfig, bool) (workspace.ErrCode, string) {
		return 0, ""
	})

	n1 := &task.Worker{}
	patches.ApplyMethod(reflect.TypeOf(n1), "ProcessDataPlaneSync", func(*task.Worker, string, string, string) {
		return
	})

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 200)

	service.URLPatterns()[7].Func(mockWriterGet, getRequest)
}

// Query All services

// Query capability
func TestAppDUpdate(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}
	TrafficRule := make([]dataplane.TrafficRule, 0)
	updateRule := dataplane.TrafficRule{
		TrafficRuleID: trafficRuleId,
		FilterType:    "FLOW",
		Priority:      5,
		TrafficFilter: []dataplane.TrafficFilter{},
		Action:        "DROP",
		State:         "INACTIVE",
	}
	TrafficRule = append(TrafficRule, updateRule)
	DNSRule := make([]dataplane.DNSRule, 0)

	updateDnsRule := dataplane.DNSRule{
		DNSRuleID:     dnsRuleId,
		DomainName:    exampleDomainName,
		IPAddressType: util.IPv4Type,
		IPAddress:     exampleIPAddress,
		TTL:           1,
		State:         util.InactiveState,
	}
	DNSRule = append(DNSRule, updateDnsRule)
	appConfig := models.AppDConfig{TrafficRule, DNSRule, true, "abc", "PUT"}
	appConfigBytes, _ := json.Marshal(appConfig)
	// Create http get request
	getRequest, _ := http.NewRequest("GET", getCapabilitiesUrl, bytes.NewReader(appConfigBytes))
	getRequest.URL.RawQuery = writeAllServices

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte(appUpdateRsp)).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)
	x := 0
	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		// To handle two db call for Task Status or taskID-->AppID database.

		if x == 0 {
			x = x + 1
			return []byte(defaultAppInstanceId), 0
		} else {
			return []byte(`
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
}`), 0
		}

	})
	defer patches.Reset()
	patches.ApplyFunc(util.FindInstanceByKey, func(result url.Values) (*proto.FindInstancesResponse, error) {
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

	patches.ApplyFunc(util.GenerateUniqueId, func() string {
		return "703e0f3b-b993-4d35-8d93-a469a4909ca3"
	})

	// 2 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[1].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query capability
func TestAppDUpdateAppNameIncorrect(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}
	TrafficRule := make([]dataplane.TrafficRule, 0)
	updateRule := dataplane.TrafficRule{
		TrafficRuleID: trafficRuleId,
		FilterType:    "FLOW",
		Priority:      5,
		TrafficFilter: []dataplane.TrafficFilter{},
		Action:        "DROP",
		State:         "INACTIVE",
	}
	TrafficRule = append(TrafficRule, updateRule)
	DNSRule := make([]dataplane.DNSRule, 0)

	updateDnsRule := dataplane.DNSRule{
		DNSRuleID:     dnsRuleId,
		DomainName:    exampleDomainName,
		IPAddressType: util.IPv4Type,
		IPAddress:     exampleIPAddress,
		TTL:           1,
		State:         util.InactiveState,
	}
	DNSRule = append(DNSRule, updateDnsRule)
	appConfig := models.AppDConfig{TrafficRule, DNSRule, true, "invalid", "PUT"}
	appConfigBytes, _ := json.Marshal(appConfig)
	// Create http get request
	getRequest, _ := http.NewRequest("GET", getCapabilitiesUrl, bytes.NewReader(appConfigBytes))
	getRequest.URL.RawQuery = writeAllServices

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte("{\"title\":\"Bad Request\",\"status\":6,\"detail\":\"app-name doesn't match\"}\n")).
		Return(0, nil)
	mockWriter.On("WriteHeader", 400)
	x := 0
	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) ([]byte, int) {
		// To handle two db call for Task Status or taskID-->AppID database.

		if x == 0 {
			x = x + 1
			return []byte(defaultAppInstanceId), 0
		} else {
			return []byte(`
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
  "appName": "abcd"
}`), 0
		}
	})
	defer patches.Reset()
	patches.ApplyFunc(util.FindInstanceByKey, func(result url.Values) (*proto.FindInstancesResponse, error) {
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
	patches.ApplyFunc(util.GenerateUniqueId, func() string {
		return "703e0f3b-b993-4d35-8d93-a469a4909ca3"
	})

	// 2 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[1].Func(mockWriter, getRequest)

	assert.Equal(t, "400", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query capability
func TestQuerySubscribeStatistic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}
	// Create http get request
	getRequest, _ := http.NewRequest("GET", getCapabilitiesUrl, bytes.NewReader([]byte("dummy")))
	getRequest.URL.RawQuery = writeAllServices

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte(subscribeInfoRsp)).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	// 2 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[10].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

// Query capability
func TestQuerySubscribeStatisticRecords(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}
	// Create http get request
	getRequest, _ := http.NewRequest("GET", getCapabilitiesUrl, bytes.NewReader([]byte("dummy")))
	getRequest.URL.RawQuery = writeAllServices

	// Mock the response writer
	mockWriter := &mockHttpWriter{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write",
		[]byte(subscribeRecords)).
		Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	patch := gomonkey.ApplyFunc(event.GetAllSubscriberInfoFromDB, func() map[string]*models.SerAvailabilityNotificationSubscription {
		records := make(map[string]*models.SerAvailabilityNotificationSubscription)
		entry := models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    subscriberId1,
			FilteringCriteria: models.FilteringCriteria{},
		}
		entry.FilteringCriteria.SerInstanceIds = append(entry.FilteringCriteria.SerInstanceIds, defCapabilityId)
		entry.FilteringCriteria.SerNames = append(entry.FilteringCriteria.SerNames, "FaceRegService6")
		records[fmt.Sprintf(recordDB, defaultAppInstanceId, subscriberId1)] = &entry
		entry2 := models.SerAvailabilityNotificationSubscription{
			SubscriptionId:    subscriberId1,
			FilteringCriteria: models.FilteringCriteria{},
		}
		entry2.FilteringCriteria.SerInstanceIds = append(entry.FilteringCriteria.SerInstanceIds, defCapabilityId)
		entry2.FilteringCriteria.SerNames = append(entry.FilteringCriteria.SerNames, "FaceRegService6")
		records[fmt.Sprintf(recordDB, defaultAppInstanceId, subscriberId1+"a")] = &entry2
		return records
	})
	defer patch.Reset()

	// 2 is the order of the DNS get one handler in the URLPattern
	service.URLPatterns()[10].Func(mockWriter, getRequest)

	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader),
		responseCheckFor200)

	mockWriter.AssertExpectations(t)
}

func TestAppInstanceTerminationErrHandler(t *testing.T) {
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

	var appDComm *appd.AppDCommon
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAnyOngoingOperationExist", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return false
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "StageNewTask", func(*appd.AppDCommon, string, string,
		*models.AppDConfig, bool) (workspace.ErrCode, string) {
		return 0, ""
	})

	n1 := &task.Worker{}
	patches.ApplyMethod(reflect.TypeOf(n1), "ProcessDataPlaneSync", func(*task.Worker, string, string, string) {
		return
	})

	patches.ApplyFunc(backend.GetRecord, func(path string) (record []byte, errorCode int) {
		TrfSts := models.RuleStatus{Id: "r123", State: 0, Method: 0}
		DnsSts := models.RuleStatus{Id: "r144", State: 0, Method: 0}
		status := models.TaskStatus{}
		status.Progress = 1
		status.Details = "Status"
		status.DNSRuleStatusLst = append(status.DNSRuleStatusLst, DnsSts)
		status.TrafficRuleStatusLst = append(status.TrafficRuleStatusLst, TrfSts)

		outBytes, _ := json.Marshal(&status)
		return outBytes, 0
	})

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 200)

	service.URLPatterns()[7].Func(mockWriterGet, getRequest)
}

func TestAppInstanceTerminationErr(t *testing.T) {
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

	var appDComm *appd.AppDCommon
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAnyOngoingOperationExist", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return false
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "StageNewTask", func(*appd.AppDCommon, string, string,
		*models.AppDConfig, bool) (workspace.ErrCode, string) {
		return 0, ""
	})

	n1 := &task.Worker{}
	patches.ApplyMethod(reflect.TypeOf(n1), "ProcessDataPlaneSync", func(*task.Worker, string, string, string) {
		return
	})

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 200)

	service.URLPatterns()[7].Func(mockWriterGet, getRequest)
}

func TestAppInstanceTerminationErrUnmarshal(t *testing.T) {
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

	var appDComm *appd.AppDCommon
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAnyOngoingOperationExist", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return false
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "StageNewTask", func(*appd.AppDCommon, string, string,
		*models.AppDConfig, bool) (workspace.ErrCode, string) {
		return 0, ""
	})

	n1 := &task.Worker{}
	patches.ApplyMethod(reflect.TypeOf(n1), "ProcessDataPlaneSync", func(*task.Worker, string, string, string) {
		return
	})

	patches.ApplyFunc(backend.GetRecord, func(path string) (record []byte, errorCode int) {
		outBytes := make([]byte, 0)
		return outBytes, 0
	})

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 200)

	service.URLPatterns()[7].Func(mockWriterGet, getRequest)
}

func TestAppInstanceTerminationNoProgress(t *testing.T) {
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

	var appDComm *appd.AppDCommon
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAnyOngoingOperationExist", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return false
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "StageNewTask", func(*appd.AppDCommon, string, string,
		*models.AppDConfig, bool) (workspace.ErrCode, string) {
		return 0, ""
	})

	n1 := &task.Worker{}
	patches.ApplyMethod(reflect.TypeOf(n1), "ProcessDataPlaneSync", func(*task.Worker, string, string, string) {
		return
	})

	patches.ApplyFunc(backend.GetRecord, func(path string) (record []byte, errorCode int) {
		TrfSts := models.RuleStatus{Id: "r123", State: 0, Method: 0}
		DnsSts := models.RuleStatus{Id: "r144", State: 0, Method: 0}
		status := models.TaskStatus{}
		status.Progress = -1
		status.Details = "Status"
		status.DNSRuleStatusLst = append(status.DNSRuleStatusLst, DnsSts)
		status.TrafficRuleStatusLst = append(status.TrafficRuleStatusLst, TrfSts)

		outBytes, _ := json.Marshal(&status)
		return outBytes, 0
	})

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 200)

	service.URLPatterns()[7].Func(mockWriterGet, getRequest)
}

func TestAppInstanceTerminationStatusDbErr(t *testing.T) {
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

	var appDComm *appd.AppDCommon
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAnyOngoingOperationExist", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return false
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "StageNewTask", func(*appd.AppDCommon, string, string,
		*models.AppDConfig, bool) (workspace.ErrCode, string) {
		return 0, ""
	})

	n1 := &task.Worker{}
	patches.ApplyMethod(reflect.TypeOf(n1), "ProcessDataPlaneSync", func(*task.Worker, string, string, string) {
		return
	})

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 200)

	service.URLPatterns()[7].Func(mockWriterGet, getRequest)
}

func TestAppInstanceTerminationStatusDbErrhandle(t *testing.T) {
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

	var appDComm *appd.AppDCommon
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAnyOngoingOperationExist", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return false
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "StageNewTask", func(*appd.AppDCommon, string, string,
		*models.AppDConfig, bool) (workspace.ErrCode, string) {
		return 0, ""
	})

	n1 := &task.Worker{}
	patches.ApplyMethod(reflect.TypeOf(n1), "ProcessDataPlaneSync", func(*task.Worker, string, string, string) {
		return
	})

	patches.ApplyFunc(backend.GetRecord, func(path string) (record []byte, errorCode int) {
		TrfSts := models.RuleStatus{Id: "r123", State: 0, Method: 0}
		DnsSts := models.RuleStatus{Id: "r144", State: 0, Method: 0}

		status := models.TaskStatus{}
		status.Progress = 1
		status.Details = "Status"
		status.DNSRuleStatusLst = append(status.DNSRuleStatusLst, DnsSts)
		status.TrafficRuleStatusLst = append(status.TrafficRuleStatusLst, TrfSts)

		outBytes, _ := json.Marshal(&status)
		return outBytes, 0
	})

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 200)

	service.URLPatterns()[7].Func(mockWriterGet, getRequest)
}

const callBack = "https://%d.%d.%d.%d:%d/example/catalogue1"

var callBackRef = fmt.Sprintf(callBack, 192, 0, 2, 1, 8080)

func TestAppInstanceTerminationInProgress(t *testing.T) {
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

	var appDComm *appd.AppDCommon
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAnyOngoingOperationExist", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return false
	})

	rec_count := 1
	patches.ApplyFunc(backend.GetRecord, func(path string) (record []byte, errorCode int) {
		if rec_count <= 3 {
			rec_count++
			return recordInDb, 0
		} else if rec_count == 4 {
			rec_count++
			TrfSts := models.RuleStatus{Id: "r123", State: 0, Method: 0}
			DnsSts := models.RuleStatus{Id: "r144", State: 0, Method: 0}

			status := models.TaskStatus{}
			status.Progress = 1
			status.Details = "Status"
			status.DNSRuleStatusLst = append(status.DNSRuleStatusLst, DnsSts)
			status.TrafficRuleStatusLst = append(status.TrafficRuleStatusLst, TrfSts)
			status.TerminationStatus = util.TerminationInProgress
			outBytes, _ := json.Marshal(&status)
			return outBytes, 0
		} else if rec_count == 5 {
			createSubscription := models.AppTerminationNotificationSubscription{
				SubscriptionType:  "AppTerminationNotificationSubscription",
				CallbackReference: "https://webhook.site/92fc3c0a-90e1-45ca-b346-ca514056fade",
				AppInstanceId:     defaultAppInstanceId,
			}
			createSubscriptionBytes, _ := json.Marshal(createSubscription)
			return createSubscriptionBytes, 0
		} else {
			rec_count++
			return nil, 0
		}
	})

	// Mock the response writer
	mockWriterGet := &mockHttpWriterWithoutWrite{}
	responseGetHeader := http.Header{} // Create http response header
	mockWriterGet.On("Header").Return(responseGetHeader)
	mockWriterGet.On("Write").Return(0, nil)
	mockWriterGet.On("WriteHeader", 200)

	service.URLPatterns()[7].Func(mockWriterGet, getRequest)
}

func TestHandleTerminationNotification(t *testing.T) {

	count := 1
	patches := gomonkey.ApplyFunc(backend.GetRecord, func(path string) (record []byte, errorCode int) {
		if path == util.AppTerminationNotificationSubscription+defaultAppInstanceId+"/" {
			createSubscription := models.AppTerminationNotificationSubscription{
				SubscriptionType:  "AppTerminationNotificationSubscription",
				CallbackReference: "https://webhook.site/92fc3c0a-90e1-45ca-b346-ca514056fade",
				AppInstanceId:     defaultAppInstanceId,
			}
			createSubscriptionBytes, _ := json.Marshal(createSubscription)
			return createSubscriptionBytes, 0
		} else if count == 20 && path == util.AppConfirmTerminationPath+defaultAppInstanceId+"/" {
			rec := models.ConfirmTerminationRecord{
				util.TERMINATING,
				util.TerminationInProgress,
			}
			recBytes, _ := json.Marshal(rec)
			return recBytes, 0
		}
		count++
		return nil, 0
	})

	defer patches.Reset()
	patches.ApplyFunc(os.Getenv, func(key string) string {
		if key == "MEPAUTH_SERVICE_PORT" {
			return "10443"
		}
		if key == "MEPAUTH_PORT_10443_TCP_ADDR" {
			return "1"
		}
		return "edgegallery"
	})
	//w := task.Worker {}
	//w.HandleTerminationNotification(defaultAppInstanceId)
}

func TestAppInstanceTerminationStatus(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()
	TrafficRule := make([]dataplane.TrafficRule, 0)
	updateRule := dataplane.TrafficRule{
		TrafficRuleID: trafficRuleId,
		FilterType:    "FLOW",
		Priority:      5,
		TrafficFilter: []dataplane.TrafficFilter{},
		Action:        "DROP",
		State:         "INACTIVE",
	}
	TrafficRule = append(TrafficRule, updateRule)
	DNSRule := make([]dataplane.DNSRule, 0)

	updateDnsRule := dataplane.DNSRule{
		DNSRuleID:     dnsRuleId,
		DomainName:    exampleDomainName,
		IPAddressType: util.IPv4Type,
		IPAddress:     exampleIPAddress,
		TTL:           1,
		State:         util.InactiveState,
	}
	DNSRule = append(DNSRule, updateDnsRule)
	appConfig := models.AppDConfig{TrafficRule, DNSRule, true, "invalid", "PUT"}
	appConfigBytes, _ := json.Marshal(appConfig)

	dnsTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		_, err2 := w.Write([]byte(""))
		if err2 != nil {
			t.Error("Error: Write Response Error")
		}
	}))

	defer dnsTestServer.Close()

	service := Mm5Service{}
	getRequest, _ := http.NewRequest("DELETE",
		fmt.Sprintf(delAppInstFormat, defaultAppInstanceId),
		nil)
	getRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	getRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)
	count := 1
	patches := gomonkey.ApplyFunc(backend.GetRecords, func(path string) (map[string][]byte, int) {
		records := make(map[string][]byte)
		createSubscription := models.AppTerminationNotificationSubscription{
			SubscriptionType:  "AppTerminationNotificationSubscription",
			CallbackReference: dnsTestServer.URL,
			AppInstanceId:     defaultAppInstanceId,
		}
		createSubscriptionBytes, _ := json.Marshal(createSubscription)
		records[path] = createSubscriptionBytes
		count++
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

	var appDComm *appd.AppDCommon
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAppInstanceAlreadyCreated", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return true
	})
	patches.ApplyMethod(reflect.TypeOf(appDComm), "IsAnyOngoingOperationExist", func(a *appd.AppDCommon,
		appInstanceId string) bool {
		// Return Success.
		return false
	})

	counter := 1
	finish := false
	patches.ApplyFunc(backend.GetRecord, func(path string) (record []byte, errorCode int) {
		fmt.Println(path)
		if counter == 2 {
			TrfSts := models.RuleStatus{Id: "r123", State: 0, Method: 0}
			DnsSts := models.RuleStatus{Id: "r144", State: 0, Method: 0}

			status := models.TaskStatus{}
			status.Progress = 1
			status.Details = "Status"
			status.DNSRuleStatusLst = append(status.DNSRuleStatusLst, DnsSts)
			status.TrafficRuleStatusLst = append(status.TrafficRuleStatusLst, TrfSts)
			status.TerminationStatus = util.TerminationInProgress

			outBytes, _ := json.Marshal(&status)
			counter++
			return outBytes, 0
		}
		if counter == 1 {
			counter++
			return appConfigBytes, 0
		}

		if path == "/cse-sr/etsi/app-confirm-termination/5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f/" {
			var result util.AppTerminateStatus
			if !finish {
				result = util.TerminationInProgress
				finish = true
			} else {
				result = util.TerminationFinish
			}
			confirm := models.ConfirmTerminationRecord{OperationAction: "TERMINATING", TerminationStatus: result}
			outBytes, _ := json.Marshal(confirm)
			counter++
			return outBytes, 0
		}

		return appConfigBytes, 0
	})
	t.Run("success", func(t *testing.T) {
		patche1 := gomonkey.ApplyFunc((*http.Client).Do, func(client *http.Client, req *http.Request) (*http.Response,
			error) {
			response := http.Response{Status: "200 OK", StatusCode: 200}
			response.Body = http.NoBody
			return &response, nil
		})
		patche1.Reset()
		// Mock the response writer
		mockWriterGet := &mockHttpWriterWithoutWrite{}
		responseGetHeader := http.Header{} // Create http response header
		mockWriterGet.On("Header").Return(responseGetHeader)
		mockWriterGet.On("Write").Return(0, nil)
		mockWriterGet.On("WriteHeader", 200)

		service.URLPatterns()[7].Func(mockWriterGet, getRequest)
		time.Sleep(time.Second * 5)
	})
}

// Query Task Status with valid values
func TestInsertHttpLog(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}
	plans.EsClient, _ = es.NewClient(es.SetSniff(false), es.SetURL("http://localhost"))

	// Create http get request
	postRequest, _ := http.NewRequest("POST",
		kongLogFormat,
		bytes.NewReader([]byte(`
		{
			"name": "update plugins",
			"request": {
				"method": "POST",
				"headers":
					{
						"host": "http://localhost",
						"authorization": "Bearer eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MzM1MzcyOTguNzk1NDI3LCJpc3MiOiJtZXBhdXRoIiwic3ViIjoiNWFiZTQ3ODItMmM3MC00ZTQ3LTlhNGUtMGVlM2ExYTBmZDFmIiwiY2xpZW50aXAiOiIyMDAuMS4xLjIifQ.f9NIZ5AtKHIadGKWodcfZ5A_IbZX_tAIZ_C9BgHgvTUtOI0QODFMtlKPxqbMW487Nyq_ROYgP19zDUBYT93de8tFLPpT8O9Kn7_YIIvWASyGP_eapO6g30j7rFX_8rjvxa21kDTQhoo6HnD_pbnh_QWex4vuDHzGmWoW_2AAq_MkkPCmDBu7nWLSElpupcsvlY0qTgR9Ay7dYvKX1-L-c0Pdjcy4sisdyneK9gg-hcBaKDfgr_yAARi92QDc7iXxZCu3hjDTq_9JQWkwuBSJ3a-yl5spzPT0xhWiTlSWlyKtTyfd85g2SglHGb9jCRqddou9poSixVhKeiH3_DJPBg",
						"content-type": "text"
					},
				"body": {
					"mode": "raw",
					"raw": "{\r\n    \"name\": \"http-log\",\r\n    \"config\": {\r\n        \"flush_timeout\": 2,\r\n        \"http_endpoint\": \"http://logstash:5044\",\r\n        \"retry_count\": 10,\r\n        \"timeout\": 1000,\r\n        \"queue_size\": 1,\r\n        \"keepalive\": 1000,\r\n        \"content_type\": \"application/json\",\r\n        \"method\": \"POST\"\r\n    }\r\n}",
					"options": {
						"raw": {
							"language": "json"
						}
					}
				},
				"url": {
					"raw": "https://{{KONG_HOST}}:8444/plugins/03b14d7e-f307-40ba-a849-f0730dce1e46",
					"protocol": "https",
					"host": [
						"{{KONG_HOST}}"
					],
					"port": "8444",
					"path": [
						"plugins",
						"03b14d7e-f307-40ba-a849-f0730dce1e46"
					]
				}
			},
			"response": []
		}`)))

	ss := &es.IndexService{}

	patches := gomonkey.ApplyMethod(reflect.TypeOf(ss), "Do", func(s *es.IndexService, ctx context.Context) (*es.IndexResponse, error) {
		return nil, nil
	})
	defer patches.Reset()
	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	// 1
	service.URLPatterns()[8].Func(mockWriter, postRequest)
	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader), responseCheckFor400)
	mockWriter.AssertExpectations(t)
}

// Query Task Status with valid values
func TestGetHttpLog(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	service := Mm5Service{}
	plans.EsClient, _ = es.NewClient(es.SetSniff(false), es.SetURL("http://localhost"))

	// Create http get request
	postRequest, _ := http.NewRequest("GET",
		kongLogFormat,
		bytes.NewReader([]byte(`
		{
			"name": "update plugins",
			"request": {
				"method": "POST",
				"headers":
					{
						"host": "http://localhost",
						"authorization": "Bearer eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MzM1MzcyOTguNzk1NDI3LCJpc3MiOiJtZXBhdXRoIiwic3ViIjoiNWFiZTQ3ODItMmM3MC00ZTQ3LTlhNGUtMGVlM2ExYTBmZDFmIiwiY2xpZW50aXAiOiIyMDAuMS4xLjIifQ.f9NIZ5AtKHIadGKWodcfZ5A_IbZX_tAIZ_C9BgHgvTUtOI0QODFMtlKPxqbMW487Nyq_ROYgP19zDUBYT93de8tFLPpT8O9Kn7_YIIvWASyGP_eapO6g30j7rFX_8rjvxa21kDTQhoo6HnD_pbnh_QWex4vuDHzGmWoW_2AAq_MkkPCmDBu7nWLSElpupcsvlY0qTgR9Ay7dYvKX1-L-c0Pdjcy4sisdyneK9gg-hcBaKDfgr_yAARi92QDc7iXxZCu3hjDTq_9JQWkwuBSJ3a-yl5spzPT0xhWiTlSWlyKtTyfd85g2SglHGb9jCRqddou9poSixVhKeiH3_DJPBg",
						"content-type": "text"
					},
				"body": {
					"mode": "raw",
					"raw": "{\r\n    \"name\": \"http-log\",\r\n    \"config\": {\r\n        \"flush_timeout\": 2,\r\n        \"http_endpoint\": \"http://logstash:5044\",\r\n        \"retry_count\": 10,\r\n        \"timeout\": 1000,\r\n        \"queue_size\": 1,\r\n        \"keepalive\": 1000,\r\n        \"content_type\": \"application/json\",\r\n        \"method\": \"POST\"\r\n    }\r\n}",
					"options": {
						"raw": {
							"language": "json"
						}
					}
				},
				"url": {
					"raw": "https://{{KONG_HOST}}:8444/plugins/03b14d7e-f307-40ba-a849-f0730dce1e46",
					"protocol": "https",
					"host": [
						"{{KONG_HOST}}"
					],
					"port": "8444",
					"path": [
						"plugins",
						"03b14d7e-f307-40ba-a849-f0730dce1e46"
					]
				}
			},
			"response": []
		}`)))

	patches := gomonkey.ApplyFunc(util.FindInstanceByKey, func(result url.Values) (*proto.FindInstancesResponse, error) {
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
	ss := &es.CountService{}

	patches.ApplyMethod(reflect.TypeOf(ss), "Do", func(s *es.CountService, ctx context.Context) (int64, error) {
		return 0, nil
	})
	defer patches.Reset()
	postRequest.URL.RawQuery = fmt.Sprintf(appInstanceQueryFormat, defaultAppInstanceId)
	postRequest.Header.Set(appInstanceIdHeader, defaultAppInstanceId)

	mockWriter := &mockHttpWriterWithoutWrite{}
	responseHeader := http.Header{} // Create http response header
	mockWriter.On("Header").Return(responseHeader)
	mockWriter.On("Write").Return(0, nil)
	mockWriter.On("WriteHeader", 200)

	// 1
	service.URLPatterns()[9].Func(mockWriter, postRequest)
	assert.Equal(t, "200", responseHeader.Get(responseStatusHeader), responseCheckFor400)
	mockWriter.AssertExpectations(t)
}

func TestInitRootKeyAndWorkKey(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()
	key := []byte("testkey")
	util.KeyComponentFromUserStr = &key
	var Encdata []byte
	var Noncedata []byte
	patches := gomonkey.ApplyFunc(ioutil.WriteFile, func(filename string, data []byte, perm os.FileMode) error {
		if filename == util.EncryptedWorkKeyFilePath {
			Encdata = data
		}
		if filename == util.WorkKeyNonceFilePath {
			Noncedata = data
		}
		return nil
	})
	count := 1
	patches.ApplyFunc(ioutil.ReadFile, func(name string) ([]byte, error) {
		if name == util.EncryptedWorkKeyFilePath {
			return Encdata, nil
		} else if name == util.WorkKeyNonceFilePath {
			return Noncedata, nil
		} else if count == 1 {
			count++
			return []byte("eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MzM1MzcyOTguNzk1NDI3LCJpc3MiOiJtZXBhdXRoIiwic3ViIjoiNWFiZTQ3ODItMmM3MC00ZTQ3LTlhNGUtMGVlM2ExYTBmZDFmIiwiY2xpZW50aXAiOiIyMDAuMS4xLjIifQ.f9NIZ5AtKHIadGKWodcfZ5A_IbZX_tAIZ_C9BgHgvTUtOI0QODFMtlKPxqbMW487Nyq_ROYg"), nil
		} else {
			count++
			return []byte("yJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MzM1MzcyOTguNzk1NDI3LCJpc3MiOiJtZXBhdXRoIiwic3ViIjoiNWFiZTQ3ODItMmM3MC00ZTQ3LTlhNGUtMGVlM2ExYTBmZDFmIiwiY2xpZW50aXAiOiIyMDAuMS4xLjIifQ.f9NIZ5AtKHIadGKWodcfZ5A_IbZX_tAIZ_C9BgHgvTUtOI0QODFMtlKPxqbMW487Nyq_ROYgk"), nil
		}
	})

	defer patches.Reset()
	util.InitRootKeyAndWorkKey()
	util.EncryptAndSaveCertPwd(&key)
	certPwd, err := util.GetCertPwd()
	fmt.Println(certPwd, err)
	assert.Equal(t, 7, count)
}
