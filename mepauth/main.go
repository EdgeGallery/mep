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
	"bufio"
	"bytes"
	log "github.com/sirupsen/logrus"
	"mepauth/controllers"
	"os"
	"path/filepath"

	"github.com/astaxie/beego"

	_ "mepauth/config"
	_ "mepauth/models"
	_ "mepauth/routers"
	"mepauth/util"
)

func readPropertiesFile(filename string) (util.AppConfigProperties, error) {
	config := util.AppConfigProperties{}

	if len(filename) == 0 {
		return config, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		log.Error("Failed to open the file.")
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if bytes.Contains(line, []byte("=")) {
			keyVal := bytes.Split(line, []byte("="))
			key := bytes.TrimSpace(keyVal[0])
			val := bytes.TrimSpace(keyVal[1])
			config[string(key)] = &val
		}
	}
	if err := scanner.Err(); err != nil {
		log.Error("Failed to read the file.")
		clearAppConfigOnExit(config)
		return nil, err
	}
	return config, nil
}

func main() {
	configFilePath := filepath.FromSlash("/usr/mep/mepauth.properties")
	appConfig, err := readPropertiesFile(configFilePath)
	if err != nil {
		log.Error("Failed to read the config parameters from properties file")
		return
	}
	// Clearing all the sensitive information on exit for error case. For the success case
	// function handling the sensitive information will clear after the usage.
	defer clearAppConfigOnExit(appConfig)
	err = os.Truncate(configFilePath, 0)
	if err != nil {
		log.Error("Failed to clear the config file")
		return
	}
	validation := util.ValidateInputArgs(appConfig)
	if !validation {
		return
	}
	keyComponentUserStr := appConfig["KEY_COMPONENT"]
	err = util.ValidateKeyComponentUserInput(keyComponentUserStr)
	if err != nil {
		log.Error("input validation failed.")
		return
	}
	util.KeyComponentFromUserStr = keyComponentUserStr

	initSuccess := doInitialization(appConfig["TRUSTED_LIST"])
	if !initSuccess {
		return
	}

	err = util.EncryptAndSaveJwtPwd(appConfig["JWT_PRIVATE_KEY"])
	if err != nil {
		log.Error("Failed to encrypt and save jwt private key password.")
		return
	}
	err = controllers.ConfigureAkAndSk(string(*appConfig["APP_INST_ID"]),
		string(*appConfig["ACCESS_KEY"]), appConfig["SECRET_KEY"])
	if err != nil {
		log.Error("failed to configure ak sk values")
		return
	}
	tlsConf, err := util.TLSConfig("HTTPSCertFile")
	if err != nil {
		log.Error("failed to config tls for beego")
		return
	}

	controllers.InitAuthInfoList()

	beego.BeeApp.Server.TLSConfig = tlsConf
	beego.ErrorController(&controllers.ErrorController{})
	beego.Run()
}

func clearAppConfigOnExit(appConfig util.AppConfigProperties) {
	for _, element := range appConfig {
		util.ClearByteArray(*element)
	}
}


func doInitialization(trustedNetworks *[]byte) bool {
	err := initAPIGateway(trustedNetworks)
	if err != nil {
		log.Error("Failed to init api gateway.")
		return false
	}
	err = util.InitRootKeyAndWorkKey()
	if err != nil {
		log.Error("Failed to init root key and work key.")
		return false
	}
	return true
}