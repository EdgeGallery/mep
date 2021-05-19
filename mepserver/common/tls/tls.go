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

//Package tls is a tls plugin for service center
package tls

import (
	"github.com/apache/servicecomb-service-center/pkg/log"
	svcutil "github.com/apache/servicecomb-service-center/pkg/util"
	mgr "github.com/apache/servicecomb-service-center/server/plugin"

	"mepserver/common/util"
)

func init() {
	mgr.RegisterPlugin(mgr.Plugin{PName: mgr.CIPHER, Name: "mepserver_tls", New: New})

}

// New plugin instance
func New() mgr.PluginInstance {
	return &MepServerTLS{}
}

type MepServerTLS struct {
}

// Encrypt data
func (c *MepServerTLS) Encrypt(src string) (string, error) {
	df, ok := mgr.DynamicPluginFunc(mgr.CIPHER, "Encrypt").(func(src string) (string, error))
	if ok {
		return df(src)
	}
	return src, nil
}

// Decrypt data
func (c *MepServerTLS) Decrypt(src string) (string, error) {

	decrypt := src
	certPwd, err := util.GetCertPwd()
	if err != nil {
		log.Errorf(err, "Get cert pwd failed.")
		return decrypt, err
	}
	decrypt = svcutil.BytesToStringWithNoCopy(certPwd)
	return decrypt, err
}
