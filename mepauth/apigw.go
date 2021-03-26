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

package main

import (
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"mepauth/util"
)

const ServicesPath string = "/services"

func initAPIGateway(trustedNetworks *[]byte) error {
	log.Info("Inside initAPIGateway")
	log.Info("GetAPIGwURL")
	apiGwUrl, getApiGwUrlErr := util.GetAPIGwURL()
	if getApiGwUrlErr != nil {
		log.Error("Failed to get api gateway url")
		return getApiGwUrlErr
	}
	log.Infof("GetAPIGwURL: %s", apiGwUrl)
	log.Info("setApiGwConsumer")
	err := setApiGwConsumer(apiGwUrl)
	if err != nil {
		log.Error("Failed to setApiGwConsumer")
		return err
	}
	log.Info("setupKongMepServer")
	err = setupKongMepServer(apiGwUrl)
	if err != nil {
		log.Error("Failed to setupKongMepServer")
		return err
	}

	log.Info("setupKongMepAuth")
	err = setupKongMepAuth(apiGwUrl, trustedNetworks)
	if err != nil {
		log.Error("Failed to setupKongMepAuth")
		return err
	}

	log.Info("Initialization of consumer is successful")
	return nil
}

func setApiGwConsumer(apiGwUrl string) error {
	log.Info("Inside setApiGwConsumer")
	// add mepauth consumer to kong
	consumerUrl := apiGwUrl + "/consumers"
	jsonConsumerByte := []byte(fmt.Sprintf(`{ "username": "%s" }`, util.MepAppJwtName))
	log.Info("SendPostRequest")
	err := util.SendPostRequest(consumerUrl, jsonConsumerByte)
	if err != nil {
		log.Error("Consumer initialization failed")
		log.Errorf("Post error: %s", err.Error())
		return err
	}

	log.Info("GetAppConfig")
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
	log.Info("SendPostRequest 2")
	err = util.SendPostRequest(kongJwtUrl, kongJwtByte)
	if err != nil {
		log.Error("Failed while adding consumer token.")
		return err
	}
	return nil
}

func setupKongMepServer(apiGwUrl string) error {
	log.Info("Inside setupKongMepServer")
	// add mep server service and route to kong.
	// since mep is also in the same pos, same ip address will work
	log.Info("GetAppConfig 1")
	mepServerHost := util.GetAppConfig("mepserver_host")
	if len(mepServerHost) == 0 {
		msg := "mep server host configuration is not set"
		log.Error(msg)
		return errors.New(msg)
	}
	log.Info("GetAppConfig 2")
	mepServerPort := util.GetAppConfig("mepserver_port")
	if len(mepServerPort) == 0 {
		msg := "mep server port configuration is not set"
		log.Error(msg)
		return errors.New(msg)
	}
	log.Info("addServiceRoute")
	err := addServiceRoute(util.MepserverName, "https://"+mepServerHost+":"+mepServerPort+"/"+util.MepserverRootPath)
	if err != nil {
		log.Error("Add mep server route to kong failed")
		return err
	}
	// enable mep server jwt plugin
	mepserverPluginUrl := apiGwUrl + ServicesPath + "/" + util.MepserverName + "/plugins"
	jwtConfig := fmt.Sprintf(`{ "name": "%s", "config": { "claims_to_verify": ["exp"] } }`, util.JwtPlugin)
	log.Info("SendPostRequest 1")
	err = util.SendPostRequest(mepserverPluginUrl, []byte(jwtConfig))
	if err != nil {
		log.Error("Enable mep server jwt plugin failed")
		return err
	}
	// enable mep server appid-header plugin
	log.Info("SendPostRequest 2")
	err = util.SendPostRequest(mepserverPluginUrl, []byte(fmt.Sprintf(`{ "name": "%s" }`, util.AppidPlugin)))
	if err != nil {
		log.Error("Enable mep server appid-header plugin failed.")
		return err
	}
	// enable mep server pre-function plugin
	log.Info("SendPostRequest 3")
	err = util.SendPostRequest(mepserverPluginUrl, []byte(fmt.Sprintf(`{ "name": "%s", "config": %s }`,
		util.PreFunctionPlugin, util.MepserverPreFunctionConf)))
	if err != nil {
		log.Error("Enable mep server pre-function plugin failed.")
		return err
	}
	// enable mep server rate-limiting plugin
	ratePluginReq := []byte(fmt.Sprintf(`{ "name": "%s", "config": %s }`,
		util.RateLimitPlugin, util.MepserverRateConf))
	log.Info("SendPostRequest 4")
	err = util.SendPostRequest(mepserverPluginUrl, ratePluginReq)
	if err != nil {
		log.Error("Enable mep server appid-header plugin failed")
		return err
	}
	// enable mep server response-transformer plugin
	respPluginReq := []byte(util.ResponseTransformerConf)
	log.Info("SendPostRequest 5")
	err = util.SendPostRequest(mepserverPluginUrl, respPluginReq)
	if err != nil {
		log.Error("Enable mep server response-transformer plugin failed")
		return err
	}
	return nil
}

func setupKongMepAuth(apiGwURL string, trustedNetworks *[]byte) error {
	// add mep auth service and route to kong
	log.Info("inside setupKongMepAuth")
	log.Info("GetAppConfig 1")
	httpsPort := util.GetAppConfig("HttpsPort")
	if len(httpsPort) == 0 {
		msg := "https port configuration is not set"
		log.Error(msg)
		return errors.New(msg)
	}
	// Since kong is also deployed in same pod, it can reach by the ip address
	log.Info("GetAppConfig 2")
	mepAuthHost := util.GetAppConfig("HTTPSAddr")
	if len(mepAuthHost) == 0 {
		msg := "mep auth host configuration is not set"
		log.Error(msg)
		return errors.New(msg)
	}
	mepAuthURL := "https://" + mepAuthHost + ":" + httpsPort
	log.Info("addServiceRoute")
	err := addServiceRoute(util.MepauthName, mepAuthURL)
	if err != nil {
		log.Error("Add mep server route to kong failed.")
		return err
	}
	// enable mep auth rate-limiting plugin
	mepAuthPluginURL := apiGwURL + ServicesPath + "/" + util.MepauthName + "/plugins"
	mepAuthRatePluReq := []byte(fmt.Sprintf(`{ "name": "%s", "config": %s }`,
		util.RateLimitPlugin, util.MepauthRateConf))
	log.Info("SendPostRequest 1")
	err = util.SendPostRequest(mepAuthPluginURL, mepAuthRatePluReq)
	if err != nil {
		log.Error("Enable mep auth appid-header plugin failed.")
		return err
	}
	// enable mep auth response-transformer plugin
	respPluginReq := []byte(util.ResponseTransformerConf)
	log.Info("SendPostRequest 2")
	err = util.SendPostRequest(mepAuthPluginURL, respPluginReq)
	if err != nil {
		log.Error("Enable mep auth response-transformer plugin failed")
		return err
	}

	log.Info("Checking trustedNetworks")

	if (trustedNetworks != nil) && (len(*trustedNetworks) > 0) {
		trustedNetworksList := strings.Split(string(*trustedNetworks), ";")
		log.Info("ValidateIpAndCidr")
		allIpValid, err := util.ValidateIpAndCidr(trustedNetworksList)
		if (err == nil) && allIpValid {
			mepIpRestrict := []byte(fmt.Sprintf(`{ "name": "%s", "config": %s }`,
				util.IpRestrictPlugin, getTrustedIpList(trustedNetworksList)))
			log.Info("SendPostRequest 3")
			err = util.SendPostRequest(mepAuthPluginURL, mepIpRestrict)
			if err != nil {
				log.Error("Ip restriction failed")
				return err
			}
		} else {
			log.Info("trusted list input is not valid, allowing all the networks")
		}
	}
	log.Info("Finished trustedNetworks")
	return nil
}

func getTrustedIpList(trustedNetworksList []string) string {
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

func addServiceRoute(serviceName string, targetURL string) error {
	apiGwURL, getApiGwUrlErr := util.GetAPIGwURL()
	if getApiGwUrlErr != nil {
		log.Error("Failed to get api gateway url.")
		return getApiGwUrlErr
	}

	kongServiceURL := apiGwURL + ServicesPath
	serviceReq := []byte(fmt.Sprintf(`{ "url": "%s", "name": "%s" }`,
		targetURL, serviceName))
	errMepService := util.SendPostRequest(kongServiceURL, serviceReq)
	if errMepService != nil {
		log.Error("Add " + serviceName + " service to kong failed.")
		return errMepService
	}

	kongRouteURL := apiGwURL + ServicesPath + "/" + serviceName + "/routes"
	preserveHost := ""
	if serviceName == util.MepauthName {
		preserveHost = ` ,"preserve_host": true`
	}
	reqStr := `{ "paths": ["/%s"], "name": "%s"` + preserveHost + `}`
	routeReq := []byte(fmt.Sprintf(reqStr, serviceName, serviceName))

	err := util.SendPostRequest(kongRouteURL, routeReq)
	if err != nil {
		log.Error("Add " + serviceName + " route to kong failed.")
		return err
	}
	return nil
}
