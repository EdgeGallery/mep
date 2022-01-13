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
	"healthcheck/controllers"
	"healthcheck/util"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

var msg = "Communicate get request result received."

func TestComGet(t *testing.T) {
	c := getComController()
	c.Get()
	// Check for success case wherein the status value will be default i.e. 0
	assert.Equal(t, 0, c.Ctx.ResponseWriter.Status, msg)
}

func TestComPostIpErr(t *testing.T) {
	c := getComController()
	c.Post()
	// Check for success case wherein the status value will be default i.e. 0
	assert.Equal(t, 400, c.Ctx.ResponseWriter.Status, msg)
}

func TestComPostUnmarshalErr(t *testing.T) {
	c := getComController()
	patch1 := gomonkey.ApplyFunc(util.ValidateSrcAddress, func(_ string) error {
		return nil
	})
	defer patch1.Reset()
	c.Post()
	// Check for success case wherein the status value will be default i.e. 0
	assert.Equal(t, 400, c.Ctx.ResponseWriter.Status, msg)
}

func TestComPost(t *testing.T) {
	c := getComController()

	patch1 := gomonkey.ApplyFunc(util.ValidateSrcAddress, func(_ string) error {
		return nil
	})
	defer patch1.Reset()
	mecHostIp := []string{"127.0.0.1", ""}
	requestBody, _ := json.Marshal(map[string]interface{}{
		"mechostIp": mecHostIp,
	})
	c.Ctx.Input.RequestBody = requestBody
	c.Post()
	// Check for success case wherein the status value will be default i.e. 0
	assert.Equal(t, 0, c.Ctx.ResponseWriter.Status, msg)
}

func TestComPostGetUnhealthy(t *testing.T) {
	c := getComController()

	patch1 := gomonkey.ApplyFunc(util.ValidateSrcAddress, func(_ string) error {
		return nil
	})
	defer patch1.Reset()

	responseGetJson, _ := json.Marshal("ok")
	responseGetBody := ioutil.NopCloser(bytes.NewReader(responseGetJson))
	patch11 := gomonkey.ApplyMethod(reflect.TypeOf(&http.Client{}), "Get", func(client *http.Client, url string) (resp *http.Response, err error) {
		return &http.Response{Body: responseGetBody}, nil
	})
	defer patch11.Reset()
	mecHostIp := []string{"127.0.0.1", ""}
	requestBody, _ := json.Marshal(map[string]interface{}{
		"mechostIp": mecHostIp,
	})
	c.Ctx.Input.RequestBody = requestBody
	c.Post()
	// Check for success case wherein the status value will be default i.e. 0
	assert.Equal(t, 500, c.Ctx.ResponseWriter.Status, msg)
}

func TestComPostGetHealthy(t *testing.T) {
	c := getComController()
	patch1 := gomonkey.ApplyFunc(util.ValidateSrcAddress, func(_ string) error {
		return nil
	})
	defer patch1.Reset()

	responseGetJson, _ := json.Marshal("ok")
	responseGetBody := ioutil.NopCloser(bytes.NewReader(responseGetJson))
	patch11 := gomonkey.ApplyMethod(reflect.TypeOf(&http.Client{}), "Get", func(client *http.Client, url string) (resp *http.Response, err error) {
		return &http.Response{Body: responseGetBody, StatusCode: 200}, nil
	})
	defer patch11.Reset()
	mecHostIp := []string{"127.0.0.1", ""}
	requestBody, _ := json.Marshal(map[string]interface{}{
		"mechostIp": mecHostIp,
	})
	c.Ctx.Input.RequestBody = requestBody
	c.Post()
	// Check for success case wherein the status value will be default i.e. 0
	assert.Equal(t, 0, c.Ctx.ResponseWriter.Status, msg)
}

func getComController() *controllers.ComController {
	getBeegoController := beego.Controller{Ctx: &context.Context{ResponseWriter: &context.Response{ResponseWriter: httptest.NewRecorder()}},
		Data: make(map[interface{}]interface{})}
	c := &controllers.ComController{BaseController: controllers.BaseController{
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
