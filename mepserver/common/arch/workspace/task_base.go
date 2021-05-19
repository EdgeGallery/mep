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

// Package workspace implements architecture work space
package workspace

type TaskBaseIf interface {
	OnRequest(data string) TaskCode
	Parse(params string)
	OnFork(wkSpace interface{}, param interface{}) int
	GetErrCode() (ErrCode, string)
	OnStop() int
	WithName(name string)
	SetSerErrInfo(serErr *SerErrInfo)
}

type TaskBase struct {
	serErr     *SerErrInfo
	errMsg     string
	Name       string
	Param      []string
	resultCode ErrCode
}

// WithName set task base name
func (t *TaskBase) WithName(name string) {
	t.Name = name
}

// Parse task base parse params
func (t *TaskBase) Parse(params string) {
	t.Param = append(t.Param, params)
}

// OnFork task base on fork
func (t *TaskBase) OnFork(wkSpace interface{}, param interface{}) int {
	return 0
}

// OnStop task base on stop
func (t *TaskBase) OnStop() int {
	return 0
}

type TaskCode int

const (
	TaskFinish TaskCode = iota
)

// OnRequest task base on request
func (t *TaskBase) OnRequest(wkSpace interface{}) TaskCode {
	return TaskFinish
}

// SetFirstErrorCode set task base error code
func (t *TaskBase) SetFirstErrorCode(code ErrCode, msg string) {
	if t.resultCode > TaskOK {
		return
	}
	t.resultCode = code
	t.errMsg = msg
}

// GetErrCode get error code
func (t *TaskBase) GetErrCode() (ErrCode, string) {
	return t.resultCode, t.errMsg
}

// SetSerErrInfo set error info
func (t *TaskBase) SetSerErrInfo(serErr *SerErrInfo) {
	t.serErr = serErr
}

// GetSerErrInfo get error info
func (t *TaskBase) GetSerErrInfo() *SerErrInfo {
	return t.serErr
}
