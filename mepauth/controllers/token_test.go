package controllers

import (
	"crypto/rsa"
	"encoding/hex"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"

	. "github.com/agiledragon/gomonkey"
	"github.com/dgrijalva/jwt-go/v4"
	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"

	"mepauth/models"
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


func TestParseAuthHeader(t *testing.T) {

	Convey("parse auth header", t, func(){
		ak, signHeader, sig := parseAuthHeader("SDK-HMAC-SHA256 Access=QVUJMSUMgS0VZLS0tLS0, " +
			"SignedHeaders=content-type;host;x-sdk-date, " +
			"Signature=62192e2ee0b871321e43a607654f93f661a91fcdedba86e45f02602c99eca052")
		So(ak, ShouldEqual,"QVUJMSUMgS0VZLS0tLS0")
		So(signHeader, ShouldEqual,"content-type;host;x-sdk-date")
		So(sig, ShouldEqual,"62192e2ee0b871321e43a607654f93f661a91fcdedba86e45f02602c99eca052")
		ak, signHeader, sig = parseAuthHeader("invalid_header")
		So(ak, ShouldEqual,"")
		So(signHeader, ShouldEqual,"")
		So(sig, ShouldEqual,"")
	})
}

func TestGetAppInsIdSk(t *testing.T) {
	authInfo := &models.AuthInfoRecord{}
	authInfo.Ak="QVUJMSUMgS0VZLS0tLS0"
	authInfo.Sk="sk"
	authInfo.AppInsId="5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"
	authInfo.Nonce="nonce"

	Convey("get app instance id and sk", t, func(){
		Convey("for success", func(){
			patches := ApplyFunc(ReadData, func(_ string) (models.AuthInfoRecord, error) {
				return *authInfo, nil
			})
			patches.ApplyFunc(hex.Decode,func(_, _ []byte) (int,error) {
				return 0,nil
			})
			patches.ApplyFunc(util.GetWorkKey, func() ([]byte, error) {
				return []byte("validKey"), nil
			})
			patches.ApplyFunc(util.DecryptByAES256GCM, func(_,_,_ []byte)([]byte, error) {
				return nil, nil
			})
			defer patches.Reset()
			appInsId, _, ok := GetAppInsIdSk("QVUJMSUMgS0VZLS0tLS0")
			So(appInsId, ShouldEqual,"5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f")
			So(ok, ShouldBeTrue)
		})
		Convey("for read fail", func(){
			patches := ApplyFunc(ReadData, func(_ string) (models.AuthInfoRecord, error) {
				return *authInfo, errors.New("read error")
			})
			defer patches.Reset()
			_, _, ok := GetAppInsIdSk("QVUJMSUMgS0VZLS0tLS0")
			So(ok, ShouldBeFalse)
		})
		Convey("for decode fail", func(){
			patches := ApplyFunc(ReadData, func(_ string) (models.AuthInfoRecord, error) {
				return *authInfo, nil
			})
			patches.ApplyFunc(hex.Decode,func(_, _ []byte) (int,error) {
				return 0,errors.New("decode fail")
			})
			defer patches.Reset()
			appInsId, _, ok := GetAppInsIdSk("QVUJMSUMgS0VZLS0tLS0")
			So(appInsId, ShouldEqual, "")
			So(ok, ShouldBeTrue)
		})
		Convey("for get work key fail", func(){
			patches := ApplyFunc(ReadData, func(_ string) (models.AuthInfoRecord, error) {
				return *authInfo, nil
			})
			patches.ApplyFunc(hex.Decode,func(_, _ []byte) (int,error) {
				return 0,nil
			})
			patches.ApplyFunc(util.GetWorkKey, func() ([]byte, error) {
				return []byte("validKey"), errors.New("get work key fail")
			})
			appInsId, _, ok := GetAppInsIdSk("QVUJMSUMgS0VZLS0tLS0")
			So(appInsId, ShouldEqual, "")
			So(ok, ShouldBeTrue)
		})
		Convey("for decrypt fail", func(){
			patches := ApplyFunc(ReadData, func(_ string) (models.AuthInfoRecord, error) {
				return *authInfo, nil
			})
			patches.ApplyFunc(hex.Decode,func(_, _ []byte) (int,error) {
				return 0,nil
			})
			patches.ApplyFunc(util.GetWorkKey, func() ([]byte, error) {
				return []byte("validKey"), errors.New("get work key fail")
			})
			patches.ApplyFunc(util.DecryptByAES256GCM, func(_,_,_ []byte)([]byte, error) {
				return nil, errors.New("for decrypt fail")
			})
			appInsId, _, ok := GetAppInsIdSk("QVUJMSUMgS0VZLS0tLS0")
			So(appInsId, ShouldEqual, "")
			So(ok, ShouldBeTrue)
		})
	})
}

func TestAkSignatureIsValid(t *testing.T) {

	ak := "QVUJMSUMgS0VZLS0tLS0"
	sk := []byte("sksksksk")
	signHeader := "content-type;host;x-sdk-date"
	sig := "signature"
	r, err := http.NewRequest("POST", "http://127.0.0.1", strings.NewReader(""))
	if err != nil {
		log.Error("prepare http request failed")
	}
	r.Header.Set("content-type", "json")
	r.Header.Set(util.HOST_HEADER, "127.0.0.1")
	r.Header.Set(util.DATE_HEADER, util.DATE_FORMAT)

	Convey("ak signature is valid", t, func(){
		Convey("for success", func(){
			var s *util.Sign
			patches := ApplyMethod(reflect.TypeOf(s), "GetSignature", func(_ *util.Sign, _ *http.Request) (string, error) {
				return "signature", nil
			})
			defer patches.Reset()
			ok, err:=akSignatureIsValid(r,ak,sk,signHeader,sig)

			So(ok, ShouldBeTrue)
			So(err, ShouldBeNil)
		})
		Convey("for fail", func(){
			var s *util.Sign
			patches := ApplyMethod(reflect.TypeOf(s), "GetSignature", func(_ *util.Sign, _ *http.Request) (string, error) {
				return "_signature", nil
			})
			defer patches.Reset()
			ok, err:=akSignatureIsValid(r,ak,sk,signHeader,sig)

			So(ok, ShouldBeFalse)
			So(err, ShouldBeNil)
		})
		Convey("for error", func(){
			var s *util.Sign
			patches := ApplyMethod(reflect.TypeOf(s), "GetSignature", func(_ *util.Sign, _ *http.Request) (string, error) {
				return "signature", errors.New("get sig error")
			})
			defer patches.Reset()
			ok, err:=akSignatureIsValid(r,ak,sk,signHeader,sig)

			So(ok, ShouldBeFalse)
			So(err, ShouldNotBeNil)
		})

	})
}

func fromBase10(base10 string) *big.Int {
	i, ok := new(big.Int).SetString(base10, 10)
	if !ok {
		panic("bad number: " + base10)
	}
	return i
}

func TestGenerateJwtToken(t *testing.T) {
	appInsId:="5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f"
	clientIp:="127.0.0.1"
	token := &jwt.Token{}
	priv := &rsa.PrivateKey{
		PublicKey: rsa.PublicKey{
			N: fromBase10("290684273230919398108010081414538931343"),
			E: 65537,
		},
		D: fromBase10("31877380284581499213530787347443987241"),
		Primes: []*big.Int{
			fromBase10("16775196964030542637"),
			fromBase10("17328218193455850539"),
		},
	}
	err := beego.LoadAppConfig("ini", "../conf/app.conf")
	if err != nil {
		panic(err)
	}
	Convey("generate jwt token", t, func(){
		Convey("for success", func(){
			patches := ApplyFunc(util.GetPrivateKey, func() (*rsa.PrivateKey, error) {
				return priv, nil
			})
			patches.ApplyMethod(reflect.TypeOf(token), "SignedString", func(_ *jwt.Token, _ interface{}, _ ...jwt.SigningOption) (string, error) {
				return "token_content", nil
			})

			defer patches.Reset()
			token, err := generateJwtToken(appInsId,clientIp)

			So(token, ShouldNotEqual,"")
			So(err, ShouldBeNil)
		})
		Convey("for fail", func(){
			patches := ApplyFunc(util.GetPrivateKey, func() (*rsa.PrivateKey, error) {
				return nil, errors.New("get private key fail")
			})
			defer patches.Reset()
			token, err := generateJwtToken(appInsId,clientIp)

			So(token, ShouldBeNil)
			So(err.Error(), ShouldEqual, "failed to get private key")
		})
	})
}

