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
	"crypto/tls"
	"errors"
	"github.com/agiledragon/gomonkey"
	"github.com/astaxie/beego/orm"
	"io"
	"mepauth/dbAdapter"
	"mepauth/util"
	"os"
	"reflect"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestInitDb(t *testing.T) {
	Convey("initDb", t, func() {
		Convey("for success", func() {
			patch1 := gomonkey.ApplyFunc(orm.RegisterDriver, func(string, orm.DriverType) error {
				return nil
			})
			defer patch1.Reset()
			patch2 := gomonkey.ApplyFunc(orm.RegisterDataBase, func(string, string, string, ...int) error {
				return nil
			})
			defer patch2.Reset()
			patch3 := gomonkey.ApplyFunc(orm.RunSyncdb, func(string, bool, bool) error {
				return nil
			})
			defer patch3.Reset()
			patch4 := gomonkey.ApplyFunc(util.GetAppConfig, func(confvar string) string {
				switch confvar {
				case "dbAdapter":
					return "pgDb"
				case "db_passwd":
					return "Test_Password"
				default:
					return ""
				}
			})
			defer patch4.Reset()

			var pgdb *dbAdapter.PgDb
			patch5 := gomonkey.ApplyMethod(reflect.TypeOf(pgdb), "InitOrmer", func(*dbAdapter.PgDb) error {
				return nil
			})

			defer patch5.Reset()
			dbAdapter.Db = dbAdapter.InitDb()
		})
	})
}

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
			var initializer *ApiGwInitializer
			patch1 := gomonkey.ApplyMethod(reflect.TypeOf(initializer), "InitAPIGateway", func(*ApiGwInitializer, *[]byte) error {
				return nil
			})
			defer patch1.Reset()
			patch2 := gomonkey.ApplyFunc(util.InitRootKeyAndWorkKey, func() error {
				return nil
			})
			defer patch2.Reset()
			patch3 := gomonkey.ApplyFunc(util.TLSConfig, func(string) (*tls.Config, error) {
				return nil, nil
			})
			defer patch3.Reset()
			res := doInitialization(&network)
			So(res, ShouldBeTrue)
		})

		Convey("for InitAPIGateway fail", func() {
			network := []byte("example.com")
			var initializer *ApiGwInitializer
			patch1 := gomonkey.ApplyMethod(reflect.TypeOf(initializer), "InitAPIGateway", func(*ApiGwInitializer, *[]byte) error {
				return errors.New("InitAPIGateway fail")
			})
			patch2 := gomonkey.ApplyFunc(util.TLSConfig, func(string) (*tls.Config, error) {
				return nil, nil
			})
			defer patch2.Reset()
			defer patch1.Reset()
			res := doInitialization(&network)
			So(res, ShouldBeFalse)
		})

		Convey("for InitRootKeyAndWorkKey fail", func() {
			network := []byte("example.com")
			var initializer *ApiGwInitializer
			patch1 := gomonkey.ApplyMethod(reflect.TypeOf(initializer), "InitAPIGateway", func(*ApiGwInitializer, *[]byte) error {
				return nil
			})
			defer patch1.Reset()
			patch2 := gomonkey.ApplyFunc(util.InitRootKeyAndWorkKey, func() error {
				return errors.New("InitRootKeyAndWorkKey fail")
			})
			patch3 := gomonkey.ApplyFunc(util.TLSConfig, func(string) (*tls.Config, error) {
				return nil, nil
			})
			defer patch3.Reset()
			defer patch2.Reset()
			res := doInitialization(&network)
			So(res, ShouldBeFalse)
		})

	})
}
