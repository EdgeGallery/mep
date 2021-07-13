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

// Package util implements mep auth utility functions and contain constants
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
	separator     string = "/"
	lineSeparator string = "\n"
	DateFormat    string = "20060102T150405Z"
	algorithm     string = "SDK-HMAC-SHA256"
	DateHeader    string = "x-sdk-date"
	HostHeader    string = "Host"
)

// Sign data structure
type Sign struct {
	SecretKey []byte
}

// GetSignature to obtain signature for a given request
func (sig *Sign) GetSignature(req *http.Request) (string, error) {
	if req == nil {
		return "", errors.New("request is nil")
	}
	// construct canonical request
	canonicalRequest, errGetCanonicalRequest := sig.getCanonicalRequest(req)
	if errGetCanonicalRequest != nil {
		return "", errGetCanonicalRequest
	}
	// create string to sign
	stringToSign, errGetStringToSign := sig.getStringToSign(canonicalRequest, req.Header.Get(DateHeader))
	if errGetStringToSign != nil {
		return "", errGetStringToSign
	}
	// calculate signature
	signature, errCalculateSignature := sig.calculateSignature(stringToSign)
	if errCalculateSignature != nil {
		return "", errCalculateSignature
	}
	return signature, nil
}

// construct canonical request and return
func (sig *Sign) getCanonicalRequest(req *http.Request) (string, error) {

	// begin construct canonical request
	// request method
	method := req.Method
	// request uri
	uri := sig.getCanonicalUri(req)
	// query string
	query := sig.getCanonicalQueryString(req)
	// request headers
	headersReq := sig.getCanonicalHeaders(req)
	// signed headers
	headersSign := sig.getSignedHeaders(req)
	// request body
	hexEncodeBody, errGetRequestBodyHash := sig.getRequestBodyHash(req)
	if errGetRequestBodyHash != nil {
		return "", errGetRequestBodyHash
	}
	// construct complete
	return strings.Join([]string{method, uri, query, headersReq, headersSign, hexEncodeBody}, lineSeparator), nil
}

// construct canonical uri can return
func (sig *Sign) getCanonicalUri(req *http.Request) string {
	// split uri to []string
	paths := strings.Split(req.URL.Path, separator)
	var uris []string
	for _, path := range paths {
		// ignore the empty string and relative path string
		if path == "" || path == "." || path == ".." {
			continue
		}
		uris = append(uris, url.QueryEscape(path))
	}
	// create canonical uri
	canonicalUri := separator + strings.Join(uris, separator)
	// check the uri suffix
	if strings.HasSuffix(canonicalUri, separator) {
		return canonicalUri
	} else {
		return canonicalUri + separator
	}
}

// construct canonical query string and return
func (sig *Sign) getCanonicalQueryString(req *http.Request) string {

	var params []string
	for key, values := range req.URL.Query() {
		for _, value := range values {
			// canonical query string with each value
			params = append(params, url.QueryEscape(key)+"="+url.QueryEscape(value))
		}
	}
	sort.Strings(params)
	return strings.Join(params, "&")
}

// construct canonical request headers and return
func (sig *Sign) getCanonicalHeaders(req *http.Request) string {

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
	return strings.Join(headers, lineSeparator) + lineSeparator
}

// return signed headers list as string
func (sig *Sign) getSignedHeaders(req *http.Request) string {

	var headers []string
	for key := range req.Header {
		headers = append(headers, strings.ToLower(key))
	}
	sort.Strings(headers)
	return strings.Join(headers, ";")
}

// get request body, do sha256 encrypt and hex encode
func (sig *Sign) getRequestBodyHash(req *http.Request) (string, error) {

	reqBody, errGetRequestBody := sig.getRequestBody(req)
	if errGetRequestBody != nil {
		return "", errGetRequestBody
	}
	hexEncode, errHexEncode := sig.hexEncodeSHA256Hash(reqBody)
	if errHexEncode != nil {
		return "", errHexEncode
	}
	return hexEncode, nil
}

// get request body bytes
func (sig *Sign) getRequestBody(req *http.Request) ([]byte, error) {

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
func (sig *Sign) hexEncodeSHA256Hash(bytes []byte) (string, error) {

	hash := sha256.New()
	_, errWrite := hash.Write(bytes)
	if errWrite != nil {
		return "", errWrite
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// construct string to sign and return
func (sig *Sign) getStringToSign(canonicalRequest string, dateTime string) (string, error) {

	// begin construct string to sign, the string contains algorithm , date time and canonical request
	// canonical request
	hexEncodeReq, errHexEncode := sig.hexEncodeSHA256Hash([]byte(canonicalRequest))
	if errHexEncode != nil {
		return "", errHexEncode
	}
	// construct complete
	return strings.Join([]string{algorithm, dateTime, hexEncodeReq}, lineSeparator), nil
}

// calculate the signature with string to sign and secret key.
func (sig *Sign) calculateSignature(stringToSign string) (encodeStr string, err error) {
	defer func() {
		if err1 := recover(); err1 != nil {
			log.Error("panic handled:", err1)
			err = fmt.Errorf("recover panic as %s", err1)
		}
	}()

	h := hmac.New(sha256.New, sig.SecretKey)
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
