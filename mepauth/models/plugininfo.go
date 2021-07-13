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

// Package model contains mep auth data model
package models

// LogPluginInfo http log plugin information
type LogPluginInfo struct {
	Name   string     `json:"name"`
	Config ConfigInfo `json:"config"`
}

// ConfigInfo http log plugin configurations
type ConfigInfo struct {
	HTTPEndpoint string `json:"http_endpoint"`
	Method       string `json:"method"`
	Timeout      int    `json:"timeout"`
	Keepalive    int    `json:"keepalive"`
}
