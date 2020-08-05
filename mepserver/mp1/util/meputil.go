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

// Package path implements mep server utility functions and constants
package util

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/rest"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"github.com/apache/servicecomb-service-center/server/core"
	"github.com/apache/servicecomb-service-center/server/core/backend"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/registry"
	svcutil "github.com/apache/servicecomb-service-center/server/service/util"
	"github.com/go-playground/validator/v10"
)

// put k,v into map
func InfoToProperties(properties map[string]string, key string, value string) {
	if value != "" {
		properties[key] = value
	}
}

// trans json to obj
func JsonTextToObj(jsonText string) (interface{}, error) {
	data := []byte(jsonText)
	var jsonMap interface{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&jsonMap)
	if err != nil {
		return nil, err
	}
	return jsonMap, nil
}

// get host port in uri
func GetHostPort(uri string) (string, int) {
	const zeroPort int = 0
	idx := strings.LastIndex(uri, ":")
	domain := uri
	port := zeroPort
	var err error
	if idx > 0 {
		port, err = strconv.Atoi(uri[idx+1:])
		if err != nil {
			port = zeroPort
		}
		domain = uri[:idx]
	}
	return domain, port
}

// get tags in http request
func GetHTTPTags(r *http.Request) (url.Values, []string) {
	var ids []string
	query := r.URL.Query()
	keys := query.Get("tags")
	if len(keys) > 0 {
		ids = strings.Split(keys, ",")
	}

	return query, ids
}

// write err response
func HttpErrResponse(w http.ResponseWriter, statusCode int, obj interface{}) {
	if obj == nil {
		w.Header().Set(rest.HEADER_RESPONSE_STATUS, strconv.Itoa(statusCode))
		w.Header().Set(rest.HEADER_CONTENT_TYPE, rest.CONTENT_TYPE_TEXT)
		w.WriteHeader(statusCode)
		return
	}

	objJSON, err := json.Marshal(obj)
	if err != nil {
		log.Errorf(nil, "json masrshalling failed")
		return
	}
	w.Header().Set(rest.HEADER_RESPONSE_STATUS, strconv.Itoa(http.StatusOK))
	w.Header().Set(rest.HEADER_CONTENT_TYPE, rest.CONTENT_TYPE_JSON)
	w.WriteHeader(statusCode)
	_, err = fmt.Fprintln(w, string(objJSON))
	if err != nil {
		log.Errorf(nil, "send http response fail")
	}
}

// heartbeat use put to update a service register info
func Heartbeat(ctx context.Context, mp1SvcId string) error {
	serviceID := mp1SvcId[:len(mp1SvcId)/2]
	instanceID := mp1SvcId[len(mp1SvcId)/2:]
	req := &proto.HeartbeatRequest{
		ServiceId:  serviceID,
		InstanceId: instanceID,
	}
	_, err := core.InstanceAPI.Heartbeat(ctx, req)
	return err
}

// get service instance by serviceId
func GetServiceInstance(ctx context.Context, serviceId string) (*proto.MicroServiceInstance, error) {
	domainProject := util.ParseDomainProject(ctx)
	serviceID := serviceId[:len(serviceId)/2]
	instanceID := serviceId[len(serviceId)/2:]
	instance, err := svcutil.GetInstance(ctx, domainProject, serviceID, instanceID)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		err = fmt.Errorf("domainProject %s sservice Id %s not exist", domainProject, serviceID)
	}
	return instance, err
}

// get instance by key
func FindInstanceByKey(result url.Values) (*proto.FindInstancesResponse, error) {
	serCategoryId := result.Get("ser_category_id")
	scopeOfLocality := result.Get("scope_of_locality")
	consumedLocalOnly := result.Get("consumed_local_only")
	isLocal := result.Get("is_local")
	isQueryAllSvc := serCategoryId == "" && scopeOfLocality == "" && consumedLocalOnly == "" && isLocal == ""
	opts := []registry.PluginOp{
		registry.OpGet(registry.WithStrKey("/cse-sr/inst/files///"), registry.WithPrefix()),
	}
	resp, err := backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("query from etch error")
	}
	var findResp []*proto.MicroServiceInstance
	for _, value := range resp.Kvs {
		var instance map[string]interface{}
		err = json.Unmarshal(value.Value, &instance)
		if err != nil {
			return nil, fmt.Errorf("string convert to instance failed")
		}
		dci := &proto.DataCenterInfo{Name: "", Region: "", AvailableZone: ""}
		instance["datacenterinfo"] = dci
		message, err := json.Marshal(&instance)
		if err != nil {
			log.Errorf(nil, "instance convert to string failed")
			return nil, err
		}
		var ins *proto.MicroServiceInstance
		err = json.Unmarshal(message, &ins)
		if err != nil {
			log.Errorf(nil, "String convert to MicroServiceInstance failed!")
			return nil, err
		}
		property := ins.Properties
		if isQueryAllSvc && property != nil {
			findResp = append(findResp, ins)
		} else if strings.EqualFold(property["serCategory/id"], serCategoryId) ||
			strings.EqualFold(property["ConsumedLocalOnly"], consumedLocalOnly) ||
			strings.EqualFold(property["ScopeOfLocality"], scopeOfLocality) ||
			strings.EqualFold(property["IsLocal"], isLocal) {
			findResp = append(findResp, ins)
		}
	}
	if len(findResp) == 0 {
		return nil, fmt.Errorf("null")
	}
	response := &proto.Response{Code: 0, Message: ""}
	ret := &proto.FindInstancesResponse{Response: response, Instances: findResp}
	return ret, nil
}

// set map value
func SetMapValue(theMap map[string]interface{}, key string, val interface{}) {
	mapValue, ok := theMap[key]
	if !ok || mapValue == nil {
		theMap[key] = val
	}
}

// get the index of the string in []string
func StringContains(arr []string, val string) (index int) {
	index = -1
	for i := 0; i < len(arr); i++ {
		if arr[i] == val {
			index = i
			return
		}
	}
	return
}

// validate UUID
func ValidateUUID(id string) error {
	if len(id) != 0 {
		validate := validator.New()
		return validate.Var(id, "required,uuid")
	}
	return nil
}

// validate serviceId
func ValidateServiceID(serID string) error {
	return ValidateRegexp(serID, "[0-9a-f]{32}",
		"service ID validation failed")
}

// validate by reg
func ValidateRegexp(strToCheck string, regexStr string, errMsg string) error {
	match, err := regexp.MatchString(regexStr, strToCheck)
	if err != nil {
		return err
	}
	if !match {
		return errors.New(errMsg)
	}
	return nil
}

// get subscribe key path
func GetSubscribeKeyPath(subscribeType string) string {
	var subscribeKeyPath string
	if subscribeType == SerAvailabilityNotificationSubscription {
		subscribeKeyPath = AvailAppSubKeyPath
	} else {
		subscribeKeyPath = EndAppSubKeyPath
	}
	return subscribeKeyPath
}

// validate appInstanceId in header
func ValidateAppInstanceIdWithHeader(id string, r *http.Request) error {
	if id == r.Header.Get("X-AppinstanceID") {
		return nil
	}
	return errors.New("UnAuthorization to access the resource")
}

// trans obj to json
func ParseToJson(v interface{}) string {
	parseResult, err := json.Marshal(v)
	if err != nil {
		log.Error("Failed to marshal service info to json.", nil)
		return ""
	}
	return string(parseResult)
}

// get resource info
func GetResourceInfo(r *http.Request) string {
	resource := r.URL.String()
	if resource == "" {
		return "UNKNOWN"
	}
	return resource
}

// get method from request
func GetMethod(r *http.Request) string {
	method := r.Method
	if method == "" {
		return "GET"
	}
	return method
}

// get appInstanceId from request
func GetAppInstanceId(r *http.Request) string {
	query, _ := GetHTTPTags(r)
	return query.Get(":appInstanceId")
}

// get clientIp from request
func GetClientIp(r *http.Request) string {
	clientIp := r.Header.Get("X-Real-Ip")
	if clientIp == "" {
		clientIp = "UNKNOWN_IP"
	}
	return clientIp
}
