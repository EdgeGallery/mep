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
		ok, err := ValidateServerName("edgegallery.org")
		So(ok, ShouldBeTrue)
		So(err, ShouldBeNil)
		tooLongServerName := "edgegallery.org.edgegallery.org.edgegallery.org." +
			"edgegallery.org.edgegallery.org.edgegallery.org.edgegallery.org." +
			"edgegallery.org.edgegallery.org.edgegallery.org.edgegallery.org." +
			"edgegallery.org.edgegallery.org.edgegallery.org.edgegallery.org.edgegallery.org."
		ok, err = ValidateServerName(tooLongServerName)
		So(ok, ShouldBeFalse)
		So(err.Error(), ShouldEqual, "server or host name validation failed")
		notMatchServerName := "abc*def.org"
		ok, err = ValidateServerName(notMatchServerName)
		So(ok, ShouldBeFalse)
		So(err, ShouldBeNil)
	})
}

func TestValidateApiGwParams(t *testing.T) {
	Convey("validate apigw params", t, func() {
		ok, err := ValidateApiGwParams("apigw.edgegallery.org", "30443")
		So(ok, ShouldBeTrue)
		So(err, ShouldBeNil)
		ok, err= ValidateApiGwParams("apigw.edgegallery.org", "304433")
		So(ok, ShouldBeFalse)
		So(err, ShouldBeNil)
		tooLongHost := "edgegallery.org.edgegallery.org.edgegallery.org." +
			"edgegallery.org.edgegallery.org.edgegallery.org.edgegallery.org." +
			"edgegallery.org.edgegallery.org.edgegallery.org.edgegallery.org." +
			"edgegallery.org.edgegallery.org.edgegallery.org.edgegallery.org.edgegallery.org."
		ok, err= ValidateApiGwParams(tooLongHost, "304433")
		So(ok, ShouldBeFalse)
		So(err, ShouldNotBeNil)
	})
}

func TestValidatePassword(t *testing.T) {
	Convey("validate password", t, func() {
		invalidPwd := []byte("1234567")
		_, err := ValidatePassword(&invalidPwd)
		So(err.Error(), ShouldEqual, "password must have minimum length of 8 and maximum of 16")
		lowcasePwd := []byte("lowcasepassword")
		ok, err := ValidatePassword(&lowcasePwd)
		So(ok, ShouldBeFalse)
		So(err, ShouldNotBeNil)
		validPwd := []byte("Validpassword")
		ok, err = ValidatePassword(&validPwd)
		So(ok, ShouldBeTrue)
		So(err, ShouldBeNil)
		allDigitPwd := []byte("0000000000")
		ok, err = ValidatePassword(&allDigitPwd)
		So(ok, ShouldBeFalse)
		So(err, ShouldNotBeNil)
		withSpecialCharPwd := []byte("00000-00000")
		ok, err = ValidatePassword(&withSpecialCharPwd)
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

		validKeyCom := []byte(ComponentContent)
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
		validKeyComUserInput := []byte(ComponentContent)
		err := ValidateKeyComponentUserInput(&validKeyComUserInput)
		inValidKeyComUserInput := []byte(ComponentContent[0:255])
		err = ValidateKeyComponentUserInput(&inValidKeyComUserInput)
		So(err, ShouldNotBeNil)
	})

}
