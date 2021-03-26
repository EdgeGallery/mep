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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"

	_ "mepauth/config"
	"mepauth/controllers"
	_ "mepauth/models"
	_ "mepauth/routers"
	"mepauth/util"
)

func initDb() {
	orm.RegisterDriver("postgres", orm.DRPostgres)
	dataSource := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s",
		util.GetAppConfig("db_user"),
		util.GetAppConfig("db_passwd"),
		util.GetAppConfig("db_name"),
		util.GetAppConfig("db_host"),
		util.GetAppConfig("db_port"),
		util.GetAppConfig("db_sslmode"))
	orm.RegisterDataBase("default", "postgres", dataSource)
	orm.RunSyncdb("default", false, true)
}

func scanConfig(r io.Reader) (util.AppConfigProperties, error) {
	config := util.AppConfigProperties{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()
		if bytes.Contains(line, []byte("=")) {
			keyVal := bytes.Split(line, []byte("="))
			key := bytes.TrimSpace(keyVal[0])
			val := bytes.TrimSpace(keyVal[1])
			config[string(key)] = &val
		}
	}
	return config, scanner.Err()
}

func readPropertiesFile(filename string) (util.AppConfigProperties, error) {

	if len(filename) == 0 {
		return nil, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		log.Error("Failed to open the file.")
		return nil, err
	}
	defer file.Close()
	config, err := scanConfig(file)
	if err != nil {
		log.Error("Failed to read the file.")
		clearAppConfigOnExit(config)
		return nil, err
	}
	return config, nil
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Main process panic: %v \n %s", r, string(debug.Stack()))
			time.Sleep(5 * time.Second)
		}
	}()
	log.Info("mepauth start")
	// Initialize database
	initDb()

	log.Info("mepauth start")
	configFilePath := filepath.FromSlash("/usr/mep/mprop/mepauth.properties")
	log.Info("readPropertiesFile")
	appConfig, err := readPropertiesFile(configFilePath)
	if err != nil {
		log.Error("Failed to read the config parameters from properties file")
		time.Sleep(5 * time.Second)
		return
	}
	// Clearing all the sensitive information on exit for error case. For the success case
	// function handling the sensitive information will clear after the usage.
	// clean of mepauth.properties file use kubectl apply -f empty-mepauth-prop.yaml
	defer clearAppConfigOnExit(appConfig)
	log.Info("ValidateInputArgs")
	validation := util.ValidateInputArgs(appConfig)
	if !validation {
		log.Error("input validation failed.")
		time.Sleep(5 * time.Second)
		return
	}
	keyComponentUserStr := appConfig["KEY_COMPONENT"]
	log.Info("ValidateKeyComponentUserInput")
	err = util.ValidateKeyComponentUserInput(keyComponentUserStr)
	if err != nil {
		log.Error("ValidateKeyComponentUserInput failed.")
		time.Sleep(5 * time.Second)
		return
	}
	util.KeyComponentFromUserStr = keyComponentUserStr

	log.Info("doInitialization")
	initSuccess := doInitialization(appConfig["TRUSTED_LIST"])
	if !initSuccess {
		log.Error("doInitialization failed.")
		time.Sleep(5 * time.Second)
		return
	}
	log.Info("EncryptAndSaveJwtPwd")
	err = util.EncryptAndSaveJwtPwd(appConfig["JWT_PRIVATE_KEY"])
	if err != nil {
		log.Error("Failed to encrypt and save jwt private key password.")
		time.Sleep(5 * time.Second)
		return
	}
	log.Info("ConfigureAkAndSk")
	err = controllers.ConfigureAkAndSk(string(*appConfig["APP_INST_ID"]),
		string(*appConfig["ACCESS_KEY"]), appConfig["SECRET_KEY"])
	if err != nil {
		log.Error("failed to configure ak sk values")
		time.Sleep(5 * time.Second)
		return
	}
	log.Info("TLSConfig")
	tlsConf, err := util.TLSConfig("HTTPSCertFile")
	if err != nil {
		log.Error("failed to config tls for beego")
		time.Sleep(5 * time.Second)
		return
	}
	log.Info("InitAuthInfoList")
	controllers.InitAuthInfoList()

	time.Sleep(5 * time.Second)
	log.Info("beego will start")
	beego.BeeApp.Server.TLSConfig = tlsConf
	beego.ErrorController(&controllers.ErrorController{})
	log.Info("before beego run")
	beego.Run()
	log.Info("after beego run")
}

func clearAppConfigOnExit(appConfig util.AppConfigProperties) {
	for _, element := range appConfig {
		util.ClearByteArray(*element)
	}
}

func doInitialization(trustedNetworks *[]byte) bool {
	log.Info("initAPIGateway")
	err := initAPIGateway(trustedNetworks)
	if err != nil {
		log.Error("Failed to init api gateway.")
		time.Sleep(5 * time.Second)
		return false
	}
	log.Info("InitRootKeyAndWorkKey")
	err = util.InitRootKeyAndWorkKey()
	if err != nil {
		log.Error("Failed to init root key and work key.")
		time.Sleep(5 * time.Second)
		return false
	}
	return true
}
