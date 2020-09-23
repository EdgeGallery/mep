package main

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestScanConfig(t *testing.T) {

	r:=strings.NewReader("JWT_PRIVATE_KEY=private_key\nACCESS_KEY=QVUJMSUMgS0VZLS0tLS0\n" +
		"SECRET_KEY=DXPb4sqElKhcHe07Kw5uorayETwId1JOjjOIRomRs5wyszoCR5R7AtVa28KT3lSc\n" +
		"APP_INST_ID=5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f\nKEY_COMPONENT=oikYVgrRbDZHZSaob" +
		"OTo8ugCKsUSdVeMsg2d9b7Qr250q2HNBiET4WmecJ0MFavRA0cBzOWu8sObLha17auHoy6ULbAOgP50bDZa" +
		"pxOylTbr1kq8Z4m8uMztciGtq4e11GA0aEh0oLCR3kxFtV4EgOm4eZb7vmEQeMtBy4jaXl6miMJugoRqcfLo9" +
		"ojDYk73lbCaP9ydUkO56fw8dUUYjeMvrzmIZPLdVjPm62R4AQFQ4CEs7vp6xafx9dRwPoym\nTRUSTED_LIST=\n")
	Convey("scan config file", t, func() {
		config, err:=scanConfig(r)
		So(err, ShouldBeNil)
		So(string(*config["JWT_PRIVATE_KEY"]), ShouldEqual, "private_key" )
	})
}

func TestReadPropertiesFile(t *testing.T) {
	Convey("read properties file", t, func() {
		Convey("for success", func() {
			config, err:=readPropertiesFile("")
			So(config, ShouldBeNil)
			So(err, ShouldBeNil)
			config, err=readPropertiesFile("main.go")
			So(config, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})

	})
}