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

// Package routers registers routes of mepauth
package routers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context/param"
)

const (
	confController  = "mepauth/controllers:ConfController"
	tokenController = "mepauth/controllers:TokenController"
)

const (
	deleteOp = "deleteOp"
	get      = "get"
	put      = "put"
	post     = "post"
)

const (
	rootPath            string = "/mep"
	authTokenPrefix     string = "/token"
	appManagePrefix     string = "/appMng/v1"
	AuthTokenPath              = rootPath + authTokenPrefix
	AppManagePath              = rootPath + appManagePrefix
	confControllerRoute        = appManagePrefix + "/applications/:applicationId/confs"
)

func init() {
	beego.GlobalControllerRouter[confController] = append(beego.GlobalControllerRouter[confController],
		beego.ControllerComments{
			Method:           "Put",
			Router:           confControllerRoute,
			AllowHTTPMethods: []string{put},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})
	beego.GlobalControllerRouter[confController] = append(beego.GlobalControllerRouter[confController],
		beego.ControllerComments{
			Method:           "deleteOp",
			Router:           confControllerRoute,
			AllowHTTPMethods: []string{deleteOp},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})
	beego.GlobalControllerRouter[confController] = append(beego.GlobalControllerRouter[confController],
		beego.ControllerComments{
			Method:           "Get",
			Router:           confControllerRoute,
			AllowHTTPMethods: []string{get},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter[tokenController] = append(beego.GlobalControllerRouter[tokenController],
		beego.ControllerComments{
			Method:           "Post",
			Router:           authTokenPrefix,
			AllowHTTPMethods: []string{post},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})
}
