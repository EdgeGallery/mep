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

// Package Data Store
package datastore

import "github.com/miekg/dns"

type ResourceRecord struct {
	Name  string   `json:"name"`
	Type  string   `json:"type"`
	Class string   `json:"class"`
	TTL   uint32   `json:"ttl"`
	RData []string `json:"rData"`
}

type ZoneEntry struct {
	Zone string            `json:"zone"`
	RR   *[]ResourceRecord `json:"rr"`
}

type DataStore interface {
	// Open - Initialize the DB by creating the database
	Open() error

	// Close - Cleanup the db
	Close() error

	// SetResourceRecord - Add or modify a A type record
	SetResourceRecord(zone string, rr *ResourceRecord) error

	// GetResourceRecord - Get A type record
	GetResourceRecord(question *dns.Question) (*[]dns.RR, error)

	// DelResourceRecord - Delete A type record
	DelResourceRecord(zone string, host string, rrtype string) error
	// IsResourceRecordExists - check the record exists
	IsResourceRecordExists(zone string, rr *ResourceRecord) bool
}
