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

package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/httplib"
	log "github.com/sirupsen/logrus"
	"mepauth/dbAdapter"

	"mepauth/models"
	"mepauth/util"
)

const RouteId string = "routeId"

type OneRouteController struct {
	beego.Controller
}

func (c *OneRouteController) Get() {
	routeId := c.Ctx.Input.Param(util.UrlRouteId)
	log.Info(routeId)
	routeRecord := &models.RouteRecord{
		RouteId: routeId,
	}
	err := dbAdapter.Db.ReadData(routeRecord, RouteId)
	if err != nil {
		c.Data["json"] = err.Error()
	}
	c.Data["json"] = routeRecord

	c.ServeJSON()
}

func (c *OneRouteController) Put() {
	routeId := c.Ctx.Input.Param(util.UrlRouteId)
	log.Info(routeId)
	var routeInfo *models.RouteInfo
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &routeInfo); err == nil {
		c.Data["json"] = routeInfo
		routeRecord := &models.RouteRecord{
			RouteId: routeId,
			AppId:   routeInfo.AppId,
			SerName: routeInfo.SerInfo.SerName,
		}
		err := dbAdapter.Db.InsertData(routeRecord)
		if err != nil {
			c.Data["json"] = err.Error()
		}
		addApigwService(routeInfo)
		addApigwRoute(routeInfo)

	} else {
		c.Data["json"] = err.Error()
	}
	c.ServeJSON()
}

func addApigwRoute(routeInfo *models.RouteInfo) {
	serName := routeInfo.SerInfo.SerName
	kongRouteUrl := fmt.Sprintf("https://%s:%s/services/%s/routes",
		util.GetAppConfig(util.ApigwHost),
		util.GetAppConfig(util.ApigwPort),
		serName)
	req := httplib.Post(kongRouteUrl)
	jsonStr := []byte(fmt.Sprintf(`{ "paths": ["/%s"], "name": "%s" }`, serName, serName))
	req.Header(util.ContentType, util.JsonUtf8)
	req.Body(jsonStr)

	str, err := req.String()
	if err != nil {
		log.Error(err)
	}
	log.Infof("addApigwRoute resp = %s", str)
}

func addApigwService(routeInfo *models.RouteInfo) {
	serName := routeInfo.SerInfo.SerName
	kongServiceUrl := fmt.Sprintf("https://%s:%s/services",
		util.GetAppConfig(util.ApigwHost),
		util.GetAppConfig(util.ApigwPort))
	req := httplib.Post(kongServiceUrl)
	serUrl := routeInfo.SerInfo.Uris[0]
	jsonStr := []byte(fmt.Sprintf(`{ "url": "%s", "name": "%s" }`, serUrl, serName))
	req.Header(util.ContentType, util.JsonUtf8)
	req.Body(jsonStr)

	str, err := req.String()
	if err != nil {
		log.Error(err)
	}
	log.Infof("addApigwService resp = %s", str)

	addJwtPlugin(serName)
}

func addJwtPlugin(serName string) {
	jwtPluginUrl := fmt.Sprintf("https://%s:%s/services/%s/plugins",
		util.GetAppConfig(util.ApigwHost),
		util.GetAppConfig(util.ApigwPort),
		serName)
	req := httplib.Post(jwtPluginUrl)
	jsonPluginStr := []byte(`{ "name": "jwt" }`)
	req.Header(util.ContentType, util.JsonUtf8)
	req.Body(jsonPluginStr)

	str, err := req.String()
	if err != nil {
		log.Error(err)
	}
	log.Infof("addJwtPlugin = %s", str)

}

func (c *OneRouteController) Delete() {
	routeId := c.Ctx.Input.Param(util.UrlRouteId)
	log.Info(routeId)
	routeRecord := &models.RouteRecord{
		RouteId: routeId,
	}
	err := dbAdapter.Db.ReadData(routeRecord, RouteId)
	if err != nil {
		c.Data["json"] = err.Error()
	}

	apigwDelRoute(routeRecord.SerName)

	err = dbAdapter.Db.DeleteData(routeRecord, RouteId)
	if err != nil {
		c.Data["json"] = err.Error()
	}
	c.Data["json"] = nil
	c.ServeJSON()
}

func apigwDelRoute(serName string) {
	kongRouteUrl := fmt.Sprintf("https://%s:%s/services/%s/routes/%s",
		util.GetAppConfig(util.ApigwHost), util.GetAppConfig(util.ApigwPort), serName, serName)
	req := httplib.Delete(kongRouteUrl)
	str, err := req.String()
	if err != nil {
		log.Error(err)
	}
	log.Infof("apigwDelRoute = %s", str)
}
