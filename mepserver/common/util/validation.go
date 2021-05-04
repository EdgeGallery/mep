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

package util

import (
	"errors"
	"regexp"

	"github.com/go-playground/validator/v10"

	"github.com/apache/servicecomb-service-center/pkg/log"
)

// ValidatePassword validates pwd
func ValidatePassword(password *[]byte) (bool, error) {
	if len(*password) >= pwdLengthMin && len(*password) <= pwdLengthMax {
		// password must satisfy any two conditions
		return ValidateValLenPswd(password)

	} else {
		return false, errors.New("password must have minimum length of 8 and maximum of 16")
	}

}

// ValidateValLenPswd validates password length
func ValidateValLenPswd(password *[]byte) (bool, error) {
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
	if pwdValidCount < pwdCount {
		return false, errors.New("password must contain at least two types of the either one lowercase " +
			"character, one uppercase character, one digit or one special character")
	}
	return true, nil
}

func validateProtocol(fl validator.FieldLevel) bool {
	err := ValidateRegexp(fl.Field().String(), "^[a-zA-Z0-9]*$|^[a-zA-Z0-9][a-zA-Z0-9_\\-\\.]*[a-zA-Z0-9]$",
		"protocol validation failed")
	return err == nil
}

func validateName(fl validator.FieldLevel) bool {
	err := ValidateRegexp(fl.Field().String(), "^[a-zA-Z0-9]*$|^[a-zA-Z0-9][a-zA-Z0-9_\\-]*[a-zA-Z0-9]$",
		"name validation failed")
	return err == nil
}

func validateId(fl validator.FieldLevel) bool {
	err := ValidateRegexp(fl.Field().String(), "^[a-zA-Z0-9]*$|^[a-zA-Z0-9][a-zA-Z0-9\\-]*[a-zA-Z0-9]$",
		"id validation failed")
	return err == nil
}

func validateVersion(fl validator.FieldLevel) bool {
	err := ValidateRegexp(fl.Field().String(), "^\\d+(\\.\\d+){0,2}$",
		"version validation failed")
	return err == nil
}

// ValidateRestBody validate rest body
func ValidateRestBody(body interface{}) error {
	validate := validator.New()
	verrs := validate.RegisterValidation("validateName", validateName)
	if verrs != nil {
		return verrs
	}
	verrs = validate.RegisterValidation("validateId", validateId)
	if verrs != nil {
		return verrs
	}
	verrs = validate.RegisterValidation("validateVersion", validateVersion)
	if verrs != nil {
		return verrs
	}
	verrs = validate.RegisterValidation("validateProtocol", validateProtocol)
	if verrs != nil {
		return verrs
	}
	verrs = validate.Struct(body)
	if verrs != nil {
		for _, verr := range verrs.(validator.ValidationErrors) {
			log.Debugf("Namespace=%s, Field=%s, StructField=%s, Tag=%s, Kind =%s, Type=%s, Value=%s",
				verr.Namespace(), verr.Field(), verr.StructField(), verr.Tag(), verr.Kind(), verr.Type(),
				verr.Value())
		}
		return verrs
	}
	return nil
}
