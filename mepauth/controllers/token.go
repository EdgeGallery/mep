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

// token controller
package controllers

import (
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unsafe"

	log "github.com/sirupsen/logrus"

	"github.com/astaxie/beego"

	"mepauth/models"
	"mepauth/util"

	"github.com/dgrijalva/jwt-go/v4"
)

const Authorization string = "Authorization"
const XRealIp string = "X-Real-Ip"
const InternalError string = "Internal server error."

type TokenController struct {
	beego.Controller
}

// Process token request
func (c *TokenController) Post() {
	header := c.Ctx.Input.Header(Authorization)
	clientIp := c.Ctx.Request.Header.Get(XRealIp)
	log.Info("Get token clientIp: %s", clientIp)
	// Below we first check the formats of the header is correct or not
	ak, signHeader, sig := parseAuthHeader(header)
	if ak == "" || signHeader == "" || sig == "" {
		log.Error("Received message from ClientIP [" + clientIp + "] Operation [" + c.Ctx.Request.Method + "]" +
			" Resource [" + c.Ctx.Input.URL() + "]")
		c.writeErrorResponse("Bad request.", util.BadRequest)
		log.Error("Response message for ClientIP [" + clientIp + "] Operation [" + c.Ctx.Request.Method + "]" +
			" Resource [" + c.Ctx.Input.URL() + "] Result [Failure: Bad auth header format.]")
		return
	}
	log.Infof("ak: %s, signHeader: %s, sig: %s", ak, signHeader, sig)

	isTimeValid := validateDateTimeFormat(c.Ctx.Request)
	if !isTimeValid {
		log.Error("Received message from ClientIP [" + clientIp + "] Operation [" + c.Ctx.Request.Method + "]" +
			" Resource [" + c.Ctx.Input.URL() + "]")
		c.writeErrorResponse("Bad request.", util.BadRequest)
		log.Error("Response message for ClientIP [" + clientIp + "] Operation [" + c.Ctx.Request.Method + "]" +
			" Resource [" + c.Ctx.Input.URL() + "] Result [Failure: Bad x-sdk-time format.]")
		return
	}

	log.Info("Received message from ClientIP [" + clientIp + "] ClientAK [" + ak + "]" +
		" Operation [" + c.Ctx.Request.Method + "] Resource [" + c.Ctx.Input.URL() + "]")

	isAkBlockListed := IsAkInBlockList(ak)
	if isAkBlockListed {
		c.writeErrorResponse("Access is locked.", util.Forbidden)
		log.Error("Response message for ClientIP [" + clientIp + "] ClientAK [" + ak + "]" +
			" Operation [" + c.Ctx.Request.Method + "] Resource [" + c.Ctx.Input.URL() + "]" +
			" Result [Failure: Ak is blockListed.]")
		return
	}

	appInsId, sk, akExist := GetAppInsIdSk(ak)
	if appInsId == "" || sk == nil || len(sk) == 0 {
		c.checkAkExistAndWriteErrorRes(akExist)
		log.Error("Response message for ClientIP [" + clientIp + "] ClientAK [" + ak + "]" +
			" Operation [" + c.Ctx.Request.Method + "] Resource [" + c.Ctx.Input.URL() + "]" +
			" Result [Failure: Matching App instance id not found.]")
		return
	}

	log.Info("Corresponding App Instance Id " + appInsId + " found for ClientAK " + ak)

	ret := c.validateSignature(ak, sk, signHeader, sig)
	if !ret {
		return
	}
	ClearAkFromBlockListing(ak)

	tokenInfo := c.getTokenInfo(appInsId, ak)
	if tokenInfo == nil {
		return
	}

	c.sendResponseMsg(ak, tokenInfo)
}

func (c *TokenController) writeErrorResponse(errMsg string, code int) {
	log.Error(errMsg)
	c.writeResponse(errMsg, code)
}

func (c *TokenController) writeResponse(msg string, code int) {
	c.Data["json"] = msg
	c.Ctx.ResponseWriter.WriteHeader(code)
	c.ServeJSON()
}

type JwtClaims struct {
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
	claims := JwtClaims{
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
		log.Error("Failed to sign the token for App InstanceId [" + appInsId + "]")
		return nil, err
	}
	return &token, nil
}

// Get app instance Id and Sk
func GetAppInsIdSk(ak string) (string, []byte, bool) {
	//authInfoRecord, readErr := ReadDataFromFile(ak)
	authInfoRecord := &models.AuthInfoRecord{
		Ak: ak,
	}
	readErr := ReadData(authInfoRecord, "ak")
	if readErr != nil && readErr.Error() != util.PgOkMsg {
		log.Error("auth info record does not exist")
		return "", nil, false
	}
	encodedSk := []byte(authInfoRecord.Sk)
	cipherSkBytes := make([]byte, hex.DecodedLen(len(encodedSk)), 200)
	_, errDecodeSk := hex.Decode(cipherSkBytes, encodedSk)
	if errDecodeSk != nil {
		log.Error("decode secret key failed")
		return "", nil, true
	}
	encodedNonce := []byte(authInfoRecord.Nonce)
	nonceBytes := make([]byte, hex.DecodedLen(len(encodedNonce)), 30)
	_, errDecodeNonce := hex.Decode(nonceBytes, encodedNonce)
	if errDecodeNonce != nil {
		log.Error("decode nonce failed")
		// clear nonce
		util.ClearByteArray(nonceBytes)
		return "", nil, true
	}
	workKey, genKeyErr := util.GetWorkKey()
	if genKeyErr != nil {
		log.Error("generate work key failed")
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
		log.Error("decrypt secret key failed")
		return "", nil, true
	}
	return authInfoRecord.AppInsId, sk, true
}

func akSignatureIsValid(r *http.Request, ak string, sk []byte, signHeader string, sig string) (bool, error) {

	s := util.Sign{
		AccessKey: ak,
		SecretKey: sk,
	}
	// since we put mepauth behind kong, mepagent would use /route_path/mepauth_url to sign
	reqUrl := "https://" + r.Host + "/" + util.MepauthName + r.URL.String()
	reqToBeSigned, errNewRequest := http.NewRequest("POST", reqUrl, strings.NewReader(""))
	if errNewRequest != nil {
		log.Error("prepare http request to generate signature is failed")
		return false, errors.New("create new request fail")
	}

	for _, h := range strings.Split(signHeader, ";") {
		reqToBeSigned.Header.Set(h, r.Header.Get(h))
	}
	reqToBeSigned.Header.Set(util.HOST_HEADER, r.Host)

	signature, err := s.GetSignature(reqToBeSigned)
	if err != nil {
		log.Error("failed to generate signature")
		return false, err
	}

	return sig == signature, nil
}

func parseAuthHeader(header string) (ak string, signHeader string, sig string) {

	defer func() {
		if err1 := recover(); err1 != nil {
			log.Error("panic handled:", err1)
		}
	}()
	authRegexp := regexp.MustCompile(util.AuthHeaderRegex)
	matchVars := authRegexp.FindStringSubmatch(header)
	if len(matchVars) <= util.MaxMatchVarSize {
		return "", "", ""
	}
	return matchVars[1], matchVars[2], matchVars[3]
}

func validateDateTimeFormat(req *http.Request) bool {
	stringXSdkTime := req.Header.Get(util.DATE_HEADER)
	if stringXSdkTime == "" {
		return false
	}
	_, err := time.Parse(util.DATE_FORMAT, stringXSdkTime)
	if err != nil {
		log.Error("validate datetimeformat failed")
		return false
	}
	return true

}

func (c *TokenController) validateSignature(ak string, sk []byte, signHeader string, sig string) bool {
	signIsValid, err := akSignatureIsValid(c.Ctx.Request, ak, sk, signHeader, sig)
	clientIp := c.Ctx.Request.Header.Get(XRealIp)

	// clear sk
	util.ClearByteArray(sk)
	if err != nil {
		c.writeResponse(InternalError, util.IntSerErr)
		log.Info("Response message for ClientIP [" + clientIp + "] ClientAK [" + ak + "]" +
			" Operation [" + c.Ctx.Request.Method + "] Resource [" + c.Ctx.Input.URL() + "]" +
			" Result[Failure: Generating signature failed.]")
		return false
	}

	if !signIsValid {
		ProcessAkForBlockListing(ak)
		c.writeResponse("Invalid access or signature.", util.Unauthorized)
		log.Info("Response message for ClientIP [" + clientIp + "] ClientAK [" + ak + "]" +
			" Operation [" + c.Ctx.Request.Method + "] Resource [" + c.Ctx.Input.URL() + "]" +
			" Result [Failure: Signature is invalid.]")
		return false
	}

	return true
}

func (c *TokenController) getTokenInfo(appInsId string, ak string) *models.TokenInfo {
	clientIp := c.Ctx.Request.Header.Get(XRealIp)
	if clientIp == "" {
		clientIp = "UNKNOWN_IP"
	}

	token, err := generateJwtToken(appInsId, clientIp)
	if err != nil {
		c.writeResponse(InternalError, util.IntSerErr)
		log.Info("Response message for ClientIP [" + clientIp + "] ClientAK [" + ak + "]" +
			" Operation [" + c.Ctx.Request.Method + "] Resource [" + c.Ctx.Input.URL() + "]" +
			" Result [Failure: Generation of jwt token failed.]")
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
		c.writeErrorResponse("Invalid access or signature.", util.Unauthorized)
	} else {
		c.writeErrorResponse(InternalError, util.IntSerErr)
	}
}

func (c *TokenController) sendResponseMsg(ak string, tokenInfo *models.TokenInfo) {
	clientIp := c.Ctx.Request.Header.Get(XRealIp)
	c.Data["json"] = tokenInfo
	c.ServeJSON()
	bKey := *(*[]byte)(unsafe.Pointer(&tokenInfo.AccessToken))
	util.ClearByteArray(bKey)
	log.Info("Response message for ClientIP [" + clientIp + "] ClientAK [" + ak + "]" +
		" Operation [" + c.Ctx.Request.Method + "] Resource [" + c.Ctx.Input.URL() + "] Result [Success]")
}
