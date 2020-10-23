package controllers

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"

	. "github.com/agiledragon/gomonkey"
	. "github.com/smartystreets/goconvey/convey"

	"mepauth/models"
	"mepauth/util"
)

func TestPut(t *testing.T) {
	appInsId := "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"
	var input *context.BeegoInput
	fmt.Println("111", reflect.TypeOf(input).NumMethod())
	patch1 := ApplyMethod(reflect.TypeOf(input), "Param", func(*context.BeegoInput, string) string {
		return appInsId
	})

	patch2 := ApplyFunc(json.Unmarshal, func([]byte, interface{}) error {
		return nil
	})

	patch3 := ApplyFunc(getCipherAndNonce, func(*[]byte) ([]byte, []byte, error) {
		return nil, nil, nil
	})

	patch4 := ApplyFunc(InsertOrUpdateData, func(interface{}, ...string) error {
		return nil
	})

	var ct *beego.Controller
	patch5 := ApplyMethod(reflect.TypeOf(ct), "ServeJSON", func(*beego.Controller, ...bool) {

	})

	defer patch1.Reset()
	defer patch2.Reset()
	defer patch3.Reset()
	defer patch4.Reset()
	defer patch5.Reset()
}

func TestConfigureAkAndSk(t *testing.T) {
	validAppInsID := "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"
	inValidAppInsID := "invalid_appinstanceid"
	validAk := "oooooooooooooooooooo"
	inValidAk := "invalidAk"
	validSk := []byte("oooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooo")
	inValidSk := []byte("invailidSk")
	Convey("configure ak and sk", t, func() {
		Convey("for success", func() {
			patches := ApplyFunc(saveAkAndSk, func(_ string, _ string, _ *[]byte) error {
				return nil
			})
			defer patches.Reset()
			err := ConfigureAkAndSk(validAppInsID, validAk, &validSk)
			So(err, ShouldBeNil)
		})
		Convey("for fail", func() {
			patches := ApplyFunc(saveAkAndSk, func(_ string, _ string, _ *[]byte) error {
				return errors.New("error")
			})
			defer patches.Reset()
			err := ConfigureAkAndSk(validAppInsID, validAk, &validSk)
			So(err, ShouldNotBeNil)
		})
		Convey("invalid ak and sk", func() {
			patches := ApplyFunc(saveAkAndSk, func(_ string, _ string, _ *[]byte) error {
				return nil
			})
			defer patches.Reset()
			err := ConfigureAkAndSk(inValidAppInsID, validAk, &validSk)
			So(err, ShouldNotBeNil)
			err = ConfigureAkAndSk(validAppInsID, inValidAk, &validSk)
			So(err, ShouldNotBeNil)
			err = ConfigureAkAndSk(validAppInsID, validAk, &inValidSk)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestSaveAkAndSk(t *testing.T) {
	validAppInsID := "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"
	validAk := "oooooooooooooooooooo"
	validSk := []byte("oooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooo")
	validKey := []byte("00000000000000000000000000000000")

	Convey("save ak and sk", t, func() {
		Convey("for success", func() {
			patches := ApplyFunc(util.GetWorkKey, func() ([]byte, error) {
				return validKey, nil
			})
			defer patches.Reset()
			err := saveAkAndSk(validAppInsID, validAk, &validSk)
			So(err, ShouldBeNil)
		})
		Convey("read fail", func() {
			patches := ApplyFunc(util.GetWorkKey, func() ([]byte, error) {
				return validKey, nil
			})
			defer patches.Reset()
			patches.ApplyFunc(rand.Read, func(_ []byte) (n int, err error) {
				return 1, errors.New("read fail")
			})
			err := saveAkAndSk(validAppInsID, validAk, &validSk)

			So(err.Error(), ShouldEqual, "read fail")
		})
		Convey("get work key fail", func() {
			patches := ApplyFunc(util.GetWorkKey, func() ([]byte, error) {
				return nil, errors.New("get work key fail")
			})
			defer patches.Reset()
			err := saveAkAndSk(validAppInsID, validAk, &validSk)

			So(err.Error(), ShouldEqual, "get work key fail")
		})
		Convey("encrypt fail", func() {
			patches := ApplyFunc(util.GetWorkKey, func() ([]byte, error) {
				return validKey, nil
			})
			patches.ApplyFunc(util.EncryptByAES256GCM, func(_ []byte, _ []byte, _ []byte) ([]byte, error) {
				return nil, errors.New("encrypt fail")
			})
			defer patches.Reset()
			err := saveAkAndSk(validAppInsID, validAk, &validSk)

			So(err.Error(), ShouldEqual, "encrypt fail")
		})
		Convey("insert fail", func() {
			patches := ApplyFunc(util.GetWorkKey, func() ([]byte, error) {
				return validKey, nil
			})
			patches.ApplyFunc(InsertOrUpdateDataToFile, func(_ *models.AuthInfoRecord) error {
				return errors.New("insert fail")
			})
			defer patches.Reset()
			err := saveAkAndSk(validAppInsID, validAk, &validSk)

			So(err.Error(), ShouldEqual, "insert fail")
		})
	})
}
