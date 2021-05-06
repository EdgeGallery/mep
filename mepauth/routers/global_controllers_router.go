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

package routers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context/param"
)

const (
	ConfController     = "mepauth/controllers:ConfController"
	OneRouteController = "mepauth/controllers:OneRouteController"
	TokenController    = "mepauth/controllers:TokenController"
)

const (
	DELETE = "delete"
	GET    = "get"
	PUT    = "put"
	POST   = "post"
)

const (
	RootPath            string = "/mep"
	AuthTokenPrefix     string = "/token"
	AppManagePrefix     string = "/appMng/v1"
	AuthTokenPath              = RootPath + AuthTokenPrefix
	AppManagePath              = RootPath + AppManagePrefix
	OneControllerRoute         = AppManagePrefix + "/routes/:routeId"
	ConfControllerRoute        = AppManagePrefix + "/applications/:applicationId/confs"
)

func init() {
	beego.GlobalControllerRouter[ConfController] = append(beego.GlobalControllerRouter[ConfController],
		beego.ControllerComments{
			Method:           "Put",
			Router:           ConfControllerRoute,
			AllowHTTPMethods: []string{PUT},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})
	beego.GlobalControllerRouter[ConfController] = append(beego.GlobalControllerRouter[ConfController],
		beego.ControllerComments{
			Method:           "Delete",
			Router:           ConfControllerRoute,
			AllowHTTPMethods: []string{DELETE},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})
	beego.GlobalControllerRouter[ConfController] = append(beego.GlobalControllerRouter[ConfController],
		beego.ControllerComments{
			Method:           "Get",
			Router:           ConfControllerRoute,
			AllowHTTPMethods: []string{GET},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter[OneRouteController] = append(beego.GlobalControllerRouter[OneRouteController],
		beego.ControllerComments{
			Method:           "Put",
			Router:           OneControllerRoute,
			AllowHTTPMethods: []string{PUT},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})
	beego.GlobalControllerRouter[OneRouteController] = append(beego.GlobalControllerRouter[OneRouteController],
		beego.ControllerComments{
			Method:           "Delete",
			Router:           OneControllerRoute,
			AllowHTTPMethods: []string{DELETE},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})
	beego.GlobalControllerRouter[OneRouteController] = append(beego.GlobalControllerRouter[OneRouteController],
		beego.ControllerComments{
			Method:           "Get",
			Router:           OneControllerRoute,
			AllowHTTPMethods: []string{GET},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter[TokenController] = append(beego.GlobalControllerRouter[TokenController],
		beego.ControllerComments{
			Method:           "Post",
			Router:           AuthTokenPrefix,
			AllowHTTPMethods: []string{POST},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})
}
