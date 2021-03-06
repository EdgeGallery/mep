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

// Package dbAdapter contains database interface and implements database adapter
package adapter

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"mepauth/util"
	"os"
)

// Db database
var Db Database

func getDbAdapter() (Database, error) {
	dbAdapter := util.GetAppConfig("dbAdapter")
	switch dbAdapter {
	case "pgDb":
		db := &PgDb{}
		err := db.InitDatabase()
		if err != nil {
			return nil, errors.New("failed to register database")
		}
		return db, nil
	default:
		return nil, errors.New("no database is found")
	}
}

// InitDb initializes database
func InitDb() () {
	db, err := getDbAdapter()
	if err != nil {
		log.Error("Unable to get DB adapter: " + err.Error())
		os.Exit(1)
	}
	Db = db
}
