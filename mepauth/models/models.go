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

// model in this package
package models

import (
	"time"

	"github.com/astaxie/beego/orm"
)

func init() {
	orm.RegisterModel(new(AuthInfoRecord))
	orm.RegisterModel(new(RouteRecord))
}

type AuthInfoRecord struct {
	AppInsId string `orm:"pk"`
	Ak       string
	Sk       string
	Nonce    string
}

type StateType string

type AkSessionInfo struct {
	ClearTimer      *time.Timer
	State           StateType
	Ak              string
	ValidateCounter int64
}

type TokenInfo struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   uint32 `json:"expires_in"`
}

type AuthInfo struct {
	Credentials Credentials `json:"credentials"`
}

type Credentials struct {
	AccessKeyId string `json:"accessKeyId"`
	SecretKey   string `json:"secretKey"`
}

type AppAuthInfo struct {
	AuthInfo AuthInfo `json:"authInfo"`
}

type RouteRecord struct {
	Id      int64  `json:"id"`
	RouteId string `json:"routeId"`
	AppId   string `json:"appId"`
	SerName string `json:"serName"`
}

type RouteInfo struct {
	Id      int64   `json:"routeId"`
	AppId   string  `json:"appId"`
	SerInfo SerInfo `orm:"type(json)" json:"serInfo"`
}

type SerInfo struct {
	SerName string   `json:"serName"`
	Uris    []string `json:"uris"`
}
