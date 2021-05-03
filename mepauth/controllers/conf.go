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

// conf controller
package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"github.com/astaxie/beego"
	log "github.com/sirupsen/logrus"
	"mepauth/dbAdapter"

	"mepauth/models"
	"mepauth/util"
)

const AppInsId string = "app_ins_id"

type ConfController struct {
	beego.Controller
}

func (c *ConfController) Put() {
	var appAuthInfo *models.AppAuthInfo
	var err error
	appInsId := c.Ctx.Input.Param(util.UrlApplicationId)
	log.Infof("conf ak/sk appInstanceId=%s", appInsId)

	if err = json.Unmarshal(c.Ctx.Input.RequestBody, &appAuthInfo); err == nil {
		c.Data["json"] = appAuthInfo
		ak := appAuthInfo.AuthInfo.Credentials.AccessKeyId
		log.Infof("conf ak/sk ak=%s", ak)
		sk := appAuthInfo.AuthInfo.Credentials.SecretKey
		skByte := []byte(sk)
		cipherSkBytes, nonceBytes, err2 := getCipherAndNonce(&skByte)
		if err2 != nil {
			c.Data["json"] = err2.Error()
			c.ServeJSON()
			return
		}
		authInfoRecord := &models.AuthInfoRecord{
			AppInsId: appInsId,
			Ak:       ak,
			Sk:       string(cipherSkBytes),
			Nonce:    string(nonceBytes),
		}
		err = dbAdapter.Db.InsertOrUpdateData(authInfoRecord, AppInsId)
		if err != nil && err.Error() != util.PgOkMsg {
			c.Data["json"] = err.Error()
		}
	} else {
		c.Data["json"] = err.Error()
	}
	c.ServeJSON()
}

func (c *ConfController) Delete() {
	appInsId := c.Ctx.Input.Param(util.UrlApplicationId)
	log.Infof("delete ak/sk appInstanceId=%s", appInsId)

	authInfoRecord := &models.AuthInfoRecord{
		AppInsId: appInsId,
	}

	err := dbAdapter.Db.DeleteData(authInfoRecord, AppInsId)
	if err != nil {
		c.writeErrorResponse("Delete fail.", util.BadRequest)
		return
	}

	c.Data["json"] = "Delete success."
	c.ServeJSON()
}

func (c *ConfController) Get() {
	appInsId := c.Ctx.Input.Param(util.UrlApplicationId)

	authInfoRecord := &models.AuthInfoRecord{
		AppInsId: appInsId,
	}

	err := dbAdapter.Db.ReadData(authInfoRecord, AppInsId)
	if err != nil && err.Error() != util.PgOkMsg {
		c.Data["json"] = err.Error()
	}
	c.Data["json"] = authInfoRecord
	c.ServeJSON()
}

func (c *ConfController) writeErrorResponse(errMsg string, code int) {
	log.Error(errMsg)
	c.writeResponse(errMsg, code)
}

func (c *ConfController) writeResponse(msg string, code int) {
	c.Data["json"] = msg
	c.Ctx.ResponseWriter.WriteHeader(code)
	c.ServeJSON()
}

// Save Ak and Sk configuration into file
func ConfigureAkAndSk(appInsID string, ak string, sk *[]byte) error {

	log.Infof("ak/sk configuration is received, the corresponding app is " + appInsID)

	if validateErr := util.ValidateUUID(appInsID); validateErr != nil {
		log.Error("AppInstanceId: " + appInsID + " is invalid.")
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
	cipherSkBytes, nonceBytes, err := getCipherAndNonce(sk)
	if err != nil {
		return err
	}
	authInfoRecord := &models.AuthInfoRecord{
		AppInsId: appInsID,
		Ak:       ak,
		Sk:       string(cipherSkBytes),
		Nonce:    string(nonceBytes),
	}
	//err = InsertOrUpdateDataToFile(authInfoRecord)
	err = dbAdapter.Db.InsertOrUpdateData(authInfoRecord, AppInsId)
	util.ClearByteArray(nonceBytes)
	if err != nil && err.Error() != util.PgOkMsg {
		log.Error("Failed to save ak and sk to file.")
		return err
	}
	return nil
}

func getCipherAndNonce(sk *[]byte) ([]byte, []byte, error) {
	nonce := make([]byte, util.NonceSize, 20)
	_, generateNonceErr := rand.Read(nonce)
	if generateNonceErr != nil {
		log.Error("Failed to generate nonce.")
		util.ClearByteArray(*sk)
		return nil, nil, generateNonceErr
	}
	workKey, genKeyErr := util.GetWorkKey()
	if genKeyErr != nil {
		log.Error("Failed to generate work key.")
		util.ClearByteArray(nonce)
		util.ClearByteArray(*sk)
		return nil, nil, genKeyErr
	}
	cipherSk, encryptErr := util.EncryptByAES256GCM(*sk, workKey, nonce)
	util.ClearByteArray(*sk)
	util.ClearByteArray(workKey)
	if encryptErr != nil {
		log.Error("Failed to encrypt secret key.")
		// clear nonce
		util.ClearByteArray(nonce)
		return nil, nil, encryptErr
	}
	cipherSkBytes := make([]byte, hex.EncodedLen(len(cipherSk)), 200)
	hex.Encode(cipherSkBytes, cipherSk)
	nonceBytes := make([]byte, hex.EncodedLen(len(nonce)), 30)
	hex.Encode(nonceBytes, nonce)
	util.ClearByteArray(nonce)
	return cipherSkBytes, nonceBytes, nil
}
