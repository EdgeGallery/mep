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
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"

	"dns-server/datastore"
	"dns-server/mgmt"
	"dns-server/util"
)

// Input placeholder
type InputParameters struct {
	dbName          *string // DB name placeholder
	port            *uint   // dns port number
	mgmtPort        *uint   // management interface port number
	connTimeOut     *uint   // connection time out value
	ipAddString     *string // dns listening ip
	ipMgmtAddString *string // management interface listening ip
	forwarder       *string // forwarder ip address
	loadBalance     *bool   // need load balancing?
}

// Input flag parameters registration
func registerInputParameters(inParam *InputParameters) {
	if inParam == nil {
		log.Fatalf( "Input config is not ready yet.")
		return
	}
	inParam.dbName = flag.String("db", "dbEgDns", "Database name")
	inParam.port = flag.Uint("port", util.DefaultDnsPort, "Port number to listens to")
	inParam.mgmtPort = flag.Uint("managementPort", util.DefaultManagementPort,
		"Management interface port number to listens to")
	inParam.connTimeOut = flag.Uint("connectionTimeout", util.DefaultConnTimeout,
		"Connection timeout(Read & Write) in seconds(2~50)")
	inParam.ipAddString = flag.String("ipAdd", util.DefaultIP, "Ipv4/Ipv6 address to listens to")
	inParam.ipMgmtAddString = flag.String("managementIpAdd", util.DefaultIP,
		"Management Ipv4/Ipv6 address to listens to")
	inParam.forwarder = flag.String("forwarder", util.DefaultIP, "Forwarder")
	inParam.loadBalance = flag.Bool("loadBalance", false, "Load balance using random shuffle")

	flag.Parse()
}

// Input parameter validation, parsing and generating configuration for running the dns server
func validateInputAndGenerateConfig(inParam *InputParameters) *Config {
	// Validate db name
	if len(*inParam.dbName) >= util.MaxDbNameLength {
		err := fmt.Errorf("error: db name should be less than 256")
		log.Fatalf("Failed to parse db name(%s).", err.Error())
	}
	if strings.ContainsAny(*inParam.dbName, util.DbStringExceptions) {
		err := fmt.Errorf("error: db name should be a single world and should not have \"%s\"",
			util.DbStringExceptions)
		log.Fatalf( "Failed to parse db name(%s). %s", *inParam.dbName, err.Error())
	}

	// Validate DNS port range
	if *inParam.port > util.MaxPortNumber || *inParam.port == 0 {
		err := fmt.Errorf("error: port number not in valid range")
		log.Fatalf( "Failed to parse port number(%s).", err.Error())
	}

	// Validate DNS management port range
	if *inParam.mgmtPort > util.MaxPortNumber || *inParam.mgmtPort == 0 {
		err := fmt.Errorf("error: management port number not in valid range")
		log.Fatalf( "Failed to parse management port number(%s).", err.Error())
	}
	if *inParam.port == *inParam.mgmtPort {
		err := fmt.Errorf("error: cannot use same port number for dns and management")
		log.Fatalf( "Port number conflict(%s).", err.Error())
	}

	// Validate connTimeOut range
	if *inParam.connTimeOut > util.MaxConnTimeout || *inParam.connTimeOut < util.MinConnTimeout {
		err := fmt.Errorf("error: connection timeout not in valid range(2~50)")
		log.Fatalf( "Failed to parse connection timeout input(%s).", err.Error())
	}

	// Validate IP address
	ipAdd := net.ParseIP(*inParam.ipAddString)
	if ipAdd == nil  {
		err := fmt.Errorf("error: parsing ip address failed, not in ipv4/ipv6 format")
		log.Fatalf( "Failed to parse ip address(%s). %s", *inParam.ipAddString, err.Error())
	}

	if ipAdd.IsMulticast() || ipAdd.Equal(net.IPv4bcast) {
		err := fmt.Errorf("error: multicast or broadcast ip address ")
		log.Fatalf( "Multicast or broadcast ip addresss(%s). %s", *inParam.ipAddString, err.Error())
	}

	// Validate Management IP address
	ipMgmtAdd := net.ParseIP(*inParam.ipMgmtAddString)
	if ipMgmtAdd == nil {
		err := fmt.Errorf("error: parsing management ip address failed, not in ipv4/ipv6 format")
		log.Fatalf( "Failed to parse management ip address(%s). %s", *inParam.ipMgmtAddString, err.Error())
	}

	if ipMgmtAdd.IsMulticast() || ipMgmtAdd.Equal(net.IPv4bcast) {
		err := fmt.Errorf("error: multicast or broadcast ip address ")
		log.Fatalf( "Multicast or broadcast ip addresss(%s). %s", *inParam.ipMgmtAddString, err.Error())
	}

	// Validate forwarder
	forwarderAdd := net.ParseIP(*inParam.forwarder)
	if forwarderAdd == nil {
		err := fmt.Errorf("error: parsing forwarder failed, not in ipv4/ipv6 format")
		log.Fatalf( "Failed to parse forwarder address(%s). %s", *inParam.forwarder, err.Error())
	}

	if forwarderAdd.IsMulticast() || forwarderAdd.Equal(net.IPv4bcast) {
		err := fmt.Errorf("error: multicast or broadcast ip address ")
		log.Fatalf( "Multicast or broadcast ip addresss(%s). %s", *inParam.forwarder, err.Error())
	}

	return &Config{dbName: *inParam.dbName,
		port:              *inParam.port,
		mgmtPort:          *inParam.mgmtPort,
		ipAdd:             ipAdd,
		ipMgmtAdd:         ipMgmtAdd,
		connectionTimeout: *inParam.connTimeOut,
		forwarder:         forwarderAdd,
		loadBalance:       *inParam.loadBalance,
	}
}

func waitForSignal() {
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case s := <-sig:
			log.Infof("Signal(%d) received, stopping dns server\n", s)
			os.Exit(0)
		}
	}
}

func main() {
	log.Info("Starting Edge-Gallery DNS-Server.")

	inputParam := &InputParameters{}
	// Register input flag parameters
	registerInputParameters(inputParam)

	config := validateInputAndGenerateConfig(inputParam)

	store := &datastore.BoltDB{FileName: config.dbName, TTL: util.DefaultTTL}
	mgmtCtl := &mgmt.Controller{}
	dnsServer := NewServer(config, store, mgmtCtl)

	err := dnsServer.Run()
	defer dnsServer.Stop()
	if err != nil {
		log.Fatal("Failed to Start the DNS server.", err)
	}

	log.Info("DNS server started successfully.")
	waitForSignal()
}
