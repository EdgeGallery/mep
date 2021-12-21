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
	"crypto/x509"
	"errors"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/astaxie/beego/httplib"
)

// TLSConfig create tls configuration
func TLSConfig(crtName string, skipInsecureVerify bool) (*tls.Config, error) {
	appConfig, err := GetAppConfig()
	if err != nil {
		log.Error("Get app config error.", nil)
		return nil, err
	}
	certNameConfig := appConfig[crtName]
	if len(certNameConfig) == 0 {
		log.Errorf(nil, "Certificate(%s) path doesn't available in the app config.", crtName)
		return nil, errors.New("cert name configuration is not set")
	}

	crt, err := ioutil.ReadFile(certNameConfig)
	if err != nil {
		log.Error("Unable to read certificate.", nil)
		return nil, err
	}

	rootCAs := x509.NewCertPool()
	ok := rootCAs.AppendCertsFromPEM(crt)
	if !ok {
		log.Error("Failed to decode the certificate file.", nil)
		return nil, errors.New("failed to decode cert file")
	}

	serverName := appConfig["server_name"]
	serverNameIsValid, validateServerNameErr := validateServerName(serverName)
	if validateServerNameErr != nil || !serverNameIsValid {
		log.Error("Validate server name error.", nil)
		return nil, validateServerNameErr
	}
	sslCiphers := appConfig["ssl_ciphers"]
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
			log.Error("Not a recommended cipher suite.", nil)
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
	log.Infof("New rest request url: %s, method: %s.", url, method)
	log.Debugf("Rest body: %s.", string(jsonStr))
	req := generateReq(url, method, jsonStr, tlsCfg)

	res, err := req.String()
	if err != nil {
		log.Errorf(nil, "Rest request failed on server(result: %s).", res)
		return res, err
	}
	log.Infof("Rest request completed(result: %s).", res)
	return res, nil
}

//SendRequest rest request return response object
func SendRequestRes(url string, method string, jsonStr []byte, tlsCfg *tls.Config) (string, error) {
	log.Infof("New rest request url: %s, method: %s.", url, method)
	log.Debugf("Rest body: %s.", string(jsonStr))
	req := generateReq(url, method, jsonStr, tlsCfg)

	resp, err := req.Response()
	if err != nil {
		log.Errorf(nil, "Rest request failed on server(result: %s).", resp.Status)
		return resp.Status, err
	}

	statusCode := resp.StatusCode
	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		log.Errorf(nil, "Rest request failed on server(statusCode: %d).", resp.StatusCode)
		return resp.Status, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf(nil, "Rest request failed on server(respBody: %s).", resp.Status)
		return resp.Status, err
	}
	log.Infof("Rest request completed(result: %s).", string(respBody))
	return string(respBody), nil
}

func generateReq(url string, method string, jsonStr []byte, tlsCfg *tls.Config) *httplib.BeegoHTTPRequest {
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
	req.Header(XRealIp, GetLocalIP())
	return req
}

// GetLocalIP get local ip
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, address := range addrs {
		// Check the IP address to determine whether to loop back the address
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
