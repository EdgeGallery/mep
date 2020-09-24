package main

import (
	"errors"
	"testing"

	. "github.com/agiledragon/gomonkey"
	"github.com/astaxie/beego"
	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"

	"mepauth/util"
)

func TestInitAPIGateway(t *testing.T) {

	Convey("init api gateway", t, func() {
		Convey("for success", func() {
			patches := ApplyFunc(util.GetAPIGwURL, func() (string, error) {
				return "https://127.0.0.1:8444", nil
			})
			defer patches.Reset()
			patches.ApplyFunc(setApiGwConsumer, func(_ string) error {
				return nil
			})
			patches.ApplyFunc(setupKongMepServer, func(_ string) error {
				return nil
			})
			patches.ApplyFunc(setupKongMepAuth, func(_ string, _ *[]byte) error {
				return nil
			})
			err := initAPIGateway(nil)
			So(err, ShouldBeNil)
		})
		Convey("for fail - get apigw url error", func() {
			patches := ApplyFunc(util.GetAPIGwURL, func() (string, error) {
				return "", errors.New("get apigw url error")
			})
			defer patches.Reset()
			err := initAPIGateway(nil)
			So(err, ShouldNotBeNil)
		})
		Convey("for fail - set apigw consumer error", func() {
			patches := ApplyFunc(util.GetAPIGwURL, func() (string, error) {
				return "https://127.0.0.1:8444", nil
			})
			defer patches.Reset()
			patches.ApplyFunc(setApiGwConsumer, func(_ string) error {
				return errors.New("set apigw consumer error")
			})
			err := initAPIGateway(nil)
			So(err, ShouldNotBeNil)
		})
		Convey("for fail - setup kong mepserver error", func() {
			patches := ApplyFunc(util.GetAPIGwURL, func() (string, error) {
				return "https://127.0.0.1:8444", nil
			})
			defer patches.Reset()
			patches.ApplyFunc(setApiGwConsumer, func(_ string) error {
				return nil
			})
			patches.ApplyFunc(setupKongMepServer, func(_ string) error {
				return errors.New("setup kong mepserver error")
			})
			err := initAPIGateway(nil)
			So(err, ShouldNotBeNil)
		})
		Convey("for fail - setup kong mepauth error", func() {
			patches := ApplyFunc(util.GetAPIGwURL, func() (string, error) {
				return "https://127.0.0.1:8444", nil
			})
			defer patches.Reset()
			patches.ApplyFunc(setApiGwConsumer, func(_ string) error {
				return nil
			})
			patches.ApplyFunc(setupKongMepServer, func(_ string) error {
				return nil
			})
			patches.ApplyFunc(setupKongMepAuth, func(_ string, _ *[]byte) error {
				return errors.New("setup kong mepauth error")
			})
			err := initAPIGateway(nil)
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
			patches := ApplyFunc(util.SendPostRequest, func(_ string, _ []byte) error {
				return nil
			})
			patches.ApplyFunc(util.GetPublicKey, func() ([]byte, error) {
				return []byte("public_key"), nil
			})
			defer patches.Reset()
			err := setApiGwConsumer("https://127.0.0.1:8444")
			So(err, ShouldBeNil)
		})
		Convey("for fail - send post request error", func() {
			patches := ApplyFunc(util.SendPostRequest, func(_ string, _ []byte) error {
				return errors.New("send post request error")
			})
			patches.ApplyFunc(util.GetPublicKey, func() ([]byte, error) {
				return []byte("public_key"), nil
			})
			defer patches.Reset()
			err := setApiGwConsumer("https://127.0.0.1:8444")
			So(err, ShouldNotBeNil)
		})
		Convey("for fail - mepauth_key empty", func() {
			patches := ApplyFunc(util.SendPostRequest, func(_ string, _ []byte) error {
				return nil
			})
			patches.ApplyFunc(util.GetPublicKey, func() ([]byte, error) {
				return []byte("public_key"), nil
			})
			beego.AppConfig.Set("mepauth_key", "")
			defer patches.Reset()
			defer beego.AppConfig.Set("mepauth_key", "mepauth")
			err := setApiGwConsumer("https://127.0.0.1:8444")
			So(err, ShouldNotBeNil)
		})
	})
}

func TestSetupKongMepServer(t *testing.T) {
	err := beego.LoadAppConfig("ini", "../conf/app.conf")
	if err != nil {
		log.Error(err.Error())
	}
	Convey("set kong mep server", t, func() {
		Convey("for success", func() {
			patches := ApplyFunc(util.SendPostRequest, func(_ string, _ []byte) error {
				return nil
			})
			patches.ApplyFunc(addServiceRoute, func(_, _ string) error {
				return nil
			})
			defer patches.Reset()
			err := setupKongMepServer("https://127.0.0.1:8444")
			So(err, ShouldBeNil)
		})
		Convey("for fail - send post request error", func() {

			patches := ApplyFunc(util.SendPostRequest, func(_ string, _ []byte) error {
				return errors.New("send post request error")
			})
			patches.ApplyFunc(addServiceRoute, func(_, _ string) error {
				return nil
			})
			defer patches.Reset()
			err := setupKongMepServer("https://127.0.0.1:8444")
			So(err, ShouldNotBeNil)
		})
	})
}

func TestSetupKongMepAuth(t *testing.T) {
	err := beego.LoadAppConfig("ini", "../conf/app.conf")
	if err != nil {
		log.Error(err.Error())
	}
	Convey("set kong mep auth", t, func() {
		Convey("for success", func() {
			patches := ApplyFunc(util.SendPostRequest, func(_ string, _ []byte) error {
				return nil
			})
			patches.ApplyFunc(addServiceRoute, func(_, _ string) error {
				return nil
			})
			beego.AppConfig.Set("HTTPSAddr", "127.0.0.1")
			defer patches.Reset()
			defer beego.AppConfig.Set("HTTPSAddr", "")
			err := setupKongMepAuth("https://127.0.0.1:8444", nil)
			So(err, ShouldBeNil)
		})
		Convey("for fail - send post request error", func() {

			patches := ApplyFunc(util.SendPostRequest, func(_ string, _ []byte) error {
				return errors.New("send post request error")
			})
			patches.ApplyFunc(addServiceRoute, func(_, _ string) error {
				return nil
			})
			defer patches.Reset()
			err := setupKongMepAuth("https://127.0.0.1:8444", nil)
			So(err, ShouldNotBeNil)
		})
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
			patches.ApplyFunc(util.SendPostRequest, func(_ string, _ []byte) error {
				return nil
			})
			defer patches.Reset()
			err := addServiceRoute("mepauth", "https://127.0.0.1:8080")
			So(err, ShouldBeNil)
		})
		Convey("for fail - get api gateway url error", func() {

			patches := ApplyFunc(util.GetAPIGwURL, func() (string, error) {
				return "https://127.0.0.1:8444", errors.New("get api gateway url error")
			})
			patches.ApplyFunc(util.SendPostRequest, func(_ string, _ []byte) error {
				return nil
			})
			defer patches.Reset()
			err := addServiceRoute("mepauth", "https://127.0.0.1:8080")
			So(err, ShouldNotBeNil)
		})
		Convey("for fail - send post request error", func() {

			patches := ApplyFunc(util.GetAPIGwURL, func() (string, error) {
				return "https://127.0.0.1:8444", nil
			})
			patches.ApplyFunc(util.SendPostRequest, func(_ string, _ []byte) error {
				return errors.New("send post request error")
			})
			defer patches.Reset()
			err := addServiceRoute("mepauth", "https://127.0.0.1:8080")
			So(err, ShouldNotBeNil)
		})
	})
}
