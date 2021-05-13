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

// Package main
package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/astaxie/beego/httplib"
	"mepauth/models"
	"mepauth/routers"
	"strings"

	log "github.com/sirupsen/logrus"

	"mepauth/util"
)

const servicesPath string = "/services"
const configFormat string = `{ "name": "%s", "config": %s }`

// API gateway initializer
type apiGwInitializer struct {
	tlsConfig *tls.Config
}

func (i *apiGwInitializer) InitAPIGateway(trustedNetworks *[]byte) error {
	apiGwUrl, getApiGwUrlErr := util.GetAPIGwURL()
	if getApiGwUrlErr != nil {
		log.Error("Failed to get api gateway url")
		return getApiGwUrlErr
	}
	err := i.SetApiGwConsumer(apiGwUrl)
	if err != nil {
		return err
	}
	err = i.SetupKongMepServer(apiGwUrl)
	if err != nil {
		return err
	}

	err = i.SetupKongMepAuth(apiGwUrl, trustedNetworks)
	if err != nil {
		return err
	}

	err = i.SetupHttpLogPlugin(apiGwUrl)
	if err != nil {
		return err
	}

	log.Info("Initialization of consumer is successful")
	return nil
}

func (i *apiGwInitializer) SetupHttpLogPlugin(apiGwUrl string) error {
	// enable global http log plugin
	pluginUrl := apiGwUrl + util.PluginPath
	err := i.SendPostRequest(pluginUrl, []byte(models.GetHttpLogPluginData()))
	if err != nil {
		log.Error("Enable http log plugin failed")
		return err
	}
	return nil
}

func (i *apiGwInitializer) SetApiGwConsumer(apiGwUrl string) error {
	// add mepauth consumer to kong
	consumerUrl := apiGwUrl + "/consumers"
	jsonConsumerByte := []byte(fmt.Sprintf(`{ "username": "%s" }`, util.MepAppJwtName))
	err := i.SendPostRequest(consumerUrl, jsonConsumerByte)
	if err != nil {
		log.Error("Consumer initialization failed")
		return err
	}

	mepAuthKey := util.GetAppConfig("mepauth_key")
	if len(mepAuthKey) == 0 {
		msg := "mep auth key configuration is not set"
		log.Error(msg)
		return errors.New(msg)
	}
	// add jwt plugin to mepauth consumer
	kongJwtUrl := consumerUrl + "/" + util.MepAppJwtName + "/jwt"
	jwtPublicKey, err := util.GetPublicKey()
	if err != nil {
		return err
	}
	kongJwtByte := []byte(fmt.Sprintf(`{ "algorithm": "RS512", "key": "%s", "rsa_public_key": "%s" }`,
		mepAuthKey, string(jwtPublicKey)))
	err = i.SendPostRequest(kongJwtUrl, kongJwtByte)
	if err != nil {
		log.Error("Failed while adding consumer token.")
		return err
	}
	return nil
}

func (i *apiGwInitializer) SetupKongMepServer(apiGwUrl string) error {
	// add mep server service and route to kong.
	// since mep is also in the same pos, same ip address will work
	mepServerHost := util.GetAppConfig("mepserver_host")
	if len(mepServerHost) == 0 {
		msg := "mep server host configuration is not set"
		log.Error(msg)
		return errors.New(msg)
	}
	mepServerPort := util.GetAppConfig("mepserver_port")
	if len(mepServerPort) == 0 {
		msg := "mep server port configuration is not set"
		log.Error(msg)
		return errors.New(msg)
	}
	err := i.AddServiceRoute(util.MepserverName, []string{util.MepServerServiceMgmt, util.MepServerAppSupport},
		"https://"+mepServerHost+":"+mepServerPort, false)
	if err != nil {
		log.Error("Add mep server route to kong failed")
		return err
	}
	// enable mep server jwt plugin
	mepServerPluginUrl := apiGwUrl + servicesPath + "/" + util.MepserverName + util.PluginPath
	jwtConfig := fmt.Sprintf(`{ "name": "%s", "config": { "claims_to_verify": ["exp"] } }`, util.JwtPlugin)
	err = i.SendPostRequest(mepServerPluginUrl, []byte(jwtConfig))
	if err != nil {
		log.Error("Enable mep server jwt plugin failed")
		return err
	}
	// enable mep server appid-header plugin
	err = i.SendPostRequest(mepServerPluginUrl, []byte(fmt.Sprintf(`{ "name": "%s" }`, util.AppidPlugin)))
	if err != nil {
		log.Error("Enable mep server appid-header plugin failed.")
		return err
	}
	// enable mep server pre-function plugin
	err = i.SendPostRequest(mepServerPluginUrl, []byte(fmt.Sprintf(configFormat,
		util.PreFunctionPlugin, util.MepserverPreFunctionConf)))
	if err != nil {
		log.Error("Enable mep server pre-function plugin failed.")
		return err
	}
	// enable mep server rate-limiting plugin
	ratePluginReq := []byte(fmt.Sprintf(configFormat,
		util.RateLimitPlugin, util.MepserverRateConf))
	err = i.SendPostRequest(mepServerPluginUrl, ratePluginReq)
	if err != nil {
		log.Error("Enable mep server appid-header plugin failed")
		return err
	}
	// enable mep server response-transformer plugin
	respPluginReq := []byte(util.ResponseTransformerConf)
	err = i.SendPostRequest(mepServerPluginUrl, respPluginReq)
	if err != nil {
		log.Error("Enable mep server response-transformer plugin failed")
		return err
	}
	return nil
}

func (i *apiGwInitializer) SetupKongMepAuth(apiGwURL string, trustedNetworks *[]byte) error {
	// add mep auth service and route to kong
	httpsPort := util.GetAppConfig("HttpsPort")
	if len(httpsPort) == 0 {
		msg := "https port configuration is not set"
		log.Error(msg)
		return errors.New(msg)
	}
	// Since kong is also deployed in same pod, it can reach by the ip address
	mepAuthHost := util.GetAppConfig("HTTPSAddr")
	if len(mepAuthHost) == 0 {
		msg := "mep auth host configuration is not set"
		log.Error(msg)
		return errors.New(msg)
	}
	mepAuthURL := "https://" + mepAuthHost + ":" + httpsPort
	err := i.AddServiceRoute(util.MepauthName, []string{routers.AuthTokenPath, routers.AppManagePath}, mepAuthURL, false)
	if err != nil {
		log.Error("Add mep server route to kong failed.")
		return err
	}
	// enable mep auth rate-limiting plugin
	mepAuthPluginURL := apiGwURL + servicesPath + "/" + util.MepauthName + util.PluginPath
	mepAuthRatePluReq := []byte(fmt.Sprintf(configFormat,
		util.RateLimitPlugin, util.MepauthRateConf))
	err = i.SendPostRequest(mepAuthPluginURL, mepAuthRatePluReq)
	if err != nil {
		log.Error("Enable mep auth appid-header plugin failed.")
		return err
	}
	// enable mep auth response-transformer plugin
	respPluginReq := []byte(util.ResponseTransformerConf)
	err = i.SendPostRequest(mepAuthPluginURL, respPluginReq)
	if err != nil {
		log.Error("Enable mep auth response-transformer plugin failed")
		return err
	}

	if (trustedNetworks != nil) && (len(*trustedNetworks) > 0) {
		trustedNetworksList := strings.Split(string(*trustedNetworks), ";")
		allIpValid, err := util.ValidateIpAndCidr(trustedNetworksList)
		if (err == nil) && allIpValid {
			mepIpRestrict := []byte(fmt.Sprintf(configFormat,
				util.IpRestrictPlugin, i.getTrustedIpList(trustedNetworksList)))
			err = i.SendPostRequest(mepAuthPluginURL, mepIpRestrict)
			if err != nil {
				log.Error("Ip restriction failed")
				return err
			}
		} else {
			log.Info("trusted list input is not valid, allowing all the networks")
		}
	}
	return nil
}

func (i *apiGwInitializer) getTrustedIpList(trustedNetworksList []string) string {
	var ipcidrList string
	ipList := `{ "whitelist": [`
	for _, ipcidr := range trustedNetworksList {
		ipcidrList += `"` + ipcidr + `",`
	}
	if strings.HasSuffix(ipcidrList, ",") {
		ipcidrList = ipcidrList[:len(ipcidrList)-len(",")]
	}
	ipList = ipList + ipcidrList + `] }`
	return ipList
}

func (i *apiGwInitializer) AddServiceRoute(serviceName string, servicePaths []string, targetURL string, needStripPath bool) error {
	apiGwURL, getApiGwUrlErr := util.GetAPIGwURL()
	if getApiGwUrlErr != nil {
		log.Error("Failed to get api gateway url.")
		return getApiGwUrlErr
	}

	paths := strings.Join(servicePaths, `", "`)

	kongServiceURL := apiGwURL + servicesPath
	serviceReq := []byte(fmt.Sprintf(`{ "url": "%s", "name": "%s" }`,
		targetURL, serviceName))
	errMepService := i.SendPostRequest(kongServiceURL, serviceReq)
	if errMepService != nil {
		log.Error("Add " + serviceName + " service to kong failed.")
		return errMepService
	}

	kongRouteURL := apiGwURL + servicesPath + "/" + serviceName + "/routes"

	preserveHost := ""
	if serviceName == util.MepauthName {
		preserveHost = ` ,"preserve_host": true`
	}
	stripPath := ""
	if !needStripPath {
		stripPath = ` ,"strip_path": false`
	}

	reqStr := `{ "paths": ["%s"], "name": "%s"` + preserveHost + stripPath + `}`
	routeReq := []byte(fmt.Sprintf(reqStr, paths, serviceName))

	err := i.SendPostRequest(kongRouteURL, routeReq)
	if err != nil {
		log.Error("Add " + serviceName + " route to kong failed.")
		return err
	}
	return nil
}

// Send post request
func (i *apiGwInitializer) SendPostRequest(consumerURL string, jsonStr []byte) error {

	req := httplib.Post(consumerURL)
	req.Header(util.ContentType, util.JsonUtf8)
	req.SetTLSClientConfig(i.tlsConfig)
	req.Body(jsonStr)
	_, err := req.String()
	if err != nil {
		log.Error("send Post Request Failed")
		return err
	}
	return nil
}