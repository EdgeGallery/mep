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
package main

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/miekg/dns"

	"dns-server/datastore"
	"dns-server/mgmt"
	"dns-server/util"
)

// DNS server configuration
type Config struct {
	dbName            string // Database name, default zone
	port              uint   // Port to listen to, default 53
	mgmtPort          uint   // Http port to listen to, default 80
	ipAdd             net.IP // IP address to listen to, default 0.0.0.0
	ipMgmtAdd         net.IP // IP address to listen to, default 0.0.0.0
	forwarder         net.IP // Forwarder dns address , default 8.8.8.8
	connectionTimeout uint   // Connection time out value, both read, and write, default 2s
	loadBalance       bool   // load balancing using random shuffle
}

type Server struct {
	config    *Config
	dataStore datastore.DataStore
	mgmtCtl   mgmt.ManagementCtrl
	tcpServer *dns.Server
	udpServer *dns.Server
}

func NewServer(config *Config, dataStore datastore.DataStore, mgmtCtl mgmt.ManagementCtrl) *Server {
	return &Server{config: config, dataStore: dataStore, mgmtCtl: mgmtCtl}
}

func (s *Server) Run() error {

	// Set dns query handler
	dns.HandleFunc(".", s.handleDNS)

	address := fmt.Sprintf("%s:%d", s.config.ipAdd.String(), s.config.port)

	s.udpServer = &dns.Server{
		Addr:         address,
		Net:          "udp",
		UDPSize:      util.DNSUDPPacketSize,
		ReadTimeout:  time.Duration(s.config.connectionTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.connectionTimeout) * time.Second,
	}

	err := s.dataStore.Open()
	if err != nil {
		return err
	}

	go s.mgmtCtl.StartController(&s.dataStore, s.config.ipMgmtAdd, s.config.mgmtPort)
	go s.start(s.udpServer)
	return nil
}

func (s *Server) start(dns *dns.Server) {
	err := dns.ListenAndServe()
	if err != nil {
		log.Fatalf(err, "Failed to listen dns %s server on %s", dns.Net, dns.Addr)
	}
	log.Infof("Dns %s server now running on %s.", dns.Net, dns.Addr, dns.ReusePort)
}

func (s *Server) Stop() {
	err := s.dataStore.Close()
	if err != nil {
		log.Error("Failed to close the data store.", err)
	}

	if s.udpServer != nil {
		err := s.udpServer.Shutdown()
		if err != nil {
			log.Error("Failed to stop the dns udp server.", err)
		}
	}

	err = s.mgmtCtl.StopController()
	if err != nil {
		log.Fatal("Failed to stop the management controller", err)
	}

	log.Info("Edge-Gallery DNS-Server stopped now.")
}

// forward request to external server
func (s *Server) forward(req *dns.Msg) (*dns.Msg, error) {
	c := new(dns.Client)
	forwarder := s.config.forwarder.String()
	if forwarder == "0.0.0.0" {
		return nil, fmt.Errorf("could not resolve the request %q and no forwarder is configured",
			req.Question[0].Name)
	}

	// Retry 3 times on failure. exchange will not retry on failure.
	for i := 0; i < util.ForwardRetryCount; i++ {
		ret, _, err := c.Exchange(req, forwarder+":53")
		if err != nil {
			continue
		}
		if ret.Rcode == dns.RcodeSuccess {
			return ret, nil
		}
	}

	return nil, fmt.Errorf("forward of request %q was not accepted", req.Question[0].Name)
}

// Handle DNS Query matching
func (s *Server) handleDNS(w dns.ResponseWriter, req *dns.Msg) {

	if len(req.Question) != 1 {
		s.writeErrorResponse(w, req, dns.RcodeFormatError)
		return
	}
	if req.Opcode == dns.OpcodeQuery {
		log.Debugf("Query lookup (%s)", req.Question[0].String())
		// Match data from db
		rrs, err := s.dataStore.GetResourceRecord(&req.Question[0])
		if err != nil {
			respMsg, err := s.forward(req)
			if err != nil {
				s.writeErrorResponse(w, req, dns.RcodeServerFailure)
				log.Debugf("Failed to find entry: %v", err)
				return
			}
			err = w.WriteMsg(respMsg)
			if err != nil {
				log.Errorf(nil, "Failed to send a response for query")
			}
			return
		}
		// Shuffle the response if load balancing is enabled
		if s.config.loadBalance && len(*rrs) > 1 {
			rand.Shuffle(len(*rrs), func(i, j int) {
				(*rrs)[i], (*rrs)[j] = (*rrs)[j], (*rrs)[i]
			})
		}
		s.writeSuccessResponse(rrs, w, req)
		return
	} else {
		log.Debugf("Unsupported DNS OpCode %s", dns.OpcodeToString[req.Opcode])
		s.writeErrorResponse(w, req, dns.RcodeRefused)
		return
	}
}

func (s *Server) writeErrorResponse(w dns.ResponseWriter, req *dns.Msg, rc int) {
	response := new(dns.Msg)
	response.SetReply(req)
	response.SetRcode(req, rc)

	err := w.WriteMsg(response)
	if err != nil {
		log.Errorf(nil, "Failed to send error response for query")
	}
}

func (s *Server) writeSuccessResponse(answer *[]dns.RR, w dns.ResponseWriter, req *dns.Msg) {
	response := new(dns.Msg)
	response.Answer = *answer
	response.Authoritative = true
	response.SetReply(req)

	err := w.WriteMsg(response)
	if err != nil {
		log.Errorf(nil, "Failed to send success response for query")
	}
}
