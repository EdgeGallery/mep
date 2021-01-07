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

// Filtering criteria to match services for which events are requested to be reported. If absent, matches all services. All child attributes are combined with the logical  \"AND\" operation.
type FilteringCriteria struct {
	SerInstanceIds []string      `json:"serInstanceIds" validate:"omitempty,min=0,dive,max=32,validateId"`
	SerNames       []string      `json:"serNames"  validate:"omitempty,min=0,dive,max=128,validateName"`
	SerCategories  []CategoryRef `json:"serCategories" validate:"omitempty,dive"`
	States         []string      `json:"states" validate:"omitempty,min=0,dive,oneof=ACTIVE INACTIVE"`
	IsLocal        bool          `json:"isLocal,omitempty"`
}
