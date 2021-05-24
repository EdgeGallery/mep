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

// Package util methods
package util

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/astaxie/beego/httplib"
)

// TLSConfig create tls configuration
func TLSConfig(crtName string, skipInsecureVerify bool) (*tls.Config, error) {
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
	serverNameIsValid, validateServerNameErr := validateServerName(serverName)
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
		RootCAs:            rootCAs,
		ServerName:         serverName,
		MinVersion:         tls.VersionTLS12,
		CipherSuites:       cipherSuites,
		InsecureSkipVerify: skipInsecureVerify,
	}, nil
}

// Validate Server Name
func validateServerName(serverName string) (bool, error) {
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

//GetAppConfig get the app-config from the configuration file
func GetAppConfig() (AppConfigProperties, error) {
	// read app.conf file to AppConfigProperties object
	cfgPath := filepath.FromSlash(ConfigFilePath)
	appConfig, err := readPropertiesFile(cfgPath)
	return appConfig, err
}

// SendPostRequest sends post request
func SendPostRequest(url string, jsonStr []byte, tlsCfg *tls.Config) (string, error) {
	return SendRequest(url, PostMethod, jsonStr, tlsCfg)
}

// SendPutRequest sends put request
func SendPutRequest(url string, jsonStr []byte, tlsCfg *tls.Config) (string, error) {
	return SendRequest(url, PutMethod, jsonStr, tlsCfg)
}

// SendDelRequest Sends delete request
func SendDelRequest(url string, tlsCfg *tls.Config) (string, error) {
	return SendRequest(url, DeleteMethod, nil, tlsCfg)
}

// SendDelRequest Sends get request
func SendGetRequest(url string, tlsCfg *tls.Config) (string, error) {
	return SendRequest(url, GetMethod, nil, tlsCfg)
}

//SendRequest rest request
func SendRequest(url string, method string, jsonStr []byte, tlsCfg *tls.Config) (string, error) {
	log.Infof("SendRequest url: %s, method: %s, jsonStr: %s", url, method, jsonStr)
	var req *httplib.BeegoHTTPRequest
	switch method {
	case PostMethod:
		req = httplib.Post(url)
		req.Header("Content-Type", "application/json; charset=utf-8")
		req.Body(jsonStr)
	case PutMethod:
		req = httplib.Put(url)
		req.Header("Content-Type", "application/json; charset=utf-8")
		req.Body(jsonStr)
	case DeleteMethod:
		req = httplib.Delete(url)
	default:
		req = httplib.Get(url)
	}

	req.SetTLSClientConfig(tlsCfg)

	res, err := req.String()
	if err != nil {
		log.Error("send request failed", nil)
		return res, err
	}
	log.Infof("res=%s", res)
	return res, nil
}
