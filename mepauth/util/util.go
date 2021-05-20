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

// Package util implements mep auth utility functions and contain constants
package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/astaxie/beego"
	"github.com/dgrijalva/jwt-go/v4"
	"golang.org/x/crypto/pbkdf2"
)

const keyFileMode os.FileMode = 0600

const keySize int = 32
const NonceSize int = 12
const iterationNum int = 100000
const componentSize int = 256

const componentFilePath string = "cprop/c_properties"
const saltFilePath string = "sprop/s_properties"
const encryptedWorkKeyFilePath string = "wprop/w_properties"
const workKeyNonceFilePath string = "wnprop/wn_properties"
const encryptedJwtPrivateKeyPwdFilePath string = "encrypted_jwt_private_key_pwd"
const jwtPrivateKeyPwdNonceFilePath string = "jwt_private_key_nonce_pwd"

// AppConfigProperties application configuration properties
type AppConfigProperties map[string]*[]byte

var KeyComponentFromUserStr *[]byte
var cipherSuiteMap = map[string]uint16{
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
}

// GetAppConfig gets app configuration
func GetAppConfig(k string) string {
	return beego.AppConfig.String(k)
}

// GetPublicKey gets public key from app configuration
func GetPublicKey() ([]byte, error) {
	jwtPublicKey := GetAppConfig("jwt_public_key")
	if len(jwtPublicKey) == 0 {
		log.Error("JWT public key configuration is not set")
		return nil, errors.New("jwt public key configuration is not set")
	}

	publicKey, err := ioutil.ReadFile(jwtPublicKey)
	if err != nil {
		log.Error("Unable to read public key file")
		return nil, errors.New("unable to read public key file")
	}
	publicKeyBlock, _ := pem.Decode(publicKey)
	if publicKeyBlock == nil || publicKeyBlock.Type != "PUBLIC KEY" {
		log.Error("Failed to decode public key file")
		return nil, errors.New("failed to decode public key file")
	}
	return publicKey, nil
}

// GetPrivateKey gets private key from app configuration
func GetPrivateKey() (*rsa.PrivateKey, error) {
	encryptKeyFile := GetAppConfig("jwt_encrypted_private_key")
	if len(encryptKeyFile) == 0 {
		log.Error("cannot fetch jwt private key from env")
		return nil, errors.New("cannot fetch jwt private key from env")
	}

	keyContent, err := ioutil.ReadFile(encryptKeyFile)
	if err != nil {
		log.Error("unable to read key file")
		return nil, errors.New("unable to read key file")
	}

	encryptKeyBlock, _ := pem.Decode(keyContent)
	if encryptKeyBlock == nil {
		log.Error("failed to decode encrypt jwt private key file")
		// clear keyContent
		ClearByteArray(keyContent)
		return nil, errors.New("failed to decode encrypt jwt private key file")
	}
	// decrypt key using plain pwd
	if x509.IsEncryptedPEMBlock(encryptKeyBlock) {
		plainPwBytes, getPwdErr := getJwtPrivateKeyPwd()
		if getPwdErr != nil {
			log.Error("failed to get jwt private key password")
			// clear keyContent
			ClearByteArray(keyContent)
			// clear encryptKeyBlock
			ClearByteArray(encryptKeyBlock.Bytes)
			return nil, errors.New("failed to get jwt private key password")
		}
		keyData, err := x509.DecryptPEMBlock(encryptKeyBlock, plainPwBytes)
		// clear plainPwBytes
		ClearByteArray(plainPwBytes)
		ClearByteArray(keyContent)
		ClearByteArray(encryptKeyBlock.Bytes)
		if err != nil {
			log.Error("failed to decrypt jwt private key file")
			return nil, errors.New("failed to decrypt jwt private key file")
		}
		decryptKeyBlock := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: keyData,
		}
		keyContent = pem.EncodeToMemory(decryptKeyBlock)
		// clear encryptKeyBlock
		ClearByteArray(keyData)
	} else {
		// clear encryptKeyBlock
		ClearByteArray(encryptKeyBlock.Bytes)
	}
	parsedKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyContent)
	// clear keyContent
	ClearByteArray(keyContent)
	if err != nil {
		log.Error("failed to parse private key")
		return nil, errors.New("failed to parse private key")
	}
	return parsedKey, nil
}

// GetAPIGwURL gets api gateway URL
func GetAPIGwURL() (string, error) {
	apiGwHost := GetAppConfig(ApigwHost)
	apiGwPort := GetAppConfig(ApigwPort)
	apiGwParamsAreValid, validateApiGwParamsErr := validateApiGwParams(apiGwHost, apiGwPort)
	if validateApiGwParamsErr != nil || !apiGwParamsAreValid {
		log.Error("Validation of consumer URL failed")
		return "", validateApiGwParamsErr
	}
	kongConsumerUrl := fmt.Sprintf("https://%s:%s", apiGwHost, apiGwPort)
	return kongConsumerUrl, nil
}

// EncryptByAES256GCM encrypts secret key using aes 256 gcm algo
func EncryptByAES256GCM(plaintext []byte, key []byte, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Error("failed to create aes cipher")
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Error("failed to wrap cipher")
		return nil, err
	}

	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptByAES256GCM encrypts secret key using aes 256 gcm algo
func DecryptByAES256GCM(ciphertext []byte, key []byte, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Error("failed to create aes cipher")
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Error("failed to wrap cipher")
		return nil, err
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		log.Error("failed to decrypt secret key")
		return nil, err
	}

	return plaintext, nil
}

// TLSConfig provides TLS configuration
func TLSConfig(crtName string) (*tls.Config, error) {
	certNameConfig := GetAppConfig(crtName)
	if len(certNameConfig) == 0 {
		log.Error( "Certificate name is not set")
		return nil, errors.New("certificate name configuration is not set")
	}

	crt, err := ioutil.ReadFile(certNameConfig)
	if err != nil {
		log.Error("Unable to read certificate file")
		return nil, err
	}

	rootCAs := x509.NewCertPool()
	ok := rootCAs.AppendCertsFromPEM(crt)
	if !ok {
		log.Error("Failed to decode certificate file")
		return nil, errors.New("failed to decode certificate file")
	}

	serverName := GetAppConfig("server_name")
	serverNameIsValid, validateServerNameErr := validateServerName(serverName)
	if validateServerNameErr != nil || !serverNameIsValid {
		log.Error("Server name validation failed")
		return nil, validateServerNameErr
	}
	sslCiphers := GetAppConfig("ssl_ciphers")
	if len(sslCiphers) == 0 {
		log.Error("SSL cipher configuration is not set")
		return nil, errors.New("ssl cipher configuration is not set")
	}
	cipherSuites := getCipherSuites(sslCiphers)
	if cipherSuites == nil {
		log.Error("Unable to get cipher suite")
		return nil, errors.New("unable to get cipher suite")
	}
	return &tls.Config{
		RootCAs:      rootCAs,
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS12,
		CipherSuites: cipherSuites,
	}, nil
}

func getCipherSuites(sslCiphers string) []uint16 {
	cipherSuiteArr := make([]uint16, 0, 5)
	cipherSuiteNameList := strings.Split(sslCiphers, ",")
	for _, cipherName := range cipherSuiteNameList {
		cipherName = strings.TrimSpace(cipherName)
		if len(cipherName) == 0 {
			continue
		}
		mapValue, ok := cipherSuiteMap[cipherName]
		if !ok {
			log.Error("Not recommended cipher suite")
			return nil
		}
		cipherSuiteArr = append(cipherSuiteArr, mapValue)
	}
	if len(cipherSuiteArr) > 0 {
		return cipherSuiteArr
	}
	return nil
}

// GetWorkKey generates work key by using root key
func GetWorkKey() ([]byte, error) {
	// get root key by key components
	rootKey, genRootKeyErr := genRootKey(componentFilePath, saltFilePath)
	if genRootKeyErr != nil {
		log.Error("Failed to generate root key by key components")
		return nil, genRootKeyErr
	}

	// decrypt work key by root key.
	workKey, decryptedWorkKeyErr := decryptKey(rootKey, encryptedWorkKeyFilePath, workKeyNonceFilePath)
	// clear root keyfv
	ClearByteArray(rootKey)
	if decryptedWorkKeyErr != nil {
		log.Error(decryptedWorkKeyErr.Error())
		return nil, decryptedWorkKeyErr
	}
	return workKey, nil
}

func getJwtPrivateKeyPwd() ([]byte, error) {
	// get work key
	workKey, getWorkKeyErr := GetWorkKey()
	if getWorkKeyErr != nil {
		log.Error("failed to get work key")
		return nil, getWorkKeyErr
	}

	// decrypt jwt private key password by root key.
	jwtPrivateKeyPwd, decryptedJwtPrivateKeyPwdErr := decryptKey(workKey, encryptedJwtPrivateKeyPwdFilePath,
		jwtPrivateKeyPwdNonceFilePath)
	// clear work key
	ClearByteArray(workKey)
	if decryptedJwtPrivateKeyPwdErr != nil {
		log.Error(decryptedJwtPrivateKeyPwdErr.Error())
		return nil, decryptedJwtPrivateKeyPwdErr
	}
	log.Info("Succeed to decrypt jwt private key password.")
	return jwtPrivateKeyPwd, nil
}

// EncryptAndSaveJwtPwd encrypts and save JWT password
func EncryptAndSaveJwtPwd(jwtPrivateKeyPwd *[]byte) error {
	pwdIsValid, err := validateJwtPassword(jwtPrivateKeyPwd)
	if err != nil || !pwdIsValid {
		log.Error(err)
		ClearByteArray(*jwtPrivateKeyPwd)
		return err
	}
	jwtPrivateKeyPwdNonce := make([]byte, NonceSize, 20)
	_, jwtPrivateKeyPwdNonceErr := rand.Read(jwtPrivateKeyPwdNonce)
	if jwtPrivateKeyPwdNonceErr != nil {
		errMsg := "failed to generate random jwt private key password nonce"
		log.Error(errMsg)
		ClearByteArray(*jwtPrivateKeyPwd)
		return errors.New(errMsg)
	}
	// get work key
	workKey, getWorkKeyErr := GetWorkKey()
	if getWorkKeyErr != nil {
		log.Error("Failed to get work key")
		ClearByteArray(*jwtPrivateKeyPwd)
		ClearByteArray(jwtPrivateKeyPwdNonce)
		return getWorkKeyErr
	}
	encryptedJwtPrivateKeyPwd, encryptedJwtPrivateKeyPwdErr := EncryptByAES256GCM(*jwtPrivateKeyPwd,
		workKey, jwtPrivateKeyPwdNonce)
	ClearByteArray(*jwtPrivateKeyPwd)
	ClearByteArray(workKey)
	if encryptedJwtPrivateKeyPwdErr != nil {
		errMsg := "failed to encrypt jwt private key password"
		log.Error(errMsg)
		ClearByteArray(jwtPrivateKeyPwdNonce)
		return errors.New(errMsg)
	}

	writeEncryptedPwdErr := ioutil.WriteFile(encryptedJwtPrivateKeyPwdFilePath,
		encryptedJwtPrivateKeyPwd, keyFileMode)
	writeNonceErr := ioutil.WriteFile(jwtPrivateKeyPwdNonceFilePath, jwtPrivateKeyPwdNonce, keyFileMode)
	ClearByteArray(encryptedJwtPrivateKeyPwd)
	ClearByteArray(jwtPrivateKeyPwdNonce)
	if writeEncryptedPwdErr != nil || writeNonceErr != nil {
		errMsg := "failed to write encrypt jwt private key password and nonce to file"
		log.Error(errMsg)
		return errors.New(errMsg)
	}
	log.Info("Succeed to encrypt and save jwt private key password and nonce to file.")
	return nil
}

// InitRootKeyAndWorkKey initializes root key and work key
func InitRootKeyAndWorkKey() error {
	// generate and save random root key components if not exist
	if !isFileOrDirExist(componentFilePath) || !isFileOrDirExist(saltFilePath) {
		genRandRootKeyComponentErr := genRandRootKeyComponent(componentFilePath, saltFilePath)
		if genRandRootKeyComponentErr != nil {
			log.Error("Failed to generate random key")
			return genRandRootKeyComponentErr
		}
		log.Info("Succeed to generate random key components and salt.")
	}

	// generate and save encrypted work key if not exist.
	if !isFileOrDirExist(encryptedWorkKeyFilePath) || !isFileOrDirExist(workKeyNonceFilePath) {
		// get root key by key components
		rootKey, genRootKeyErr := genRootKey(componentFilePath, saltFilePath)
		if genRootKeyErr != nil {
			log.Error("Failed to generate root key")
			return genRootKeyErr
		}
		log.Info("Succeed to generate root key by key components.")
		workKey, genAndSaveWorkKeyErr := genAndSaveWorkKey(rootKey, encryptedWorkKeyFilePath, workKeyNonceFilePath)
		ClearByteArray(workKey)
		ClearByteArray(rootKey)
		if genAndSaveWorkKeyErr != nil {
			log.Error("Failed to generate and save work key")
			return genAndSaveWorkKeyErr
		}
		log.Info("Succeed to generate and save encrypted work key and nonce.")
	}
	return nil
}

func genAndSaveWorkKey(rootKey []byte, encryptedWorkKeyFilePath string, workKeyNonceFilePath string) ([]byte, error) {
	workKey := make([]byte, keySize, 50)
	_, workKeyErr := rand.Read(workKey)
	if workKeyErr != nil {
		return nil, fmt.Errorf("failed to generate random work secret key")
	}
	workKeyNonce := make([]byte, NonceSize, 20)
	_, workKeyNonceErr := rand.Read(workKeyNonce)
	if workKeyNonceErr != nil {
		ClearByteArray(workKey)
		return nil, fmt.Errorf("failed to generate random work key nonce")
	}
	encryptedWorkKey, encryptedWorkKeyErr := EncryptByAES256GCM(workKey, rootKey, workKeyNonce)
	if encryptedWorkKeyErr != nil {
		ClearByteArray(workKey)
		ClearByteArray(workKeyNonce)
		return nil, fmt.Errorf("failed to encrypt work secret key")
	}

	writeEncryptedWorkKeyErr := ioutil.WriteFile(encryptedWorkKeyFilePath, encryptedWorkKey, keyFileMode)
	writeWorkKeyNonceErr := ioutil.WriteFile(workKeyNonceFilePath, workKeyNonce, keyFileMode)
	ClearByteArray(encryptedWorkKey)
	ClearByteArray(workKeyNonce)
	if writeEncryptedWorkKeyErr != nil || writeWorkKeyNonceErr != nil {
		ClearByteArray(workKey)
		return nil, fmt.Errorf("failed to write work secret key and nonce to file")
	}
	return workKey, nil
}

func decryptKey(key []byte, encryptedKeyFilePath string, keyNonceFilePath string) ([]byte, error) {
	encryptedKey, readEncryptedKeyErr := ioutil.ReadFile(encryptedKeyFilePath)
	if readEncryptedKeyErr != nil {
		return nil, fmt.Errorf("failed to read encrypted key from file")
	}

	keyNonce, readKeyNonceErr := ioutil.ReadFile(keyNonceFilePath)
	if readKeyNonceErr != nil {
		ClearByteArray(encryptedKey)
		return nil, fmt.Errorf("failed to read nonce from file")
	}
	key, decryptedKeyErr := DecryptByAES256GCM(encryptedKey, key, keyNonce)
	ClearByteArray(encryptedKey)
	ClearByteArray(keyNonce)
	if decryptedKeyErr != nil {
		return nil, fmt.Errorf("failed to decrypt secret key")
	}
	return key, nil
}

func genRootKey(componentFilePath string, saltFilePath string) ([]byte, error) {
	// get component from user input
	if len(*KeyComponentFromUserStr) == 0 {
		log.Error("Parameter of key is not provided")
		return nil, fmt.Errorf("parameter of key is not provided")
	}
	componentFromUser := make([]byte, componentSize, 300)
	for i := 0; i < componentSize && i < len(*KeyComponentFromUserStr); i++ {
		componentFromUser[i] = (*KeyComponentFromUserStr)[i]
	}

	// get component from file
	componentFromFile, readComponentErr := ioutil.ReadFile(componentFilePath)
	if readComponentErr != nil {
		ClearByteArray(componentFromUser)
		return nil, fmt.Errorf("failed to read random key components from file")
	}
	salt, readSaltErr := ioutil.ReadFile(saltFilePath)
	if readSaltErr != nil {
		ClearByteArray(componentFromUser)
		ClearByteArray(componentFromFile)
		return nil, fmt.Errorf("failed to read random key salt from file")
	}

	// get component from hard code
	componentFromHardCode := make([]byte, componentSize, 300)
	componentFromHardCodeTmp := []byte(componentContent)
	for i := 0; i < componentSize && i < len(componentFromHardCodeTmp); i++ {
		componentFromHardCode[i] = componentFromHardCodeTmp[i]
	}

	// generate root key by key components
	tmpComponent := make([]byte, componentSize, 300)
	for i := 0; i < componentSize; i++ {
		tmpComponent[i] = componentFromUser[i] ^ componentFromFile[i] ^ componentFromHardCode[i]
	}
	rootKey := pbkdf2.Key(tmpComponent, salt, iterationNum, keySize, sha256.New)
	ClearByteArray(componentFromUser)
	ClearByteArray(componentFromFile)
	ClearByteArray(componentFromHardCode)
	ClearByteArray(componentFromHardCodeTmp)
	ClearByteArray(salt)
	ClearByteArray(tmpComponent)
	return rootKey, nil
}

func genRandRootKeyComponent(componentFilePath string, saltFilePath string) error {
	component := make([]byte, componentSize, 300)
	_, generateComponentErr := rand.Read(component)
	if generateComponentErr != nil {
		return fmt.Errorf("failed to generate random key component")
	}

	salt := make([]byte, componentSize, 300)
	_, generateSaltErr := rand.Read(salt)
	if generateSaltErr != nil {
		ClearByteArray(component)
		return fmt.Errorf("failed to generate random key salt")
	}
	writeComponent1FileErr := ioutil.WriteFile(componentFilePath, component, keyFileMode)
	writeSaltFileErr := ioutil.WriteFile(saltFilePath, salt, keyFileMode)
	// clear component
	ClearByteArray(component)
	// clear salt
	ClearByteArray(salt)
	if writeComponent1FileErr != nil || writeSaltFileErr != nil {
		return fmt.Errorf("failed to write random key component and salt to file")
	}
	return nil
}

func isFileOrDirExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

// ClearByteArray clears byte array from memory
func ClearByteArray(data []byte) {
	if data == nil {
		return
	}
	for i := 0; i < len(data); i++ {
		data[i] = 0
	}
}

// ValidateDbParams validates db parameters
func ValidateDbParams(dbPwd string) (bool, error) {
	dbPwdBytes := []byte(dbPwd)
	dbPwdIsValid, validateDbPwdErr := validatePassword(&dbPwdBytes)
	if validateDbPwdErr != nil || !dbPwdIsValid {
		return dbPwdIsValid, validateDbPwdErr
	}
	return true, nil
}

// Validate password
func validatePassword(password *[]byte) (bool, error) {
	if len(*password) >= minPasswordSize && len(*password) <= maxPasswordSize {
		// password must satisfy any two conditions
		pwdValidCount := getPasswordValidCount(password)
		if pwdValidCount < maxPasswordCount {
			return false, errors.New("password must contain at least two types of the either one lowercase" +
				" character, one uppercase character, one digit or one special character")
		}
	} else {
		return false, errors.New("password must have minimum length of 8 and maximum of 16")
	}
	return true, nil
}

// To get password valid count
func getPasswordValidCount(password *[]byte) int {
	var pwdValidCount = 0
	pwdIsValid, err := regexp.Match(singleDigitRegex, *password)
	if pwdIsValid && err == nil {
		pwdValidCount++
	}
	pwdIsValid, err = regexp.Match(lowerCaseRegex, *password)
	if pwdIsValid && err == nil {
		pwdValidCount++
	}
	pwdIsValid, err = regexp.Match(upperCaseRegex, *password)
	if pwdIsValid && err == nil {
		pwdValidCount++
	}
	// space validation for password complexity is not added
	// as jwt decrypt fails if space is included in password
	pwdIsValid, err = regexp.Match(specialCharRegex, *password)
	if pwdIsValid && err == nil {
		pwdValidCount++
	}
	return pwdValidCount
}
