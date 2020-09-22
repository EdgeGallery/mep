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

// util package
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

// Validate UUID
func ValidateUUID(id string) error {
	if len(id) != 0 {
		validate := validator.New()
		res := validate.Var(id, "required,uuid")
		if res != nil {
			return errors.New("UUID validate failed")
		}
	} else {
		return errors.New("UUID validate failed")
	}
	return nil
}

// Validate Ak
func ValidateAk(ak string) error {
	isMatch, errMatch := regexp.MatchString(AkRegex, ak)
	if errMatch != nil || !isMatch {
		return errors.New("validate ak failed")
	}
	return nil
}

// Validate Sk
func ValidateSk(sk *[]byte) error {
	isMatch, errMatch := regexp.Match(SkRegex, *sk)
	if errMatch != nil || !isMatch {
		return errors.New("validate sk failed")
	}
	return nil
}

// Validate Server Name
func ValidateServerName(serverName string) (bool, error) {
	if len(serverName) > maxHostNameLen {
		return false, errors.New("server or host name validation failed")
	}
	return regexp.MatchString(ServerNameRegex, serverName)
}

// Validate Api gateway IP address and port
func ValidateApiGwParams(apiGwHost string, apiGwPort string) (bool, error) {
	apiGwHostIsValid, validateApiGwErr := ValidateServerName(apiGwHost)
	if validateApiGwErr != nil || !apiGwHostIsValid {
		return apiGwHostIsValid, validateApiGwErr
	}
	apiGwPortIsValid, validateApiGwPortErr := regexp.MatchString(PortRegex, apiGwPort)
	if validateApiGwPortErr != nil || !apiGwPortIsValid {
		return apiGwPortIsValid, validateApiGwPortErr
	}
	return true, nil
}

// Validate password
func ValidatePassword(password *[]byte) (bool, error) {
	if len(*password) >= minPasswordSize && len(*password) <= maxPasswordSize {
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
		if pwdIsValid && err == nil  {
			pwdValidCount++
		}
		// space validation for password complexity is not added
		// as jwt decrypt fails if space is included in password
		pwdIsValid, err = regexp.Match(specialCharRegex, *password)
		if pwdIsValid && err == nil {
			pwdValidCount++
		}
		if pwdValidCount < maxPasswordCount {
			return false, errors.New("password must contain at least two types of the either one lowercase" +
				" character, one uppercase character, one digit or one special character")
		}
	} else {
		return false, errors.New("password must have minimum length of 8 and maximum of 16")
	}
	return true, nil
}

// Validate IP address adn CIDR
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

func ValidateKeyComponentUserInput(keyComponentUserStr *[]byte) error {
	if len(*keyComponentUserStr) < ComponentSize {
		log.Error("key component user string length is not valid")
		return  fmt.Errorf("key component user string length is not valid")
	}
	return nil
}
