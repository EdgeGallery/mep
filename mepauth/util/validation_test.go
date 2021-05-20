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
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestValidateUUID(t *testing.T) {
	Convey("validate uuid", t, func() {
		So(ValidateUUID("5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"), ShouldBeNil)
		So(ValidateUUID("invalid-uuid-sample").Error(), ShouldNotBeNil)
		So(ValidateUUID("").Error(), ShouldNotBeNil)
	})
}

func TestValidateAk(t *testing.T) {
	Convey("validate ak", t, func() {
		So(ValidateAk("QVUJMSUMgS0VZLS0tLS0"), ShouldBeNil)
		So(ValidateAk("lessthan20strings"), ShouldNotBeNil)
	})
}

func TestValidateSk(t *testing.T) {
	Convey("validate sk", t, func() {
		validSk := []byte("DXPb4sqElKhcHe07Kw5uorayETwId1JOjjOIRomRs5wyszoCR5R7AtVa28KT3lSc")
		err := ValidateSk(&validSk)
		So(err, ShouldBeNil)
		notValidSk := []byte("lessthan64strings")
		err = ValidateSk(&notValidSk)
		So(err, ShouldNotBeNil)
	})
}

func TestValidateServerName(t *testing.T) {
	Convey("validate server name", t, func() {
		ok, err := validateServerName("edgegallery.org")
		So(ok, ShouldBeTrue)
		So(err, ShouldBeNil)
		tooLongServerName := "edgegallery.org.edgegallery.org.edgegallery.org." +
			"edgegallery.org.edgegallery.org.edgegallery.org.edgegallery.org." +
			"edgegallery.org.edgegallery.org.edgegallery.org.edgegallery.org." +
			"edgegallery.org.edgegallery.org.edgegallery.org.edgegallery.org.edgegallery.org."
		ok, err = validateServerName(tooLongServerName)
		So(ok, ShouldBeFalse)
		So(err.Error(), ShouldEqual, "server or host name length validation failed")
		notMatchServerName := "abc*def.org"
		ok, err = validateServerName(notMatchServerName)
		So(ok, ShouldBeFalse)
		So(err, ShouldBeNil)
	})
}

func TestValidateApiGwParams(t *testing.T) {
	Convey("validate apigw params", t, func() {
		ok, err := validateApiGwParams("apigw.edgegallery.org", "30443")
		So(ok, ShouldBeTrue)
		So(err, ShouldBeNil)
		ok, err = validateApiGwParams("apigw.edgegallery.org", "304433")
		So(ok, ShouldBeFalse)
		So(err, ShouldBeNil)
		tooLongHost := "edgegallery.org.edgegallery.org.edgegallery.org." +
			"edgegallery.org.edgegallery.org.edgegallery.org.edgegallery.org." +
			"edgegallery.org.edgegallery.org.edgegallery.org.edgegallery.org." +
			"edgegallery.org.edgegallery.org.edgegallery.org.edgegallery.org.edgegallery.org."
		ok, err = validateApiGwParams(tooLongHost, "304433")
		So(ok, ShouldBeFalse)
		So(err, ShouldNotBeNil)
	})
}

func TestValidatePassword(t *testing.T) {
	Convey("validate password", t, func() {
		invalidPwd := []byte("1234567")
		_, err := validateJwtPassword(&invalidPwd)
		So(err.Error(), ShouldEqual, "password must have minimum length of 8 and maximum of 16")
		lowcasePwd := []byte("lowcasepassword")
		ok, err := validateJwtPassword(&lowcasePwd)
		So(ok, ShouldBeFalse)
		So(err, ShouldNotBeNil)
		validPwd := []byte("Validpassword")
		ok, err = validateJwtPassword(&validPwd)
		So(ok, ShouldBeTrue)
		So(err, ShouldBeNil)
		allDigitPwd := []byte("0000000000")
		ok, err = validateJwtPassword(&allDigitPwd)
		So(ok, ShouldBeFalse)
		So(err, ShouldNotBeNil)
		withSpecialCharPwd := []byte("00000-00000")
		ok, err = validateJwtPassword(&withSpecialCharPwd)
		So(ok, ShouldBeTrue)
		So(err, ShouldBeNil)
	})
}

func TestValidateIpAndCidr(t *testing.T) {
	Convey("valid ip and cidr", t, func() {
		ip := []string{"192.0.2.1/24"}
		ok, err := ValidateIpAndCidr(ip)
		So(ok, ShouldBeTrue)
		So(err, ShouldBeNil)
	})
	Convey("invalid ip and cidr", t, func() {
		ip := []string{"192.0.2.256"}
		ok, err := ValidateIpAndCidr(ip)
		So(ok, ShouldBeFalse)
		So(err, ShouldNotBeNil)
	})
}

func TestValidateInputArgs(t *testing.T) {
	Convey("validate input args", t, func() {
		config := AppConfigProperties{}
		config["KEY_COMPONENT"] = nil
		So(ValidateInputArgs(config), ShouldBeFalse)

		spaceKeyCom := []byte(" ")
		config["KEY_COMPONENT"] = &spaceKeyCom
		So(ValidateInputArgs(config), ShouldBeFalse)

		validKeyCom := []byte(componentContent)
		validJwtKey := []byte("te9Fmv%qaq")
		validAk := []byte("QVUJMSUMgS0VZLS0tLS0")
		validSk := []byte("DXPb4sqElKhcHe07Kw5uorayETwId1JOjjOIRomRs5wyszoCR5R7AtVa28KT3lSc")
		validAppInsID := []byte("5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f")
		config["KEY_COMPONENT"] = &validKeyCom
		config["JWT_PRIVATE_KEY"] = &validJwtKey
		config["APP_INST_ID"] = &validAppInsID
		config["ACCESS_KEY"] = &validAk
		config["SECRET_KEY"] = &validSk
		So(ValidateInputArgs(config), ShouldBeTrue)
	})
}

func TestValidateKeyComponentUserInput(t *testing.T) {
	Convey("validate key component user input", t, func() {
		validKeyComUserInput := []byte(componentContent)
		err := ValidateKeyComponentUserInput(&validKeyComUserInput)
		inValidKeyComUserInput := []byte(componentContent[0:255])
		err = ValidateKeyComponentUserInput(&inValidKeyComUserInput)
		So(err, ShouldNotBeNil)
	})

}
