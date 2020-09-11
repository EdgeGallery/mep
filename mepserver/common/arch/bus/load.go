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

// Package path implements architecture data bus
package bus

import (
	"reflect"
	"strings"
)

func parseTags(tag reflect.StructTag) (string, string) {
	fieldName := tag.Get("json")
	if fieldName == "" {
		return "", ""
	}
	fieldNames := strings.Split(fieldName, ",")
	secTag := DataIn
	if len(fieldNames) > 1 {
		secTag = fieldNames[1]
	}
	return fieldNames[0], secTag
}

// pass data according to direction tag
func LoadObjByInd(dst interface{}, src interface{}, direction string) bool {
	rflDst := reflect.ValueOf(dst).Elem()
	rflSrc := reflect.ValueOf(src).Elem()
	vType := rflDst.Type()
	for i := 0; i < rflDst.NumField(); i++ {
		fieldName, secTag := parseTags(vType.Field(i).Tag)
		if fieldName == "" {
			continue
		}
		if secTag != direction {
			continue
		}

		valDst := rflDst.Field(i)
		srcNode := objReflectPath(rflSrc, rflSrc, fieldName)
		if srcNode.e != nil || srcNode.CurNode.Kind() == reflect.Invalid {
			continue
		}

		valSrc := srcNode.CurNode
		if direction == DataIn {
			valDst.Set(valSrc)
		} else {
			valSrc.Set(valDst)
		}
	}
	return true
}
