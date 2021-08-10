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

package mgmt

import (
	"encoding/json"
	"fmt"
	"github.com/agiledragon/gomonkey"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"

	"dns-server/datastore"
)

// Query dns rules request in mp1 interface
var url = "/mep/dns_server_mgmt/v1/rrecord"
var cont = "Content-Type"
var appj = "application/json"
var eg = "www.example.com."
var eg172 = "www.example.com.\t30\tIN\tA\t172.168.15.100"
var errRecord = "Error in deleting the record"
var eg1 = "www.example1.com."
var eg1_49 = "www.example1.com.\t30\tIN\tA\t172.168.15.49"
var egAbc = "www.example.abc."
var eg1Abc = "www.example1.abc."
var mepA = "/mep/dns_server_mgmt/v1/rrecord/www.example.abc./A"
var egE1 = "[{\"zone\":\".\",\"rr\":[{\"name\":\"www.example.com.\",\"type\":\"A\",\"class\":\"IN\","
var egE2 = "\"ttl\":30,\"rData\":[\"172.168.15.100\"]},{\"name\":\"www.example1.com.\",\"type\":\"A\","
var egE3 = "\"class\":\"IN\",\"ttl\":30,\"rData\":[\"172.168.15.49\",\"172.168.15.50\",\"172.168.15.51\"]}]}]"

var rr_entry = "{\"name\": \"www.example.com.\",\"type\": \"A\",\"class\": \"IN\",\"ttl\": 30,\"rData\": [\"172.168.15.100\"]}"
var rr_entry1 = "{\"name\": \"www.example1.com.\",\"type\": \"A\",\"class\": \"IN\",\"ttl\": 30,\"rData\": [\"172.168.15.101\"]}"
var rr_entry2 = "{\"name\": \"www.example.org.\",\"type\": \"A\",\"class\": \"IN\",\"ttl\": 30,\"rData\": [\"172.168.15.102\"]}"
var rr_entrySet = "{\"name\": \"www.example.com.\",\"type\": \"A\",\"class\": \"IN\",\"ttl\": 32,\"rData\": [\"172.168.15.100\", \"152.168.15.102\"]}"
var rr_eg100 = "www.example.com.\t30\tIN\tA\t172.168.15.100"
var rr_eg101 = "www.example1.com.\t30\tIN\tA\t172.168.15.101"
var rr_eg102 = "www.example.org.\t30\tIN\tA\t172.168.15.102"
var invalidZone = "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890" +
	"1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890" +
	"1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"
var rr_entry_invalid = "{\"name\": \"www.example.c" + invalidZone + "om.\",\"type\": \"A\",\"class\": \"IN\",\"ttl\": 30,\"rData\": [\"172.168.15.100\"]}"
var egOrg = "www.example.org."
var rr_invalidIP = "{\"name\": \"www.e.com.\",\"type\": \"A\",\"class\": \"IN\",\"ttl\": 30,\"rData\": [\"255.255.255.255\"]}"
var rr_invalidrrtype = "{\"name\": \"www.e.com.\",\"type\": \"AAB\",\"class\": \"IN\",\"ttl\": 30,\"rData\": [\"172.168.15.100\"]}"
var rr_invalidTTL = "{\"name\": \"www.e.com.\",\"type\": \"A\",\"class\": \"IN\",\"ttl\": 0,\"rData\": [\"172.168.15.1005\"]}"
var rr_setinvalidrrtype = "{\"name\": \"www.example.com.\",\"type\": \"AAB\",\"class\": \"IN\",\"ttl\": 30,\"rData\": [\"172.168.15.100\"]}"

func TestRestControllerOperations(t *testing.T) {
	defer func() {
		_ = os.RemoveAll(datastore.DBPath)
		if r := recover(); r != nil {
			t.Errorf("Panic: %v", r)
		}
	}()

	store := &datastore.BoltDB{FileName: "testdb", TTL: 30}
	err := store.Open()
	assert.Equal(t, nil, err, "Error in opening the db")
	defer store.Close()

	mgmtCtl := &Controller{dataStore: store}

	t.Run("BasicOperationsOnAddRecord", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url+"/./www.example.com/A", strings.NewReader(rr_entry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		rrResponse, _ := store.GetResourceRecord(&dns.Question{Name: eg,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, rr_eg100, (*rrResponse)[0].String(),
			"Error")

		err = store.DelResourceRecord("", eg, "A")
		assert.Equal(t, nil, err, errRecord)
		err = store.DelResourceRecord("", eg1, "A")
		assert.NotEqual(t, nil, err, errRecord)
	})

	t.Run("DeleteZone", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(rr_entry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		newRequest, err = http.NewRequest(http.MethodPost, url, strings.NewReader(rr_entry1))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder = httptest.NewRecorder()
		c = e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		// zone name given as record
		err = store.DelResourceRecord("", "www.example.org.", "A")
		assert.NotEqual(t, nil, err, errRecord)

		err = store.DelResourceRecord("", eg, "A")
		assert.Equal(t, nil, err, errRecord)
		err = store.DelResourceRecord("", eg1, "A")
		assert.Equal(t, nil, err, errRecord)
	})

	t.Run("RequestWithOutZoneInfo", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(rr_entry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		newRequest, err = http.NewRequest(http.MethodPost, url, strings.NewReader(rr_entry1))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder = httptest.NewRecorder()
		c = e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		rrResponse, _ := store.GetResourceRecord(&dns.Question{Name: eg,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, rr_eg100, (*rrResponse)[0].String(),
			"Error")

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: eg1,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, rr_eg101, (*rrResponse)[0].String(),
			"Error")

		err = store.DelResourceRecord("", eg, "A")
		assert.Equal(t, nil, err, errRecord)
		err = store.DelResourceRecord("", eg1, "A")
		assert.Equal(t, nil, err, errRecord)
	})

	t.Run("NonDefaultZone", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(rr_entry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		newRequest, err = http.NewRequest(http.MethodPost, url, strings.NewReader(rr_entry1))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder = httptest.NewRecorder()
		c = e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		rrResponse, _ := store.GetResourceRecord(&dns.Question{Name: eg,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example.com.\t30\tIN\tA\t172.168.15.100", (*rrResponse)[0].String(),
			"Error")

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: eg1,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example1.com.\t30\tIN\tA\t172.168.15.101", (*rrResponse)[0].String(),
			"Error")

		// zone name given as record
		err = store.DelResourceRecord("", ".", "A")
		assert.NotEqual(t, nil, err, errRecord)
		err = store.DelResourceRecord("", "com.", "A")
		assert.NotEqual(t, nil, err, errRecord)

		err = store.DelResourceRecord("", eg, "A")
		assert.Equal(t, nil, err, errRecord)
		err = store.DelResourceRecord("", eg1, "A")
		assert.Equal(t, nil, err, errRecord)
	})

	t.Run("MultiZone", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(rr_entry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		c.SetParamNames("zone")
		c.SetParamValues(".")
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		newRequest, err = http.NewRequest(http.MethodPost, url, strings.NewReader(rr_entry2))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder = httptest.NewRecorder()
		c = e.NewContext(newRequest, recorder)
		c.SetParamNames("zone")
		c.SetParamValues("org.")
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		rrResponse, _ := store.GetResourceRecord(&dns.Question{Name: eg,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, eg172, (*rrResponse)[0].String(),
			"Error")

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: egOrg,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, rr_eg102, (*rrResponse)[0].String(),
			"Error")

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: eg,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example.com.\t30\tIN\tA\t172.168.15.100", (*rrResponse)[0].String(),
			"Error")

		// zone name given as record
		err = store.DelResourceRecord("invalid", ".", "A")
		assert.NotEqual(t, nil, err, errRecord)
		err = store.DelResourceRecord("", "org.", "A")
		assert.NotEqual(t, nil, err, errRecord)

		err = store.DelResourceRecord("", eg, "A")
		assert.Equal(t, nil, err, errRecord)
		err = store.DelResourceRecord("org", egOrg, "A")
		assert.Equal(t, nil, err, errRecord)
	})

	t.Run("DeleteRequest", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(rr_entry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		c.SetParamNames("zone")
		c.SetParamValues(".")
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		newRequest, err = http.NewRequest(http.MethodPost, url, strings.NewReader(rr_entry1))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder = httptest.NewRecorder()
		c = e.NewContext(newRequest, recorder)
		c.SetParamNames("zone")
		c.SetParamValues(".")
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		newRequest, err = http.NewRequest(http.MethodPost, url, strings.NewReader(rr_entry2))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder = httptest.NewRecorder()
		c = e.NewContext(newRequest, recorder)
		c.SetParamNames("zone")
		c.SetParamValues("org.")
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		rrResponse, _ := store.GetResourceRecord(&dns.Question{Name: eg,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, rr_eg100, (*rrResponse)[0].String(),
			"Error")

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: eg1,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, rr_eg101, (*rrResponse)[0].String(),
			"Error")

		deleteUrl1 := mepA
		deleteRequest1, err := http.NewRequest(http.MethodPost, deleteUrl1, strings.NewReader(""))
		assert.Equal(t, nil, err, "Error")
		deleteRequest1.Header.Set(cont, appj)
		deleteRecorder1 := httptest.NewRecorder()
		delContext1 := e.NewContext(deleteRequest1, deleteRecorder1)
		delContext1.SetParamNames("fqdn", "rrtype")
		delContext1.SetParamValues(eg, "A")
		err = mgmtCtl.handleDeleteResourceRecord(delContext1)
		assert.Equal(t, nil, err, "Error")
		assert.Equal(t, http.StatusOK, delContext1.Response().Status, "Error")
		err = store.DelResourceRecord("", eg, "A")
		assert.NotEqual(t, nil, err, errRecord)

		deleteUrl2 := "/mep/dns_server_mgmt/v1/rrecord/www.example.org./A"
		deleteRequest2, err := http.NewRequest(http.MethodPost, deleteUrl2, strings.NewReader(""))
		assert.Equal(t, nil, err, "Error")
		deleteRequest2.Header.Set(cont, appj)
		deleteRecorder2 := httptest.NewRecorder()
		delContext2 := e.NewContext(deleteRequest2, deleteRecorder2)
		delContext2.SetParamNames("fqdn", "rrtype")
		delContext2.SetParamValues(egOrg, "A")
		err = mgmtCtl.handleDeleteResourceRecord(delContext2)
		assert.Equal(t, nil, err, "Error")
		assert.Equal(t, http.StatusOK, delContext2.Response().Status, "Error")
		err = store.DelResourceRecord("", egOrg, "A")
		assert.NotEqual(t, nil, err, errRecord)

		deleteUrl4 := "/mep/dns_server_mgmt/v1/rrecord/www.example1.com./A"
		deleteRequest4, err := http.NewRequest(http.MethodPost, deleteUrl4, strings.NewReader(""))
		assert.Equal(t, nil, err, "Error")
		deleteRequest4.Header.Set(cont, appj)
		deleteRecorder4 := httptest.NewRecorder()
		delContext4 := e.NewContext(deleteRequest4, deleteRecorder4)
		delContext4.SetParamNames("fqdn", "rrtype")
		delContext4.SetParamValues(eg1, "A")
		err = mgmtCtl.handleDeleteResourceRecord(delContext4)
		assert.Equal(t, nil, err, "Error")
		assert.Equal(t, http.StatusOK, delContext4.Response().Status, "Error")
		err = store.DelResourceRecord("", eg1, "A")
		assert.NotEqual(t, nil, err, errRecord)
	})
	t.Run("DeleteRequestEmptyFqdn", func(t *testing.T) {
		e := echo.New()
		deleteUrl1 := mepA
		deleteRequest1, err := http.NewRequest(http.MethodPost, deleteUrl1, strings.NewReader(""))
		assert.Equal(t, nil, err, "Error")
		deleteRequest1.Header.Set(cont, appj)
		deleteRecorder1 := httptest.NewRecorder()
		delContext1 := e.NewContext(deleteRequest1, deleteRecorder1)
		// delContext1.SetParamNames("fqdn", "rrtype") Commented to make it empty
		// delContext1.SetParamValues(egAbc, "A") Commented to make it empty
		err = mgmtCtl.handleDeleteResourceRecord(delContext1)
		assert.Equal(t, nil, err, "Error")
		assert.Equal(t, http.StatusBadRequest, delContext1.Response().Status, "Error")

	})

	t.Run("DeleteRequestOnNonExistingRecord", func(t *testing.T) {
		e := echo.New()
		deleteUrl1 := mepA
		deleteRequest1, err := http.NewRequest(http.MethodPost, deleteUrl1, strings.NewReader(""))
		assert.Equal(t, nil, err, "Error")
		deleteRequest1.Header.Set(cont, appj)
		deleteRecorder1 := httptest.NewRecorder()
		delContext1 := e.NewContext(deleteRequest1, deleteRecorder1)
		delContext1.SetParamNames("fqdn", "rrtype")
		delContext1.SetParamValues(egAbc, "A")
		err = mgmtCtl.handleDeleteResourceRecord(delContext1)
		assert.Equal(t, nil, err, "Error")
		assert.Equal(t, http.StatusInternalServerError, delContext1.Response().Status, "Error")
	})

	t.Run("BasicOperationsOnSetRecord", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url+"/./www.example.com/A", strings.NewReader(rr_entry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		rrResponse, _ := store.GetResourceRecord(&dns.Question{Name: eg,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, rr_eg100, (*rrResponse)[0].String(),
			"Error")

		newRequest, err = http.NewRequest(http.MethodPut, url+"/./www.example.com/A", strings.NewReader(rr_entrySet))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder = httptest.NewRecorder()
		c = e.NewContext(newRequest, recorder)
		c.SetParamNames("fqdn", "rrtype")
		c.SetParamValues(eg, "A")
		err = mgmtCtl.handleSetResourceRecords(c)
		assert.Equal(t, nil, err, "Error")
		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: eg,
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example.com.\t32\tIN\tA\t172.168.15.100", (*rrResponse)[0].String(),
			"Error")
		assert.Equal(t, "www.example.com.\t32\tIN\tA\t152.168.15.102", (*rrResponse)[1].String(),
			"Error")
		err = store.DelResourceRecord("", eg, "A")
		assert.Equal(t, nil, err, errRecord)
		err = store.DelResourceRecord("", eg1, "A")
		assert.NotEqual(t, nil, err, errRecord)
	})
	t.Run("BasicOperationsOnAddRecordBindErr", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url+"/./www.example.com/A", strings.NewReader(egE1))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")
	})

	t.Run("BasicOperationsOnAddRecordInvalidZone", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(rr_entry2))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		c.SetParamNames("zone")
		c.SetParamValues(invalidZone)
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")
	})

	t.Run("BasicOperationsOnAddRecordInvalidFqdn", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(rr_entry_invalid))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		c.SetParamNames("zone")
		c.SetParamValues("com")
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")
	})
	t.Run("BasicOperationsOnAddRecordInvalidRdata", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(rr_invalidIP))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		c.SetParamNames("zone")
		c.SetParamValues("com")
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")
	})

	t.Run("BasicOperationsOnAddRecordExists", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(rr_entry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		c.SetParamNames("zone")
		c.SetParamValues("org")
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		newRequest, err = http.NewRequest(http.MethodPost, url, strings.NewReader(rr_entry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder = httptest.NewRecorder()
		c = e.NewContext(newRequest, recorder)
		c.SetParamNames("zone")
		c.SetParamValues("org")
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")
		err = store.DelResourceRecord("", eg, "A")
		assert.Equal(t, nil, err, errRecord)
	})

	t.Run("BasicOperationsOnAddRecordInvalidRRtype", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(rr_invalidrrtype))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		c.SetParamNames("zone")
		c.SetParamValues("org")
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")
	})
	t.Run("BasicOperationsOnAddRecordInvalidRRtype", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(rr_invalidTTL))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		c.SetParamNames("zone")
		c.SetParamValues("org")
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")
	})
	t.Run("BasicOperationsOnSetRecordBindErr", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url+"/./www.example.com/A", strings.NewReader(rr_entry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		//set
		newRequest, err = http.NewRequest(http.MethodPut, url+"/./www.example.com/A", strings.NewReader(eg172))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder = httptest.NewRecorder()
		c = e.NewContext(newRequest, recorder)
		c.SetParamNames("fqdn", "rrtype")
		c.SetParamValues(eg, "A")
		err = mgmtCtl.handleSetResourceRecords(c)
		assert.Equal(t, nil, err, "Error")
		err = store.DelResourceRecord("", eg, "A")
		assert.Equal(t, nil, err, errRecord)
	})
	t.Run("BasicOperationsOnSetRecordInvalidInput", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url+"/./www.example.com/A", strings.NewReader(rr_entry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		//set
		newRequest, err = http.NewRequest(http.MethodPut, url+"/./www.example.com/A", strings.NewReader(rr_entrySet))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder = httptest.NewRecorder()
		c = e.NewContext(newRequest, recorder)
		c.SetParamNames("fqdn")
		c.SetParamValues(eg)
		err = mgmtCtl.handleSetResourceRecords(c)
		assert.Equal(t, nil, err, "Error")
		//set
		newRequest, err = http.NewRequest(http.MethodPut, url+"/./www.example.com/A", strings.NewReader(rr_entrySet))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder = httptest.NewRecorder()
		c = e.NewContext(newRequest, recorder)
		c.SetParamNames("rrtype")
		c.SetParamValues("A")
		err = mgmtCtl.handleSetResourceRecords(c)
		assert.Equal(t, nil, err, "Error")
		err = store.DelResourceRecord("", eg, "A")
		assert.Equal(t, nil, err, errRecord)
	})

	t.Run("BasicOperationsOnSetRecordInvalidInput", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url+"/./www.example.com/A", strings.NewReader(rr_entry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		//set
		newRequest, err = http.NewRequest(http.MethodPut, url+"/./www.example.com/AB", strings.NewReader(rr_entrySet))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder = httptest.NewRecorder()
		c = e.NewContext(newRequest, recorder)
		c.SetParamNames("fqdn", "rrtype")
		c.SetParamValues(eg, "AB")
		err = mgmtCtl.handleSetResourceRecords(c)
		assert.Equal(t, nil, err, "Error")
		//set
		newRequest, err = http.NewRequest(http.MethodPut, url+"/./www.invalid.com/A", strings.NewReader(rr_entrySet))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder = httptest.NewRecorder()
		c = e.NewContext(newRequest, recorder)
		c.SetParamNames("fqdn", "rrtype")
		c.SetParamValues(eg+".in", "A")
		err = mgmtCtl.handleSetResourceRecords(c)
		assert.Equal(t, nil, err, "Error")
		err = store.DelResourceRecord("", eg, "A")
		assert.Equal(t, nil, err, errRecord)
	})
	t.Run("BasicOperationsOnSetRecordInvalidRdata", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(rr_entry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		c.SetParamNames("zone")
		c.SetParamValues("com")
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		newRequest, err = http.NewRequest(http.MethodPut, url+"/./www.invalid.com/A", strings.NewReader(rr_invalidIP))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder = httptest.NewRecorder()
		c = e.NewContext(newRequest, recorder)
		c.SetParamNames("fqdn", "rrtype")
		c.SetParamValues("www.e.com.", "A")
		err = mgmtCtl.handleSetResourceRecords(c)
		assert.Equal(t, nil, err, "Error")
		err = store.DelResourceRecord("", eg, "A")
		assert.Equal(t, nil, err, errRecord)
	})
	t.Run("BasicOperationsOnSetRecordRecordNotExists", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPut, url+"/./www.invalid.com/A", strings.NewReader(rr_entrySet))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		c.SetParamNames("fqdn", "rrtype")
		c.SetParamValues(eg, "A")
		err = mgmtCtl.handleSetResourceRecords(c)
		assert.Equal(t, nil, err, "Error")
	})
	t.Run("BasicOperationsOnSetRecordInvalidTTL", func(t *testing.T) {
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(rr_entry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		c.SetParamNames("zone")
		c.SetParamValues("")
		err = mgmtCtl.handleAddResourceRecords(c)
		assert.Equal(t, nil, err, "Error")
		count := 1
		patch := gomonkey.ApplyFunc(json.Marshal, func(v interface{}) ([]byte, error) {
			if count == 1 {
				count++
				bytes := "{\"host\":\"www.example.com.\",\"rrType\":1}"
				return []byte(bytes), nil
			} else {
				return nil, fmt.Errorf("test")
			}
		})
		defer patch.Reset()
		newRequest, err = http.NewRequest(http.MethodPut, url+"/./www.invalid.com/A", strings.NewReader(rr_entry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set(cont, appj)
		recorder = httptest.NewRecorder()
		c = e.NewContext(newRequest, recorder)
		c.SetParamNames("fqdn", "rrtype")
		c.SetParamValues(eg, "A")
		err = mgmtCtl.handleSetResourceRecords(c)
		assert.Equal(t, nil, err, "Error")
		patch.Reset()
		err = store.DelResourceRecord("", eg, "A")
		assert.Equal(t, nil, err, errRecord)
	})
	//Cleanup Db
	_ = os.RemoveAll(datastore.DBPath)
}
