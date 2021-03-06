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

// Package controllers implements mep auth controller
package controllers

import (
	"errors"
	"github.com/astaxie/beego"
	"github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"
)

const (
	operation = "] operation ["
	resource  = " resource ["
)

// BaseController base controller
type BaseController struct {
	beego.Controller
}

// To display log for received message
func (c *BaseController) logReceivedMsg(clientIp string) {
	log.Info("Received message from ClientIP [" + clientIp + operation + c.Ctx.Request.Method + "]" +
		resource + c.Ctx.Input.URL() + "]")
}

// To display log for received message
func (c *BaseController) logReceivedMsgWithAk(clientIp string, ak string) {
	log.Info("Received message from ClientIP [" + clientIp + "] ClientAK [" + ak + "]" + operation +
		c.Ctx.Request.Method + "]" + resource + c.Ctx.Input.URL() + "]")
}

// Handled logging for error case
func (c *BaseController) handleLoggingForError(clientIp string, code int, errMsg string) {
	c.writeErrorResponse(errMsg, code)
	c.logErrResponseMsg(clientIp, errMsg)
}

func (c *BaseController) logErrResponseMsg(clientIp string, errMsg string) {
	log.Info("Response message for ClientIP [" + clientIp + operation +
		c.Ctx.Request.Method + "]" + resource + c.Ctx.Input.URL() + "] Result [Failure: " + errMsg + ".]")
}

func (c *BaseController) logErrResponseMsgWithAk(clientIp string, errMsg string, ak string) {
	log.Info("Response message for ClientIP [" + clientIp + "] ClientAK [" + ak + "]" + operation +
		c.Ctx.Request.Method + "]" + resource + c.Ctx.Input.URL() + "] Result [Failure: " + errMsg + ".]")
}

// Write error response
func (c *BaseController) writeErrorResponse(errMsg string, code int) {
	log.Error(errMsg)
	c.writeResponse(errMsg, code)
}

// Write response
func (c *BaseController) writeResponse(msg string, code int) {
	c.Data["json"] = msg
	c.Ctx.ResponseWriter.WriteHeader(code)
	c.ServeJSON()
}

// Handled logging for success case
func (c *BaseController) handleLoggingForSuccess(clientIp string, msg string) {
	c.ServeJSON()
	if msg != "" {
		log.Info("Response message for ClientIP [" + clientIp + operation + c.Ctx.Request.Method + "]" +
			resource + c.Ctx.Input.URL() + "] Result [Success: " + msg + "]")
	} else {
		log.Info("Response message for ClientIP [" + clientIp + operation + c.Ctx.Request.Method + "]" +
			resource + c.Ctx.Input.URL() + "] Result [Success]")
	}
}

// Validate source address
func (c *BaseController) validateSrcAddress(id string) error {
	if id == "" {
		log.Error("Source IP address validation failed as input is nil")
		return errors.New("source IP address validation failed as input is nil")
	}

	validate := validator.New()
	err := validate.Var(id, "required,ipv4")
	if err != nil {
		return validate.Var(id, "required,ipv6")
	}
	return nil
}
