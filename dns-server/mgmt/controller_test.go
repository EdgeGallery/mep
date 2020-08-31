/*
 *  Copyright 2020 Huawei Technologies Co., Ltd.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package mgmt

import (
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

	t.Run("BasicOperationsOnSetRecord", func(t *testing.T) {
		url := "/mep/dns_server_mgmt/v1/rrecord"
		exampleEntry := "[{\"zone\":\".\",\"rr\":[{\"name\":\"www.example.com.\",\"type\":\"A\",\"class\":\"IN\"," +
			"\"ttl\":30,\"rData\":[\"172.168.15.100\"]},{\"name\":\"www.example1.com.\",\"type\":\"A\"," +
			"\"class\":\"IN\",\"ttl\":30,\"rData\":[\"172.168.15.49\",\"172.168.15.50\",\"172.168.15.51\"]}]}]"
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(exampleEntry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleSetResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		rrResponse, _ := store.GetResourceRecord(&dns.Question{Name: "www.example.com.",
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example.com.\t30\tIN\tA\t172.168.15.100", (*rrResponse)[0].String(),
			"Error")

		err = store.DelResourceRecord("www.example.com.", "A")
		assert.Equal(t, nil, err, "Error in deleting the record")
		err = store.DelResourceRecord("www.example1.com.", "A")
		assert.Equal(t, nil, err, "Error in deleting the record")
	})

	t.Run("DeleteZone", func(t *testing.T) {
		url := "/mep/dns_server_mgmt/v1/rrecord"
		exampleEntry := "[{\"zone\":\".\",\"rr\":[{\"name\":\"www.example.com.\",\"type\":\"A\",\"class\":\"IN\"," +
			"\"ttl\":30,\"rData\":[\"172.168.15.100\"]},{\"name\":\"www.example1.com.\",\"type\":\"A\"," +
			"\"class\":\"IN\",\"ttl\":30,\"rData\":[\"172.168.15.49\",\"172.168.15.50\",\"172.168.15.51\"]}]}]"
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(exampleEntry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleSetResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		// zone name given as record
		err = store.DelResourceRecord(".", "A")
		assert.NotEqual(t, nil, err, "Error in deleting the record")

		err = store.DelResourceRecord("www.example.com.", "A")
		assert.Equal(t, nil, err, "Error in deleting the record")
		err = store.DelResourceRecord("www.example1.com.", "A")
		assert.Equal(t, nil, err, "Error in deleting the record")
	})

	t.Run("RequestWithOutZoneInfo", func(t *testing.T) {
		url := "/mep/dns_server_mgmt/v1/rrecord"
		exampleEntry := "[{\"rr\":[{\"name\":\"www.example.com.\",\"type\":\"A\",\"class\":\"IN\"," +
			"\"ttl\":30,\"rData\":[\"172.168.15.100\"]},{\"name\":\"www.example1.com.\",\"type\":\"A\"," +
			"\"class\":\"IN\",\"ttl\":30,\"rData\":[\"172.168.15.49\",\"172.168.15.50\",\"172.168.15.51\"]}]}]"
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(exampleEntry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleSetResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		rrResponse, _ := store.GetResourceRecord(&dns.Question{Name: "www.example.com.",
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example.com.\t30\tIN\tA\t172.168.15.100", (*rrResponse)[0].String(),
			"Error")

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: "www.example1.com.",
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example1.com.\t30\tIN\tA\t172.168.15.49", (*rrResponse)[0].String(),
			"Error")

		err = store.DelResourceRecord("www.example.com.", "A")
		assert.Equal(t, nil, err, "Error in deleting the record")
		err = store.DelResourceRecord("www.example1.com.", "A")
		assert.Equal(t, nil, err, "Error in deleting the record")
	})

	t.Run("NonDefaultZone", func(t *testing.T) {
		url := "/mep/dns_server_mgmt/v1/rrecord"
		exampleEntry := "[{\"zone\":\"abc.\",\"rr\":[{\"name\":\"www.example.abc.\",\"type\":\"A\",\"class\":\"IN\"," +
			"\"ttl\":30,\"rData\":[\"172.168.15.100\"]},{\"name\":\"www.example1.abc.\",\"type\":\"A\"," +
			"\"class\":\"IN\",\"ttl\":30,\"rData\":[\"172.168.15.49\",\"172.168.15.50\",\"172.168.15.51\"]}]}]"
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(exampleEntry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleSetResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		rrResponse, _ := store.GetResourceRecord(&dns.Question{Name: "www.example.abc.",
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example.abc.\t30\tIN\tA\t172.168.15.100", (*rrResponse)[0].String(),
			"Error")

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: "www.example1.abc.",
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example1.abc.\t30\tIN\tA\t172.168.15.49", (*rrResponse)[0].String(),
			"Error")
		assert.Equal(t, "www.example1.abc.\t30\tIN\tA\t172.168.15.50", (*rrResponse)[1].String(),
			"Error")
		assert.Equal(t, "www.example1.abc.\t30\tIN\tA\t172.168.15.51", (*rrResponse)[2].String(),
			"Error")

		// zone name given as record
		err = store.DelResourceRecord(".", "A")
		assert.NotEqual(t, nil, err, "Error in deleting the record")
		err = store.DelResourceRecord("abc.", "A")
		assert.NotEqual(t, nil, err, "Error in deleting the record")

		err = store.DelResourceRecord("www.example.abc.", "A")
		assert.Equal(t, nil, err, "Error in deleting the record")
		err = store.DelResourceRecord("www.example1.abc.", "A")
		assert.Equal(t, nil, err, "Error in deleting the record")
	})

	t.Run("MultiZone", func(t *testing.T) {
		url := "/mep/dns_server_mgmt/v1/rrecord"
		exampleEntry := "[{\"zone\":\".\",\"rr\":[{\"name\":\"www.example.com.\",\"type\":\"A\",\"class\":\"IN\"," +
			"\"ttl\":30,\"rData\":[\"172.168.15.100\"]},{\"name\":\"www.example1.com.\",\"type\":\"A\"," +
			"\"class\":\"IN\",\"ttl\":30,\"rData\":[\"172.168.15.49\",\"172.168.15.50\",\"172.168.15.51\"]}]}," +
			"{\"zone\":\"abc.\",\"rr\":[{\"name\":\"www.example.abc.\",\"type\":\"A\",\"class\":\"IN\",\"ttl\":30," +
			"\"rData\":[\"162.168.15.100\"]},{\"name\":\"www.example1.abc.\",\"type\":\"A\",\"class\":\"IN\"," +
			"\"ttl\":30,\"rData\":[\"162.168.15.49\",\"162.168.15.50\",\"162.168.15.51\"]}]}]"
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(exampleEntry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleSetResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		rrResponse, _ := store.GetResourceRecord(&dns.Question{Name: "www.example.com.",
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example.com.\t30\tIN\tA\t172.168.15.100", (*rrResponse)[0].String(),
			"Error")

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: "www.example1.com.",
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example1.com.\t30\tIN\tA\t172.168.15.49", (*rrResponse)[0].String(),
			"Error")
		assert.Equal(t, "www.example1.com.\t30\tIN\tA\t172.168.15.50", (*rrResponse)[1].String(),
			"Error")
		assert.Equal(t, "www.example1.com.\t30\tIN\tA\t172.168.15.51", (*rrResponse)[2].String(),
			"Error")

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: "www.example.abc.",
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example.abc.\t30\tIN\tA\t162.168.15.100", (*rrResponse)[0].String(),
			"Error")

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: "www.example1.abc.",
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example1.abc.\t30\tIN\tA\t162.168.15.49", (*rrResponse)[0].String(),
			"Error")
		assert.Equal(t, "www.example1.abc.\t30\tIN\tA\t162.168.15.50", (*rrResponse)[1].String(),
			"Error")
		assert.Equal(t, "www.example1.abc.\t30\tIN\tA\t162.168.15.51", (*rrResponse)[2].String(),
			"Error")

		// zone name given as record
		err = store.DelResourceRecord(".", "A")
		assert.NotEqual(t, nil, err, "Error in deleting the record")
		err = store.DelResourceRecord("abc.", "A")
		assert.NotEqual(t, nil, err, "Error in deleting the record")

		err = store.DelResourceRecord("www.example.abc.", "A")
		assert.Equal(t, nil, err, "Error in deleting the record")
		err = store.DelResourceRecord("www.example1.abc.", "A")
		assert.Equal(t, nil, err, "Error in deleting the record")

		err = store.DelResourceRecord("www.example.com.", "A")
		assert.Equal(t, nil, err, "Error in deleting the record")
		err = store.DelResourceRecord("www.example1.com.", "A")
		assert.Equal(t, nil, err, "Error in deleting the record")
	})

	t.Run("DeleteRequest", func(t *testing.T) {
		url := "/mep/dns_server_mgmt/v1/rrecord"
		exampleEntry := "[{\"zone\":\".\",\"rr\":[{\"name\":\"www.example.com.\",\"type\":\"A\",\"class\":\"IN\"," +
			"\"ttl\":30,\"rData\":[\"172.168.15.100\"]},{\"name\":\"www.example1.com.\",\"type\":\"A\"," +
			"\"class\":\"IN\",\"ttl\":30,\"rData\":[\"172.168.15.49\",\"172.168.15.50\",\"172.168.15.51\"]}]}," +
			"{\"zone\":\"abc.\",\"rr\":[{\"name\":\"www.example.abc.\",\"type\":\"A\",\"class\":\"IN\",\"ttl\":30," +
			"\"rData\":[\"162.168.15.100\"]},{\"name\":\"www.example1.abc.\",\"type\":\"A\",\"class\":\"IN\"," +
			"\"ttl\":30,\"rData\":[\"162.168.15.49\",\"162.168.15.50\",\"162.168.15.51\"]}]}]"
		e := echo.New()
		newRequest, err := http.NewRequest(http.MethodPost, url, strings.NewReader(exampleEntry))
		assert.Equal(t, nil, err, "Error")
		newRequest.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		c := e.NewContext(newRequest, recorder)
		err = mgmtCtl.handleSetResourceRecords(c)
		assert.Equal(t, nil, err, "Error")

		rrResponse, _ := store.GetResourceRecord(&dns.Question{Name: "www.example.com.",
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example.com.\t30\tIN\tA\t172.168.15.100", (*rrResponse)[0].String(),
			"Error")

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: "www.example1.com.",
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example1.com.\t30\tIN\tA\t172.168.15.49", (*rrResponse)[0].String(),
			"Error")
		assert.Equal(t, "www.example1.com.\t30\tIN\tA\t172.168.15.50", (*rrResponse)[1].String(),
			"Error")
		assert.Equal(t, "www.example1.com.\t30\tIN\tA\t172.168.15.51", (*rrResponse)[2].String(),
			"Error")

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: "www.example.abc.",
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example.abc.\t30\tIN\tA\t162.168.15.100", (*rrResponse)[0].String(),
			"Error")

		rrResponse, _ = store.GetResourceRecord(&dns.Question{Name: "www.example1.abc.",
			Qtype: dns.TypeA, Qclass: dns.ClassINET})
		assert.Equal(t, "www.example1.abc.\t30\tIN\tA\t162.168.15.49", (*rrResponse)[0].String(),
			"Error")
		assert.Equal(t, "www.example1.abc.\t30\tIN\tA\t162.168.15.50", (*rrResponse)[1].String(),
			"Error")
		assert.Equal(t, "www.example1.abc.\t30\tIN\tA\t162.168.15.51", (*rrResponse)[2].String(),
			"Error")

		deleteUrl1 := "/mep/dns_server_mgmt/v1/rrecord/www.example.abc./A"
		deleteRequest1, err := http.NewRequest(http.MethodPost, deleteUrl1, strings.NewReader(""))
		assert.Equal(t, nil, err, "Error")
		deleteRequest1.Header.Set("Content-Type", "application/json")
		deleteRecorder1 := httptest.NewRecorder()
		delContext1 := e.NewContext(deleteRequest1, deleteRecorder1)
		delContext1.SetParamNames("fqdn", "rrtype")
		delContext1.SetParamValues("www.example.abc.", "A")
		err = mgmtCtl.handleDeleteResourceRecord(delContext1)
		assert.Equal(t, nil, err, "Error")
		assert.Equal(t, http.StatusOK, delContext1.Response().Status, "Error")
		err = store.DelResourceRecord("www.example.abc.", "A")
		assert.NotEqual(t, nil, err, "Error in deleting the record")

		deleteUrl2 := "/mep/dns_server_mgmt/v1/rrecord/www.example1.abc./A"
		deleteRequest2, err := http.NewRequest(http.MethodPost, deleteUrl2, strings.NewReader(""))
		assert.Equal(t, nil, err, "Error")
		deleteRequest2.Header.Set("Content-Type", "application/json")
		deleteRecorder2 := httptest.NewRecorder()
		delContext2 := e.NewContext(deleteRequest2, deleteRecorder2)
		delContext2.SetParamNames("fqdn", "rrtype")
		delContext2.SetParamValues("www.example1.abc.", "A")
		err = mgmtCtl.handleDeleteResourceRecord(delContext2)
		assert.Equal(t, nil, err, "Error")
		assert.Equal(t, http.StatusOK, delContext2.Response().Status, "Error")
		err = store.DelResourceRecord("www.example1.abc.", "A")
		assert.NotEqual(t, nil, err, "Error in deleting the record")

		deleteUrl3 := "/mep/dns_server_mgmt/v1/rrecord/www.example.com./A"
		deleteRequest3, err := http.NewRequest(http.MethodPost, deleteUrl3, strings.NewReader(""))
		assert.Equal(t, nil, err, "Error")
		deleteRequest3.Header.Set("Content-Type", "application/json")
		deleteRecorder3 := httptest.NewRecorder()
		delContext3 := e.NewContext(deleteRequest3, deleteRecorder3)
		delContext3.SetParamNames("fqdn", "rrtype")
		delContext3.SetParamValues("www.example.com.", "A")
		err = mgmtCtl.handleDeleteResourceRecord(delContext3)
		assert.Equal(t, nil, err, "Error")
		assert.Equal(t, http.StatusOK, delContext3.Response().Status, "Error")
		err = store.DelResourceRecord("www.example.com.", "A")
		assert.NotEqual(t, nil, err, "Error in deleting the record")

		deleteUrl4 := "/mep/dns_server_mgmt/v1/rrecord/www.example1.com./A"
		deleteRequest4, err := http.NewRequest(http.MethodPost, deleteUrl4, strings.NewReader(""))
		assert.Equal(t, nil, err, "Error")
		deleteRequest4.Header.Set("Content-Type", "application/json")
		deleteRecorder4 := httptest.NewRecorder()
		delContext4 := e.NewContext(deleteRequest4, deleteRecorder4)
		delContext4.SetParamNames("fqdn", "rrtype")
		delContext4.SetParamValues("www.example1.com.", "A")
		err = mgmtCtl.handleDeleteResourceRecord(delContext4)
		assert.Equal(t, nil, err, "Error")
		assert.Equal(t, http.StatusOK, delContext4.Response().Status, "Error")
		err = store.DelResourceRecord("www.example1.com.", "A")
		assert.NotEqual(t, nil, err, "Error in deleting the record")
	})
	t.Run("DeleteRequestEmptyFqdn", func(t *testing.T) {
		e := echo.New()
		deleteUrl1 := "/mep/dns_server_mgmt/v1/rrecord/www.example.abc./A"
		deleteRequest1, err := http.NewRequest(http.MethodPost, deleteUrl1, strings.NewReader(""))
		assert.Equal(t, nil, err, "Error")
		deleteRequest1.Header.Set("Content-Type", "application/json")
		deleteRecorder1 := httptest.NewRecorder()
		delContext1 := e.NewContext(deleteRequest1, deleteRecorder1)
		// delContext1.SetParamNames("fqdn", "rrtype") Commented to make it empty
		// delContext1.SetParamValues("www.example.abc.", "A") Commented to make it empty
		err = mgmtCtl.handleDeleteResourceRecord(delContext1)
		assert.Equal(t, nil, err, "Error")
		assert.Equal(t, http.StatusBadRequest, delContext1.Response().Status, "Error")

	})

	t.Run("DeleteRequestOnNonExistingRecord", func(t *testing.T) {
		e := echo.New()
		deleteUrl1 := "/mep/dns_server_mgmt/v1/rrecord/www.example.abc./A"
		deleteRequest1, err := http.NewRequest(http.MethodPost, deleteUrl1, strings.NewReader(""))
		assert.Equal(t, nil, err, "Error")
		deleteRequest1.Header.Set("Content-Type", "application/json")
		deleteRecorder1 := httptest.NewRecorder()
		delContext1 := e.NewContext(deleteRequest1, deleteRecorder1)
		delContext1.SetParamNames("fqdn", "rrtype")
		delContext1.SetParamValues("www.example.abc.", "A")
		err = mgmtCtl.handleDeleteResourceRecord(delContext1)
		assert.Equal(t, nil, err, "Error")
		assert.Equal(t, http.StatusInternalServerError, delContext1.Response().Status, "Error")

	})

	_ = os.RemoveAll(datastore.DBPath)

}
