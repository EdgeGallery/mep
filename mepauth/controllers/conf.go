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

// Package controllers implements mep auth controller
package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"mepauth/adapter"
	"net/http"

	"mepauth/models"
	"mepauth/util"
)

const appInstanceID string = "app_ins_id"

// ConfController configuration controller
type ConfController struct {
	BaseController
}

// @Title Adds AK/SK configuration
// @Description addition of ak & sk configuration
// @Param   Content-Type   header  string  true   "MIME type, fill in application/json"
// @Param   applicationId  path  string  true   "APP instance ID"
// @Param   body body models.AppAuthInfo true "User Info"
// @Success 200 ok
// @Failure 400 bad request
// @router /appMng/v1/applications/:applicationId/confs [put]
func (c *ConfController) Put() {
	log.Info("Put AK/SK configuration request received.")
	clientIp := c.Ctx.Request.Header.Get(xRealIp)
	err := c.validateSrcAddress(clientIp)
	if err != nil {
		c.handleLoggingForError(clientIp, http.StatusBadRequest, util.ClientIpaddressInvalid)
		return
	}
	c.logReceivedMsg(clientIp)
	var appAuthInfo *models.AppAuthInfo
	// Get application instance ID from param
	appInsId := c.Ctx.Input.Param(util.UrlApplicationId)

	if err = json.Unmarshal(c.Ctx.Input.RequestBody, &appAuthInfo); err == nil {
		c.Data["json"] = appAuthInfo
		// Get AK
		ak := appAuthInfo.AuthInfo.Credentials.AccessKeyId
		// Get SK
		sk := appAuthInfo.AuthInfo.Credentials.SecretKey
		skByte := []byte(sk)
		err := ConfigureAkAndSk(appInsId, ak, &skByte)
		if err != nil {
			switch err.Error() {
			case util.AppIDFailMsg:
				c.handleLoggingForError(clientIp, http.StatusBadRequest, "Invalid input for application instance ID")
				return
			case util.AkFailMsg:
			case util.SkFailMsg:
				c.handleLoggingForError(clientIp, http.StatusBadRequest, "Invalid input for ak or sk")
				return
			default:
				c.handleLoggingForError(clientIp, http.StatusInternalServerError, "Error while saving configuration")
			}
		}
	} else {
		c.handleLoggingForError(clientIp, http.StatusBadRequest, err.Error())
	}
	c.handleLoggingForSuccess(clientIp, "")
}

// @Title Deletes AK/SK configuration
// @Description deletion of ak & sk configuration
// @Param   Content-Type   header  string  true   "MIME type, fill in application/json"
// @Param   applicationId  path  string  true   "APP instance ID"
// @Success 200 ok
// @Failure 400 bad request
// @router /appMng/v1/applications/:applicationId/confs [delete]
func (c *ConfController) Delete() {

	log.Info("Delete AK/SK configuration request received.")
	clientIp := c.Ctx.Request.Header.Get(xRealIp)
	err := c.validateSrcAddress(clientIp)
	if err != nil {
		c.handleLoggingForError(clientIp, http.StatusBadRequest, util.ClientIpaddressInvalid)
		return
	}
	c.logReceivedMsg(clientIp)

	appInsId := c.Ctx.Input.Param(util.UrlApplicationId)
	if validateErr := util.ValidateUUID(appInsId); validateErr != nil {
		c.handleLoggingForError(clientIp, http.StatusBadRequest, util.AppIDFailMsg)
		return
	}

	authInfoRecord := &models.AuthInfoRecord{
		AppInsId: appInsId,
	}

	err = adapter.Db.DeleteData(authInfoRecord, appInstanceID)
	if err != nil {
		c.handleLoggingForError(clientIp, http.StatusBadRequest, err.Error())
		return
	}
	c.handleLoggingForSuccess(clientIp, "")
}

// @Title Gets AK/SK configuration
// @Description get ak & sk configuration
// @Param   Content-Type   header  string  true   "MIME type, fill in application/json"
// @Param   applicationId  path  string  true   "APP instance ID"
// @Success 200 ok
// @Failure 400 bad request
// @router /appMng/v1/applications/:applicationId/confs [get]
func (c *ConfController) Get() {

	log.Info("Get AK/SK configuration request received.")
	clientIp := c.Ctx.Request.Header.Get(xRealIp)
	err := c.validateSrcAddress(clientIp)
	if err != nil {
		c.handleLoggingForError(clientIp, http.StatusBadRequest, util.ClientIpaddressInvalid)
		return
	}
	c.logReceivedMsg(clientIp)

	appInsId := c.Ctx.Input.Param(util.UrlApplicationId)
	if validateErr := util.ValidateUUID(appInsId); validateErr != nil {
		c.handleLoggingForError(clientIp, http.StatusBadRequest, util.AppIDFailMsg)
		return
	}

	authInfoRecord := &models.AuthInfoRecord{
		AppInsId: appInsId,
	}

	err = adapter.Db.ReadData(authInfoRecord, appInstanceID)
	if err != nil && err.Error() != util.PgOkMsg {
		c.handleLoggingForError(clientIp, http.StatusBadRequest, err.Error())
		return
	}
	c.Data["json"] = authInfoRecord
	c.ServeJSON()
	log.Info("Response message for ClientIP [" + clientIp + operation + c.Ctx.Request.Method + "]" +
		resource + c.Ctx.Input.URL() + "] Result [Success]")
}

// ConfigureAkAndSk save Ak and Sk configuration into file
func ConfigureAkAndSk(appInsID string, ak string, sk *[]byte) error {

	log.Infof("AK/SK configuration is received, the corresponding app is " + appInsID)

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
	err = adapter.Db.InsertOrUpdateData(authInfoRecord, appInstanceID)
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
