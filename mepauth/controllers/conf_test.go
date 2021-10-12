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

package controllers

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"mepauth/adapter"
	"mepauth/models"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"

	. "github.com/agiledragon/gomonkey"
	. "github.com/smartystreets/goconvey/convey"

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

	var pgdb *adapter.PgDb
	patch4 := ApplyMethod(reflect.TypeOf(pgdb), "InsertOrUpdateData", func(*adapter.PgDb, interface{}, ...string) error {
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
	appName := "appNmae"
	requiredServices := ""
	Convey("configure ak and sk", t, func() {
		Convey("for success", func() {
			patches := ApplyFunc(saveAkAndSk, func(_ string, _ string, _ *[]byte, _ string, _ string) error {
				return nil
			})
			defer patches.Reset()
			err := ConfigureAkAndSk(validAppInsID, validAk, &validSk, appName, requiredServices)
			So(err, ShouldBeNil)
		})
		Convey("for fail", func() {
			patches := ApplyFunc(saveAkAndSk, func(_ string, _ string, _ *[]byte, _ string, _ string) error {
				return errors.New("error")
			})
			defer patches.Reset()
			err := ConfigureAkAndSk(validAppInsID, validAk, &validSk, appName, requiredServices)
			So(err, ShouldNotBeNil)
		})
		Convey("invalid ak and sk", func() {
			patches := ApplyFunc(saveAkAndSk, func(_ string, _ string, _ *[]byte, _ string, _ string) error {
				return nil
			})
			defer patches.Reset()
			err := ConfigureAkAndSk(inValidAppInsID, validAk, &validSk, appName, requiredServices)
			So(err, ShouldNotBeNil)
			err = ConfigureAkAndSk(validAppInsID, inValidAk, &validSk, appName, requiredServices)
			So(err, ShouldNotBeNil)
			err = ConfigureAkAndSk(validAppInsID, validAk, &inValidSk, appName, requiredServices)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestSaveAkAndSk(t *testing.T) {
	validAppInsID := "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"
	validAk := "oooooooooooooooooooo"
	validSk := []byte("oooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooo")
	validKey := []byte("00000000000000000000000000000000")
	appName := "appNmae"
	requiredServices := ""
	adapter.Db = &adapter.PgDb{}

	Convey("save ak and sk", t, func() {
		Convey("for success", func() {
			patch1 := ApplyFunc(util.GetWorkKey, func() ([]byte, error) {
				return validKey, nil
			})

			var pgdb *adapter.PgDb
			patch2 := ApplyMethod(reflect.TypeOf(pgdb), "InsertOrUpdateData", func(*adapter.PgDb, interface{}, ...string) error {
				return nil
			})

			defer patch1.Reset()
			defer patch2.Reset()
			err := saveAkAndSk(validAppInsID, validAk, &validSk, appName, requiredServices)
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
			err := saveAkAndSk(validAppInsID, validAk, &validSk, appName, requiredServices)

			So(err.Error(), ShouldEqual, "read fail")
		})
		Convey("get work key fail", func() {
			patches := ApplyFunc(util.GetWorkKey, func() ([]byte, error) {
				return nil, errors.New("get work key fail")
			})
			defer patches.Reset()
			err := saveAkAndSk(validAppInsID, validAk, &validSk, appName, requiredServices)

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
			err := saveAkAndSk(validAppInsID, validAk, &validSk, appName, requiredServices)

			So(err.Error(), ShouldEqual, "encrypt fail")
		})
		Convey("insert fail", func() {
			patch1 := ApplyFunc(util.GetWorkKey, func() ([]byte, error) {
				return validKey, nil
			})
			var pgdb *adapter.PgDb
			patch2 := ApplyMethod(reflect.TypeOf(pgdb), "InsertOrUpdateData", func(*adapter.PgDb, interface{}, ...string) error {
				return errors.New("insert fail")
			})

			defer patch1.Reset()
			defer patch2.Reset()

			err := saveAkAndSk(validAppInsID, validAk, &validSk, appName, requiredServices)

			So(err.Error(), ShouldEqual, "insert fail")
		})
	})
}

func getConfController() *ConfController {
	c := &ConfController{}
	c.Init(context.NewContext(), "", "", nil)
	req, err := http.NewRequest("POST", "http://127.0.0.1", strings.NewReader(""))
	if err != nil {
		log.Error("prepare http request failed")
	}

	c.Ctx.Request = req
	c.Ctx.Request.Header.Set("X-Real-Ip", "127.0.0.1")
	c.Ctx.ResponseWriter = &context.Response{}
	c.Ctx.ResponseWriter.ResponseWriter = httptest.NewRecorder()
	c.Ctx.Output = context.NewOutput()
	c.Ctx.Input = context.NewInput()
	c.Ctx.Output.Reset(c.Ctx)
	c.Ctx.Input.Reset(c.Ctx)
	return c
}

// Test conf PUT
func TestPutSuccess(t *testing.T) {
	Convey("Test delete", t, func() {
		c := getConfController()
		cred := models.Credentials{}
		cred.AccessKeyId = "AK"
		cred.SecretKey = "SK"
		authInfo := models.AuthInfo{Credentials: cred}
		appInstanceInfo := &models.AppInstanceInfo{}
		appInstanceInfo.AuthInfo = authInfo

		adapter.Db = &adapter.PgDb{}

		bytes, _ := json.Marshal(appInstanceInfo)
		c.Ctx.Input.RequestBody = bytes

		patches := ApplyFunc(ConfigureAkAndSk, func(_ string, _ string, _ *[]byte, _ string, _ string) error {
			return nil
		})
		patches.Reset()
		c.Put()
		out := c.Data["json"]
		So(out, ShouldNotBeNil)
	})
}

// Test conf PUT Failure
func TestPutFailure(t *testing.T) {
	Convey("Test put failure", t, func() {
		c := getConfController()
		c.Ctx.Request.Header.Set("X-Real-Ip", "")
		c.Put()
		out := c.Data["json"]
		So(out, ShouldContainSubstring, "clientIp address is invalid")
	})
}

// Test conf Delete
func TestDeleteSuccess(t *testing.T) {
	Convey("Test delete", t, func() {
		validAppInsID := "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"
		c := getConfController()
		cred := models.Credentials{}
		cred.AccessKeyId = "AK"
		cred.SecretKey = "SK"
		authInfo := models.AuthInfo{Credentials: cred}
		appInstanceInfo := &models.AppInstanceInfo{}
		appInstanceInfo.AuthInfo = authInfo

		adapter.Db = &adapter.PgDb{}

		bytes, _ := json.Marshal(appInstanceInfo)
		c.Ctx.Input.RequestBody = bytes
		c.Ctx.Input.SetParam(util.UrlApplicationId, validAppInsID)
		var pgdb *adapter.PgDb
		patches := ApplyMethod(reflect.TypeOf(pgdb), "DeleteData", func(*adapter.PgDb, interface{}, ...string) error {
			return nil
		})
		defer patches.Reset()
		c.Delete()
		out := c.Data["json"]
		So(out, ShouldEqual, "Delete success.")
	})
}

func TestDeleteFailure(t *testing.T) {
	Convey("Test delete failure", t, func() {
		c := getConfController()
		c.Ctx.Request.Header.Set("X-Real-Ip", "")
		c.Delete()
		out := c.Data["json"]
		So(out, ShouldContainSubstring, "clientIp address is invalid")
	})
}

func TestDeleteApplicationInstanceIDFailure(t *testing.T) {
	Convey("Test delete", t, func() {
		inValidAppInsID := "5abe478223-2c70-4e47-9a4e-0ee3a1a0fd1f"
		c := getConfController()
		c.Ctx.Input.SetParam(util.UrlApplicationId, inValidAppInsID)
		c.Delete()
		out := c.Data["json"]
		So(out, ShouldContainSubstring, "Application Instance ID validation failed")
	})
}

// Test conf PUT
func TestGetSuccess(t *testing.T) {
	Convey("Test Get", t, func() {
		validAppInsID := "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"
		c := getConfController()
		cred := models.Credentials{}
		cred.AccessKeyId = "AK"
		cred.SecretKey = "SK"
		authInfo := models.AuthInfo{Credentials: cred}
		appInstanceInfo := &models.AppInstanceInfo{}
		appInstanceInfo.AuthInfo = authInfo

		adapter.Db = &adapter.PgDb{}

		bytes, _ := json.Marshal(appInstanceInfo)
		c.Ctx.Input.RequestBody = bytes
		c.Ctx.Input.SetParam(util.UrlApplicationId, validAppInsID)
		var pgdb *adapter.PgDb
		patches := ApplyMethod(reflect.TypeOf(pgdb), "ReadData", func(*adapter.PgDb, interface{}, ...string) error {
			return nil
		})
		defer patches.Reset()
		c.Get()
		out := c.Data["json"]
		So(out, ShouldNotBeNil)
	})
}

// Test conf Get failure
func TestGetFailure(t *testing.T) {
	Convey("Test get failure", t, func() {
		c := getConfController()
		c.Ctx.Request.Header.Set("X-Real-Ip", "")
		c.Get()
		out := c.Data["json"]
		So(out, ShouldContainSubstring, "clientIp address is invalid")
	})
	Convey("Test Get failure with invalid application instance id", t, func() {
		inValidAppInsID := "5abe478223-2c70-4e47-9a4e-0ee3a1a0fd1f"
		c := getConfController()
		c.Ctx.Input.SetParam(util.UrlApplicationId, inValidAppInsID)
		c.Get()
		out := c.Data["json"]
		So(out, ShouldContainSubstring, "Application Instance ID validation failed")
	})
}
