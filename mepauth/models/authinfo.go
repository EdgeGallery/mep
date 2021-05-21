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

// Package model contains mep auth data model
package models

import (
	"time"

	"github.com/astaxie/beego/orm"
)

func init() {
	orm.RegisterModel(new(AuthInfoRecord))
}

// AuthInfoRecord authentication information record data structure
type AuthInfoRecord struct {
	AppInsId string `orm:"pk"`
	Ak       string
	Sk       string
	Nonce    string
}

// AkSessionInfo AK session information data structure
type AkSessionInfo struct {
	ClearTimer      *time.Timer
	State           string
	Ak              string
	ValidateCounter int64
}

// TokenInfo token information data structure
type TokenInfo struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   uint32 `json:"expires_in"`
}

// AuthInfo authentication information data structure
type AuthInfo struct {
	Credentials Credentials `json:"credentials"`
}

// Credentials data structure
type Credentials struct {
	AccessKeyId string `json:"accessKeyId"`
	SecretKey   string `json:"secretKey"`
}

// AppAuthInfo application authentication information data structure
type AppAuthInfo struct {
	AuthInfo AuthInfo `json:"authInfo"`
}
