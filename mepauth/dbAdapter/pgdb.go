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

// db controller
package dbAdapter

import (
	"errors"
	"fmt"
	"mepauth/util"
	"unsafe"

	"github.com/astaxie/beego/orm"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

const DEFAULT string = "default"
const DRIVER string = "postgres"

// Pg database
type PgDb struct {
	ormer orm.Ormer
}

// Constructor of ORM
func (db *PgDb) InitOrmer() (err1 error) {
	defer func() {
		if err := recover(); err != nil {
			log.Error("panic handled:", err)
			err1 = fmt.Errorf("recover panic as %s", err)
		}
	}()
	o := orm.NewOrm()
	err1 = o.Using(DEFAULT)
	if err1 != nil {
		return err1
	}
	db.ormer = o

	return nil
}

// Insert data into pg database
func (db *PgDb) InsertData(data interface{}) (err error) {
	_, err = db.ormer.Insert(data)
	return err
}

// Insert or update data into pg database
func (db *PgDb) InsertOrUpdateData(data interface{}, cols ...string) (err error) {
	_, err = db.ormer.InsertOrUpdate(data, cols...)
	return err
}

// Read data from pg database
func (db *PgDb) ReadData(data interface{}, cols ...string) (err error) {
	err = db.ormer.Read(data, cols...)
	return err
}

// Delete data from pg database
func (db *PgDb) DeleteData(data interface{}, cols ...string) (err error) {
	_, err = db.ormer.Delete(data, cols...)
	return err
}

// Init database
func (db *PgDb) InitDatabase() error {

	// Validate password
	dbPwd := []byte(util.GetAppConfig("db_passwd"))
	dbPwdStr := string(dbPwd)
	util.ClearByteArray(dbPwd)
	dbParamsAreValid, validateDbParamsErr := util.ValidateDbParams(dbPwdStr)
	if validateDbParamsErr != nil || !dbParamsAreValid {
		log.Error("Password validation failed")
		return errors.New("failed to validate db parameters")
	}

	registerDriverErr := orm.RegisterDriver(DRIVER, orm.DRPostgres)
	if registerDriverErr != nil {
		log.Error("Failed to register driver")
		return registerDriverErr
	}

	dataSource := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s",
		util.GetAppConfig("db_user"),
		dbPwdStr,
		util.GetAppConfig("db_name"),
		util.GetAppConfig("db_host"),
		util.GetAppConfig("db_port"),
		util.GetAppConfig("db_sslmode"))

	registerDataBaseErr := orm.RegisterDataBase(DEFAULT, DRIVER, dataSource)
	//clear bStr
	bKey1 := *(*[]byte)(unsafe.Pointer(&dataSource))
	util.ClearByteArray(bKey1)

	bKey := *(*[]byte)(unsafe.Pointer(&dbPwdStr))
	util.ClearByteArray(bKey)

	if registerDataBaseErr != nil {
		log.Error("Failed to register database")
		return registerDataBaseErr
	}

	errRunSyncdb := orm.RunSyncdb(DEFAULT, false, true)
	if errRunSyncdb != nil {
		log.Error("Failed to sync database.")
		return errRunSyncdb
	}

	err := db.InitOrmer()
	if err != nil {
		log.Error("Failed to init ormer")
		return err
	}

	return nil
}
