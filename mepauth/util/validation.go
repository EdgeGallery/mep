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
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"regexp"

	"github.com/go-playground/validator/v10"
)

// ValidateUUID validates UUID
func ValidateUUID(id string) error {
	if len(id) != 0 {
		validate := validator.New()
		res := validate.Var(id, "required,uuid")
		if res != nil {
			return errors.New(AppIDFailMsg)
		}
	} else {
		return errors.New("UUID validate failed")
	}
	return nil
}

// ValidateAk validates Ak
func ValidateAk(ak string) error {
	isMatch, errMatch := regexp.MatchString(akRegex, ak)
	if errMatch != nil || !isMatch {
		return errors.New(AkFailMsg)
	}
	return nil
}

// ValidateSk validates Sk
func ValidateSk(sk *[]byte) error {
	isMatch, errMatch := regexp.Match(skRegex, *sk)
	if errMatch != nil || !isMatch {
		return errors.New(SkFailMsg)
	}
	return nil
}

// Validate Server Name
func validateServerName(serverName string) (bool, error) {
	if len(serverName) > maxHostNameLen {
		log.Error("Server name length validation failed")
		return false, errors.New("server or host name length validation failed")
	}
	return regexp.MatchString(serverNameRegex, serverName)
}

// Validate Api gateway IP address and port
func validateApiGwParams(apiGwHost string, apiGwPort string) (bool, error) {
	apiGwHostIsValid, validateApiGwErr := validateServerName(apiGwHost)
	if validateApiGwErr != nil || !apiGwHostIsValid {
		return apiGwHostIsValid, validateApiGwErr
	}
	apiGwPortIsValid, validateApiGwPortErr := regexp.MatchString(portRegex, apiGwPort)
	if validateApiGwPortErr != nil || !apiGwPortIsValid {
		log.Error("API gateway port doesn't match the expected pattern")
		return apiGwPortIsValid, validateApiGwPortErr
	}
	return true, nil
}

// Validate password
func validateJwtPassword(password *[]byte) (bool, error) {
	if validateLength(password) {
		pwdValidCount := validateRegex(password)
		if pwdValidCount < maxPasswordCount {
			return false, errors.New("password must contain at least two types of the either one lowercase" +
				" character, one uppercase character, one digit or one special character")
		}
	} else {
		return false, errors.New("password must have minimum length of 8 and maximum of 16")
	}
	return true, nil
}

func validateRegex(password *[]byte) int {
	// password must satisfy any two conditions
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

func validateLength(password *[]byte) bool {
	return len(*password) >= minPasswordSize && len(*password) <= maxPasswordSize
}

// ValidateIpAndCidr validates IP address and CIDR
func ValidateIpAndCidr(trustedNetworkList []string) (bool, error) {
	for _, ipcidr := range trustedNetworkList {
		isValidIp := true
		isValidCidr := true
		if net.ParseIP(ipcidr) == nil {
			isValidIp = false
		}
		_, _, errorCidr := net.ParseCIDR(ipcidr)
		if errorCidr != nil {
			isValidCidr = false
		}
		if !isValidIp && !isValidCidr {
			log.Error("ip/cidr parsing failed")
			return false, errors.New("ip/cidr parsing failed")
		}
	}
	return true, nil
}

// ValidateInputArgs validates application configurations related arguments
func ValidateInputArgs(appConfig AppConfigProperties) bool {
	args := []string{"KEY_COMPONENT", "JWT_PRIVATE_KEY", "APP_INST_ID", "ACCESS_KEY", "SECRET_KEY"}
	for _, s := range args {
		input := appConfig[s]
		if input == nil {
			log.Error(s + " input is nil.")
			return false
		}
		res1 := bytes.TrimSpace(*input)
		if len(res1) == 0 {
			log.Error(s + " input is empty.")
			return false
		}
	}
	return true
}

// ValidateKeyComponentUserInput validates key component user string against minimum length
func ValidateKeyComponentUserInput(keyComponentUserStr *[]byte) error {
	if len(*keyComponentUserStr) < componentSize {
		log.Error("Key component user string length is not valid")
		return  fmt.Errorf("key component user string length is not valid")
	}
	return nil
}
