package main

import (
	"errors"
	"io"
	"mepauth/util"
	"os"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey"

	. "github.com/smartystreets/goconvey/convey"
)

func TestScanConfig(t *testing.T) {

	r := strings.NewReader("JWT_PRIVATE_KEY=private_key\nACCESS_KEY=QVUJMSUMgS0VZLS0tLS0\n" +
		"SECRET_KEY=DXPb4sqElKhcHe07Kw5uorayETwId1JOjjOIRomRs5wyszoCR5R7AtVa28KT3lSc\n" +
		"APP_INST_ID=5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f\nKEY_COMPONENT=oikYVgrRbDZHZSaob" +
		"OTo8ugCKsUSdVeMsg2d9b7Qr250q2HNBiET4WmecJ0MFavRA0cBzOWu8sObLha17auHoy6ULbAOgP50bDZa" +
		"pxOylTbr1kq8Z4m8uMztciGtq4e11GA0aEh0oLCR3kxFtV4EgOm4eZb7vmEQeMtBy4jaXl6miMJugoRqcfLo9" +
		"ojDYk73lbCaP9ydUkO56fw8dUUYjeMvrzmIZPLdVjPm62R4AQFQ4CEs7vp6xafx9dRwPoym\nTRUSTED_LIST=\n")
	Convey("scan config file", t, func() {
		config, err := scanConfig(r)
		So(err, ShouldBeNil)
		So(string(*config["JWT_PRIVATE_KEY"]), ShouldEqual, "private_key")
	})
}

func TestReadPropertiesFile(t *testing.T) {
	Convey("read properties file", t, func() {
		Convey("for success", func() {
			config, err := readPropertiesFile("")
			So(config, ShouldBeNil)
			So(err, ShouldBeNil)
			config, err = readPropertiesFile("main.go")
			So(config, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})
		Convey("for open file fail", func() {
			patch1 := gomonkey.ApplyFunc(os.Open, func(string) (*os.File, error) {
				return nil, errors.New("open file fail")
			})
			defer patch1.Reset()
			_, err := readPropertiesFile("abc.go")
			So(err, ShouldNotBeNil)
		})
		Convey("scan config fail", func() {
			patch1 := gomonkey.ApplyFunc(scanConfig, func(io.Reader) (util.AppConfigProperties, error) {
				return util.AppConfigProperties{}, errors.New("scan config fail")
			})
			defer patch1.Reset()
			_, err := readPropertiesFile("main.go")
			So(err, ShouldNotBeNil)
		})
	})
}

func TestClearAppConfigOnExit(t *testing.T) {
	Convey("Clear AppConfig", t, func() {
		Convey("for success", func() {
			trustedNetworks := util.AppConfigProperties{}
			network := []byte("example.com")
			trustedNetworks["network1"] = &network
			clearAppConfigOnExit(trustedNetworks)
		})
	})
}

func TestDoInitialization(t *testing.T) {
	Convey("Do Initialization", t, func() {
		Convey("for success", func() {
			network := []byte("example.com")
			patch1 := gomonkey.ApplyFunc(initAPIGateway, func(*[]byte) error {
				return nil
			})
			defer patch1.Reset()
			patch2 := gomonkey.ApplyFunc(util.InitRootKeyAndWorkKey, func() error {
				return nil
			})
			defer patch2.Reset()
			res := doInitialization(&network)
			So(res, ShouldBeTrue)
		})

		Convey("for initAPIGateway fail", func() {
			network := []byte("example.com")
			patch1 := gomonkey.ApplyFunc(initAPIGateway, func(*[]byte) error {
				return errors.New("initAPIGateway fail")
			})
			defer patch1.Reset()
			res := doInitialization(&network)
			So(res, ShouldBeFalse)
		})

		Convey("for InitRootKeyAndWorkKey fail", func() {
			network := []byte("example.com")
			patch1 := gomonkey.ApplyFunc(initAPIGateway, func(*[]byte) error {
				return nil
			})
			defer patch1.Reset()
			patch2 := gomonkey.ApplyFunc(util.InitRootKeyAndWorkKey, func() error {
				return errors.New("InitRootKeyAndWorkKey fail")
			})
			defer patch2.Reset()
			res := doInitialization(&network)
			So(res, ShouldBeFalse)
		})

	})
}
