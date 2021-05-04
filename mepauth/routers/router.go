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

// MEP Auth APIs
// @APIVersion 1.0.0
// @Title MEP Auth API
// @Description APIs for MEP authentication
// @TermsOfServiceUrl http://beego.me/
package routers

import (
	"github.com/astaxie/beego"
	"mepauth/controllers"
)

// Init mepauth APIs
func init() {

	ns := beego.NewNamespace("/mep/",
		beego.NSInclude(
			&controllers.ConfController{},
			&controllers.OneRouteController{},
			&controllers.TokenController{},
		),
	)
	beego.AddNamespace(ns)
}
