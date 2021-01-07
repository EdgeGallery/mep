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

// Package path implements mep server object models
package models

// This type represents the general information of a MEC service.
type TransportInfo struct {
	ID          string         `json:"id" validate:"omitempty,max=64,validateId"`
	Name        string         `json:"name" validate:"omitempty,max=128,validateName"`
	Description string         `json:"description" validate:"omitempty,max=128"`
	TransType   TransportTypes `json:"type" validate:"omitempty,oneof=REST_HTTP MB_TOPIC_BASED MB_ROUTING MB_PUBSUB RPC RPC_STREAMING WEBSOCKET"`
	Protocol    string         `json:"protocol" validate:"omitempty,max=32,validateProtocol"`
	Version     string         `json:"version" validate:"omitempty,max=32,validateVersion"`
	// This type represents information about a transport endpoint
	Endpoint         EndPointInfo `json:"endpoint,omitempty"`
	Security         SecurityInfo `json:"security,omitempty"`
	ImplSpecificInfo interface{}  `json:"implSpecificInfo,omitempty"`
}

type TransportTypes string
