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
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const DataIn string = "in"
const FormatIntBase int = 10

type JSONPathInfo struct {
	ParentNode reflect.Value
	CurNode    reflect.Value
	CurName    string
	e          error
}

type JpErr struct {
	ErrDes string
	JPath  string
}

// json path error
func (e *JpErr) Error() string {
	return fmt.Sprintf("jpath error info:%s, json path:%s", e.ErrDes, e.JPath)
}

func objReflectPath(p reflect.Value, v reflect.Value, path string) JSONPathInfo {
	fieldName, subPath := getFirstName(path)

	switch v.Kind() {
	case reflect.Invalid:
		return JSONPathInfo{e: &JpErr{"reflect.Invalid", path}}

	case reflect.Slice, reflect.Array:
		return objReflectPathArray(v, fieldName, subPath)
	case reflect.Struct:
		return objReflectPathStruct(v, fieldName, subPath)

	case reflect.Map:
		return objReflectPathMap(v, fieldName, subPath)

	case reflect.Ptr:
		if v.IsNil() {
			return JSONPathInfo{e: &JpErr{"pointer is null", path}}
		}
		return objReflectPath(p, v.Elem(), path)
	case reflect.Interface:
		if v.IsNil() {
			return JSONPathInfo{e: &JpErr{"kind is interface, nil", path}}
		}

		if subPath == "" {
			return reflectSafeAddr(v, v.Elem())
		}
		return objReflectPath(p, v.Elem(), path)

	default:
		return getFieldFromPath(p, v, path)
	}
}

func objReflectPathArray(v reflect.Value, fieldName string, subPath string) JSONPathInfo {
	if fieldName == "-" {
		mapInfo := reflectSafeAddr(v, reflect.ValueOf(nil))
		mapInfo.CurName = "-"
		return mapInfo
	}

	idx, err := strconv.Atoi(fieldName)
	if err != nil {
		return JSONPathInfo{e: &JpErr{"Atoi error", fieldName}}
	}
	if idx >= v.Len() {
		return JSONPathInfo{e: &JpErr{"Index out of range", fieldName}}
	}
	if subPath == "" {
		return reflectSafeAddr(v, v.Index(idx))
	}
	return objReflectPath(v, v.Index(idx), subPath)
}

func objReflectPathStruct(v reflect.Value, fieldName string, subPath string) JSONPathInfo {

	vType := v.Type()
	for i := 0; i < v.NumField(); i++ {
		if !matchJSONFieldName(vType, i, fieldName) {
			continue
		}
		if subPath == "" {
			return reflectSafeAddr(v, v.Field(i))
		}
		return objReflectPath(v, v.Field(i), subPath)
	}
	return JSONPathInfo{e: &JpErr{"can not find field in struct", fieldName}}

}

func objReflectPathMap(v reflect.Value, fieldName string, subPath string) JSONPathInfo {
	for _, key := range v.MapKeys() {
		if reflectValueToString(key) != fieldName {
			continue
		}

		if subPath == "" {
			mapInfo := reflectSafeAddr(v, reflect.ValueOf(nil))
			mapInfo.CurName = fieldName
			return mapInfo
		}
		return objReflectPath(v, v.MapIndex(key), subPath)

	}
	if subPath == "" {
		mapInfo := reflectSafeAddr(v, reflect.ValueOf(nil))
		mapInfo.CurName = fieldName
		return mapInfo
	}

	return JSONPathInfo{e: &JpErr{"path not in map:" + fieldName, subPath}}
}

func getFirstName(path string) (string, string) {
	if len(path) == 0 {
		return "", ""
	}
	newPath := path
	if path[0] == '/' {
		newPath = path[1:]
	}
	pos := strings.IndexByte(newPath, '/')
	if pos < 0 {
		pos = len(newPath)
	}
	subPath := newPath[pos:]
	firstName := newPath[0:pos]
	escape := strings.IndexByte(firstName, '~')
	if escape >= 0 {
		firstName = strings.Replace(firstName, "~1", "/", -1)
		firstName = strings.Replace(firstName, "~0", "~", -1)
	}
	return firstName, subPath
}

func matchJSONFieldName(vType reflect.Type, i int, jsonName string) bool {
	tag := vType.Field(i).Tag
	if !strings.Contains(string(tag), jsonName) {
		return false
	}
	name := tag.Get("json")
	if name == "" {
		name = strings.ToLower(vType.Field(i).Name)
	} else {
		pos := strings.IndexByte(name, ',')
		if pos > 0 {
			name = name[0:pos]
		}
	}
	if name == jsonName {
		return true
	}

	return false
}

func getFieldFromPath(p reflect.Value, v reflect.Value, path string) JSONPathInfo {
	var info JSONPathInfo
	if !v.CanAddr() {
		return JSONPathInfo{e: &JpErr{"CanAddr false", path}}
	}

	switch v.Kind() {
	case reflect.Invalid:
		return JSONPathInfo{e: &JpErr{"Kind invalid", path}}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		info.ParentNode = p
		info.CurNode = v
		return info

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		info.ParentNode = p
		info.CurNode = v
		return info

	case reflect.Bool:
		info.ParentNode = p
		info.CurNode = v
		return info
	case reflect.String:
		info.ParentNode = p
		info.CurNode = v
		return info
	case reflect.Slice, reflect.Map:
		info.ParentNode = p
		info.CurNode = v
		return info
	case reflect.Ptr:
		info.ParentNode = p
		info.CurNode = v
		return info
	case reflect.Chan, reflect.Func:
		return JSONPathInfo{e: &JpErr{"Kind Chan or Func", path}}
	default:
		return JSONPathInfo{e: &JpErr{"unexpect Kind: reflect.Array, reflect.Struct, reflect.Interface", path}}
	}
}

func reflectValueToString(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Invalid:
		return "invalid"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), FormatIntBase)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), FormatIntBase)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.String:
		return v.String()
	case reflect.Chan, reflect.Func, reflect.Ptr, reflect.Slice, reflect.Map:
		return "invalid"
	default:
		return "invalid"
	}
}

func reflectSafeAddr(p reflect.Value, v reflect.Value) JSONPathInfo {
	var info JSONPathInfo

	if p.CanAddr() {
		info.ParentNode = p
	}

	if v.CanAddr() {
		info.CurNode = v
	}
	return info
}
