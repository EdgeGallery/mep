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

// Package path implements architecture work space
package workspace

import (
	"sync"
)

type PlanIf interface {
	SetErrorStep(stepName string)
	UsingSpace(spaceName string)
	RunBackground(task ...interface{})
	RunParallel(task ...interface{})
	RunSerial(task ...interface{})
}

type SpaceIf interface {
	PlanIf
	getPlan() *PlanBase
}

type ErrCode int

const (
	TaskOK ErrCode = iota
	TaskFail
)

const FINALLY string = "finally"

type SubGrp struct {
	Policy     GoPolicy
	CurStepIdx int
	StepNames  []string
	StepObjs   []interface{}
}

func (s *SubGrp) install(task []interface{}) bool {
	for _, stepObj := range task {
		if stepObj == nil {
			return false
		}
		s.StepObjs = append(s.StepObjs, stepObj)
	}
	return true
}

type SerErrInfo struct {
	ErrCode int
	Message string
	GrpIdx  int
	TaskIdx int
}

type PlanBase struct {
	SerError  *SerErrInfo
	PlanName  string
	SpaceName string
	ErrStep   string
	PlanGrp   []SubGrp
	CurGrpIdx int
	WtPlan    sync.WaitGroup
}

// set error step
func (b *PlanBase) SetErrorStep(stepName string) {
	b.ErrStep = stepName
}

// set space name
func (b *PlanBase) UsingSpace(spaceName string) {
	b.SpaceName = spaceName
}

// run task in background
func (b *PlanBase) RunBackground(task ...interface{}) {
	b.loadTask(GoBackground, task)
}

// run task in parallel
func (b *PlanBase) RunParallel(task ...interface{}) {
	b.loadTask(GoParallel, task)
}

// run task in serial
func (b *PlanBase) RunSerial(task ...interface{}) {
	b.loadTask(GoSerial, task)
}

// run task in serial with name
func (b *PlanBase) RunSerialName(task interface{}, name string) {
	var subGrp SubGrp
	subGrp.Policy = GoSerial
	subGrp.install([]interface{}{task})
	b.loadData([]interface{}{task})
	subGrp.StepNames = []string{name}
	b.PlanGrp = append(b.PlanGrp, subGrp)
}

// run a task group
func (b *PlanBase) Try(task ...interface{}) {
	b.SetErrorStep(FINALLY)
	b.loadTask(GoSerial, task)
}

// finally handle exception
func (b *PlanBase) Finally(task interface{}) {
	var subGrp SubGrp
	subGrp.Policy = GoSerial
	subGrp.install([]interface{}{task})
	b.loadData([]interface{}{task})
	subGrp.StepNames = []string{FINALLY}
	b.PlanGrp = append(b.PlanGrp, subGrp)
}

func (b *PlanBase) loadTask(policy GoPolicy, task []interface{}) {
	var subGrp SubGrp
	subGrp.Policy = policy
	subGrp.install(task)
	b.loadData(task)
	b.PlanGrp = append(b.PlanGrp, subGrp)
}

func (b *PlanBase) loadData(task []interface{}) bool {
	for _, stepObj := range task {
		if stepObj == nil {
			return false
		}
		stepIf, ok := stepObj.(TaskBaseIf)
		if !ok {
			continue
		}
		stepIf.SetSerErrInfo(b.SerError)
	}
	return true
}

type SpaceBase struct {
	PlanBase
}

// init space base
func (s *SpaceBase) Init() {
	s.SerError = &SerErrInfo{}
}

func (s *SpaceBase) getPlan() *PlanBase {
	return &s.PlanBase
}

// go to error handling step
func GotoErrorStep(curPlan *PlanBase, grpNum int) bool {
	if grpNum >= len(curPlan.PlanGrp) {
		return false
	}
	for idx, stepName := range curPlan.PlanGrp[grpNum].StepNames {
		if stepName == curPlan.ErrStep {
			if curPlan.CurGrpIdx > grpNum {
				return true
			}
			curPlan.CurGrpIdx = grpNum
			if curPlan.PlanGrp[grpNum].CurStepIdx < idx {
				curPlan.PlanGrp[grpNum].CurStepIdx = idx
			}
			return true
		}
	}
	return false
}

// record error info
func RecordErrInfo(curPlan *PlanBase, stepIdx int) {
	if curPlan.CurGrpIdx >= len(curPlan.PlanGrp) {
		return
	}
	curGrp := curPlan.PlanGrp[curPlan.CurGrpIdx]
	if stepIdx < 0 || stepIdx >= len(curGrp.StepObjs) {
		return
	}

	curPlan.SerError.GrpIdx = curPlan.CurGrpIdx
	curPlan.SerError.TaskIdx = stepIdx
	curStep := curGrp.StepObjs[stepIdx]
	stepIf, ok := curStep.(TaskBaseIf)
	if !ok {
		return
	}
	errCode, msg := stepIf.GetErrCode()
	curPlan.SerError.ErrCode = int(errCode)
	curPlan.SerError.Message = msg
}

// go to error handling process
func GotoErrorProc(curPlan *PlanBase) {
	curPlan.CurGrpIdx++
	for grpNum := 0; grpNum < len(curPlan.PlanGrp); grpNum++ {
		done := GotoErrorStep(curPlan, grpNum)
		if done {
			break
		}
	}
}
