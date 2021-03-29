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

// Package path implements dns client
package backend

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/server/core/backend"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/registry"

	meputil "mepserver/common/util"
)

// Read a single record from the data store on given path
func GetRecord(path string) (record []byte, errorCode int) {
	log.Debugf("DB: Read request: %v", path)
	opts := []registry.PluginOp{
		registry.OpGet(registry.WithStrKey(path), registry.WithPrefix()),
	}
	resp, err := backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if err != nil {
		log.Errorf(nil, "get single entry from data-store failed")
		return nil, meputil.OperateDataWithEtcdErr
	}
	if len(resp.Kvs) == 0 {
		log.Errorf(nil, "record does not exists on given path")
		return nil, meputil.SubscriptionNotFound
	}
	return resp.Kvs[0].Value, 0
}

// Read multiple records on the given path
func GetRecords(path string) (records map[string][]byte, errorCode int) {
	log.Debugf("DB: Read requests: %v", path)
	opts := []registry.PluginOp{
		registry.OpGet(registry.WithStrKey(path), registry.WithPrefix()),
	}
	resp, err := backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if err != nil {
		log.Errorf(nil, "get entries from data-store failed")
		return nil, meputil.OperateDataWithEtcdErr
	}
	resultList := make(map[string][]byte)
	for _, kvs := range resp.Kvs {
		resultList[filepath.Base(string(kvs.Key))] = kvs.Value
	}
	return resultList, 0
}

// Read multiple records on the given path
func GetRecordsWithCompleteKeyPath(path string) (records map[string][]byte, errorCode int) {
	log.Debugf("DB: Read requests: %v", path)
	opts := []registry.PluginOp{
		registry.OpGet(registry.WithStrKey(path), registry.WithPrefix()),
	}
	resp, err := backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if err != nil {
		log.Errorf(nil, "get entries with path from data-store failed")
		return nil, meputil.OperateDataWithEtcdErr
	}
	resultList := make(map[string][]byte)
	for _, kvs := range resp.Kvs {
		resultList[string(kvs.Key)] = kvs.Value
	}
	return resultList, 0
}

// Write new record to the given path
func PutRecord(path string, value []byte) int {
	log.Debugf("DB: Write request: %v", path)
	opts := []registry.PluginOp{
		registry.OpPut(registry.WithStrKey(path), registry.WithValue(value)),
	}
	_, err := backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if err != nil {
		log.Errorf(nil, "write to data-store failed")
		return meputil.OperateDataWithEtcdErr
	}
	return 0
}

// Delete a record on the given path
func DeleteRecord(path string) int {
	log.Debugf("DB: Delete request: %v", path)
	opts := []registry.PluginOp{
		registry.OpDel(registry.WithStrKey(path), registry.WithPrefix()),
	}
	_, err := backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if err != nil {
		log.Errorf(nil, "delete entries from data-store failed")
		return meputil.OperateDataWithEtcdErr
	}
	return 0
}

// Delete the db entries one by one as per the input paths
func DeletePaths(paths []string, continueOnFailure bool) int {
	for _, pathEntry := range paths {
		errCode := DeleteRecord(pathEntry)
		if errCode != 0 {
			log.Errorf(nil, "cache(path: %s) delete from etcd failed, "+
				"this might lead to data inconsistency!", strings.TrimPrefix(pathEntry, meputil.DBRootPath))
			if continueOnFailure {
				continue
			}
			return errCode
		}
	}
	return 0
}
