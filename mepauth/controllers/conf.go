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

// conf controller
package controllers

import (
	"crypto/rand"
	"encoding/hex"
	log "github.com/sirupsen/logrus"

	"mepauth/models"
	"mepauth/util"
)

// Save Ak and Sk configuration into file
func ConfigureAkAndSk(appInsID string, ak string, sk *[]byte) error {

	log.Info("ak/sk configuration is received, the corresponding app is "+ appInsID)

	if validateErr := util.ValidateUUID(appInsID); validateErr != nil {
		log.Error("AppInstanceId: "+ appInsID + " is invalid.")
		return validateErr
	}

	validateAkErr := util.ValidateAk(ak)
	if validateAkErr != nil {
		log.Error("Ak is invalid, appInstanceId is " + appInsID + ".")
		return validateAkErr
	}
	validateSkErr := util.ValidateSk(sk)
	if validateSkErr != nil {
		log.Error("Sk is invalid, appInstanceId is " + appInsID + ".")
		return validateSkErr
	}

	saveAkAndSkErr := saveAkAndSk(appInsID, ak, sk)
	if saveAkAndSkErr != nil {
		log.Error("Failed to save ak and sk to file, appInstanceId is " + appInsID + ".")
		return saveAkAndSkErr
	}
	log.Info("Succeed to save ak and sk, appInstanceId is " + appInsID + ".")
	return nil
}

func saveAkAndSk(appInsID string, ak string, sk *[]byte) error {
	nonce := make([]byte, util.NonceSize, 20)
	_, generateNonceErr := rand.Read(nonce)
	if generateNonceErr != nil {
		log.Error("Failed to generate nonce.")
		util.ClearByteArray(*sk)
		return generateNonceErr
	}
	workKey, genKeyErr := util.GetWorkKey()
	if genKeyErr != nil {
		log.Error("Failed to generate work key.")
		util.ClearByteArray(nonce)
		util.ClearByteArray(*sk)
		return genKeyErr
	}
	cipherSk, encryptErr := util.EncryptByAES256GCM(*sk, workKey, nonce)
	util.ClearByteArray(*sk)
	util.ClearByteArray(workKey)
	if encryptErr != nil {
		log.Error("Failed to encrypt secret key.")
		// clear nonce
		util.ClearByteArray(nonce)
		return encryptErr
	}
	cipherSkBytes := make([]byte, hex.EncodedLen(len(cipherSk)), 200)
 	hex.Encode(cipherSkBytes, cipherSk)
	nonceBytes := make([]byte, hex.EncodedLen(len(nonce)), 30)
	hex.Encode(nonceBytes, nonce)
	util.ClearByteArray(nonce)
	authInfoRecord := &models.AuthInfoRecord{
		AppInsId: appInsID,
		Ak:       ak,
		Sk:       string(cipherSkBytes),
		Nonce:    string(nonceBytes),
	}
	err := InsertOrUpdateData(authInfoRecord)
	util.ClearByteArray(nonceBytes)
	if err != nil {
		log.Error("Failed to save ak and sk to file.")
		return err
	}
	return nil
}
