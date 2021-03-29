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

package config

import (
	"github.com/agiledragon/gomonkey"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

const panicFormatString = "Panic: %v"
const responseNilError = "Error must be nil"

func TestBasicConfig(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	patch1 := gomonkey.ApplyFunc(ioutil.ReadFile, func(filename string) ([]byte, error) {
		mepConfigYaml := `
# dns agent configuration
dnsAgent:
  # values: local, dataplane, all
  type: all
  # local dns server end point
  endPoint:
    address:
      host: localhost
      port: 80


# data plane option to use in Mp2 interface
dataplane:
  # values: none
  type: none
`
		return []byte(mepConfigYaml), nil
	})
	defer patch1.Reset()

	config, err := LoadMepServerConfig()
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}
	assert.Equal(t, "all", config.DNSAgent.Type, responseNilError)
	assert.Equal(t, "none", config.DataPlane.Type, responseNilError)
}

func TestDnsAgentWrongTypeConfig1(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	patch1 := gomonkey.ApplyFunc(ioutil.ReadFile, func(filename string) ([]byte, error) {
		mepConfigYaml := `
# dns agent configuration
dnsAgent:
  # values: local, dataplane, all
  type: non-exist
  # local dns server end point
  endPoint:
    address:
      host: localhost
      port: 80


# data plane option to use in Mp2 interface
dataplane:
  # values: none
  type: none
`
		return []byte(mepConfigYaml), nil
	})
	defer patch1.Reset()

	config, err := LoadMepServerConfig()
	assert.EqualError(t, err, "Key: 'MepServerConfig.DNSAgent.Type' Error:Field validation for 'Type' failed on the 'oneof' tag", responseNilError)
	assert.Equal(t, (*MepServerConfig)(nil), config)
}

func TestDnsAgentTypeConfig2(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	patch1 := gomonkey.ApplyFunc(ioutil.ReadFile, func(filename string) ([]byte, error) {
		mepConfigYaml := `
# dns agent configuration
dnsAgent:
  # values: local, dataplane, all
  type: local
  # local dns server end point
  endPoint:
    address:
      host: localhost
      port: 80


# data plane option to use in Mp2 interface
dataplane:
  # values: none
  type: none
`
		return []byte(mepConfigYaml), nil
	})
	defer patch1.Reset()

	config, err := LoadMepServerConfig()
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}
	assert.Equal(t, "local", config.DNSAgent.Type, responseNilError)
	assert.Equal(t, "none", config.DataPlane.Type, responseNilError)
}

func TestDnsAgentTypeConfig3(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	patch1 := gomonkey.ApplyFunc(ioutil.ReadFile, func(filename string) ([]byte, error) {
		mepConfigYaml := `
# dns agent configuration
dnsAgent:
  # values: local, dataplane, all
  type: dataplane
  # local dns server end point
  endPoint:
    address:
      host: localhost
      port: 80


# data plane option to use in Mp2 interface
dataplane:
  # values: none
  type: none
`
		return []byte(mepConfigYaml), nil
	})
	defer patch1.Reset()

	config, err := LoadMepServerConfig()
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}
	assert.Equal(t, "dataplane", config.DNSAgent.Type, responseNilError)
	assert.Equal(t, "none", config.DataPlane.Type, responseNilError)
}

func TestDnsAgentWrongTypeConfig4(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(panicFormatString, r)
		}
	}()

	patch1 := gomonkey.ApplyFunc(ioutil.ReadFile, func(filename string) ([]byte, error) {
		mepConfigYaml := `
# dns agent configuration
dnsAgent:
  # values: local, dataplane, all
  type: 
  # local dns server end point
  endPoint:
    address:
      host: localhost
      port: 80


# data plane option to use in Mp2 interface
dataplane:
  # values: none
  type: none
`
		return []byte(mepConfigYaml), nil
	})
	defer patch1.Reset()

	config, err := LoadMepServerConfig()
	assert.EqualError(t, err, "Key: 'MepServerConfig.DNSAgent.Type' Error:Field validation for 'Type' failed on the 'oneof' tag", responseNilError)
	assert.Equal(t, (*MepServerConfig)(nil), config)
}
