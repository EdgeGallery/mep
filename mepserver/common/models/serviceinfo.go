/*
 * Copyright 2021 Huawei Technologies Co., Ltd.
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

// Package models implements mep server object models
package models

import (
	"encoding/json"
	"fmt"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/server/core/proto"

	meputil "mepserver/common/util"
)

const PropertiesMapSize = 5
const FormatIntBase = 10
const serviceLivenessInterval = "livenessInterval"
const serviceGatewayURIFormatString = "%s://mep-api-gw.mep:%d/%s"

// ServiceInfo holds the service info response/request body
type ServiceInfo struct {
	SerInstanceId     string        `json:"serInstanceId,omitempty"`
	SerName           string        `json:"serName" validate:"required,max=128,validateName"`
	SerCategory       CategoryRef   `json:"serCategory" validate:"omitempty"`
	Version           string        `json:"version" validate:"required,max=32"`
	State             string        `json:"state" validate:"required,oneof=ACTIVE INACTIVE"`
	TransportID       string        `json:"transportId" validate:"omitempty,max=64,validateId"`
	TransportInfo     TransportInfo `json:"transportInfo" validate:"omitempty"`
	Serializer        string        `json:"serializer" validate:"required,oneof=JSON XML PROTOBUF3"`
	ScopeOfLocality   string        `json:"scopeOfLocality" validate:"omitempty,oneof=MEC_SYSTEM MEC_HOST NFVI_POP ZONE ZONE_GROUP NFVI_NODE"`
	ConsumedLocalOnly bool          `json:"consumedLocalOnly,omitempty"`
	IsLocal           bool          `json:"isLocal,omitempty"`
	LivenessInterval  int           `json:"livenessInterval" validate:"omitempty,gte=0,max=2147483646"`
	Links             Link          `json:"_links,omitempty"`
}
type Link struct {
	Self          Selves `json:"self"`
	AppInstanceId string `json:"appInstanceId"`
}
type Selves struct {
	Href string `json:"liveness,omitempty"`
}

// GenerateServiceRequest transform ServiceInfo to CreateServiceRequest
func (s *ServiceInfo) GenerateServiceRequest(req *proto.CreateServiceRequest) {
	if req != nil {
		if req.Service == nil {
			req.Service = &proto.MicroService{}
		}
		req.Service.AppId = ""
		req.Service.ServiceName = s.SerName
		req.Service.Version = meputil.ServiceVersion
		req.Service.Status = "UP"
		if s.State == meputil.InactiveState {
			req.Service.Status = "DOWN"
		}
	} else {
		log.Warn("create service request nil")
	}
}

// GenerateRegisterInstance transform ServiceInfo to RegisterInstanceRequest
func (s *ServiceInfo) GenerateRegisterInstance(req *proto.RegisterInstanceRequest, isUpdateReq bool) {
	if req != nil {
		if req.Instance == nil {
			req.Instance = &proto.MicroServiceInstance{}
		}
		if req.Instance.Properties == nil {
			req.Instance.Properties = make(map[string]string, PropertiesMapSize)
		}
		req.Instance.Properties["serName"] = s.SerName
		s.serCategoryToProperties(req.Instance.Properties)
		req.Instance.Version = meputil.ServiceVersion
		req.Instance.Properties["version"] = s.Version
		req.Instance.Timestamp = strconv.FormatInt(time.Now().Unix(), FormatIntBase)
		req.Instance.ModTimestamp = req.Instance.Timestamp

		req.Instance.Status = "UP"
		if s.State == meputil.InactiveState {
			req.Instance.Status = "DOWN"
		}
		properties := req.Instance.Properties
		meputil.UpdatePropertiesMap(properties, "transportId", s.TransportID)
		meputil.UpdatePropertiesMap(properties, "serializer", s.Serializer)
		meputil.UpdatePropertiesMap(properties, "ScopeOfLocality", s.ScopeOfLocality)
		meputil.UpdatePropertiesMap(properties, "ConsumedLocalOnly", strconv.FormatBool(s.ConsumedLocalOnly))
		meputil.UpdatePropertiesMap(properties, "IsLocal", strconv.FormatBool(s.IsLocal))
		meputil.UpdatePropertiesMap(properties, serviceLivenessInterval, strconv.Itoa(0))
		if s.LivenessInterval != 0 {
			meputil.UpdatePropertiesMap(properties, serviceLivenessInterval, strconv.Itoa(meputil.DefaultHeartbeatInterval))
			s.LivenessInterval = meputil.DefaultHeartbeatInterval
		}
		meputil.UpdatePropertiesMap(properties, "mecState", s.State)
		secNanoSec := strconv.FormatInt(time.Now().UTC().UnixNano(), FormatIntBase)
		meputil.UpdatePropertiesMap(properties, "timestamp/seconds", secNanoSec[:len(secNanoSec)/2+1])
		meputil.UpdatePropertiesMap(properties, "timestamp/nanoseconds", secNanoSec[len(secNanoSec)/2+1:])
		req.Instance.HostName = "default"
		var epType string
		req.Instance.Endpoints, epType = s.registerEndpoints(isUpdateReq, req.Instance)
		req.Instance.Properties["endPointType"] = epType

		healthCheck := &proto.HealthCheck{
			Mode:     proto.CHECK_BY_HEARTBEAT,
			Port:     0,
			Interval: math.MaxInt32 - 1,
			Times:    0,
			Url:      "",
		}
		req.Instance.HealthCheck = healthCheck
		s.transportInfoToProperties(req.Instance.Properties)
	} else {
		log.Warn("register instance request nil")
	}
}

func (s *ServiceInfo) registerEndpoints(modReq bool, instance *proto.MicroServiceInstance) ([]string, string) {
	if len(s.TransportInfo.Endpoint.Uris) != 0 {
		return s.handleEndPointUri(s.TransportInfo.Endpoint.Uris, modReq, instance)
	} else if len(s.TransportInfo.Endpoint.Addresses) != 0 {
		var nUris []string
		for _, address := range s.TransportInfo.Endpoint.Addresses {
			uri := fmt.Sprintf("%s://%s:%d/", s.TransportInfo.Protocol, address.Host, address.Port)
			nUris = append(nUris, uri)
		}
		return s.handleEndPointUri(nUris, modReq, instance)
	} else if s.TransportInfo.Endpoint.Alternative != nil {
		jsonBytes, err := json.Marshal(s.TransportInfo.Endpoint.Alternative)
		if err != nil {
			return nil, ""
		}
		jsonText := string(jsonBytes)
		endPoints := make([]string, 0, 1)
		endPoints = append(endPoints, jsonText)
		return endPoints, meputil.Alternatives
	}
	// On modification request if there is no entry, clean up existing entries
	if modReq {
		for k, v := range instance.Properties {
			if !strings.HasPrefix(k, meputil.EndPointPropPrefix) {
				continue
			}
			delete(instance.Properties, k)
			meputil.ApiGWInterface.CleanUpApiGwEntry(v)
		}
	}
	return nil, ""
}
func (s *ServiceInfo) generateServiceIdAndName() (string, string) {
	serviceId := util.GenerateUuid()[0:20]
	return serviceId, s.SerName + serviceId
}

func (s *ServiceInfo) handleEndPointUri(nUris []string, modReq bool, instance *proto.MicroServiceInstance) (
	[]string, string) {
	var serviceUris []string
	appConfig, err := meputil.GetAppConfig()
	if err != nil {
		log.Error("Get app config failed.", err)
		return nil, ""
	}
	httpProtocol := appConfig["http_protocol"]
	sslEnable := appConfig["ssl_enabled"]
	var proxyPort int
	if strings.EqualFold(sslEnable, "true") {
		proxyPort = 8443
	} else {
		proxyPort = 8000
	}

	newEn, unmEn, delEn := s.compareEndPointsWithDB(nUris, modReq, instance)
	for uri, serName := range newEn {
		meputil.UpdatePropertiesMap(instance.Properties, meputil.EndPointPropPrefix+uri, serName)
		serviceUris = append(serviceUris, fmt.Sprintf(serviceGatewayURIFormatString, httpProtocol, proxyPort, serName))
		s.RegisterToApiGw(uri, serName, modReq)
	}
	if modReq {
		for _, serName := range unmEn {
			serviceUris = append(serviceUris, fmt.Sprintf(serviceGatewayURIFormatString, httpProtocol, proxyPort, serName))
		}
		for uri, serName := range delEn {
			delete(instance.Properties, meputil.EndPointPropPrefix+uri)
			meputil.ApiGWInterface.CleanUpApiGwEntry(serName)
		}
	}
	return serviceUris, meputil.Uris
}

func (s *ServiceInfo) compareEndPointsWithDB(nUris []string, modReq bool, instance *proto.MicroServiceInstance) (
	map[string]string, map[string]string, map[string]string) {
	newEn := make(map[string]string) // new entries
	if !modReq {
		for _, uri := range nUris {
			_, gwSerName := s.generateServiceIdAndName()
			newEn[uri] = gwSerName // not found, adding new
		}
		return newEn, nil, nil
	}
	unmEn := make(map[string]string) // un-modified
	delEn := make(map[string]string) // to delete
	for _, uri := range nUris {
		if serName, found := instance.Properties[meputil.EndPointPropPrefix+uri]; !found {
			_, gwSerName := s.generateServiceIdAndName()
			newEn[uri] = gwSerName // not found, adding new
		} else {
			unmEn[uri] = serName // found, so keeping as unmodified
		}
	}
	// look for deleted entries
	for k, v := range instance.Properties {
		if !strings.HasPrefix(k, meputil.EndPointPropPrefix) {
			continue
		}
		uri := k[len(meputil.EndPointPropPrefix):]
		if _, found := newEn[uri]; found {
			continue
		}
		if _, found := unmEn[uri]; found {
			continue
		}
		delEn[uri] = v
	}
	return newEn, unmEn, delEn
}

func (s *ServiceInfo) transportInfoToProperties(properties map[string]string) {
	if properties == nil {
		return
	}
	meputil.UpdatePropertiesMap(properties, "transportInfo/id", s.TransportInfo.ID)
	meputil.UpdatePropertiesMap(properties, "transportInfo/name", s.TransportInfo.Name)
	meputil.UpdatePropertiesMap(properties, "transportInfo/description", s.TransportInfo.Description)
	meputil.UpdatePropertiesMap(properties, "transportInfo/type", string(s.TransportInfo.TransType))
	meputil.UpdatePropertiesMap(properties, "transportInfo/protocol", s.TransportInfo.Protocol)
	meputil.UpdatePropertiesMap(properties, "transportInfo/version", s.TransportInfo.Version)
	grantTypes := strings.Join(s.TransportInfo.Security.OAuth2Info.GrantTypes, "，")
	meputil.UpdatePropertiesMap(properties, "transportInfo/security/oAuth2Info/grantTypes", grantTypes)
	meputil.UpdatePropertiesMap(properties, "transportInfo/security/oAuth2Info/tokenEndpoint",
		s.TransportInfo.Security.OAuth2Info.TokenEndpoint)

}

func (s *ServiceInfo) serCategoryToProperties(properties map[string]string) {
	if properties == nil {
		return
	}
	meputil.UpdatePropertiesMap(properties, "serCategory/href", s.SerCategory.Href)
	meputil.UpdatePropertiesMap(properties, "serCategory/id", s.SerCategory.ID)
	meputil.UpdatePropertiesMap(properties, "serCategory/name", s.SerCategory.Name)
	meputil.UpdatePropertiesMap(properties, "serCategory/version", s.SerCategory.Version)
}

// FromServiceInstance transform MicroServiceInstance to ServiceInfo
func (s *ServiceInfo) FromServiceInstance(inst *proto.MicroServiceInstance) {
	if inst == nil || inst.Properties == nil {
		return
	}
	s.SerInstanceId = inst.ServiceId + inst.InstanceId
	s.serCategoryFromProperties(inst.Properties)
	s.Version = inst.Properties["version"]
	s.State = inst.Properties["mecState"]

	s.Links = Link{
		AppInstanceId: inst.Properties["appInstanceId"],
	}
	s.SerName = inst.Properties["serName"]
	s.TransportID = inst.Properties["transportId"]
	s.Serializer = inst.Properties["serializer"]
	s.ScopeOfLocality = inst.Properties["ScopeOfLocality"]
	var err error
	s.LivenessInterval, err = strconv.Atoi(inst.Properties[serviceLivenessInterval])
	if err != nil {
		log.Warn("parse int liveness Interval fail")
	}
	if s.LivenessInterval != 0 {
		s.Links.Self.Href = inst.Properties["liveness"]
	}
	s.ConsumedLocalOnly, err = strconv.ParseBool(inst.Properties["ConsumedLocalOnly"])
	if err != nil {
		log.Warn("parse bool ConsumedLocalOnly fail")
	}
	s.IsLocal, err = strconv.ParseBool(inst.Properties["IsLocal"])
	if err != nil {
		log.Warn("parse bool IsLocal fail")
	}
	s.fromEndpoints(inst.Endpoints, inst.Properties["endPointType"])
	s.transportInfoFromProperties(inst.Properties)
}

func (s *ServiceInfo) RegisterToApiGw(uri string, serviceName string, isUpdateReq bool) {
	log.Infof("API gateway registration for new service(name: %s, uri: %s).", serviceName, uri)
	serInfo := meputil.SerInfo{
		SerName: serviceName,
		Uri:     uri,
	}
	meputil.ApiGWInterface.AddOrUpdateApiGwService(serInfo)
	meputil.ApiGWInterface.AddOrUpdateApiGwRoute(serInfo)
	if !isUpdateReq {
		meputil.ApiGWInterface.EnableJwtPlugin(serInfo)
	}
}

func (s *ServiceInfo) serCategoryFromProperties(properties map[string]string) {
	if properties == nil {
		return
	}
	s.SerCategory.Href = properties["serCategory/href"]
	s.SerCategory.ID = properties["serCategory/id"]
	s.SerCategory.Name = properties["serCategory/name"]
	s.SerCategory.Version = properties["serCategory/version"]
}

func (s *ServiceInfo) fromEndpoints(uris []string, epType string) {
	if epType == meputil.Uris {
		s.TransportInfo.Endpoint.Uris = uris
		return
	}
	if epType == meputil.Addresses {

		s.TransportInfo.Endpoint.Addresses = make([]EndPointInfoAddress, 0, 1)
		for _, v := range uris {
			host, port, err := meputil.GetHostPort(v)
			if err != nil { // Exclude if uri is not correct
				continue
			}
			tmp := EndPointInfoAddress{
				Host: host,
				Port: uint32(port),
			}
			s.TransportInfo.Endpoint.Addresses = append(s.TransportInfo.Endpoint.Addresses, tmp)
		}
	}
	if epType == meputil.Alternatives {
		if len(uris) == 0 {
			return
		}
		jsonObj, err := meputil.JsonTextToObj(uris[0])
		if err != nil {
			s.TransportInfo.Endpoint.Alternative = jsonObj
		}
		return
	}
}

func (s *ServiceInfo) transportInfoFromProperties(properties map[string]string) {
	if properties == nil {
		return
	}
	s.TransportInfo.ID = properties["transportInfo/id"]
	s.TransportInfo.Name = properties["transportInfo/name"]
	s.TransportInfo.Description = properties["transportInfo/description"]
	s.TransportInfo.TransType = TransportTypes(properties["transportInfo/type"])
	s.TransportInfo.Protocol = properties["transportInfo/protocol"]
	s.TransportInfo.Version = properties["transportInfo/version"]
	grantTypes := properties["transportInfo/security/oAuth2Info/grantTypes"]
	s.TransportInfo.Security.OAuth2Info.GrantTypes = strings.Split(grantTypes, ",")
	s.TransportInfo.Security.OAuth2Info.TokenEndpoint = properties["transportInfo/security/oAuth2Info/tokenEndpoint"]
}
