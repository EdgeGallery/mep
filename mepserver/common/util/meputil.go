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

// Package path implements mep server utility functions and constants
package util

import (
	"bufio"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/pbkdf2"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/rest"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"github.com/apache/servicecomb-service-center/server/core"
	"github.com/apache/servicecomb-service-center/server/core/backend"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/registry"
	svcutil "github.com/apache/servicecomb-service-center/server/service/util"
	"github.com/go-playground/validator/v10"
)

const KeyFileMode os.FileMode = 0600

const KeySize int = 32
const NonceSize int = 12
const IterationNum int = 100000
const ComponentSize int = 256

const ComponentFilePath string = "cprop/c_properties"
const SaltFilePath string = "sprop/s_properties"
const EncryptedWorkKeyFilePath string = "wprop/w_properties"
const WorkKeyNonceFilePath string = "wnprop/wn_properties"
const EncryptedCertSecFilePath string = "ssl/cert_pwd"
const CertSecNonceFilePath string = "ssl/cert_pwd_nonce"

var KeyComponentFromUserStr *[]byte

// put k,v into map
func InfoToProperties(properties map[string]string, key string, value string) {
	if value != "" {
		properties[key] = value
	}
}

// trans json to obj
func JsonTextToObj(jsonText string) (interface{}, error) {
	data := []byte(jsonText)
	var jsonMap interface{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&jsonMap)
	if err != nil {
		return nil, err
	}
	return jsonMap, nil
}

// get host port in uri
func GetHostPort(uri string) (string, int) {
	const zeroPort int = 0
	idx := strings.LastIndex(uri, ":")
	domain := uri
	port := zeroPort
	var err error
	if idx > 0 {
		port, err = strconv.Atoi(uri[idx+1:])
		if err != nil {
			port = zeroPort
		}
		domain = uri[:idx]
	}
	return domain, port
}

// get tags in http request
func GetHTTPTags(r *http.Request) (url.Values, []string) {
	var ids []string
	query := r.URL.Query()
	keys := query.Get("tags")
	if len(keys) > 0 {
		ids = strings.Split(keys, ",")
	}

	return query, ids
}

// write err response
func HttpErrResponse(w http.ResponseWriter, statusCode int, obj interface{}) {
	if obj == nil {
		w.Header().Set(rest.HEADER_RESPONSE_STATUS, strconv.Itoa(statusCode))
		w.Header().Set(rest.HEADER_CONTENT_TYPE, rest.CONTENT_TYPE_TEXT)
		w.WriteHeader(statusCode)
		return
	}

	objJSON, err := json.Marshal(obj)
	if err != nil {
		log.Errorf(nil, "json masrshalling failed")
		return
	}
	w.Header().Set(rest.HEADER_RESPONSE_STATUS, strconv.Itoa(statusCode))
	w.Header().Set(rest.HEADER_CONTENT_TYPE, rest.CONTENT_TYPE_JSON)
	w.WriteHeader(statusCode)
	_, err = fmt.Fprintln(w, string(objJSON))
	if err != nil {
		log.Errorf(nil, "send http response fail")
	}
}

// heartbeat use put to update a service register info
func Heartbeat(ctx context.Context, mp1SvcId string) error {
	serviceID := mp1SvcId[:len(mp1SvcId)/2]
	instanceID := mp1SvcId[len(mp1SvcId)/2:]
	req := &proto.HeartbeatRequest{
		ServiceId:  serviceID,
		InstanceId: instanceID,
	}
	_, err := core.InstanceAPI.Heartbeat(ctx, req)
	return err
}

// get service instance by serviceId
func GetServiceInstance(ctx context.Context, serviceId string) (*proto.MicroServiceInstance, error) {
	domainProject := util.ParseDomainProject(ctx)
	serviceID := serviceId[:len(serviceId)/2]
	instanceID := serviceId[len(serviceId)/2:]
	instance, err := svcutil.GetInstance(ctx, domainProject, serviceID, instanceID)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		err = fmt.Errorf("domainProject %s sservice Id %s not exist", domainProject, serviceID)
	}
	return instance, err
}

// get instance by key
func FindInstanceByKey(result url.Values) (*proto.FindInstancesResponse, error) {
	serCategoryId := result.Get("ser_category_id")
	scopeOfLocality := result.Get("scope_of_locality")
	consumedLocalOnly := result.Get("consumed_local_only")
	isLocal := result.Get("is_local")
	isQueryAllSvc := serCategoryId == "" && scopeOfLocality == "" && consumedLocalOnly == "" && isLocal == ""
	opts := []registry.PluginOp{
		registry.OpGet(registry.WithStrKey("/cse-sr/inst/files///"), registry.WithPrefix()),
	}
	resp, err := backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("query from etch error")
	}
	var findResp []*proto.MicroServiceInstance
	for _, value := range resp.Kvs {
		var instance map[string]interface{}
		err = json.Unmarshal(value.Value, &instance)
		if err != nil {
			return nil, fmt.Errorf("string convert to instance failed")
		}
		dci := &proto.DataCenterInfo{Name: "", Region: "", AvailableZone: ""}
		instance["datacenterinfo"] = dci
		message, err := json.Marshal(&instance)
		if err != nil {
			log.Errorf(nil, "instance convert to string failed")
			return nil, err
		}
		var ins *proto.MicroServiceInstance
		err = json.Unmarshal(message, &ins)
		if err != nil {
			log.Errorf(nil, "String convert to MicroServiceInstance failed!")
			return nil, err
		}
		property := ins.Properties
		if isQueryAllSvc && property != nil {
			findResp = append(findResp, ins)
		} else if strings.EqualFold(property["serCategory/id"], serCategoryId) ||
			strings.EqualFold(property["ConsumedLocalOnly"], consumedLocalOnly) ||
			strings.EqualFold(property["ScopeOfLocality"], scopeOfLocality) ||
			strings.EqualFold(property["IsLocal"], isLocal) {
			findResp = append(findResp, ins)
		}
	}
	if len(findResp) == 0 {
		return nil, fmt.Errorf("null")
	}
	response := &proto.Response{Code: 0, Message: ""}
	ret := &proto.FindInstancesResponse{Response: response, Instances: findResp}
	return ret, nil
}

// set map value
func SetMapValue(theMap map[string]interface{}, key string, val interface{}) {
	mapValue, ok := theMap[key]
	if !ok || mapValue == nil {
		theMap[key] = val
	}
}

// get the index of the string in []string
func StringContains(arr []string, val string) (index int) {
	index = -1
	for i := 0; i < len(arr); i++ {
		if arr[i] == val {
			index = i
			return
		}
	}
	return
}

// validate UUID
func ValidateUUID(id string) error {
	if len(id) != 0 {
		validate := validator.New()
		return validate.Var(id, "required,uuid")
	}
	return nil
}

// validate serviceId
func ValidateServiceID(serID string) error {
	return ValidateRegexp(serID, "[0-9a-f]{32}",
		"service ID validation failed")
}

// validate by reg
func ValidateRegexp(strToCheck string, regexStr string, errMsg string) error {
	match, err := regexp.MatchString(regexStr, strToCheck)
	if err != nil {
		return err
	}
	if !match {
		return errors.New(errMsg)
	}
	return nil
}

// get subscribe key path
func GetSubscribeKeyPath(subscribeType string) string {
	var subscribeKeyPath string
	if subscribeType == SerAvailabilityNotificationSubscription {
		subscribeKeyPath = AvailAppSubKeyPath
	} else {
		subscribeKeyPath = EndAppSubKeyPath
	}
	return subscribeKeyPath
}

// validate appInstanceId in header
func ValidateAppInstanceIdWithHeader(id string, r *http.Request) error {
	if id == r.Header.Get("X-AppinstanceID") {
		return nil
	}
	return errors.New("UnAuthorization to access the resource")
}

// get resource info
func GetResourceInfo(r *http.Request) string {
	resource := r.URL.String()
	if resource == "" {
		return "UNKNOWN"
	}
	return resource
}

// get method from request
func GetMethod(r *http.Request) string {
	method := r.Method
	if method == "" {
		return "GET"
	}
	return method
}

// get appInstanceId from request
func GetAppInstanceId(r *http.Request) string {
	query, _ := GetHTTPTags(r)
	return query.Get(":appInstanceId")
}

// get clientIp from request
func GetClientIp(r *http.Request) string {
	clientIp := r.Header.Get("X-Real-Ip")
	if clientIp == "" {
		clientIp = "UNKNOWN_IP"
	}
	return clientIp
}

func ValidateKeyComponentUserInput(keyComponentUserStr *[]byte) error {
	if len(*keyComponentUserStr) < ComponentSize {
		log.Errorf(nil, "key component user string length is not valid")
		return fmt.Errorf("key component user string length is not valid")
	}
	return nil
}

// use aes 256 gcm algo to encrypt secret keys
func EncryptByAES256GCM(plaintext []byte, key []byte, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Errorf(nil, "failed to create aes cipher")
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Errorf(nil, "failed to wrap cipher")
		return nil, err
	}

	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nil
}

// use aes 256 gcm algo to decrypt secret keys
func DecryptByAES256GCM(ciphertext []byte, key []byte, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Errorf(nil, "failed to create aes cipher")
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Errorf(nil, "failed to wrap cipher")
		return nil, err
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		log.Errorf(nil, "failed to decrypt secret key")
		return nil, err
	}

	return plaintext, nil
}

// Generate work key by using root key
func GetWorkKey() ([]byte, error) {
	// get root key by key components
	rootKey, genRootKeyErr := genRootKey(ComponentFilePath, SaltFilePath)
	if genRootKeyErr != nil {
		log.Errorf(nil, "failed to generate root key by key components")
		return nil, genRootKeyErr
	}
	log.Info("Succeed to generate root key by key components.")

	// decrypt work key by root key.
	workKey, decryptedWorkKeyErr := decryptKey(rootKey, EncryptedWorkKeyFilePath, WorkKeyNonceFilePath)
	// clear root key
	ClearByteArray(rootKey)
	if decryptedWorkKeyErr != nil {
		log.Errorf(nil, decryptedWorkKeyErr.Error())
		return nil, decryptedWorkKeyErr
	}
	log.Info("Succeed to decrypt work key.")
	return workKey, nil
}

// Init root key and work key
func InitRootKeyAndWorkKey() error {
	// generate and save random root key components if not exist
	if !IsFileOrDirExist(ComponentFilePath) || !IsFileOrDirExist(SaltFilePath) {
		genRandRootKeyComponentErr := genRandRootKeyComponent(ComponentFilePath, SaltFilePath)
		if genRandRootKeyComponentErr != nil {
			log.Errorf(nil, "failed to generate random key")
			return genRandRootKeyComponentErr
		}
		log.Info("Succeed to generate random key components and salt.")
	}

	// generate and save encrypted work key if not exist.
	if !IsFileOrDirExist(EncryptedWorkKeyFilePath) || !IsFileOrDirExist(WorkKeyNonceFilePath) {
		// get root key by key components
		rootKey, genRootKeyErr := genRootKey(ComponentFilePath, SaltFilePath)
		if genRootKeyErr != nil {
			log.Errorf(nil, "failed to generate root key")
			return genRootKeyErr
		}
		log.Info("Succeed to generate root key by key components.")
		workKey, genAndSaveWorkKeyErr := genAndSaveWorkKey(rootKey, EncryptedWorkKeyFilePath, WorkKeyNonceFilePath)
		ClearByteArray(workKey)
		ClearByteArray(rootKey)
		if genAndSaveWorkKeyErr != nil {
			log.Errorf(nil, "failed to generate and save work key")
			return genAndSaveWorkKeyErr
		}
		log.Info("Succeed to generate and save encrypted work key and nonce.")
	}
	return nil
}

func genAndSaveWorkKey(rootKey []byte, encryptedWorkKeyFilePath string, workKeyNonceFilePath string) ([]byte, error) {
	workKey := make([]byte, KeySize, 50)
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

	writeEncryptedWorkKeyErr := ioutil.WriteFile(encryptedWorkKeyFilePath, encryptedWorkKey, KeyFileMode)
	writeWorkKeyNonceErr := ioutil.WriteFile(workKeyNonceFilePath, workKeyNonce, KeyFileMode)
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
		log.Errorf(nil, "parameter of key is not provided")
		return nil, fmt.Errorf("parameter of key is not provided")
	}
	componentFromUser := make([]byte, ComponentSize, 300)
	for i := 0; i < ComponentSize && i < len(*KeyComponentFromUserStr); i++ {
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
	componentFromHardCode := make([]byte, ComponentSize, 300)
	componentFromHardCodeTmp := []byte(ComponentContent)
	for i := 0; i < ComponentSize && i < len(componentFromHardCodeTmp); i++ {
		componentFromHardCode[i] = componentFromHardCodeTmp[i]
	}

	// generate root key by key components
	tmpComponent := make([]byte, ComponentSize, 300)
	for i := 0; i < ComponentSize; i++ {
		tmpComponent[i] = componentFromUser[i] ^ componentFromFile[i] ^ componentFromHardCode[i]
	}
	rootKey := pbkdf2.Key(tmpComponent, salt, IterationNum, KeySize, sha256.New)
	ClearByteArray(componentFromUser)
	ClearByteArray(componentFromFile)
	ClearByteArray(componentFromHardCode)
	ClearByteArray(componentFromHardCodeTmp)
	ClearByteArray(salt)
	ClearByteArray(tmpComponent)
	return rootKey, nil
}

func genRandRootKeyComponent(componentFilePath string, saltFilePath string) error {
	component := make([]byte, ComponentSize, 300)
	_, generateComponentErr := rand.Read(component)
	if generateComponentErr != nil {
		return fmt.Errorf("failed to generate random key component")
	}

	salt := make([]byte, ComponentSize, 300)
	_, generateSaltErr := rand.Read(salt)
	if generateSaltErr != nil {
		ClearByteArray(component)
		return fmt.Errorf("failed to generate random key salt")
	}
	writeComponent1FileErr := ioutil.WriteFile(componentFilePath, component, KeyFileMode)
	writeSaltFileErr := ioutil.WriteFile(saltFilePath, salt, KeyFileMode)
	// clear component
	ClearByteArray(component)
	// clear salt
	ClearByteArray(salt)
	if writeComponent1FileErr != nil || writeSaltFileErr != nil {
		return fmt.Errorf("failed to write random key component and salt to file")
	}
	return nil
}

// check file of dir exist
func IsFileOrDirExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

// Clear byte array from memory
func ClearByteArray(data []byte) {
	if data == nil {
		return
	}
	for i := 0; i < len(data); i++ {
		data[i] = 0
	}
}

// Encrypt and save cert password
func EncryptAndSaveCertPwd(certPwd *[]byte) error {
	certPwdNonce := make([]byte, NonceSize, 20)
	_, certPwdNonceErr := rand.Read(certPwdNonce)
	if certPwdNonceErr != nil {
		errMsg := "failed to generate random cert password nonce"
		log.Errorf(nil, errMsg)
		ClearByteArray(*certPwd)
		return errors.New(errMsg)
	}
	// get work key
	workKey, getWorkKeyErr := GetWorkKey()
	if getWorkKeyErr != nil {
		log.Errorf(nil, "failed to get work key")
		ClearByteArray(*certPwd)
		ClearByteArray(certPwdNonce)
		return getWorkKeyErr
	}
	encryptedCertPwd, encryptedCertPwdErr := EncryptByAES256GCM(*certPwd, workKey, certPwdNonce)
	ClearByteArray(*certPwd)
	ClearByteArray(workKey)
	if encryptedCertPwdErr != nil {
		errMsg := "failed to encrypt cert password"
		log.Errorf(nil, errMsg)
		ClearByteArray(certPwdNonce)
		return errors.New(errMsg)
	}

	writeEncryptedPwdErr := ioutil.WriteFile(EncryptedCertSecFilePath,
		encryptedCertPwd, KeyFileMode)
	writeNonceErr := ioutil.WriteFile(CertSecNonceFilePath, certPwdNonce, KeyFileMode)
	ClearByteArray(encryptedCertPwd)
	ClearByteArray(certPwdNonce)
	if writeEncryptedPwdErr != nil || writeNonceErr != nil {
		errMsg := "failed to write encrypt cert password and nonce to file"
		log.Errorf(nil, errMsg)
		return errors.New(errMsg)
	}
	log.Info("Succeed to encrypt and save cert password and nonce to file.")
	return nil
}

// get cert pwd
func GetCertPwd() ([]byte, error) {
	// get work key
	workKey, getWorkKeyErr := GetWorkKey()
	if getWorkKeyErr != nil {
		log.Errorf(nil, "failed to get work key")
		return nil, getWorkKeyErr
	}

	// decrypt cert password by work key.
	certPwd, decryptedCertPwdErr := decryptKey(workKey, EncryptedCertSecFilePath,
		CertSecNonceFilePath)
	// clear work key
	ClearByteArray(workKey)
	if decryptedCertPwdErr != nil {
		log.Errorf(nil, decryptedCertPwdErr.Error())
		return nil, decryptedCertPwdErr
	}
	log.Info("Succeed to decrypt cert password.")
	return certPwd, nil
}

// Generate a strong ETag for the http message header. Using sha256
// hashing for generating the code.
// Example: etag -> "958028a29507104f180515b53eb29bdc15d1212679e6c2c8074782c3c1db1868"
func GenerateStrongETag(body []byte) string {
	return fmt.Sprintf("\"%x\"", sha256.Sum256(body))
}

// Checks whether the status code is in the success range from 200 to 299
func IsHttpStatusOK(statusCode int) bool {
	return statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices
}

type AppConfigProperties map[string]string

// read app.conf file to AppConfigProperties object
func readPropertiesFile(filename string) (AppConfigProperties, error) {

	if len(filename) == 0 {
		return nil, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		log.Errorf(nil, "Failed to open the file.")
		return nil, err
	}
	defer file.Close()
	config, err := scanConfig(file)
	if err != nil {
		log.Errorf(nil, "Failed to read the file.")
		return nil, err
	}
	return config, nil
}

func scanConfig(r io.Reader) (AppConfigProperties, error) {
	config := AppConfigProperties{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()
		if bytes.Contains(line, []byte("=")) {
			keyVal := bytes.Split(line, []byte("="))
			key := bytes.TrimSpace(keyVal[0])
			val := string(bytes.TrimSpace(keyVal[1]))
			config[string(key)] = val
		}
	}
	return config, scanner.Err()
}
