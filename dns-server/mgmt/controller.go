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

//
package mgmt

import (
	"fmt"
	"net"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	log "github.com/sirupsen/logrus"

	"dns-server/datastore"
	"dns-server/util"
)

type Controller struct {
	dataStore datastore.DataStore
	echo      *echo.Echo
}

const invalidInputErr = "invalid input!"

func (e *Controller) StartController(store *datastore.DataStore, ipAddr net.IP, port uint) {
	// Echo instance
	e.echo = echo.New()

	// Middleware
	e.echo.Use(middleware.Logger())
	e.echo.Use(middleware.Recover())
	e.echo.Use(middleware.BodyLimit(util.MaxPacketSize))

	// Routes
	e.echo.POST("/mep/dns_server_mgmt/v1/rrecord", e.handleAddResourceRecords)
	e.echo.PUT("/mep/dns_server_mgmt/v1/rrecord/:fqdn/:rrtype", e.handleSetResourceRecords)
	e.echo.DELETE("/mep/dns_server_mgmt/v1/rrecord/:fqdn/:rrtype", e.handleDeleteResourceRecord)
	e.echo.GET("/health", e.handleHealthResult)

	e.dataStore = *store

	// Start server
	e.echo.Logger.Fatal(e.echo.Start(fmt.Sprintf("%s:%d", ipAddr.String(), port)))
}

func (e *Controller) StopController() error {
	e.dataStore = nil
	if e.echo == nil {
		return nil
	}

	return e.echo.Close()
}

func (e *Controller) handleAddResourceRecords(c echo.Context) error {
	// Input Example:
	//{
	//	"name": "www.example.com.",
	//	"type": "A",
	//	"class": "IN",
	//	"ttl": 30,
	//	"rData": [
	//      "172.168.15.101"
	//     ]
	//}

	zone := c.QueryParam("zone")

	rr := datastore.ResourceRecord{}
	if nil != c.Bind(&rr) {
		log.Error("Error in parsing the rr post request body.", nil)
		return c.String(http.StatusBadRequest, invalidInputErr)
	}

	if len(zone) == 0 {
		zone = "."
	}

	err := e.validateSetRecordInput(zone, &rr)
	if err != nil {
		log.Error("Error in validating the rr post request body.", err)
		return c.String(http.StatusBadRequest, invalidInputErr)
	}

	// Check already exists, then no need to add again
	exists := e.dataStore.IsResourceRecordExists(zone, &rr)
	if exists == true {
		log.Error("Record already exist.")
		return c.String(http.StatusBadRequest, "record already exists!")
	}

	err = e.dataStore.SetResourceRecord(zone, &rr)
	if err != nil {
		log.Error("Failed to set the zone entries.")
		return c.String(http.StatusInternalServerError, err.Error())
	}
	log.Debugf("Added new resource record entry(zone: %s, name: %s, type: %s, class: %s, ttl: %d).",
		zone, rr.Name, rr.Type, rr.Class, rr.TTL)

	return c.String(http.StatusOK, "success in adding rr entry.")
}

func (e *Controller) handleSetResourceRecords(c echo.Context) error {
	// Input Example:
	//{
	//	"name": "www.example.com.",
	//	"type": "A",
	//	"class": "IN",
	//	"ttl": 30,
	//	"rData": [
	//      "172.168.15.101"
	//     ]
	//}
	zone := c.QueryParam("zone")
	fqdn := c.Param("fqdn")
	rrtype := c.Param("rrtype")

	rr := datastore.ResourceRecord{}
	if nil != c.Bind(&rr) {
		log.Error("Error in parsing the rr post request body.", nil)
		return c.String(http.StatusBadRequest, invalidInputErr)
	}

	if len(fqdn) == 0 || len(rrtype) == 0 {
		return c.String(http.StatusBadRequest, "invalid input parameters!")
	}

	if fqdn != rr.Name || rrtype != rr.Type {
		return c.String(http.StatusBadRequest, "input not match with rr resource")
	}

	if len(zone) == 0 {
		zone = "."
	}
	//Update the input param
	err := e.validateSetRecordInput(zone, &rr)
	if err != nil {
		log.Error("Error in validating the rr post request body.", err)
		return c.String(http.StatusBadRequest, invalidInputErr)
	}

	// Check already exists, if not exist then cant update
	exists := e.dataStore.IsResourceRecordExists(zone, &rr)
	if exists != true {
		log.Error("Record not exist, cannot update.", nil)
		return c.String(http.StatusBadRequest, "record not exist, cannot update!")
	}
	// Store in DB
	err = e.dataStore.SetResourceRecord(zone, &rr)
	if err != nil {
		log.Error("Failed to set the zone entries.", nil)
		return c.String(http.StatusInternalServerError, err.Error())
	}
	log.Debugf("Updated new resource record entry(zone: %s, name: %s, type: %s, class: %s, ttl: %d).",
		zone, rr.Name, rr.Type, rr.Class, rr.TTL)

	return c.String(http.StatusOK, "success in updating rr entry.")
}

func (e *Controller) validateSetRecordInput(zone string, rr *datastore.ResourceRecord) error {
	// Input Example:
	//{
	//	"name": "www.example.com.",
	//	"type": "A",
	//	"class": "IN",
	//	"ttl": 30,
	//	"rData": [
	//      "172.168.15.101"
	//     ]
	//}

	if len(zone) >= util.MaxDNSFQDNLength {
		return fmt.Errorf("invalid zone value")
	}

	if err := e.validateResourceRecords(rr); err != nil {
		return err
	}

	return nil
}

func (e *Controller) validateResourceRecords(rr *datastore.ResourceRecord) error {
	if rr.TTL == 0 ||
		len(rr.Name) == 0 || len(rr.Name) > util.MaxDNSFQDNLength || len(rr.RData) == 0 ||
		len(rr.Type) == 0 || len(rr.Type) > util.MaxDNSFQDNLength ||
		len(rr.Class) == 0 || len(rr.Class) > util.MaxDNSFQDNLength {
		return fmt.Errorf("invalid resource record value")
	}
	for _, rData := range rr.RData {
		mgmtIP := net.ParseIP(rData)
		if len(rData) == 0 ||
			len(rData) > util.MaxIPLength ||
			nil == mgmtIP || mgmtIP.IsMulticast() || mgmtIP.Equal(net.IPv4bcast) {
			return fmt.Errorf("invalid resource record value")
		}
	}

	return nil
}

func (e *Controller) handleDeleteResourceRecord(c echo.Context) error {
	zone := c.QueryParam("zone")
	fqdn := c.Param("fqdn")
	rrtype := c.Param("rrtype")

	if len(fqdn) == 0 || len(rrtype) == 0 || len(zone) >= util.MaxDNSFQDNLength {
		return c.String(http.StatusBadRequest, "invalid input parameters!")
	}

	if len(zone) == 0 {
		zone = "."
	}

	err := e.dataStore.DelResourceRecord(zone, fqdn, rrtype)
	if err != nil {
		log.Error("Failed to Delete Resource.", nil)
		return c.String(http.StatusInternalServerError, "Error in retrieving the data.")
	}

	return c.String(http.StatusOK, "Success")
}

func (e *Controller) handleHealthResult(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}
