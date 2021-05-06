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

package util

import (
	"net/http"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetSignature(t *testing.T) {
	Convey("clear byte array", t, func() {
		s := Sign{
			AccessKey: "00000000",
			SecretKey: []byte("00000000"),
		}
		host := "127.0.0.1:8080"
		reqUrl := "https://" + host + "/mepauth/mepauth/v1/token"
		reqToBeSigned, errNewRequest := http.NewRequest("POST", reqUrl, strings.NewReader(""))
		if errNewRequest != nil {
			log.Error("prepare http request to generate signature is failed")
		}
		reqToBeSigned.Header.Set("content-type", "json")
		reqToBeSigned.Header.Set(HostHeader, host)
		reqToBeSigned.Header.Set(DateHeader, DateFormat)
		signature, err := s.GetSignature(reqToBeSigned)
		So(signature, ShouldNotBeNil)
		So(err, ShouldBeNil)
	})
}
