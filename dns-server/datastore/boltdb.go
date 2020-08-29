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
package datastore

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/miekg/dns"
	bolt "go.etcd.io/bbolt"
)

const (
	ZoneConfig  = "zone"
	DefaultZone = "."
	DBPath      = "data"
)

type DNSConfigRRKey struct {
	Host   string `json:"host"`
	RRType uint16 `json:"rrType"`
}

type DNSConfigRRValue struct {
	RRClass uint16   `json:"rrClass"`
	PointTo []string `json:"pointTo"`
	Ttl     uint32   `json:"ttl"`
}

var rrTypeMap = map[string]uint16{"A": dns.TypeA}
var rrClassMap = map[string]uint16{"IN": dns.ClassINET, "CS": dns.ClassCSNET, "CH": dns.ClassCHAOS,
	"HS": dns.ClassHESIOD, "*": dns.ClassANY}

type BoltDB struct {
	FileName string
	TTL      uint32
	db       *bolt.DB
}

func (b *BoltDB) Open() error {
	var err error

	_, err = os.Stat(DBPath)
	if os.IsNotExist(err) {
		err := os.Mkdir(DBPath, 0700)
		if err != nil {
			log.Fatal("Data path does not exists and could not create a new one", err)
		}
	}

	b.db, err = bolt.Open(path.Join(DBPath, b.FileName), 0600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return err
	}

	// Create the default zone if not exists one
	err = b.db.Update(func(tx *bolt.Tx) error {
		bZone, err := tx.CreateBucketIfNotExists([]byte(ZoneConfig))
		if err != nil {
			log.Error("Failed to create the zone bucket.", err)
			return fmt.Errorf("error creating zone bucket: %s", err)
		}
		_, err = bZone.CreateBucketIfNotExists([]byte(DefaultZone))
		if err != nil {
			log.Error("Failed to create the default(.) zone bucket.", err)
			return fmt.Errorf("error creating default zone(.) bucket: %s", err)
		}
		return nil
	})

	log.Debugf("Initialize bolt db(%s) success.", b.FileName)
	return err
}

func (b *BoltDB) Close() error {
	if b.db != nil {
		err := b.db.Close()
		if err != nil {
			log.Errorf(nil, "Failed to close the bolt db(%s).", b.FileName)
			return err
		}
	}
	log.Debugf("Closed bolt db(%s) as part of shutdown service.", b.FileName)
	return nil
}

func (b *BoltDB) SetResourceRecord(zone string, rr *ResourceRecord) error {
	rrType, ok := rrTypeMap[rr.Type]
	if !ok {
		return fmt.Errorf("unsupported rrtype(%s) entry", rr.Type)
	}
	rrClass, ok := rrClassMap[rr.Class]
	if !ok {
		return fmt.Errorf("unsupported rrclass(%s) entry", rr.Class)
	}
	if rrClass == dns.ClassANY {
		return fmt.Errorf("unsupported rrclass(%s) entry", rr.Class)
	}

	host := strings.ToLower(rr.Name)

	// Add new entry to the db
	return b.db.Update(func(tx *bolt.Tx) error {
		zoneBkt, err := tx.Bucket([]byte(ZoneConfig)).CreateBucketIfNotExists([]byte(zone))
		if err != nil {
			return fmt.Errorf("zone(%s) retrieval failed", zone)
		}
		dnsCfgKey := DNSConfigRRKey{Host: host, RRType: rrType}

		confKeyBytes, err := json.Marshal(dnsCfgKey)
		if err != nil {
			return fmt.Errorf("internal error, could not parse dns config json")
		}

		confValueBytes := zoneBkt.Get(confKeyBytes)
		if confValueBytes != nil {
			dnsCfgValue := &DNSConfigRRValue{}
			// Update if exists
			if err := json.Unmarshal(confValueBytes, dnsCfgValue); err != nil {
				return fmt.Errorf("parsing failed on data retrieval")
			}
			if len(rr.Class) != 0 {
				rrClass, ok := rrClassMap[rr.Class]
				if !ok {
					return fmt.Errorf("unsupported rrclass(%s) entry", rr.Class)
				}
				dnsCfgValue.RRClass = rrClass
			}
			if rr.TTL != 0 {
				dnsCfgValue.Ttl = rr.TTL
			}
			if len(rr.RData) != 0 {
				dnsCfgValue.PointTo = rr.RData
			}
			confValueBytes, err = json.Marshal(dnsCfgValue)
			if err != nil {
				return fmt.Errorf("data store could not marshal dns config json")
			}
		} else {
			if len(rr.Name) == 0 || len(rr.Type) == 0 || len(rr.Class) == 0 || len(rr.RData) == 0 {
				log.Error("DNS create input not complete.", nil)
				return fmt.Errorf("input data error for create dns entry")
			}
			// Create since not exists
			dnsCfgValue := DNSConfigRRValue{PointTo: rr.RData, RRClass: rrClass, Ttl: rr.TTL}
			confValueBytes, err = json.Marshal(dnsCfgValue)
			if err != nil {
				return fmt.Errorf("data store could not marshal dns config json")
			}
		}
		if err = zoneBkt.Put(confKeyBytes, confValueBytes); err != nil {
			return fmt.Errorf("saving dns entry to data store failed")
		}

		return nil
	})
}

func (b *BoltDB) GetResourceRecord(question *dns.Question) (*[]dns.RR, error) {
	q := strings.ToLower(question.Name)
	var (
		off int
		end bool
	)

	dnsCfgKey := DNSConfigRRKey{Host: q, RRType: question.Qtype}
	dnsCfgKeyBytes, err := json.Marshal(dnsCfgKey)
	if err != nil {
		return nil, fmt.Errorf("parsing dns query failed")
	}

	zones := make(map[string]bool)
	// Get a  zone entries from the input question
	for {
		zones[q[off:]] = true
		off, end = dns.NextLabel(q, off)
		if end {
			break
		}
	}
	zones["."] = true // Add the default zone at end to process

	var records []dns.RR

	err = b.db.View(func(tx *bolt.Tx) error {
		var zoneBkt *bolt.Bucket
		for zone, _ := range zones {
			zoneBkt = tx.Bucket([]byte(ZoneConfig)).Bucket([]byte(zone))
			if zoneBkt == nil {
				// Zone not available in the db
				continue
			}
			dnsCfgBytes := zoneBkt.Get(dnsCfgKeyBytes)
			if dnsCfgBytes == nil {
				continue
			}
			dnsCfg := &DNSConfigRRValue{}
			if err := json.Unmarshal(dnsCfgBytes, dnsCfg); err != nil {
				return fmt.Errorf("parsing the dns record failed")
			}
			// rrClass filtering
			if dnsCfg.RRClass != question.Qclass {
				continue
			}
			for _, pointToIP := range dnsCfg.PointTo {
				records = append(records, &dns.A{Hdr: dns.RR_Header{Name: question.Name, Rrtype: dns.TypeA,
					Class: dns.ClassINET, Ttl: b.TTL}, A: net.ParseIP(pointToIP)})
			}
			break // Found the entry, so stop iterating
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading dns entry from data store failed")
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("could not process/retrieve the query")
	}

	return &records, nil
}

func (b *BoltDB) DelResourceRecord(host string, rrtypestr string) error {
	// panic("implement me")
	var found bool
	rrType, ok := rrTypeMap[rrtypestr]
	if !ok {
		return fmt.Errorf("unsupported rrtype(%s) entry", rrtypestr)
	}

	dnsCfgKey := &DNSConfigRRKey{Host: strings.ToLower(host), RRType: rrType}
	dnsCfgKeyBytes, err := json.Marshal(dnsCfgKey)
	if err != nil {
		return fmt.Errorf("failed to parse input request")
	}

	err = b.db.Update(func(tx *bolt.Tx) error {
		var zoneBkt *bolt.Bucket
		err := tx.Bucket([]byte(ZoneConfig)).ForEach(func(zone, v []byte) error {
			if found {
				return nil
			}
			zoneBkt = tx.Bucket([]byte(ZoneConfig)).Bucket(zone)
			if zoneBkt == nil {
				// Zone not available in the db
				return fmt.Errorf("failed to read the zone entry")
			}
			if zoneBkt.Get(dnsCfgKeyBytes) != nil {
				found = true
				return zoneBkt.Delete(dnsCfgKeyBytes)
			}
			return nil
		})
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to delete dns entry")
	}
	if !found {
		return fmt.Errorf("not found")
	}
	return nil
}
