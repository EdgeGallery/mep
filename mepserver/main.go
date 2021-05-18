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

// Package works for the mep server entry
package main

import (
	"errors"
	"mepserver/mp1/plans"
	"os"

	_ "mepserver/common/tls"
	"mepserver/common/util"
	_ "mepserver/mm5"
	_ "mepserver/mm5/plans"
	_ "mepserver/mp1"
	_ "mepserver/mp1/event"
	_ "mepserver/mp1/uuid"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/server"
	_ "github.com/apache/servicecomb-service-center/server/bootstrap"
	_ "github.com/apache/servicecomb-service-center/server/init"
)

var rootKey = "ROOT_KEY"
var tlsKey = "TLS_KEY"

func main() {

	err := initialEncryptComponent()
	if err != nil {
		log.Errorf(err, "Initial encrypt component failed.")
		return
	}
	if !util.IsFileOrDirExist(util.EncryptedCertSecFilePath) {
		err := encryptCertPwd()
		if err != nil {
			log.Errorf(err, "Certificate password encryption or validation failed.")
			return
		}

	}
	go plans.HeartbeatProcess()
	util.ApiGWInterface = util.NewApiGwIf()
	server.Run()
}

func encryptCertPwd() error {
	pwd := []byte(os.Getenv(tlsKey))
	if len(os.Getenv(tlsKey)) == 0 {
		err := errors.New("tls password is not set in environment variable")
		log.Errorf(err, "Read certificate password failed.")
		return err
	}
	os.Unsetenv(tlsKey)
	_, verifyErr := util.ValidatePassword(&pwd)
	if verifyErr != nil {
		log.Errorf(verifyErr, "Certificate password complexity validation failed.")
		return verifyErr
	}
	encryptCertPwdErr := util.EncryptAndSaveCertPwd(&pwd)
	if encryptCertPwdErr != nil {
		log.Errorf(encryptCertPwdErr, "Encrypt certificate password failed.")
		return encryptCertPwdErr
	}
	return nil
}

func initialEncryptComponent() error {
	keyComponentFromUser := []byte(os.Getenv(rootKey))
	if len(os.Getenv(rootKey)) == 0 {
		err := errors.New("root key is not present inside environment variable")
		log.Errorf(err, "Read root key component failed.")
		return err
	}
	_ = os.Unsetenv(rootKey)

	verifyErr := util.ValidateKeyComponentUserInput(&keyComponentFromUser)
	if verifyErr != nil {
		log.Errorf(verifyErr, "Root key component validation failed.")
		return verifyErr
	}
	util.KeyComponentFromUserStr = &keyComponentFromUser

	err := util.InitRootKeyAndWorkKey()
	if err != nil {
		log.Errorf(err, "Failed to initialize root key and work key.")
		return err
	}
	return nil
}
