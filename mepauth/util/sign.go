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

// signature service
package util

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	SEPARATOR     string = "/"
	LineSeparator string = "\n"
	DateFormat    string = "20060102T150405Z"
	ALGORITHM     string = "SDK-HMAC-SHA256"
	DateHeader    string = "x-sdk-date"
	HostHeader    string = "Host"
)

type Sign struct {
	AccessKey string
	SecretKey []byte
}

// get signature from request
func (sig *Sign) GetSignature(req *http.Request) (string, error) {
	if req == nil {
		return "", errors.New("request is nil")
	}
	// construct canonical request
	canonicalRequest, errGetCanonicalRequest := getCanonicalRequest(req)
	if errGetCanonicalRequest != nil {
		return "", errGetCanonicalRequest
	}
	// create string to sign
	stringToSign, errGetStringToSign := getStringToSign(canonicalRequest, req.Header.Get(DateHeader))
	if errGetStringToSign != nil {
		return "", errGetStringToSign
	}
	// calculate signature
	signature, errCalculateSignature := calculateSignature(stringToSign, sig.SecretKey)
	if errCalculateSignature != nil {
		return "", errCalculateSignature
	}
	return signature, nil
}

// construct canonical request and return
func getCanonicalRequest(req *http.Request) (string, error) {

	// begin construct canonical request
	// request method
	method := req.Method
	// request uri
	uri := getCanonicalUri(req)
	// query string
	query := getCanonicalQueryString(req)
	// request headers
	headersReq := getCanonicalHeaders(req)
	// signed headers
	headersSign := getSignedHeaders(req)
	// request body
	hexEncodeBody, errGetRequestBodyHash := getRequestBodyHash(req)
	if errGetRequestBodyHash != nil {
		return "", errGetRequestBodyHash
	}
	// construct complete
	return strings.Join([]string{method, uri, query, headersReq, headersSign, hexEncodeBody}, LineSeparator), nil
}

// construct canonical uri can return
func getCanonicalUri(req *http.Request) string {
	// split uri to []string
	paths := strings.Split(req.URL.Path, SEPARATOR)
	var uris []string
	for _, path := range paths {
		// ignore the empty string and relative path string
		if path == "" || path == "." || path == ".." {
			continue
		}
		uris = append(uris, url.QueryEscape(path))
	}
	// create canonical uri
	canonicalUri := SEPARATOR + strings.Join(uris, SEPARATOR)
	// check the uri suffix
	if strings.HasSuffix(canonicalUri, SEPARATOR) {
		return canonicalUri
	} else {
		return canonicalUri + SEPARATOR
	}
}

// construct canonical query string and return
func getCanonicalQueryString(req *http.Request) string {

	var params []string
	for key, values := range req.URL.Query() {
		for _, value := range values {
			// canonical query string with each value
			params = append(params, url.QueryEscape(key) + "=" + url.QueryEscape(value))
		}
	}
	sort.Strings(params)
	return strings.Join(params, "&")
}

// construct canonical request headers and return
func getCanonicalHeaders(req *http.Request) string {

	var headers []string
	for key, values := range req.Header {
		sort.Strings(values)
		var val []string
		for _, value := range values {
			// trim the each header value
			val = append(val, strings.TrimSpace(value))
		}
		// canonical header by one key and all values
		headers = append(headers, strings.ToLower(key)+":"+strings.Join(val, ","))
	}
	sort.Strings(headers)
	return strings.Join(headers, LineSeparator) + LineSeparator
}

// return signed headers list as string
func getSignedHeaders(req *http.Request) string {

	var headers []string
	for key := range req.Header {
		headers = append(headers, strings.ToLower(key))
	}
	sort.Strings(headers)
	return strings.Join(headers, ";")
}

// get request body, do sha256 encrypt and hex encode
func getRequestBodyHash(req *http.Request) (string, error) {

	reqBody, errGetRequestBody := getRequestBody(req)
	if errGetRequestBody != nil {
		return "", errGetRequestBody
	}
	hexEncode, errHexEncode := hexEncodeSHA256Hash(reqBody)
	if errHexEncode != nil {
		return "", errHexEncode
	}
	return hexEncode, nil
}

// get request body bytes
func getRequestBody(req *http.Request) ([]byte, error) {

	if req.Body == nil {
		return []byte(""), nil
	}
	body, errReadAll := ioutil.ReadAll(req.Body)
	if errReadAll != nil {
		return []byte(""), errReadAll
	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return body, nil
}

// HexEncode(Hash(bytes)) with SHA256
func hexEncodeSHA256Hash(bytes []byte) (string, error) {

	hash := sha256.New()
	_, errWrite := hash.Write(bytes)
	if errWrite != nil {
		return "", errWrite
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// construct string to sign and return
func getStringToSign(canonicalRequest string, dateTime string) (string, error) {

	// begin construct string to sign, the string contains algorithm , date time and canonical request
	// canonical request
	hexEncodeReq, errHexEncode := hexEncodeSHA256Hash([]byte(canonicalRequest))
	if errHexEncode != nil {
		return "", errHexEncode
	}
	// construct complete
	return strings.Join([]string{ALGORITHM, dateTime, hexEncodeReq}, LineSeparator), nil
}

// calculate the signature with string to sign and secret key.
func calculateSignature(stringToSign string, secretKey []byte) (encodeStr string, err error) {
	defer func() {
		if err1 := recover(); err1 != nil {
			log.Error("panic handled:", err1)
			err = fmt.Errorf("recover panic as %s", err1)
		}
	}()

	h := hmac.New(sha256.New, secretKey)
	_, errWrite := h.Write([]byte(stringToSign))
	if errWrite != nil {
		return "", errWrite
	}
	encodeStr = hex.EncodeToString(h.Sum(nil))
	rs := reflect.ValueOf(h).Elem()
	ClearByteArray(rs.FieldByName("ipad").Bytes())
	ClearByteArray(rs.FieldByName("opad").Bytes())
	return encodeStr, nil
}
