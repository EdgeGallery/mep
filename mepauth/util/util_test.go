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

package util

import (
	"encoding/pem"
	"errors"
	"io/ioutil"
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
		err := genRandRootKeyComponent("tmpComponentFile", "tmpSaltFile")
		data := []byte{'a', 'b', 'c'}
		ClearByteArray(data)
		So(isFileOrDirExist("tmpComponentFile"), ShouldBeTrue)
		So(isFileOrDirExist("tmpSaltFile"), ShouldBeTrue)
		So(err, ShouldBeNil)

		keyComponentFromUserStrByte := []byte("")
		KeyComponentFromUserStr = &keyComponentFromUserStrByte
		_, err2 := genRootKey("tmpComponentFile", "tmpSaltFile")
		So(err2, ShouldNotBeNil)

		keyComponentFromUserStrByte = []byte(componentContent)
		rootKey, err3 := genRootKey("tmpComponentFile", "tmpSaltFile")
		So(err3, ShouldBeNil)
		So(rootKey, ShouldNotBeNil)
		So(len(rootKey), ShouldBeGreaterThan, 0)

		if err := os.Remove("tmpComponentFile"); err != nil {
			log.Error("remove tmpComponentFile failed")
		}
		if err := os.Remove("tmpSaltFile"); err != nil {
			log.Error("remove tmpSaltFile failed")
		}
	})
}

func TestGetPublicKey(t *testing.T) {
	Convey("get public key", t, func() {
		Convey("for success", func() {
			patch1 := ApplyFunc(GetAppConfig, func(_ string) string {
				return "abed"
			})
			defer patch1.Reset()

			patch2 := ApplyFunc(ioutil.ReadFile, func(string) ([]byte, error) {
				return []byte("abc"), nil
			})
			defer patch2.Reset()

			patch3 := ApplyFunc(pem.Decode, func([]byte) (*pem.Block, []byte) {
				block := &pem.Block{
					Type: "PUBLIC KEY",
				}
				return block, nil
			})
			defer patch3.Reset()
			_, err := GetPublicKey()
			So(err, ShouldBeNil)

		})
		Convey("for fail1", func() {
			patches := ApplyFunc(GetAppConfig, func(_ string) string {
				return ""
			})
			defer patches.Reset()
			_, err := GetPublicKey()
			So(err, ShouldNotBeNil)
		})
		Convey("for fail2", func() {
			patch1 := ApplyFunc(GetAppConfig, func(_ string) string {
				return "abed"
			})
			defer patch1.Reset()

			patch2 := ApplyFunc(ioutil.ReadFile, func(string) ([]byte, error) {
				return nil, errors.New("ReadFile error")
			})
			defer patch2.Reset()
			_, err := GetPublicKey()
			So(err, ShouldNotBeNil)
		})
		Convey("for fail3", func() {
			patch1 := ApplyFunc(GetAppConfig, func(_ string) string {
				return "abed"
			})
			defer patch1.Reset()

			patch2 := ApplyFunc(ioutil.ReadFile, func(string) ([]byte, error) {
				return []byte("abc"), nil
			})
			defer patch2.Reset()

			patch3 := ApplyFunc(pem.Decode, func([]byte) (*pem.Block, []byte) {
				return nil, nil
			})
			defer patch3.Reset()
			_, err := GetPublicKey()
			So(err, ShouldNotBeNil)
		})
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
		_, err := EncryptByAES256GCM([]byte("plaintext"), nil, nil)
		So(err, ShouldNotBeNil)
	})
}

func TestDecryptByAES256GCM(t *testing.T) {
	Convey("decrypt by aes 256 gcm", t, func() {
		_, err := DecryptByAES256GCM([]byte("ciphertext"), nil, nil)
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

func TestInitRootKeyAndWorkKey(t *testing.T) {
	Convey("InitRootKeyAndWorkKey", t, func() {
		Convey("for success", func() {
			patch1 := ApplyFunc(isFileOrDirExist, func(string) bool {
				return true
			})
			defer patch1.Reset()

			err := InitRootKeyAndWorkKey()
			So(err, ShouldBeNil)
		})
	})
}

func TestGenAndSaveWorkKey(t *testing.T) {
	Convey("genAndSaveWorkKey", t, func() {
		Convey("for success", func() {
			patch1 := ApplyFunc(EncryptByAES256GCM, func([]byte, []byte, []byte) ([]byte, error) {
				return []byte("value"), nil
			})
			defer patch1.Reset()

			patch2 := ApplyFunc(ioutil.WriteFile, func(string, []byte, os.FileMode) error {
				return nil
			})
			defer patch2.Reset()

			_, err := genAndSaveWorkKey(nil, "", "")
			So(err, ShouldBeNil)
		})
	})
}

func TestDecryptKey(t *testing.T) {
	Convey("decryptKey", t, func() {
		Convey("for success", func() {
			patch1 := ApplyFunc(ioutil.ReadFile, func(string) ([]byte, error) {
				return []byte("value"), nil
			})
			defer patch1.Reset()

			patch2 := ApplyFunc(DecryptByAES256GCM, func([]byte, []byte, []byte) ([]byte, error) {
				return []byte("value"), nil
			})
			defer patch2.Reset()

			_, err := decryptKey(nil, "", "")
			So(err, ShouldBeNil)
		})
	})

}
