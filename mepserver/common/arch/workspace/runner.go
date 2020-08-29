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

	"mepserver/common/arch/bus"
)

type GoPolicy int

const (
	_ GoPolicy = iota
	GoParallel
	GoBackground
	GoSerial
)

const DataIn string = "in"
const DataOut string = "out"

// run plans in workspace
func WkRun(plan SpaceIf) ErrCode {
	curPlan := plan.getPlan()
	for {
		if curPlan.CurGrpIdx >= len(curPlan.PlanGrp) {
			break
		}
		curSub := &curPlan.PlanGrp[curPlan.CurGrpIdx]
		retCode, stepIdx := grpRun(curSub, plan, &curPlan.WtPlan)
		if retCode <= TaskOK {
			curPlan.CurGrpIdx++
			continue
		}
		RecordErrInfo(curPlan, stepIdx)
		GotoErrorProc(curPlan)
	}
	// wait background job finish
	curPlan.WtPlan.Wait()
	return TaskOK

}

// workspace task runner
func taskRunner(wkSpace interface{}, stepIf TaskBaseIf) int {
	for {
		bus.LoadObjByInd(stepIf, wkSpace, DataIn)
		retCode := stepIf.OnRequest("")
		if retCode <= TaskFinish {
			bus.LoadObjByInd(stepIf, wkSpace, DataOut)
			break
		}
	}
	return 0
}

// step run policy: background, parallel, serial
func stepPolicy(wg *sync.WaitGroup, curSub *SubGrp, plan SpaceIf, wtPlan *sync.WaitGroup, stepIf TaskBaseIf) ErrCode {
	taskRet := TaskOK
	switch curSub.Policy {
	case GoBackground:
		wtPlan.Add(1)
		go func() {
			taskRunner(plan, stepIf)
			wtPlan.Done()
		}()

	case GoParallel:
		wg.Add(1)
		go func() {
			taskRunner(plan, stepIf)
			wg.Done()
		}()
	default:
		taskRunner(plan, stepIf)
		taskRet, _ = stepIf.GetErrCode()
	}

	return taskRet
}

// run one step
func grpOneStep(wg *sync.WaitGroup, curSub *SubGrp, plan SpaceIf, wtPlan *sync.WaitGroup) bool {
	if curSub.CurStepIdx >= len(curSub.StepObjs) {
		return false
	}
	curStep := curSub.StepObjs[curSub.CurStepIdx]
	if curStep == nil {
		curSub.CurStepIdx++
		return true
	}
	stepIf, ok := curStep.(TaskBaseIf)
	if !ok {
		return false
	}
	taskRet := stepPolicy(wg, curSub, plan, wtPlan, stepIf)
	curSub.CurStepIdx++

	return taskRet <= TaskOK
}

func grpGetRetCode(curSub *SubGrp) (ErrCode, int) {
	for idx, curStep := range curSub.StepObjs {
		stepIf, ok := curStep.(TaskBaseIf)
		if !ok {
			continue
		}
		errCode, _ := stepIf.GetErrCode()
		if errCode > TaskOK {
			return errCode, idx
		}
	}

	return TaskOK, -1
}

func grpRun(curSub *SubGrp, plan SpaceIf, wtPlan *sync.WaitGroup) (ErrCode, int) {
	var wg sync.WaitGroup
	for {
		nextStep := grpOneStep(&wg, curSub, plan, wtPlan)
		if !nextStep {
			break
		}
	}
	if curSub.Policy == GoParallel {
		wg.Wait()
	}
	return grpGetRetCode(curSub)
}
