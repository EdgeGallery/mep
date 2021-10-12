/*
 * Copyright 2021 Huawei Technologies Co., Ltd.
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

package controllers

import (
	"github.com/astaxie/beego/context"
	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func getBaseController() *BaseController {
	c := &BaseController{}
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

func TestLogReceivedMsgWithAk(t *testing.T) {
	c := getBaseController()
	Convey("Validate log received msg with ak", t, func() {
		Convey("for success", func() {
			c.logReceivedMsgWithAk("1.1.1.1", "test_string")
			out := c.Data["json"]
			So(out, ShouldBeNil)
		})
	})
}

func TestValidateSrcAddress(t *testing.T) {
	c := getBaseController()
	Convey("Validate source address", t, func() {
		Convey("for failure", func() {
			err := c.validateSrcAddress("")
			So(err, ShouldBeError)
		})

		Convey("for invalid ip", func() {
			err := c.validateSrcAddress("1.1.1")
			So(err, ShouldBeError)
		})
	})
}

func TestHandleLoggingForSuccess(t *testing.T) {
	c := getBaseController()
	Convey("Validate logging for success", t, func() {
		Convey("for success", func() {
			c.handleLoggingForSuccess("1.1.1.1", "")
			out := c.Data["json"]
			So(out, ShouldBeNil)
		})
	})
}
