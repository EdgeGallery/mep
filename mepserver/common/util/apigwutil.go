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

package util

import (
	"crypto/tls"
	"fmt"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/astaxie/beego/httplib"
)

const serviceUrl string = "/services/"

var cipherSuiteMap = map[string]uint16{
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
}

type RouteInfo struct {
	Id      int64   `json:"routeId"`
	AppId   string  `json:"appId"`
	SerInfo SerInfo `orm:"type(json)" json:"serInfo"`
}

type SerInfo struct {
	SerName string   `json:"serName"`
	Uris    []string `json:"uris"`
}

var ApiGWInterface *ApiGwIf

type ApiGwIf struct {
	baseURL string
	tlsCfg  *tls.Config
}

func NewApiGwIf() *ApiGwIf {
	a := &ApiGwIf{}
	baseUrl := a.getApiGwUrl()
	if len(baseUrl) == 0 {
		return nil
	}
	tlsCfg, err := TLSConfig(ApiGwCaCertName, false)
	if err != nil {
		return nil
	}
	a.baseURL = baseUrl
	a.tlsCfg = tlsCfg
	return a
}

func (a *ApiGwIf) getApiGwUrl() string {
	appConfig, err := GetAppConfig()
	if err != nil {
		log.Error("Get App Config failed.", err)
		return ""
	}
	kongUrl := fmt.Sprintf("https://%s:%s", appConfig["apigw_host"], appConfig["apigw_port"])
	return kongUrl

}

func (a *ApiGwIf) AddApiGwService(routeInfo RouteInfo) {
	kongServiceUrl := a.baseURL + "/services"
	serName := routeInfo.SerInfo.SerName
	serUrl := routeInfo.SerInfo.Uris[0]
	jsonStr := []byte(fmt.Sprintf(`{ "url": "%s", "name": "%s" }`, serUrl, serName))
	err := SendPostRequest(kongServiceUrl, jsonStr, a.tlsCfg)
	if err != nil {
		log.Error("failed to add API gateway service", err)
	}
}

func (a *ApiGwIf) AddApiGwRoute(routeInfo RouteInfo) {
	serName := routeInfo.SerInfo.SerName
	kongRouteUrl := a.baseURL + serviceUrl + serName + "/routes"
	jsonStr := []byte(fmt.Sprintf(`{ "paths": ["/%s"], "name": "%s" }`, serName, serName))
	err := SendPostRequest(kongRouteUrl, jsonStr, a.tlsCfg)
	if err != nil {
		log.Error("failed to add API gateway route", err)
	}
}

// enable kong jwt plugin
func (a *ApiGwIf) EnableJwtPlugin(routeInfo RouteInfo) {
	serName := routeInfo.SerInfo.SerName
	kongPluginUrl := a.baseURL + serviceUrl + serName + "/plugins"
	jwtConfig := fmt.Sprintf(`{ "name": "%s", "config": { "claims_to_verify": ["exp"] } }`, JwtPlugin)
	err := SendPostRequest(kongPluginUrl, []byte(jwtConfig), a.tlsCfg)
	if err != nil {
		log.Error("Enable kong jwt plugin failed", err)
	}
}

func (a *ApiGwIf) ApiGwDelRoute(serName string) {
	kongRouteUrl := a.baseURL + serviceUrl + serName + "/routes/" + serName
	req := httplib.Delete(kongRouteUrl)
	str, err := req.String()
	if err != nil {
		log.Error("failed to delete API gateway route", err)
	}
	log.Infof("res=%s", str)
}
