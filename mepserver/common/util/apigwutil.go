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

// Package util implements mep server utility functions and constants
package util

import (
	"crypto/tls"
	"fmt"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/astaxie/beego/httplib"
)

const serviceUrl string = "/services/"
const routeUrl string = "/routes/"

var cipherSuiteMap = map[string]uint16{
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
}

// RouteInfo represents api gateway route model
type RouteInfo struct {
	Id      int64   `json:"routeId"`
	AppId   string  `json:"appId"`
	SerInfo SerInfo `orm:"type(json)" json:"serInfo"`
}

// SerInfo represents api gateway service info
type SerInfo struct {
	SerName string `json:"serName"`
	Uri     string `json:"uri"`
}

// ApiGWInterface holds an api gateway instance
var ApiGWInterface *ApiGwIf

// ApiGwIf represents api gateway interface
type ApiGwIf struct {
	baseURL string
	tlsCfg  *tls.Config
}

// NewApiGwIf initialize new api gate way instance
func NewApiGwIf() *ApiGwIf {
	a := &ApiGwIf{}
	baseUrl := a.getApiGwUrl()
	if len(baseUrl) == 0 {
		return nil
	}
	if GetAppConfigByKey("ssl_mode") == "1" {
		tlsCfg, err := TLSConfig(ApiGwCaCertName, false)
		if err != nil {
			return nil
		}
		a.tlsCfg = tlsCfg
	}

	a.baseURL = baseUrl
	return a
}

func (a *ApiGwIf) getApiGwUrl() string {
	appConfig, err := GetAppConfig()
	if err != nil {
		log.Error("Get app config failed.", err)
		return ""
	}
	apiGwUrl := fmt.Sprintf("https://%s:%s", appConfig["apigw_host"], appConfig["apigw_port"])
	return apiGwUrl

}

// AddOrUpdateApiGwService add/update new service in the api gateway for application
func (a *ApiGwIf) AddOrUpdateApiGwService(serInfo SerInfo) {
	serName := serInfo.SerName
	serUrl := serInfo.Uri
	jsonStr := []byte(fmt.Sprintf(`{ "url": "%s", "name": "%s" }`, serUrl, serName))
	apiGwServiceUrl := a.baseURL + serviceUrl + serName
	_, err := SendPutRequest(apiGwServiceUrl, jsonStr, a.tlsCfg)
	if err != nil {
		log.Error("Failed to add or update API gateway service", err)
	}
}

//DeleteApiGwService delete service from api  gateway
func (a *ApiGwIf) DeleteApiGwService(serviceName string) {
	apiGwServiceUrl := a.baseURL + serviceUrl + serviceName
	_, err := SendDelRequest(apiGwServiceUrl, a.tlsCfg)
	if err != nil {
		log.Error("Failed to delete API gateway service.", err)
	}
}

// AddOrUpdateApiGwRoute add/update new route in the api gateway for application
func (a *ApiGwIf) AddOrUpdateApiGwRoute(serInfo SerInfo) {
	serName := serInfo.SerName
	apiGwRouteUrl := a.baseURL + serviceUrl + serName + routeUrl + serName
	jsonStr := []byte(fmt.Sprintf(`{ "paths": ["/%s"], "name": "%s" }`, serName, serName))
	_, err := SendPutRequest(apiGwRouteUrl, jsonStr, a.tlsCfg)
	if err != nil {
		log.Error("Failed to add or update API gateway route", err)
	}
}

// DeleteApiGwRoute delete API gateway route
func (a *ApiGwIf) DeleteApiGwRoute(serviceName string) {
	apiGwRouteUrl := a.baseURL + serviceUrl + serviceName + routeUrl + serviceName
	_, err := SendDelRequest(apiGwRouteUrl, a.tlsCfg)
	if err != nil {
		log.Error("Failed to delete API gateway route.", err)
	}
}

// EnableJwtPlugin enables kong jwt plugin
func (a *ApiGwIf) EnableJwtPlugin(serInfo SerInfo) {
	serName := serInfo.SerName
	apiGwPluginUrl := a.baseURL + serviceUrl + serName + "/plugins"
	jwtConfig := fmt.Sprintf(`{ "name": "%s", "config": { "claims_to_verify": ["exp"] } }`, JwtPlugin)
	_, err := SendPostRequest(apiGwPluginUrl, []byte(jwtConfig), a.tlsCfg)
	if err != nil {
		log.Error("Enable apiGw jwt plugin failed", err)
	}
}

func (a *ApiGwIf) DeleteJwtPlugin(serviceName string) {
	apiGwPluginUrl := a.baseURL + serviceUrl + serviceName + "/plugins"
	jwtConfig := fmt.Sprintf(`{ "name": "%s", "config": { "claims_to_verify": ["exp"] } }`, JwtPlugin)
	_, err := SendPostRequest(apiGwPluginUrl, []byte(jwtConfig), a.tlsCfg)
	if err != nil {
		log.Error("Register API GW jwt plugin failed.", err)
	}
}

// ApiGwDelRoute delete application route from api gateway
func (a *ApiGwIf) ApiGwDelRoute(serName string) {
	apiGwRouteUrl := a.baseURL + serviceUrl + serName + routeUrl + serName
	req := httplib.Delete(apiGwRouteUrl)
	str, err := req.String()
	if err != nil {
		log.Error("Failed to delete API gateway route.", err)
	}
	log.Infof("Deleted service route from API Gateway(result=%s).", str)
}

// CleanUpApiGwEntry deleted the api gateway entries for a service name
func (a *ApiGwIf) CleanUpApiGwEntry(serName string) {
	if serName == "" {
		return
	}
	// delete service route from apiGw
	a.DeleteApiGwRoute(serName)
	// delete service plugin from apiGw
	a.DeleteJwtPlugin(serName)
	// delete service from apiGw
	a.DeleteApiGwService(serName)
}
