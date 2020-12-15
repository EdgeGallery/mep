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
	"errors"
	"mepauth/util"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/astaxie/beego/httplib"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"

	"github.com/agiledragon/gomonkey"

	"github.com/smartystreets/goconvey/convey"
)

func TestRouteGet(t *testing.T) {
	convey.Convey("route get", t, func() {
		convey.Convey("for success", func() {
			c := getBeegoController()
			patch1 := gomonkey.ApplyFunc(ReadData, func(data interface{}, cols ...string) error {
				return nil
			})
			defer patch1.Reset()
			var s *beego.Controller
			patch2 := gomonkey.ApplyMethod(reflect.TypeOf(s), "ServeJSON", func(_ *beego.Controller, encoding ...bool) {

			})
			defer patch2.Reset()

			routeController := &OneRouteController{Controller: c}
			routeController.Get()
			res := routeController.Data["json"]

			convey.So(res, convey.ShouldNotBeNil)
		})

		convey.Convey("for fail", func() {
			c := getBeegoController()
			patch1 := gomonkey.ApplyFunc(ReadData, func(data interface{}, cols ...string) error {
				return errors.New("ReadData fail")
			})
			defer patch1.Reset()
			var s *beego.Controller
			patch2 := gomonkey.ApplyMethod(reflect.TypeOf(s), "ServeJSON", func(_ *beego.Controller, encoding ...bool) {

			})
			defer patch2.Reset()

			routeController := &OneRouteController{Controller: c}
			routeController.Get()
			res := routeController.Data["json"]

			convey.So(res, convey.ShouldNotBeNil)
		})
	})
}

func TestRoutePut(t *testing.T) {
	convey.Convey("route put", t, func() {
		convey.Convey("for success", func() {
			c := getBeegoController()
			patch1 := gomonkey.ApplyFunc(InsertData, func(data interface{}) error {
				return nil
			})
			defer patch1.Reset()
			patch2 := gomonkey.ApplyFunc(httplib.Post, func(url string) *httplib.BeegoHTTPRequest {
				return &httplib.BeegoHTTPRequest{}
			})
			defer patch2.Reset()
			var s *httplib.BeegoHTTPRequest
			patch3 := gomonkey.ApplyMethod(reflect.TypeOf(s), "String", func(_ *httplib.BeegoHTTPRequest) (string, error) {
				return "success", nil
			})
			defer patch3.Reset()
			patch4 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Header", func(_ *httplib.BeegoHTTPRequest, key, value string) *httplib.BeegoHTTPRequest {
				return &httplib.BeegoHTTPRequest{}
			})
			defer patch4.Reset()
			patch5 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Body", func(_ *httplib.BeegoHTTPRequest, data interface{}) *httplib.BeegoHTTPRequest {
				return &httplib.BeegoHTTPRequest{}
			})
			defer patch5.Reset()
			var ser *beego.Controller
			patch6 := gomonkey.ApplyMethod(reflect.TypeOf(ser), "ServeJSON", func(_ *beego.Controller, encoding ...bool) {

			})
			defer patch6.Reset()

			routeController := &OneRouteController{Controller: c}
			routeController.Put()
			res := routeController.Data["json"]
			convey.So(res, convey.ShouldNotBeNil)
		})
	})
}

func getBeegoController() beego.Controller {
	c := beego.Controller{Ctx: &context.Context{ResponseWriter: &context.Response{ResponseWriter: httptest.NewRecorder()}},
		Data: make(map[interface{}]interface{})}
	c.Init(context.NewContext(), "", "", nil)
	c.Ctx.Input.SetParam(util.UrlRouteId, "123")
	c.Ctx.Input.RequestBody = []byte("{\n    \"routeId\": 123,\n    \"appId\": \"12345\",\n    \"serInfo\": {\n        \"serName\": \"serName\",\n        \"uris\": [\n            \"http://127.0.0.1:8080\"\n        ]\n    }\n}")
	return c
}
