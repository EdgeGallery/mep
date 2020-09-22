package util

import (
	"os"
	"testing"

	. "github.com/agiledragon/gomonkey"
	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
)

func TestClearByteArray(t *testing.T) {
	Convey("clear byte array", t, func() {
		data := []byte{'a', 'b', 'c'}
		ClearByteArray(data)
		So(data[0], ShouldEqual, 0)
		So(data[1], ShouldEqual, 0)
		So(data[2], ShouldEqual, 0)
		ClearByteArray(nil)
	})
}

func TestGenRandRootKeyComponent(t *testing.T) {
	Convey("genrate random root key component", t, func() {
		err := genRandRootKeyComponent("tmpComponentFile","tmpSaltFile")
		data := []byte{'a', 'b', 'c'}
		ClearByteArray(data)
		So(isFileOrDirExist("tmpComponentFile"), ShouldBeTrue)
		So(isFileOrDirExist("tmpSaltFile"), ShouldBeTrue)
		So(err, ShouldBeNil)

		keyComponentFromUserStrByte := []byte("")
		KeyComponentFromUserStr = &keyComponentFromUserStrByte
		_, err2 := genRootKey("tmpComponentFile","tmpSaltFile")
		So(err2, ShouldNotBeNil)

		keyComponentFromUserStrByte = []byte(ComponentContent)
		rootKey, err3 := genRootKey("tmpComponentFile","tmpSaltFile")
		So(err3, ShouldBeNil)
		So(rootKey, ShouldNotBeNil)
		So(len(rootKey), ShouldBeGreaterThan, 0)

		if err := os.Remove("tmpComponentFile"); err!=nil{
			log.Error("remove tmpComponentFile failed")
		}
		if err := os.Remove("tmpSaltFile"); err!=nil{
			log.Error("remove tmpSaltFile failed")
		}
	})
}

func TestGetPublicKey(t *testing.T) {
	Convey("get public key", t, func() {
		patches := ApplyFunc(GetAppConfig, func(_ string) string {
			return ""
		})
		defer patches.Reset()
		_, err := GetPublicKey()
		So(err, ShouldNotBeNil)
	})
}

func TestGetPrivateKey(t *testing.T) {
	Convey("get private key", t, func() {
		patches := ApplyFunc(GetAppConfig, func(_ string) string {
			return ""
		})
		defer patches.Reset()
		_, err := GetPrivateKey()
		So(err, ShouldNotBeNil)
	})
}

func TestEncryptByAES256GCM(t *testing.T) {
	Convey("encrypt by aes 256 gcm", t, func() {
		_, err :=  EncryptByAES256GCM([]byte("plaintext"),nil,nil)
		So(err, ShouldNotBeNil)
	})
}

func TestDecryptByAES256GCM(t *testing.T) {
	Convey("decrypt by aes 256 gcm", t, func() {
		_, err :=  DecryptByAES256GCM([]byte("ciphertext"),nil,nil)
		So(err, ShouldNotBeNil)
	})
}

func TestGetCipherSuites(t *testing.T) {
	Convey("get cipher suites", t, func() {
		suite := getCipherSuites("")
		So(suite, ShouldBeNil)
		suite = getCipherSuites("TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA512")
		So(suite, ShouldBeNil)
		suite = getCipherSuites("TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256")
		So(suite, ShouldNotBeNil)
	})
}


