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

// db controller
package controllers

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"mepauth/models"
	"mepauth/util"

	"github.com/astaxie/beego/orm"

	log "github.com/sirupsen/logrus"
)

const dataFile string = "/usr/mep/mepauthdata.json"

// Insert or update data into mepauth data file
func InsertOrUpdateDataToFile(data *models.AuthInfoRecord) error {

	dataBytes, errMarshal := json.Marshal(data)
	if errMarshal != nil {
		log.Error("data marshal error")
		return errors.New("data marshal error")
	}
	err := ioutil.WriteFile(dataFile, dataBytes, util.KeyFileMode)
	if err != nil {
		return err
	}
	return nil
}

// Read data from mepauth data file
func ReadDataFromFile(ak string) (models.AuthInfoRecord, error) {
	data, errRead := ioutil.ReadFile(dataFile)
	if errRead != nil {
		log.Error("read data file error")
		return models.AuthInfoRecord{}, errRead
	}
	var res models.AuthInfoRecord
	errUnmarshal := json.Unmarshal(data, &res)
	if errUnmarshal != nil {
		log.Error("unmarshal data file error")
		return models.AuthInfoRecord{}, errUnmarshal
	}
	if res.Ak != ak {
		log.Error("the ak is not same as the one in the file")
		return models.AuthInfoRecord{}, errors.New("the ak is not same as the one in the file")
	}
	return res, nil
}

func InsertData(data interface{}) error {
	o := orm.NewOrm()
	o.Using("default")
	_, err := o.Insert(data)
	return err
}

func InsertOrUpdateData(data interface{}, cols ...string) error {
	o := orm.NewOrm()
	o.Using("default")
	_, err := o.InsertOrUpdate(data, cols...)
	return err
}

func DeleteData(data interface{}, cols ...string) error {
	o := orm.NewOrm()
	o.Using("default")
	_, err := o.Delete(data, cols...)
	return err
}

func ReadData(data interface{}, cols ...string) error {
	o := orm.NewOrm()
	o.Using("default")
	err := o.Read(data, cols...)
	return err
}
