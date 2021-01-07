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

package util

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

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

func GetApigwUrl() string {
	appConfig, err := GetAppConfig()
	if err != nil {
		log.Error("Get App Config failed.", err)
		return ""
	}
	kongUrl := fmt.Sprintf("https://%s:%s", appConfig["apigw_host"], appConfig["apigw_port"])
	return kongUrl

}

func AddApigwService(routeInfo RouteInfo) {
	kongServiceUrl := GetApigwUrl() + "/services"
	serName := routeInfo.SerInfo.SerName
	serUrl := routeInfo.SerInfo.Uris[0]
	jsonStr := []byte(fmt.Sprintf(`{ "url": "%s", "name": "%s" }`, serUrl, serName))
	err := SendPostRequest(kongServiceUrl, jsonStr)
	if err != nil {
		log.Error("failed to add API gateway service", err)
	}
}

func AddApigwRoute(routeInfo RouteInfo) {
	serName := routeInfo.SerInfo.SerName
	kongRouteUrl := GetApigwUrl() + serviceUrl + serName + "/routes"
	jsonStr := []byte(fmt.Sprintf(`{ "paths": ["/%s"], "name": "%s" }`, serName, serName))
	err := SendPostRequest(kongRouteUrl, jsonStr)
	if err != nil {
		log.Error("failed to add API gateway route", err)
	}
}

// enable kong jwt plugin
func EnableJwtPlugin(routeInfo RouteInfo) {
	serName := routeInfo.SerInfo.SerName
	kongPluginUrl := GetApigwUrl() + serviceUrl + serName + "/plugins"
	jwtConfig := fmt.Sprintf(`{ "name": "%s", "config": { "claims_to_verify": ["exp"] } }`, JwtPlugin)
	err := SendPostRequest(kongPluginUrl, []byte(jwtConfig))
	if err != nil {
		log.Error("Enable kong jwt plugin failed", err)
	}
}

func ApigwDelRoute(serName string) {
	kongRouteUrl := GetApigwUrl() + serviceUrl + serName + "/routes/" + serName
	req := httplib.Delete(kongRouteUrl)
	str, err := req.String()
	if err != nil {
		log.Error("failed to delete API gateway route", err)
	}
	log.Infof("res=%s", str)
}

func GetAppConfig() (AppConfigProperties, error) {
	// read app.conf file to AppConfigProperties object
	configFilePath := filepath.FromSlash("/usr/mep/conf/app.conf")
	appConfig, err := readPropertiesFile(configFilePath)
	return appConfig, err
}

// Send post request
func SendPostRequest(url string, jsonStr []byte) error {
	return SendRequest(url, PostMethod, jsonStr)
}

// Send delete request
func SendDelRequest(url string) error {
	return SendRequest(url, DeleteMethod, nil)
}

func SendRequest(url string, method string, jsonStr []byte) error {
	log.Infof("SendRequest url: %s, method: %s, jsonStr: %s", url, method, jsonStr)
	var req *httplib.BeegoHTTPRequest
	switch method {
	case PostMethod:
		req = httplib.Post(url)
		req.Header("Content-Type", "application/json; charset=utf-8")
		req.Body(jsonStr)
	case DeleteMethod:
		req = httplib.Delete(url)
	default:
		req = httplib.Get(url)
	}

	config, err := TLSConfig("apigw_cacert")
	if err != nil {
		log.Error("unable to read certificate", nil)
		return err
	}
	req.SetTLSClientConfig(config)

	res, err := req.String()
	if err != nil {
		log.Error("send request failed", nil)
		return err
	}
	log.Infof("res=%s", res)
	return nil
}

// Update tls configuration
func TLSConfig(crtName string) (*tls.Config, error) {
	appConfig, err := GetAppConfig()
	if err != nil {
		log.Error("get app config error", nil)
		return nil, err
	}
	certNameConfig := string(appConfig[crtName])
	if len(certNameConfig) == 0 {
		log.Error(crtName+" configuration is not set", nil)
		return nil, errors.New("cert name configuration is not set")
	}

	crt, err := ioutil.ReadFile(certNameConfig)
	if err != nil {
		log.Error("unable to read certificate", nil)
		return nil, err
	}

	rootCAs := x509.NewCertPool()
	ok := rootCAs.AppendCertsFromPEM(crt)
	if !ok {
		log.Error("failed to decode cert file", nil)
		return nil, errors.New("failed to decode cert file")
	}

	serverName := string(appConfig["server_name"])
	serverNameIsValid, validateServerNameErr := ValidateServerName(serverName)
	if validateServerNameErr != nil || !serverNameIsValid {
		log.Error("validate server name error", nil)
		return nil, validateServerNameErr
	}
	sslCiphers := string(appConfig["ssl_ciphers"])
	if len(sslCiphers) == 0 {
		return nil, errors.New("TLS cipher configuration is not recommended or invalid")
	}
	cipherSuites := getCipherSuites(sslCiphers)
	if cipherSuites == nil {
		return nil, errors.New("TLS cipher configuration is not recommended or invalid")
	}
	return &tls.Config{
		RootCAs:      rootCAs,
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS12,
		CipherSuites: cipherSuites,
	}, nil
}

// Validate Server Name
func ValidateServerName(serverName string) (bool, error) {
	if len(serverName) > maxHostNameLen {
		return false, errors.New("server or host name validation failed")
	}
	return regexp.MatchString(ServerNameRegex, serverName)
}

func getCipherSuites(sslCiphers string) []uint16 {
	cipherSuiteArr := make([]uint16, 0, 5)
	cipherSuiteNameList := strings.Split(sslCiphers, ",")
	for _, cipherName := range cipherSuiteNameList {
		cipherName = strings.TrimSpace(cipherName)
		if len(cipherName) == 0 {
			continue
		}
		mapValue, ok := cipherSuiteMap[cipherName]
		if !ok {
			log.Error("not recommended cipher suite", nil)
			return nil
		}
		cipherSuiteArr = append(cipherSuiteArr, mapValue)
	}
	if len(cipherSuiteArr) > 0 {
		return cipherSuiteArr
	}
	return nil
}
