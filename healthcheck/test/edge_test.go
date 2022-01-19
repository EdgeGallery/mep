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

package test

import (
	"github.com/agiledragon/gomonkey"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"healthcheck/controllers"
	"healthcheck/util"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetEdgeIpErr(t *testing.T) {
	c := getEdgeController()
	c.Get()
	// Check for success case wherein the status value will be default i.e. 0
	assert.Equal(t, 400, c.Ctx.ResponseWriter.Status, msg)
}

func TestGetEdgeIpOk(t *testing.T) {
	c := getEdgeController()
	patch2 := gomonkey.ApplyFunc(util.ValidateSrcAddress, func(_ string) error {
		return nil
	})
	defer patch2.Reset()
	c.Get()
	// Check for success case wherein the status value will be default i.e. 0
	assert.Equal(t, 500, c.Ctx.ResponseWriter.Status, msg)
}

func getEdgeController() *controllers.EdgeController {
	getBeegoController := beego.Controller{Ctx: &context.Context{ResponseWriter: &context.Response{ResponseWriter: httptest.NewRecorder()}},
		Data: make(map[interface{}]interface{})}
	c := &controllers.EdgeController{BaseController: controllers.BaseController{
		Controller: getBeegoController}}
	c.Init(context.NewContext(), "", "", nil)
	req, err := http.NewRequest("GET", "http://127.0.0.1", strings.NewReader(""))
	if err != nil {
		log.Error("Prepare http request failed")
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
