/*
 * Copyright 2021 Huawei Technologies Co., Ltd.
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
package util

import (
	"errors"
	"github.com/go-playground/validator/v10"
)

const (
	BadRequest                int = 400
	StatusUnauthorized        int = 401
	StatusInternalServerError int = 500
	StatusNotFound            int = 404
	StatusForbidden           int = 403
	DELETE                        = "delete"
	GET                           = "get"
	POST                          = "post"
	Operation                     = "] Operation ["
	Resource                      = " Resource ["
	ClientIpaddressInvalid        = "clientIp address is invalid"
	MepPort                       = 30443
	LcmPort                       = 31252
	EdgeHealthPort                = 33666

	ErrCallFromLcm string = "failed to execute rest calling, check if lcm service is ready."
	ErrCallFromMep string = "failed to execute rest calling, check if mep service is ready."
	ErrCallForEdge string = "fail to call this edge"
	ErrSetResult   string = "fail to set communicate result from other edge"
	FailedToUnmarshal        string = "failed to unmarshal request"
	LcmHealthQuery string = "https://119.8.47.5:31252/lcmcontroller/v1/health"
	MepHealthQuery string = "https://mep-mm5.mep/health3"
)

var LocalIp string

// Validate source address
func ValidateSrcAddress(id string) error {
	if id == "" {
		return errors.New("require ip address")
	}

	validate := validator.New()
	err := validate.Var(id, "required,ipv4")
	if err != nil {
		return validate.Var(id, "required,ipv6")
	}
	return nil
}
