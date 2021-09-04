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

// Package dns implements dns client
package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mepserver/common/config"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/apache/servicecomb-service-center/pkg/log"

	meputil "mepserver/common/util"
)

const ServerURLFormat = "http://%s:%d/mep/dns_server_mgmt/v1/"
const zone = "?zone=."
const contentTypeValue = "application/json; charset=utf-8"
const contentType = "Content-Type"

// ResourceRecord represents the dns resource record
type ResourceRecord struct {
	Name  string   `json:"name"`
	Type  string   `json:"type"`
	Class string   `json:"class"`
	TTL   uint32   `json:"ttl"`
	RData []string `json:"rData"`
}

// ZoneEntry represents the dns zone
type ZoneEntry struct {
	Zone string            `json:"zone"`
	RR   *[]ResourceRecord `json:"rr"`
}

// RestDNSAgent dns agent
type RestDNSAgent struct {
	DNSAgent
	ServerEndPoint *url.URL `json:"serverEndPoint"`
	client         http.Client
}

// NewRestDNSAgent creates and initialize a dns agent
func NewRestDNSAgent(*config.MepServerConfig) *RestDNSAgent {
	log.Info("New DNS agent initialization.")
	agent := RestDNSAgent{}
	err := agent.initDnsAgent()
	if err != nil {
		return &agent
	}
	return &agent
}

func (d *RestDNSAgent) initDnsAgent() error {
	var remoteServerHost = meputil.DefaultDnsHost
	var remoteServerPort = meputil.DefaultDnsManagementPort

	host := os.Getenv("DNS_SERVER_HOST")
	if len(host) > meputil.MaxFQDNLength {
		log.Warn("invalid dns remote server host configured, reset back to default")
	} else {
		remoteServerHost = host
	}

	port := os.Getenv("DNS_SERVER_PORT")
	if len(port) > meputil.MaxPortLength {
		log.Warn("Invalid dns remote server port configured, reset back to default.")
	} else if num, err := strconv.Atoi(port); err == nil {
		if num <= 0 || num > meputil.MaxPortNumber {
			log.Warn("Invalid dns remote server port range, reset back to default.")
		} else {
			remoteServerPort = num
		}
	}

	u, err := url.Parse(fmt.Sprintf(ServerURLFormat, remoteServerHost, remoteServerPort))
	if err != nil {
		log.Errorf(nil, "Could not parse the DNS server endpoint.")
		return err
	}
	d.ServerEndPoint = u
	return nil
}

// BuildDNSEndpoint generates the dns server endpoint
func (d *RestDNSAgent) BuildDNSEndpoint(paths ...string) string {
	return meputil.JoinURL(d.ServerEndPoint.String(), paths...)
}

// AddResourceRecord update a dns entry in dns server
func (d *RestDNSAgent) AddResourceRecord(host, rrType, class string, pointTo []string, ttl uint32) error {
	if d.ServerEndPoint == nil {
		log.Errorf(nil, "Invalid DNS remote end point in add.")
		return fmt.Errorf("invalid dns server endpoint in add")
	}

	hostName := host
	if !strings.HasSuffix(host, ".") {
		hostName = host + "."
	}

	rr := ResourceRecord{Name: hostName, Type: rrType, Class: class, TTL: ttl, RData: pointTo}
	rrJSON, err := json.Marshal(rr)
	if err != nil {
		log.Errorf(nil, "Marshal DNS info failed.")
		return err
	}

	httpReq, err := http.NewRequest(http.MethodPost, d.BuildDNSEndpoint("rrecord")+zone,
		bytes.NewBuffer(rrJSON))
	if err != nil {
		log.Errorf(nil, "Http request creation for DNS add failed.")
		return err
	}
	httpReq.Header.Set(contentType, contentTypeValue)

	httpResp, err := d.client.Do(httpReq)
	if err != nil {
		log.Errorf(nil, "Request to DNS server failed in add.")
		return err
	}
	if !meputil.IsHttpStatusOK(httpResp.StatusCode) {
		log.Errorf(nil, "DNS rule add failed on server(%d: %s).", httpResp.StatusCode, httpResp.Status)
		return fmt.Errorf("add request to dns server failed")
	}
	return nil
}

// SetResourceRecord update a dns entry in dns server
func (d *RestDNSAgent) SetResourceRecord(host, rrType, class string, pointTo []string, ttl uint32) error {
	if d.ServerEndPoint == nil {
		log.Errorf(nil, "Invalid DNS remote end point in modify.")
		return fmt.Errorf("invalid dns server endpoint in modify")
	}

	hostName := host
	if !strings.HasSuffix(host, ".") {
		hostName = host + "."
	}

	rr := ResourceRecord{Name: hostName, Type: rrType, Class: class, TTL: ttl, RData: pointTo}
	rrJSON, err := json.Marshal(rr)
	if err != nil {
		log.Errorf(nil, "Marshal DNS info failed.")
		return err
	}

	httpReq, err := http.NewRequest(http.MethodPut, d.BuildDNSEndpoint("rrecord", hostName, rrType)+zone,
		bytes.NewBuffer(rrJSON))
	if err != nil {
		log.Errorf(nil, "Http request creation for DNS update failed.")
		return err
	}
	httpReq.Header.Set(contentType, contentTypeValue)

	httpResp, err := d.client.Do(httpReq)
	if err != nil {
		log.Errorf(nil, "Request to DNS server failed in update.")
		return err
	}
	if !meputil.IsHttpStatusOK(httpResp.StatusCode) {
		log.Errorf(nil, "DNS rule update failed on server(%d: %s).", httpResp.StatusCode, httpResp.Status)
		return fmt.Errorf("update request to dns server failed")
	}
	return nil

}

// DeleteResourceRecord deletes an entry from dns server
func (d *RestDNSAgent) DeleteResourceRecord(host, rrtype string) error {
	if d.ServerEndPoint == nil {
		log.Errorf(nil, "Invalid DNS remote end point in delete.")
		return fmt.Errorf("invalid dns server endpoint in delete")
	}
	hostName := host
	if !strings.HasSuffix(host, ".") {
		hostName = host + "."
	}

	httpReq, err := http.NewRequest(http.MethodDelete, d.BuildDNSEndpoint("rrecord", hostName, rrtype)+zone,
		bytes.NewBuffer([]byte("{}")))
	if err != nil {
		log.Errorf(nil, "Http request creation for DNS delete failed.")
		return err
	}
	httpReq.Header.Set(contentType, contentTypeValue)

	httpResp, err := d.client.Do(httpReq)
	if err != nil {
		log.Errorf(nil, "Request to DNS server failed in delete.")
		return err
	}
	if !meputil.IsHttpStatusOK(httpResp.StatusCode) {
		log.Errorf(nil, "DNS rule delete failed on server(%d: %s).", httpResp.StatusCode, httpResp.Status)
		return fmt.Errorf("delete request to dns server failed")
	}
	return nil
}
