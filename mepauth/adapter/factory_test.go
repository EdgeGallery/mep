/*
 * Copyright 2020-2021 Huawei Technologies Co., Ltd.
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

package adapter

import (
	"context"
	"database/sql"
	"fmt"
	. "github.com/agiledragon/gomonkey"
	"github.com/astaxie/beego/orm"
	"mepauth/util"
	"os"
	"reflect"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestInitDatabase(t *testing.T) {
	Convey("init db", t, func() {
		Convey("success", func() {
			db := &PgDb{}
			patches := ApplyFunc(util.ValidateDbParams, func(dbPwd string) (bool, error) {
				return true, nil
			})
			defer patches.Reset()
			patches.ApplyFunc(orm.RegisterDataBase, func(aliasName, driverName, dataSource string, params ...int) error {
				return nil
			})
			patches.ApplyFunc(orm.RunSyncdb, func(name string, force bool, verbose bool) error {
				return nil
			})
			patches.ApplyMethod(reflect.TypeOf(db), "InitOrmer", func(*PgDb) error {
				return nil
			})
			err := db.InitDatabase()
			So(err, ShouldBeNil)
		})
		Convey("invalid password", func() {
			patches := ApplyFunc(util.ValidateDbParams, func(dbPwd string) (bool, error) {
				return false, nil
			})
			defer patches.Reset()
			db := &PgDb{}
			err := db.InitDatabase()
			So(err, ShouldNotBeNil)
		})
		Convey("register failure", func() {
			db := &PgDb{}
			patches := ApplyFunc(util.ValidateDbParams, func(dbPwd string) (bool, error) {
				return true, nil
			})
			defer patches.Reset()
			patches.ApplyFunc(orm.RegisterDriver, func(driverName string, typ orm.DriverType) error {
				return fmt.Errorf("register error")
			})
			err := db.InitDatabase()
			So(err, ShouldNotBeNil)
		})
		Convey("register db failure", func() {
			db := &PgDb{}
			patches := ApplyFunc(util.ValidateDbParams, func(dbPwd string) (bool, error) {
				return true, nil
			})
			defer patches.Reset()
			patches.ApplyFunc(orm.RegisterDataBase, func(aliasName, driverName, dataSource string, params ...int) error {
				return fmt.Errorf("register error")
			})
			err := db.InitDatabase()
			So(err, ShouldNotBeNil)
		})
		Convey("sync db failure", func() {
			db := &PgDb{}
			patches := ApplyFunc(util.ValidateDbParams, func(dbPwd string) (bool, error) {
				return true, nil
			})
			defer patches.Reset()
			patches.ApplyFunc(orm.RegisterDataBase, func(aliasName, driverName, dataSource string, params ...int) error {
				return nil
			})
			patches.ApplyFunc(orm.RunSyncdb, func(name string, force bool, verbose bool) error {
				return fmt.Errorf("register error")
			})
			err := db.InitDatabase()
			So(err, ShouldNotBeNil)
		})
		Convey("ormer failure", func() {
			db := &PgDb{}
			patches := ApplyFunc(util.ValidateDbParams, func(dbPwd string) (bool, error) {
				return true, nil
			})
			defer patches.Reset()
			patches.ApplyFunc(orm.RegisterDataBase, func(aliasName, driverName, dataSource string, params ...int) error {
				return nil
			})
			patches.ApplyFunc(orm.RunSyncdb, func(name string, force bool, verbose bool) error {
				return nil
			})
			patches.ApplyMethod(reflect.TypeOf(db), "InitOrmer", func(*PgDb) error {
				return fmt.Errorf("register error")
			})
			err := db.InitDatabase()
			So(err, ShouldNotBeNil)
		})
	})
}

func TestInitDB(t *testing.T) {
	Convey("init db", t, func() {
		Convey("failure", func() {
			patches := ApplyFunc(os.Exit, func(code int) {
				return
			})
			defer patches.Reset()
			InitDb()
			So(Db, ShouldBeNil)
		})

	})
}

type OrSample struct {
}

func (o *OrSample) Read(md interface{}, cols ...string) error {
	return nil
}

func (o *OrSample) ReadForUpdate(md interface{}, cols ...string) error {
	return nil
}

func (o *OrSample) ReadOrCreate(md interface{}, col1 string, cols ...string) (bool, int64, error) {
	return true, 0, nil
}

func (o *OrSample) Insert(i interface{}) (int64, error) {
	return 0, nil
}

func (o *OrSample) InsertOrUpdate(md interface{}, colConflitAndArgs ...string) (int64, error) {
	return 0, nil
}

func (o *OrSample) InsertMulti(bulk int, mds interface{}) (int64, error) {
	return 0, nil
}

func (o *OrSample) Update(md interface{}, cols ...string) (int64, error) {
	return 0, nil
}

func (o *OrSample) Delete(md interface{}, cols ...string) (int64, error) {
	return 0, nil
}

func (o *OrSample) LoadRelated(md interface{}, name string, args ...interface{}) (int64, error) {
	return 0, nil
}

func (o *OrSample) QueryM2M(md interface{}, name string) orm.QueryM2Mer {
	panic("implement me")
}

func (o *OrSample) QueryTable(ptrStructOrTableName interface{}) orm.QuerySeter {
	panic("implement me")
}

func (o *OrSample) Using(name string) error {
	return nil
}

func (o *OrSample) Begin() error {
	return nil
}

func (o *OrSample) BeginTx(ctx context.Context, opts *sql.TxOptions) error {
	return nil
}

func (o *OrSample) Commit() error {
	return nil
}

func (o *OrSample) Rollback() error {
	return nil
}

func (o *OrSample) Raw(query string, args ...interface{}) orm.RawSeter {
	panic("implement me")
}

func (o *OrSample) Driver() orm.Driver {
	panic("implement me")
}

func (o *OrSample) DBStats() *sql.DBStats {
	panic("implement me")
}

func TestInitOrmer(t *testing.T) {
	Convey("init ormer", t, func() {
		Convey("success", func() {
			//var o *orm.Ormer
			patches := ApplyFunc(orm.NewOrm, func() orm.Ormer {
				return &OrSample{}
			})
			defer patches.Reset()
			db := &PgDb{}
			So(db.InitOrmer(), ShouldBeNil)
		})
		Convey("panic", func() {
			type OrSam struct {
				orm.Ormer
			}
			//var o *orm.Ormer
			patches := ApplyFunc(orm.NewOrm, func() orm.Ormer {
				return OrSam{}
			})
			defer patches.Reset()
			db := &PgDb{}
			So(db.InitOrmer(), ShouldNotBeNil)
		})

	})
}

func TestInsertData(t *testing.T) {
	Convey("insert data", t, func() {
		Convey("success", func() {
			//var o *orm.Ormer
			patches := ApplyFunc(orm.NewOrm, func() orm.Ormer {
				return &OrSample{}
			})
			defer patches.Reset()
			db := &PgDb{}
			So(db.InitOrmer(), ShouldBeNil)
			So(db.InsertData(nil), ShouldBeNil)
			So(db.InsertOrUpdateData(nil), ShouldBeNil)
			So(db.ReadData(nil), ShouldBeNil)
			So(db.DeleteData(nil), ShouldBeNil)
		})
	})
}
