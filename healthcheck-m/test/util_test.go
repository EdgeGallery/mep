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
	"github.com/stretchr/testify/assert"
	"healthcheck-m/util"
	"testing"
)

func TestValidateSrcAddressNull(t *testing.T) {
	err := util.ValidateSrcAddress("")
	assert.Error(t, err, "TestValidateSrcAddressNull execution result")
}

func TestValidateSrcAddressIPv4Success(t *testing.T) {
	err := util.ValidateSrcAddress("127.0.0.1")
	assert.Nil(t, err, "TestValidateSrcAddressIPv4Success execution result")
}

func TestValidateSrcAddressIPv6Success(t *testing.T) {
	err := util.ValidateSrcAddress("1:1:1:1:1:1:1:1")
	assert.Nil(t, err, "TestValidateSrcAddressIPv6Success execution result")
}

func TestGetDbName(t *testing.T) {
	err := util.GetLocalIp()
	assert.Equal(t, "", err, "TestGetDbName execution result")
}

func TestGetPort(t *testing.T) {
	port := util.GetInventoryPort()
	assert.Equal(t, "30203", port, "TestGetDbName execution result")
}
