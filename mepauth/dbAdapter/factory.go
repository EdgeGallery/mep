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
package dbAdapter

import (
	"errors"
	"mepauth/util"
)

// Init Db adapter
func GetDbAdapter() (Database, error) {
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
