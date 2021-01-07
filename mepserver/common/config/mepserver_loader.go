/*
 * Copyright 2020 Huawei Technologies Co., Ltd.
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
	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/ghodss/yaml"
	"github.com/go-playground/validator/v10"
	"io/ioutil"
	"mepserver/common/util"
	"path/filepath"
)

type MepServerConfig struct {
	DNSAgent  DNSAgent  `yaml:"dnsAgent"`
	DataPlane DataPlane `yaml:"dataplane"`
}
type Address struct {
	Host string `yaml:"host" validate:"omitempty,min=1,max=253"`
	Port int    `yaml:"port" validate:"omitempty,min=1,max=65535"`
}
type EndPoint struct {
	Address Address `yaml:"address"`
}
type DNSAgent struct {
	Type     string   `yaml:"type" validate:"oneof=local dataplane all"`
	Endpoint EndPoint `yaml:"endPoint" validate:"required_unless=type dataplane"`
}

type DataPlane struct {
	Type string `yaml:"type" validate:"oneof=none"`
}

// Read and load the mep server configurations
func LoadMepServerConfig() (*MepServerConfig, error) {
	configFilePath := filepath.FromSlash(util.MepServerConfigPath)
	configData, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		log.Error("Reading configuration file error", nil)
		return nil, err
	}
	var mepConfig MepServerConfig
	err = yaml.Unmarshal(configData, &mepConfig)
	if err != nil {
		log.Error("Parsing configuration file error", nil)
		return nil, err
	}
	err = validateConfig(&mepConfig)
	if err != nil {
		log.Error("Config validation failed.", err)
		return nil, err
	}
	return &mepConfig, nil
}

func validateConfig(config *MepServerConfig) error {
	validate := validator.New()
	err := validate.Struct(config)
	if err != nil {
		return err
	}
	return nil
}
