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
	"bytes"
	"encoding/json"
	"github.com/agiledragon/gomonkey"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"healthcheck-m/controllers"
	"healthcheck-m/util"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

var msg = "Health Check center side connection is ok."

func TestGetEdge(t *testing.T) {
	c := getMecMController()
	c.Get()
	// Check for success case wherein the status value will be default i.e. 0
	assert.Equal(t, 0, c.Ctx.ResponseWriter.Status, msg)
}

func TestGetIpErr(t *testing.T) {
	c := getRunController()
	c.Get()
	// Check for success case wherein the status value will be default i.e. 0
	assert.Equal(t, 400, c.Ctx.ResponseWriter.Status, msg)
}

func TestGetIpGetErr(t *testing.T) {
	c := getRunController()
	patch1 := gomonkey.ApplyFunc(util.ValidateSrcAddress, func(_ string) error {
		return nil
	})
	defer patch1.Reset()

	c.Get()
	// Check for success case wherein the status value will be default i.e. 0
	assert.Equal(t, 500, c.Ctx.ResponseWriter.Status, msg)
}

func TestGetIpNoErr(t *testing.T) {
	c := getRunController()
	patch1 := gomonkey.ApplyFunc(util.ValidateSrcAddress, func(_ string) error {
		return nil
	})
	defer patch1.Reset()
	// Test query

	mecHostInfoK8s := controllers.MecHostInfo{
		MechostIp:   "127.0.0.1",
		MechostName: "k8s",
		City:        "xian",
		Vim:         "K8S",
	}
	mecHostInfoOs := controllers.MecHostInfo{
		MechostIp:   "127.0.0.2",
		MechostName: "OS",
		City:        "xian",
		Vim:         "OpenStack",
	}
	response := []controllers.MecHostInfo{mecHostInfoK8s, mecHostInfoOs}
	responseGetJson, _ := json.Marshal(response)
	responseGetBody := ioutil.NopCloser(bytes.NewReader(responseGetJson))
	patch4 := gomonkey.ApplyMethod(reflect.TypeOf(&http.Client{}), "Do",
		func(_ *http.Client, _ *http.Request) (*http.Response, error) {
			return &http.Response{Body: responseGetBody}, nil
		})
	defer patch4.Reset()
	c.Get()
	// Check for success case wherein the status value will be default i.e. 0
	assert.Equal(t, 400, c.Ctx.ResponseWriter.Status, msg)
}

func getMecMController() *controllers.MecMController {
	getBeegoController := beego.Controller{Ctx: &context.Context{ResponseWriter: &context.Response{ResponseWriter: httptest.NewRecorder()}},
		Data: make(map[interface{}]interface{})}
	c := &controllers.MecMController{BaseController: controllers.BaseController{
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

func getRunController() *controllers.RunController {
	getBeegoController := beego.Controller{Ctx: &context.Context{ResponseWriter: &context.Response{ResponseWriter: httptest.NewRecorder()}},
		Data: make(map[interface{}]interface{})}
	c := &controllers.RunController{BaseController: controllers.BaseController{
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
