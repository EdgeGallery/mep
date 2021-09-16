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

// Package models implements mep server object models
package models

import (
	"github.com/agiledragon/gomonkey"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	"github.com/stretchr/testify/assert"
	meputil "mepserver/common/util"
	"reflect"
	"testing"
)

//Query traffic rule gets in mp1 interface
func TestGetTrafficRules(t *testing.T) {
	service := ServiceInfo{}

	var s *ServiceInfo
	patches := gomonkey.ApplyMethod(reflect.TypeOf(s), "RegisterToApiGw",
		func(s *ServiceInfo, uri string, serviceName string, isUpdateReq bool) {
			return
		})
	defer patches.Reset()
	var a *meputil.ApiGwIf
	patches.ApplyMethod(reflect.TypeOf(a), "CleanUpApiGwEntry",
		func(a *meputil.ApiGwIf, serName string) {
			return
		})

	t.Run("Create new uri", func(t *testing.T) {
		instance := &proto.MicroServiceInstance{}
		instance.Properties = make(map[string]string, 0)

		service.TransportInfo.Endpoint.Uris = make([]string, 0)
		service.TransportInfo.Endpoint.Uris = append(service.TransportInfo.Endpoint.Uris, "/abc/end1")
		service.TransportInfo.Endpoint.Uris = append(service.TransportInfo.Endpoint.Uris, "/abc/end2")

		eps, epType := service.registerEndpoints(false, instance)

		assert.Equal(t, 2, len(eps))
		assert.Equal(t, meputil.Uris, epType)
	})

	t.Run("Modify uri add new", func(t *testing.T) {
		instance := &proto.MicroServiceInstance{}
		instance.Properties = make(map[string]string, 0)
		instance.Properties[meputil.EndPointPropPrefix+"/abc/end1"] = "abc1"

		service.TransportInfo.Endpoint.Uris = make([]string, 0)
		service.TransportInfo.Endpoint.Uris = append(service.TransportInfo.Endpoint.Uris, "/abc/end1")
		service.TransportInfo.Endpoint.Uris = append(service.TransportInfo.Endpoint.Uris, "/abc/end2")

		eps, epType := service.registerEndpoints(true, instance)

		assert.Equal(t, 2, len(eps))
		assert.Equal(t, meputil.Uris, epType)
	})

	t.Run("Modify uri add 1 new, delete 1", func(t *testing.T) {
		instance := &proto.MicroServiceInstance{}
		instance.Properties = make(map[string]string, 0)
		instance.Properties[meputil.EndPointPropPrefix+"/abc/end1"] = "abc1"
		instance.Properties[meputil.EndPointPropPrefix+"/abc/end2"] = "abc2"

		service.TransportInfo.Endpoint.Uris = make([]string, 0)
		service.TransportInfo.Endpoint.Uris = append(service.TransportInfo.Endpoint.Uris, "/abc/end2")
		service.TransportInfo.Endpoint.Uris = append(service.TransportInfo.Endpoint.Uris, "/abc/end3")

		eps, epType := service.registerEndpoints(true, instance)

		assert.Equal(t, 2, len(eps))
		assert.Equal(t, meputil.Uris, epType)
	})

	t.Run("Modify address add 1 new, delete 1", func(t *testing.T) {
		instance := &proto.MicroServiceInstance{}
		instance.Properties = make(map[string]string, 0)
		instance.Properties[meputil.EndPointPropPrefix+"/abc/end1"] = "abc1"
		instance.Properties[meputil.EndPointPropPrefix+"/abc/end2"] = "abc2"

		service.TransportInfo.Endpoint.Addresses = make([]EndPointInfoAddress, 0)
		service.TransportInfo.Endpoint.Addresses = append(service.TransportInfo.Endpoint.Addresses,
			EndPointInfoAddress{Host: "1.1.1.1", Port: 3030})
		service.TransportInfo.Endpoint.Addresses = append(service.TransportInfo.Endpoint.Addresses,
			EndPointInfoAddress{Host: "1.1.1.2", Port: 3031})
		eps, epType := service.registerEndpoints(true, instance)

		assert.Equal(t, 2, len(eps))
		assert.Equal(t, meputil.Uris, epType)
	})

}
