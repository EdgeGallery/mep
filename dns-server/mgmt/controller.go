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
	"fmt"
	"net"
	"net/http"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"dns-server/datastore"
)

type Controller struct {
	dataStore datastore.DataStore
	echo      *echo.Echo
}

func (e *Controller) StartController(store *datastore.DataStore, ipAddr net.IP, port uint) {
	// Echo instance
	e.echo = echo.New()

	// Middleware
	e.echo.Use(middleware.Logger())
	e.echo.Use(middleware.Recover())

	// Routes
	e.echo.PUT("/mep/dns_server_mgmt/v1/rrecord", e.handleSetResourceRecords)
	e.echo.DELETE("/mep/dns_server_mgmt/v1/rrecord/:fqdn/:rrtype", e.handleDeleteResourceRecord)

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

func (e *Controller) handleSetResourceRecords(c echo.Context) error {
	zrs := new([]datastore.ZoneEntry)
	if err := c.Bind(zrs); err != nil {
		log.Error("Error in parsing the rr post request body.", nil)
		return c.String(http.StatusBadRequest, "invalid input!")
	}

	// Store in DB
	for _, zr := range *zrs {
		if len(zr.Zone) == 0 {
			zr.Zone = "."
		}
		for _, rr := range *zr.RR {
			err := e.dataStore.SetResourceRecord(zr.Zone, &rr)
			if err != nil {
				log.Error("Failed to set the zone entries.", nil)
				return c.String(http.StatusInternalServerError, err.Error())
			}
			log.Debugf("New resource record entry(zone: %s, name: %s, type: %s, class: %s, ttl: %d).",
				zr.Zone, rr.Name, rr.Type, rr.Class, rr.TTL)
		}
	}

	return c.String(http.StatusOK, "success in adding/updating rr entry.")

}

func (e *Controller) handleDeleteResourceRecord(c echo.Context) error {
	fqdn := c.Param("fqdn")
	rrtype := c.Param("rrtype")
	if len(fqdn) == 0 || len(rrtype) == 0 {
		return c.String(http.StatusBadRequest, "invalid input parameters!")
	}
	err := e.dataStore.DelResourceRecord(fqdn, rrtype)
	if err != nil {
		log.Error("Failed to get the zone entry.", nil)
		return c.String(http.StatusInternalServerError, "Error in retrieving the data.")
	}
	return c.String(http.StatusOK, "Success")
}
