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

// Package works for the mep server entry
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/server"
	_ "github.com/apache/servicecomb-service-center/server/bootstrap"
	_ "github.com/apache/servicecomb-service-center/server/init"

	"golang.org/x/crypto/ssh/terminal"

	_ "mepserver/mp1"
	"mepserver/mp1/util"
	_ "mepserver/mp1/uuid"
)

func main() {

	pwd, err := readPassword("Please input tls certificates password: ")
	if err != nil {
		log.Errorf(err, "read password failed")
		return
	}
	confirm, err := readPassword("Confirm the password: ")
	if err != nil || !bytes.Equal(pwd, confirm) {
		log.Errorf(err, "confirm password failed")
		return
	}
	_, verifyErr := util.ValidatePassword(&pwd)
	if verifyErr != nil {
		log.Errorf(verifyErr, "Certificate password complexity validation failed.")
		return
	}

	encryptCertPwd := encryptPass(&pwd)
	wErr := ioutil.WriteFile(util.Cert_Pwd_Path, encryptCertPwd, 0600)
	if wErr != nil {
		log.Errorf(wErr, "write encrypt cert pwd failed")
	}

	server.Run()
}

func encryptPass(pwd *[]byte) []byte {

	// clear password
	for i := 0; i < len(*pwd); i++ {
		(*pwd)[i] = 0
	}
	// encypt password
	encryptPass := []byte("encryptPass")
	return encryptPass
}

func readPassword(prompt string) ([]byte, error) {
	fmt.Print("\n" + prompt)
	pass, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}
	return pass, nil
}
