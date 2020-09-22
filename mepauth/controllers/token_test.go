package controllers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/astaxie/beego/context"

	. "github.com/agiledragon/gomonkey"
	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"

	"mepauth/util"
)

func TestValidateDateTimeFormat(t *testing.T) {
	Convey("validate date time format", t, func() {
		req, err := http.NewRequest("POST", "http://127.0.0.1", strings.NewReader(""))
		if err != nil {
			log.Error("prepare http request failed")
		}
		req.Header.Set(util.DATE_HEADER, util.DATE_FORMAT)
		ok := validateDateTimeFormat(req)
		So(ok, ShouldBeTrue)
		req.Header.Set(util.DATE_HEADER, "20200930")
		ok = validateDateTimeFormat(req)
		So(ok, ShouldBeFalse)
	})
}

func TestGetTokenInfo(t *testing.T) {
	Convey("get token info", t, func() {
		appInsId := "5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"
		ak := "QVUJMSUMgS0VZLS0tLS0"
		token := "jwtToken"
		c := getController()
		Convey("for success", func() {
			c.Ctx.Request.Header.Set("X-Real-Ip", "127.0.0.1")
			patches := ApplyFunc(generateJwtToken, func(_ string, _ string) (*string, error) {
				return &token, nil
			})
			defer patches.Reset()
			So(c.getTokenInfo(appInsId, ak), ShouldNotBeNil)
		})
		Convey("for fail", func() {

			patches := ApplyFunc(generateJwtToken, func(_ string, _ string) (*string, error) {
				return nil, errors.New("generate token fail")
			})
			defer patches.Reset()
			So(c.getTokenInfo(appInsId, ak), ShouldBeNil)
		})
	})
}

func TestValidateSignature(t *testing.T) {

	ak := "QVUJMSUMgS0VZLS0tLS0"
	sk := []byte("sksksksk")
	signHeader := "content-type;host;x-sdk-date"
	sig := "signature"
	c := getController()

	Convey("validate signature", t, func() {
		Convey("for success", func() {
			patches := ApplyFunc(akSignatureIsValid, func(_ *http.Request, _ string, _ []byte, _ string, _ string) (bool, error) {
				return true, nil
			})
			defer patches.Reset()
			ok := c.validateSignature(ak, sk, signHeader, sig)
			So(ok, ShouldBeTrue)
		})
		Convey("for fail - sig invalid", func() {
			patches := ApplyFunc(akSignatureIsValid, func(_ *http.Request, _ string, _ []byte, _ string, _ string) (bool, error) {
				return true, errors.New("ak is invalid")
			})
			defer patches.Reset()
			ok := c.validateSignature(ak, sk, signHeader, sig)
			So(ok, ShouldBeFalse)
		})
		Convey("for fail - sig invalid 2", func() {
			patches := ApplyFunc(akSignatureIsValid, func(_ *http.Request, _ string, _ []byte, _ string, _ string) (bool, error) {
				return false, nil
			})
			patches.ApplyFunc(ProcessAkForBlockListing, func(_ string) {
				return
			})
			defer patches.Reset()
			ok := c.validateSignature(ak, sk, signHeader, sig)
			So(ok, ShouldBeFalse)
		})
	})
}

func getController() *TokenController {
	c := &TokenController{}
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
