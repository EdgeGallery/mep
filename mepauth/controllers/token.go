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
	"encoding/hex"
	"errors"
	"mepauth/adapter"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unsafe"

	log "github.com/sirupsen/logrus"

	"mepauth/models"
	"mepauth/util"

	"github.com/dgrijalva/jwt-go/v4"
)

const authorization string = "authorization"
const xRealIp string = "X-Real-Ip"
const internalError string = "Internal server error."

// TokenController token controller
type TokenController struct {
	BaseController
}

// @Title Process token information
// @Description create token and return the same
// @Param   Content-Type   header  string  true   "MIME type, fill in application/json"
// @Param   authorization  header  string  true   "Certification Information"
// @Param   x-sdk-date     header  string  true   "Signature time, current timestamp, format: YYYYMMDDTHHMMSSZ"
// @Param   Host           header  string  true   "Consistent with the host field used to generate the authentication information signature"
// @Success 200 ok
// @Failure 400 bad request
// @router /token [post]
func (c *TokenController) Post() {
	log.Info("Get token request received.")
	clientIp := c.Ctx.Request.Header.Get(xRealIp)
	err := c.validateSrcAddress(clientIp)
	if err != nil {
		c.handleLoggingForError(clientIp, http.StatusBadRequest, util.ClientIpaddressInvalid)
		return
	}
	// Below we first check the formats of the header is correct or not
	header := c.Ctx.Input.Header(authorization)
	ak, signHeader, sig := parseAuthHeader(header)
	if ak == "" || signHeader == "" || sig == "" {
		c.logReceivedMsg(clientIp)
		c.handleLoggingForError(clientIp, http.StatusBadRequest, "Bad auth header format")
		return
	}

	if !isDateTimeFormatValid(c.Ctx.Request) {
		c.logReceivedMsg(clientIp)
		c.handleLoggingForError(clientIp, http.StatusBadRequest, "Bad x-sdk-time format")
		return
	}

	c.logReceivedMsgWithAk(clientIp, ak)

	if isAkInBlockList(ak) {
		c.writeErrorResponse("Access is locked.", http.StatusForbidden)
		c.logErrResponseMsgWithAk(clientIp, "Ak is blockListed", ak)
		return
	}

	appInsId, sk, akExist := getAppInsIdSk(ak)
	if appInsId == "" || sk == nil || len(sk) == 0 {
		c.checkAkExistAndWriteErrorRes(akExist)
		c.logErrResponseMsgWithAk(clientIp, "Matching App instance id not found", ak)
		return
	}

	log.Info("Corresponding App Instance Id " + appInsId + " found for ClientAK " + ak)

	if !c.isSignatureValid(ak, sk, signHeader, sig, clientIp) {
		return
	}
	clearAkFromBlockListing(ak)

	tokenInfo := c.getTokenInfo(appInsId, ak)
	if tokenInfo == nil {
		return
	}
	c.sendResponseMsg(ak, tokenInfo, clientIp)
}

type jwtClaims struct {
	jwt.StandardClaims
	ClientIp string `json:"clientip"`
}

func generateJwtToken(appInsId string, clientIp string) (*string, error) {
	privateKey, err := util.GetPrivateKey()
	if privateKey == nil || err != nil {
		return nil, errors.New("failed to get private key")
	}

	mepAuthKey := util.GetAppConfig("mepauth_key")
	if len(mepAuthKey) == 0 {
		msg := "mep auth key configuration is not set"
		log.Error(msg)
		// Clear the private key
		privateKeyBits := privateKey.D.Bits()
		for i := 0; i < len(privateKeyBits); i++ {
			privateKeyBits[i] = 0
		}
		return nil, errors.New(msg)
	}
	claims := jwtClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: jwt.At(time.Now().Add(time.Hour * 1)),
			Issuer:    mepAuthKey,
			Subject:   appInsId,
		},
		ClientIp: clientIp,
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)

	token, err := jwtToken.SignedString(privateKey)
	// Clear the private key
	privateKeyBits := privateKey.D.Bits()
	for i := 0; i < len(privateKeyBits); i++ {
		privateKeyBits[i] = 0
	}
	if err != nil || token == "" {
		log.Error("Failed to sign the token for application Instance ID [" + appInsId + "]")
		return nil, err
	}
	return &token, nil
}

// Get app instance Id and Sk
func getAppInsIdSk(ak string) (string, []byte, bool) {
	//authInfoRecord, readErr := ReadDataFromFile(ak)
	authInfoRecord := &models.AuthInfoRecord{
		Ak: ak,
	}
	readErr := adapter.Db.ReadData(authInfoRecord, "ak")
	if readErr != nil && readErr.Error() != util.PgOkMsg {
		log.Error("Auth info record does not exist")
		return "", nil, false
	}
	encodedSk := []byte(authInfoRecord.Sk)
	cipherSkBytes := make([]byte, hex.DecodedLen(len(encodedSk)), http.StatusOK)
	_, errDecodeSk := hex.Decode(cipherSkBytes, encodedSk)
	if errDecodeSk != nil {
		log.Error("Decode of secret key failed")
		return "", nil, true
	}
	encodedNonce := []byte(authInfoRecord.Nonce)
	nonceBytes := make([]byte, hex.DecodedLen(len(encodedNonce)), 30)
	_, errDecodeNonce := hex.Decode(nonceBytes, encodedNonce)
	if errDecodeNonce != nil {
		log.Error("Decode nonce failed")
		// clear nonce
		util.ClearByteArray(nonceBytes)
		return "", nil, true
	}
	workKey, genKeyErr := util.GetWorkKey()
	if genKeyErr != nil {
		log.Error("Generate work key failed")
		// clear nonce
		util.ClearByteArray(nonceBytes)
		return "", nil, true
	}
	sk, errDecryptSk := util.DecryptByAES256GCM(cipherSkBytes, workKey, nonceBytes)
	// clear work key
	util.ClearByteArray(workKey)
	// clear nonce
	util.ClearByteArray(nonceBytes)
	if errDecryptSk != nil {
		log.Error("Decrypt secret key failed")
		return "", nil, true
	}
	return authInfoRecord.AppInsId, sk, true
}

func isAkSignatureValid(r *http.Request, sk []byte, signHeader string, sig string) (bool, error) {

	s := util.Sign{
		SecretKey: sk,
	}
	// since we put mepauth behind kong, mepagent would use /route_path/mepauth_url to sign
	reqUrl := "https://" + r.Host + r.URL.String()
	reqToBeSigned, errNewRequest := http.NewRequest("POST", reqUrl, strings.NewReader(""))
	if errNewRequest != nil {
		log.Error("Preparation of http request to generate signature failed")
		return false, errors.New("create new request fail")
	}

	for _, h := range strings.Split(signHeader, ";") {
		reqToBeSigned.Header.Set(h, r.Header.Get(h))
	}
	reqToBeSigned.Header.Set(util.HostHeader, r.Host)

	signature, err := s.GetSignature(reqToBeSigned)
	if err != nil {
		log.Error("Failed to generate signature")
		return false, err
	}

	return sig == signature, nil
}

func parseAuthHeader(header string) (ak string, signHeader string, sig string) {

	defer func() {
		if err1 := recover(); err1 != nil {
			log.Error("Panic handled:", err1)
		}
	}()
	authRegexp := regexp.MustCompile(util.AuthHeaderRegex)
	matchVars := authRegexp.FindStringSubmatch(header)
	if len(matchVars) <= util.MaxMatchVarSize {
		return "", "", ""
	}
	return matchVars[1], matchVars[2], matchVars[3]
}

func isDateTimeFormatValid(req *http.Request) bool {
	stringXSdkTime := req.Header.Get(util.DateHeader)
	if stringXSdkTime == "" {
		return false
	}
	_, err := time.Parse(util.DateFormat, stringXSdkTime)
	if err != nil {
		log.Error("Validation of date & time format failed")
		return false
	}
	return true
}

func (c *TokenController) isSignatureValid(ak string, sk []byte, signHeader string, sig string, clientIp string) bool {
	signIsValid, err := isAkSignatureValid(c.Ctx.Request, sk, signHeader, sig)

	// clear sk
	util.ClearByteArray(sk)
	if err != nil {
		c.writeErrorResponse(internalError, http.StatusInternalServerError)
		c.logErrResponseMsgWithAk(clientIp, "Generating signature failed", ak)
		return false
	}

	if !signIsValid {
		processAkForBlockListing(ak)
		c.writeErrorResponse("Invalid access or signature.", http.StatusUnauthorized)
		c.logErrResponseMsgWithAk(clientIp, "Signature is invalid", ak)
		return false
	}
	return true
}

func (c *TokenController) getTokenInfo(appInsId string, ak string) *models.TokenInfo {
	clientIp := c.Ctx.Request.Header.Get(xRealIp)
	if clientIp == "" {
		clientIp = "UNKNOWN_IP"
	}

	token, err := generateJwtToken(appInsId, clientIp)
	if err != nil {
		c.writeErrorResponse(internalError, http.StatusInternalServerError)
		c.logErrResponseMsgWithAk(clientIp, "Generation of jwt token failed", ak)
		return nil
	}

	tokenInfo := &models.TokenInfo{
		AccessToken: *token,
		TokenType:   "Bearer",
		ExpiresIn:   util.ExpiresVal,
	}
	return tokenInfo
}

func (c *TokenController) checkAkExistAndWriteErrorRes(akExist bool) {
	if !akExist {
		c.writeErrorResponse("Invalid access or signature.", http.StatusUnauthorized)
	} else {
		c.writeErrorResponse(internalError, http.StatusInternalServerError)
	}
}

func (c *TokenController) sendResponseMsg(ak string, tokenInfo *models.TokenInfo, clientIp string) {
	c.Data["json"] = tokenInfo
	c.ServeJSON()
	bKey := *(*[]byte)(unsafe.Pointer(&tokenInfo.AccessToken))
	util.ClearByteArray(bKey)
	log.Info("Response message for ClientIP [" + clientIp + "] ClientAK [" + ak + "]" +
		" operation [" + c.Ctx.Request.Method + "] resource [" + c.Ctx.Input.URL() + "] Result [Success]")
}
