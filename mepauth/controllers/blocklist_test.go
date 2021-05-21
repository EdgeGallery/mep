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

package controllers

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestInitAuthInfoList(t *testing.T) {
	convey.Convey("Init AuthInfo List", t, func() {
		convey.Convey("for success", func() {
			InitAuthInfoList()
		})
	})
}

func TestIsAkInBlockList(t *testing.T) {
	convey.Convey("isAkInBlockList", t, func() {
		convey.Convey("for success", func() {
			InitAuthInfoList()
			startValidatingAk("ak")
			startBlockListingAk("ak")
			res := isAkInBlockList("ak")
			convey.So(res, convey.ShouldBeTrue)
		})

		convey.Convey("for fail state", func() {
			InitAuthInfoList()
			startValidatingAk("ak")
			res := isAkInBlockList("ak")
			convey.So(res, convey.ShouldBeFalse)
		})

		convey.Convey("for fail", func() {
			res := isAkInBlockList("ak2")
			convey.So(res, convey.ShouldBeFalse)
		})
	})
}

func TestIsAkInValidationList(t *testing.T) {
	convey.Convey("isAkInValidationList", t, func() {
		convey.Convey("for success", func() {
			InitAuthInfoList()
			startValidatingAk("ak")
			res := isAkInValidationList("ak")
			convey.So(res, convey.ShouldBeTrue)
		})
		convey.Convey("for fail", func() {
			InitAuthInfoList()
			res := isAkInValidationList("ak")
			convey.So(res, convey.ShouldBeFalse)
		})
	})
}

func TestStopValidatingAk(t *testing.T) {
	convey.Convey("stopValidatingAk", t, func() {
		convey.Convey("for success", func() {
			InitAuthInfoList()
			startValidatingAk("ak")
			stopValidatingAk("ak")
		})
	})
}

func TestClearAkFromBlockListing(t *testing.T) {
	convey.Convey("clearAkFromBlockListing", t, func() {
		convey.Convey("for success", func() {
			InitAuthInfoList()
			startValidatingAk("ak")
			clearAkFromBlockListing("ak")
		})
	})
}

func TestProcessAkForBlockListing(t *testing.T) {
	convey.Convey("processAkForBlockListing", t, func() {
		convey.Convey("for success", func() {
			startValidatingAk("ak")
			processAkForBlockListing("ak")
		})
		convey.Convey("for fail", func() {
			processAkForBlockListing("ak")
		})
	})
}
