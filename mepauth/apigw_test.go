/*
 * Copyright 2020-2021 Huawei Technologies Co., Ltd.
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

package main

import (
	"errors"
	. "github.com/agiledragon/gomonkey"
	"github.com/astaxie/beego"
	log "github.com/sirupsen/logrus"
	"mepauth/util"
	"reflect"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestInitAPIGateway(t *testing.T) {

	Convey("init api gateway", t, func() {
		Convey("for success", func() {
			var initializer *apiGwInitializer
			patches := ApplyFunc(util.GetAPIGwURL, func() (string, error) {
				return "https://127.0.0.1:8444", nil
			})
			patches.ApplyMethod(reflect.TypeOf(initializer), "SetApiGwConsumer", func(*apiGwInitializer, string) error {
				return nil
			})
			patches.ApplyMethod(reflect.TypeOf(initializer), "SetupApiGwMepServer", func(*apiGwInitializer, string) error {
				return nil
			})
			patches.ApplyMethod(reflect.TypeOf(initializer), "SetupApiGwMepAuth", func(*apiGwInitializer, string, *[]byte) error {
				return nil
			})
			patches.ApplyMethod(reflect.TypeOf(initializer), "SetupHttpLogPlugin", func(*apiGwInitializer, string) error {
				return nil
			})
			defer patches.Reset()

			config, err := util.TLSConfig("apigw_cacert")
			i := apiGwInitializer{tlsConfig: config}
			err = i.InitAPIGateway(nil)
			So(err, ShouldBeNil)
		})
		Convey("for fail - get apigw url error", func() {
			patches := ApplyFunc(util.GetAPIGwURL, func() (string, error) {
				return "", errors.New("get apigw url error")
			})
			defer patches.Reset()
			i := &apiGwInitializer{}
			err := i.InitAPIGateway(nil)
			So(err, ShouldNotBeNil)
		})
		Convey("for fail - set apigw consumer error", func() {
			patches := ApplyFunc(util.GetAPIGwURL, func() (string, error) {
				return "https://127.0.0.1:8444", nil
			})
			defer patches.Reset()
			var initializer *apiGwInitializer
			patches.ApplyMethod(reflect.TypeOf(initializer), "SetApiGwConsumer", func(*apiGwInitializer, string) error {
				return errors.New("set apigw consumer error")
			})
			i := &apiGwInitializer{}
			err := i.InitAPIGateway(nil)
			So(err, ShouldNotBeNil)
		})
		Convey("for fail - setup apiGw mepserver error", func() {
			patches := ApplyFunc(util.GetAPIGwURL, func() (string, error) {
				return "https://127.0.0.1:8444", nil
			})
			defer patches.Reset()
			var initializer *apiGwInitializer
			patches.ApplyMethod(reflect.TypeOf(initializer), "SetApiGwConsumer", func(*apiGwInitializer, string) error {
				return nil
			})
			patches.ApplyMethod(reflect.TypeOf(initializer), "SetupApiGwMepServer", func(*apiGwInitializer, string) error {
				return errors.New("setup apiGw mepserver error")
			})
			i := &apiGwInitializer{}
			err := i.InitAPIGateway(nil)
			So(err, ShouldNotBeNil)
		})
		Convey("for fail - setup apiGw mepauth error", func() {
			patches := ApplyFunc(util.GetAPIGwURL, func() (string, error) {
				return "https://127.0.0.1:8444", nil
			})
			defer patches.Reset()
			var initializer *apiGwInitializer
			patches.ApplyMethod(reflect.TypeOf(initializer), "SetApiGwConsumer", func(*apiGwInitializer, string) error {
				return nil
			})
			patches.ApplyMethod(reflect.TypeOf(initializer), "SetupApiGwMepServer", func(*apiGwInitializer, string) error {
				return nil
			})
			patches.ApplyMethod(reflect.TypeOf(initializer), "SetupApiGwMepAuth", func(*apiGwInitializer, string, *[]byte) error {
				return errors.New("setup apiGw mepauth error")
			})
			i := &apiGwInitializer{}
			err := i.InitAPIGateway(nil)
			So(err, ShouldNotBeNil)
		})

	})
}

func TestSetupHttpLogPlugin(t *testing.T) {
	Convey("Setup HttpLog Plugin", t, func() {
		Convey("for success", func() {
			var initializer *apiGwInitializer
			patch1 := ApplyMethod(reflect.TypeOf(initializer), "SendPostRequest", func(*apiGwInitializer, string, []byte) error {
				return nil
			})
			defer patch1.Reset()
			i := &apiGwInitializer{}
			err := i.SetupHttpLogPlugin("")
			So(err, ShouldBeNil)
		})
		Convey("for fail", func() {
			var initializer *apiGwInitializer
			patch1 := ApplyMethod(reflect.TypeOf(initializer), "SendPostRequest", func(*apiGwInitializer, string, []byte) error {
				return errors.New("error")
			})
			defer patch1.Reset()
			i := &apiGwInitializer{}
			err := i.SetupHttpLogPlugin("")
			So(err, ShouldNotBeNil)
		})
	})
}

func TestSetApiGwConsumer(t *testing.T) {
	err := beego.LoadAppConfig("ini", "../conf/app.conf")
	if err != nil {
		log.Error(err.Error())
	}
	Convey("set api gateway consumer", t, func() {
		Convey("for success", func() {
			var initializer *apiGwInitializer
			patches := ApplyMethod(reflect.TypeOf(initializer), "SendPostRequest", func(*apiGwInitializer, string, []byte) error {
				return nil
			})
			patches.ApplyFunc(util.GetPublicKey, func() ([]byte, error) {
				return []byte("public_key"), nil
			})
			defer patches.Reset()
			i := &apiGwInitializer{}
			err := i.SetApiGwConsumer("https://127.0.0.1:8444")
			So(err, ShouldBeNil)
		})
		Convey("for fail - send post request error", func() {
			var initializer *apiGwInitializer
			patches := ApplyMethod(reflect.TypeOf(initializer), "SendPostRequest", func(*apiGwInitializer, string, []byte) error {
				return errors.New("send post request error")
			})
			patches.ApplyFunc(util.GetPublicKey, func() ([]byte, error) {
				return []byte("public_key"), nil
			})
			defer patches.Reset()
			i := &apiGwInitializer{}
			err := i.SetApiGwConsumer("https://127.0.0.1:8444")
			So(err, ShouldNotBeNil)
		})
		Convey("for fail - mepauth_key empty", func() {
			var initializer *apiGwInitializer
			patches := ApplyMethod(reflect.TypeOf(initializer), "SendPostRequest", func(*apiGwInitializer, string, []byte) error {
				return nil
			})
			patches.ApplyFunc(util.GetPublicKey, func() ([]byte, error) {
				return []byte("public_key"), nil
			})
			beego.AppConfig.Set("mepauth_key", "")
			defer patches.Reset()
			defer beego.AppConfig.Set("mepauth_key", "mepauth")
			i := &apiGwInitializer{}
			err := i.SetApiGwConsumer("https://127.0.0.1:8444")
			So(err, ShouldNotBeNil)
		})
	})
}

func TestSetupApiGwMepServer(t *testing.T) {
	err := beego.LoadAppConfig("ini", "../conf/app.conf")
	if err != nil {
		log.Error(err.Error())
	}
	Convey("set apiGw mep server", t, func() {
		Convey("for success", func() {
			var initializer *apiGwInitializer
			patches := ApplyMethod(reflect.TypeOf(initializer), "SendPostRequest", func(*apiGwInitializer, string, []byte) error {
				return nil
			})
			patches.ApplyMethod(reflect.TypeOf(initializer), "AddServiceRoute", func(*apiGwInitializer, string, []string, string, bool) error {
				return nil
			})
			defer patches.Reset()
			i := &apiGwInitializer{}
			err := i.SetupApiGwMepServer("https://127.0.0.1:8444")
			So(err, ShouldBeNil)
		})
		Convey("for fail - send post request error", func() {
			var initializer *apiGwInitializer
			patches := ApplyMethod(reflect.TypeOf(initializer), "SendPostRequest", func(*apiGwInitializer, string, []byte) error {
				return errors.New("send post request error")
			})
			patches.ApplyMethod(reflect.TypeOf(initializer), "AddServiceRoute", func(*apiGwInitializer, string, []string, string, bool) error {
				return nil
			})

			defer patches.Reset()
			i := &apiGwInitializer{}
			err := i.SetupApiGwMepServer("https://127.0.0.1:8444")
			So(err, ShouldNotBeNil)
		})
	})
}

func TestSetupApiGwMepAuth(t *testing.T) {
	err := beego.LoadAppConfig("ini", "../conf/app.conf")
	if err != nil {
		log.Error(err.Error())
	}
	Convey("set apiGw mep auth", t, func() {
		Convey("for success", func() {
			var initializer *apiGwInitializer
			patches := ApplyMethod(reflect.TypeOf(initializer), "SendPostRequest", func(*apiGwInitializer, string, []byte) error {
				return nil
			})
			patches.ApplyMethod(reflect.TypeOf(initializer), "AddServiceRoute", func(*apiGwInitializer, string, []string, string, bool) error {
				return nil
			})
			beego.AppConfig.Set("HTTPSAddr", "127.0.0.1")
			defer patches.Reset()
			defer beego.AppConfig.Set("HTTPSAddr", "")
			i := &apiGwInitializer{}
			err := i.SetupApiGwMepAuth("https://127.0.0.1:8444", nil)
			So(err, ShouldBeNil)
		})
		Convey("for fail - send post request error", func() {
			var initializer *apiGwInitializer
			patches := ApplyMethod(reflect.TypeOf(initializer), "SendPostRequest", func(*apiGwInitializer, string, []byte) error {
				return errors.New("send post request error")
			})
			patches.ApplyMethod(reflect.TypeOf(initializer), "AddServiceRoute", func(*apiGwInitializer, string, []string, string, bool) error {
				return nil
			})
			defer patches.Reset()
			i := &apiGwInitializer{}
			err := i.SetupApiGwMepAuth("https://127.0.0.1:8444", nil)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestGetTrustedIpList(t *testing.T) {
	Convey("Get TrustedIp List", t, func() {
		list := []string{"abc.com"}
		i := &apiGwInitializer{}
		ipList := i.getTrustedIpList(list)
		So(ipList, ShouldNotBeNil)
	})
}

func TestAddServiceRoute(t *testing.T) {
	err := beego.LoadAppConfig("ini", "../conf/app.conf")
	if err != nil {
		log.Error(err.Error())
	}
	Convey("add service route", t, func() {
		Convey("for success", func() {
			patches := ApplyFunc(util.GetAPIGwURL, func() (string, error) {
				return "https://127.0.0.1:8444", nil
			})
			var initializer *apiGwInitializer
			patches.ApplyMethod(reflect.TypeOf(initializer), "SendPostRequest", func(*apiGwInitializer, string, []byte) error {
				return nil
			})
			defer patches.Reset()
			i := &apiGwInitializer{}
			err := i.AddServiceRoute("mepauth", []string{"test1", "test2"}, "https://127.0.0.1:8080", false)
			So(err, ShouldBeNil)
		})
		Convey("for fail - get api gateway url error", func() {

			patches := ApplyFunc(util.GetAPIGwURL, func() (string, error) {
				return "https://127.0.0.1:8444", errors.New("get api gateway url error")
			})
			var initializer *apiGwInitializer
			patches.ApplyMethod(reflect.TypeOf(initializer), "SendPostRequest", func(*apiGwInitializer, string, []byte) error {
				return nil
			})
			defer patches.Reset()
			i := &apiGwInitializer{}
			err := i.AddServiceRoute("mepauth", []string{"test1", "test2"}, "https://127.0.0.1:8080", false)
			So(err, ShouldNotBeNil)
		})
		Convey("for fail - send post request error", func() {

			patches := ApplyFunc(util.GetAPIGwURL, func() (string, error) {
				return "https://127.0.0.1:8444", nil
			})
			var initializer *apiGwInitializer
			patches.ApplyMethod(reflect.TypeOf(initializer), "SendPostRequest", func(*apiGwInitializer, string, []byte) error {
				return errors.New("send post request error")
			})
			defer patches.Reset()
			i := &apiGwInitializer{}
			err := i.AddServiceRoute("mepauth", []string{"test1", "test2"}, "https://127.0.0.1:8080", false)
			So(err, ShouldNotBeNil)
		})
	})
}
