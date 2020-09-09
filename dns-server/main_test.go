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
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"dns-server/datastore"
	"dns-server/mgmt"
	"dns-server/util"
)

var dbName = "test_db"
var port uint = util.DefaultDnsPort
var mgmtPort uint = util.DefaultManagementPort
var connTimeOut uint = util.DefaultConnTimeout
var ipAddString = util.DefaultIP
var ipMgmtAddString = util.DefaultIP
var forwarder = util.DefaultIP
var loadBalance = false
var epanic = "Panic expected"
var eerror = "Error expected"
var panicProblem = "a problem"
var finish = "Finished processing"

func TestMainDnsServer(t *testing.T) {
	var panicString = "Panic: "
	defer func() {
		_ = os.RemoveAll(datastore.DBPath)
	}()

	var s *Server
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Run", func(*Server) error {
		go func() {
			time.Sleep(100 * time.Millisecond)
			log.Fatal(finish)
		}()
		return nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Stop", func(*Server) {
		return
	})
	defer patch2.Reset()

	server := Server{}
	patch3 := gomonkey.ApplyFunc(NewServer, func(config *Config, dataStore datastore.DataStore,
		mgmtCtl mgmt.ManagementCtrl) *Server {
		return &server
	})
	defer patch3.Reset()

	patch4 := gomonkey.ApplyFunc(waitForSignal, func() { // Empty Impl
	})
	defer patch4.Reset()

	patch6 := gomonkey.ApplyFunc(os.Exit, func(code int) { // Empty Impl
		assert.Equal(t, 1, code, eerror)
		panic(panicProblem)
	})
	defer patch6.Reset()

	t.Run("ManagementPortNumber0", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s", epanic)
			}
			if r != panicProblem {
				t.Errorf("%s %v", panicString, r)
			}
		}()
		var invalidPortNo uint = 0
		parameters := InputParameters{&dbName, &port, &invalidPortNo, &connTimeOut,
			&ipAddString, &ipMgmtAddString, &forwarder, &loadBalance}

		patch5 := gomonkey.ApplyFunc(registerInputParameters, func(inParam *InputParameters) {
			inParam.dbName = parameters.dbName
			inParam.port = parameters.port
			inParam.mgmtPort = parameters.mgmtPort
			inParam.connTimeOut = parameters.connTimeOut
			inParam.ipAddString = parameters.ipAddString
			inParam.ipMgmtAddString = parameters.ipMgmtAddString
			inParam.forwarder = parameters.forwarder
			inParam.loadBalance = parameters.loadBalance
			return
		})
		defer patch5.Reset()

		main()
	})
	t.Run("ManagementPortNumberMaxCheck", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s", epanic)
			}
			if r != panicProblem {
				t.Errorf("%s %v", panicString, r)
			}
		}()
		var invalidPortNo uint = 65536
		parameters := InputParameters{&dbName, &port, &invalidPortNo, &connTimeOut,
			&ipAddString, &ipMgmtAddString, &forwarder, &loadBalance}

		patch5 := gomonkey.ApplyFunc(registerInputParameters, func(inParam *InputParameters) {
			inParam.dbName = parameters.dbName
			inParam.port = parameters.port
			inParam.mgmtPort = parameters.mgmtPort
			inParam.connTimeOut = parameters.connTimeOut
			inParam.ipAddString = parameters.ipAddString
			inParam.ipMgmtAddString = parameters.ipMgmtAddString
			inParam.forwarder = parameters.forwarder
			inParam.loadBalance = parameters.loadBalance
			return
		})
		defer patch5.Reset()

		main()
	})
}

func TestMainDnsServer1(t *testing.T) {
	var panicSting = "Panic :"
	defer func() {
		_ = os.RemoveAll(datastore.DBPath)
	}()

	var s *Server
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Run", func(*Server) error {
		go func() {
			time.Sleep(100 * time.Millisecond)
			log.Fatal(finish)
		}()
		return nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Stop", func(*Server) {
		return
	})
	defer patch2.Reset()

	server := Server{}
	patch3 := gomonkey.ApplyFunc(NewServer, func(config *Config, dataStore datastore.DataStore,
		mgmtCtl mgmt.ManagementCtrl) *Server {
		return &server
	})
	defer patch3.Reset()

	patch4 := gomonkey.ApplyFunc(waitForSignal, func() { // Empty Impl
	})
	defer patch4.Reset()

	patch6 := gomonkey.ApplyFunc(os.Exit, func(code int) { // Empty Impl
		assert.Equal(t, 1, code, eerror)
		panic(panicProblem)
	})
	defer patch6.Reset()

	t.Run("ForwardIPAddressParsing", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s", epanic)
			}
			if r != panicProblem {
				t.Errorf("%s %v", panicSting, r)
			}
		}()
		var invalidIpAdd = "127.0.0.256"
		parameters := InputParameters{&dbName, &port, &mgmtPort, &connTimeOut,
			&ipAddString, &ipMgmtAddString, &invalidIpAdd, &loadBalance}

		patch5 := gomonkey.ApplyFunc(registerInputParameters, func(inParam *InputParameters) {
			inParam.dbName = parameters.dbName
			inParam.port = parameters.port
			inParam.mgmtPort = parameters.mgmtPort
			inParam.connTimeOut = parameters.connTimeOut
			inParam.ipAddString = parameters.ipAddString
			inParam.ipMgmtAddString = parameters.ipMgmtAddString
			inParam.forwarder = parameters.forwarder
			inParam.loadBalance = parameters.loadBalance
			return
		})
		defer patch5.Reset()

		main()
	})

	t.Run("SamePortValidation", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s", epanic)
			}
			if r != panicProblem {
				t.Errorf("%s %v", panicSting, r)
			}
		}()
		parameters := InputParameters{&dbName, &port, &port, &connTimeOut,
			&ipAddString, &ipMgmtAddString, &forwarder, &loadBalance}

		patch5 := gomonkey.ApplyFunc(registerInputParameters, func(inParam *InputParameters) {
			inParam.dbName = parameters.dbName
			inParam.port = parameters.port
			inParam.mgmtPort = parameters.mgmtPort
			inParam.connTimeOut = parameters.connTimeOut
			inParam.ipAddString = parameters.ipAddString
			inParam.ipMgmtAddString = parameters.ipMgmtAddString
			inParam.forwarder = parameters.forwarder
			inParam.loadBalance = parameters.loadBalance
			return
		})
		defer patch5.Reset()

		main()
	})
}

func TestMainDnsServer2(t *testing.T) {
	var panicSting = "Panic:"
	defer func() {
		_ = os.RemoveAll(datastore.DBPath)
	}()

	var s *Server
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Run", func(*Server) error {
		go func() {
			time.Sleep(100 * time.Millisecond)
			log.Fatal(finish)
		}()
		return nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Stop", func(*Server) {
		return
	})
	defer patch2.Reset()

	server := Server{}
	patch3 := gomonkey.ApplyFunc(NewServer, func(config *Config, dataStore datastore.DataStore,
		mgmtCtl mgmt.ManagementCtrl) *Server {
		return &server
	})
	defer patch3.Reset()

	patch4 := gomonkey.ApplyFunc(waitForSignal, func() { // Empty Impl
	})
	defer patch4.Reset()

	patch6 := gomonkey.ApplyFunc(os.Exit, func(code int) { // Empty Impl
		assert.Equal(t, 1, code, eerror)
		panic(panicProblem)
	})
	defer patch6.Reset()

	t.Run("InvalidDbName", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s", epanic)
			}
			if r != panicProblem {
				t.Errorf("%s %v", panicSting, r)
			}
		}()

		var invalidDbName = "test.db"
		parameters := InputParameters{&invalidDbName, &port, &mgmtPort, &connTimeOut,
			&ipAddString, &ipMgmtAddString, &forwarder, &loadBalance}

		patch5 := gomonkey.ApplyFunc(registerInputParameters, func(inParam *InputParameters) {
			inParam.dbName = parameters.dbName
			inParam.port = parameters.port
			inParam.mgmtPort = parameters.mgmtPort
			inParam.connTimeOut = parameters.connTimeOut
			inParam.ipAddString = parameters.ipAddString
			inParam.ipMgmtAddString = parameters.ipMgmtAddString
			inParam.forwarder = parameters.forwarder
			inParam.loadBalance = parameters.loadBalance
			return
		})
		defer patch5.Reset()

		main()
	})

	t.Run("ManagementIPAddressParsing", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s", epanic)
			}
			if r != panicProblem {
				t.Errorf("%s %v", panicSting, r)
			}
		}()
		var invalidIpAdd = "127.0.0.256"
		parameters := InputParameters{&dbName, &port, &mgmtPort, &connTimeOut,
			&ipAddString, &invalidIpAdd, &forwarder, &loadBalance}

		patch5 := gomonkey.ApplyFunc(registerInputParameters, func(inParam *InputParameters) {
			inParam.dbName = parameters.dbName
			inParam.port = parameters.port
			inParam.mgmtPort = parameters.mgmtPort
			inParam.connTimeOut = parameters.connTimeOut
			inParam.ipAddString = parameters.ipAddString
			inParam.ipMgmtAddString = parameters.ipMgmtAddString
			inParam.forwarder = parameters.forwarder
			inParam.loadBalance = parameters.loadBalance
			return
		})
		defer patch5.Reset()

		main()
	})
}

func TestMainDnsServer3(t *testing.T) {
	var panicSting = " Panic:"
	defer func() {
		_ = os.RemoveAll(datastore.DBPath)
	}()

	var s *Server
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Run", func(*Server) error {
		go func() {
			time.Sleep(100 * time.Millisecond)
			log.Fatal(finish)
		}()
		return nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Stop", func(*Server) {
		return
	})
	defer patch2.Reset()

	server := Server{}
	patch3 := gomonkey.ApplyFunc(NewServer, func(config *Config, dataStore datastore.DataStore,
		mgmtCtl mgmt.ManagementCtrl) *Server {
		return &server
	})
	defer patch3.Reset()

	patch4 := gomonkey.ApplyFunc(waitForSignal, func() { // Empty Impl
	})
	defer patch4.Reset()

	patch6 := gomonkey.ApplyFunc(os.Exit, func(code int) { // Empty Impl
		assert.Equal(t, 1, code, eerror)
		panic(panicProblem)
	})
	defer patch6.Reset()

	t.Run("DnsIPAddressParsing3", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s", epanic)
			}
			if r != panicProblem {
				t.Errorf("%s %v", panicSting, r)
			}
		}()
		var invalidIpAdd = "128.15.47.299"
		parameters := InputParameters{&dbName, &port, &mgmtPort, &connTimeOut,
			&invalidIpAdd, &ipMgmtAddString, &forwarder, &loadBalance}

		patch5 := gomonkey.ApplyFunc(registerInputParameters, func(inParam *InputParameters) {
			inParam.dbName = parameters.dbName
			inParam.port = parameters.port
			inParam.mgmtPort = parameters.mgmtPort
			inParam.connTimeOut = parameters.connTimeOut
			inParam.ipAddString = parameters.ipAddString
			inParam.ipMgmtAddString = parameters.ipMgmtAddString
			inParam.forwarder = parameters.forwarder
			inParam.loadBalance = parameters.loadBalance
			return
		})
		defer patch5.Reset()

		main()
	})
	t.Run("DnsIPAddressParsing4", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s", epanic)
			}
			if r != panicProblem {
				t.Errorf("%s %v", panicSting, r)
			}
		}()
		var invalidIpAdd = "1::2lkh"
		parameters := InputParameters{&dbName, &port, &mgmtPort, &connTimeOut,
			&invalidIpAdd, &ipMgmtAddString, &forwarder, &loadBalance}

		patch5 := gomonkey.ApplyFunc(registerInputParameters, func(inParam *InputParameters) {
			inParam.dbName = parameters.dbName
			inParam.port = parameters.port
			inParam.mgmtPort = parameters.mgmtPort
			inParam.connTimeOut = parameters.connTimeOut
			inParam.ipAddString = parameters.ipAddString
			inParam.ipMgmtAddString = parameters.ipMgmtAddString
			inParam.forwarder = parameters.forwarder
			inParam.loadBalance = parameters.loadBalance
			return
		})
		defer patch5.Reset()

		main()
	})
}

func TestMainDnsServer4(t *testing.T) {
	var panicSting = "Panic : "
	defer func() {
		_ = os.RemoveAll(datastore.DBPath)
	}()

	var s *Server
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Run", func(*Server) error {
		go func() {
			time.Sleep(100 * time.Millisecond)
			log.Fatal(finish)
		}()
		return nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Stop", func(*Server) {
		return
	})
	defer patch2.Reset()

	server := Server{}
	patch3 := gomonkey.ApplyFunc(NewServer, func(config *Config, dataStore datastore.DataStore,
		mgmtCtl mgmt.ManagementCtrl) *Server {
		return &server
	})
	defer patch3.Reset()

	patch4 := gomonkey.ApplyFunc(waitForSignal, func() { // Empty Impl
	})
	defer patch4.Reset()

	patch6 := gomonkey.ApplyFunc(os.Exit, func(code int) { // Empty Impl
		assert.Equal(t, 1, code, eerror)
		panic(panicProblem)
	})
	defer patch6.Reset()

	t.Run("DnsIPAddressParsing1", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s", epanic)
			}
			if r != panicProblem {
				t.Errorf("%s %v", panicSting, r)
			}
		}()
		var invalidIpAdd = ""
		parameters := InputParameters{&dbName, &port, &mgmtPort, &connTimeOut,
			&invalidIpAdd, &ipMgmtAddString, &forwarder, &loadBalance}

		patch5 := gomonkey.ApplyFunc(registerInputParameters, func(inParam *InputParameters) {
			inParam.dbName = parameters.dbName
			inParam.port = parameters.port
			inParam.mgmtPort = parameters.mgmtPort
			inParam.connTimeOut = parameters.connTimeOut
			inParam.ipAddString = parameters.ipAddString
			inParam.ipMgmtAddString = parameters.ipMgmtAddString
			inParam.forwarder = parameters.forwarder
			inParam.loadBalance = parameters.loadBalance
			return
		})
		defer patch5.Reset()

		main()
	})
	t.Run("DnsIPAddressParsing2", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s", epanic)
			}
			if r != panicProblem {
				t.Errorf("%s %v", panicSting, r)
			}
		}()
		var invalidIpAdd = "a"
		parameters := InputParameters{&dbName, &port, &mgmtPort, &connTimeOut,
			&invalidIpAdd, &ipMgmtAddString, &forwarder, &loadBalance}

		patch5 := gomonkey.ApplyFunc(registerInputParameters, func(inParam *InputParameters) {
			inParam.dbName = parameters.dbName
			inParam.port = parameters.port
			inParam.mgmtPort = parameters.mgmtPort
			inParam.connTimeOut = parameters.connTimeOut
			inParam.ipAddString = parameters.ipAddString
			inParam.ipMgmtAddString = parameters.ipMgmtAddString
			inParam.forwarder = parameters.forwarder
			inParam.loadBalance = parameters.loadBalance
			return
		})
		defer patch5.Reset()

		main()
	})
}

func TestMainDnsServer5(t *testing.T) {
	var panicSting = " Panic :"
	defer func() {
		_ = os.RemoveAll(datastore.DBPath)
	}()

	var s *Server
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Run", func(*Server) error {
		go func() {
			time.Sleep(100 * time.Millisecond)
			log.Fatal(finish)
		}()
		return nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Stop", func(*Server) {
		return
	})
	defer patch2.Reset()

	server := Server{}
	patch3 := gomonkey.ApplyFunc(NewServer, func(config *Config, dataStore datastore.DataStore,
		mgmtCtl mgmt.ManagementCtrl) *Server {
		return &server
	})
	defer patch3.Reset()

	patch4 := gomonkey.ApplyFunc(waitForSignal, func() { // Empty Impl
	})
	defer patch4.Reset()

	patch6 := gomonkey.ApplyFunc(os.Exit, func(code int) { // Empty Impl
		assert.Equal(t, 1, code, eerror)
		panic(panicProblem)
	})
	defer patch6.Reset()

	t.Run("DefaultParameters", func(t *testing.T) {
		defer func() {
			r := recover()
			if r != nil {
				t.Errorf("%s %v", panicSting, r)
			}
		}()
		parameters := InputParameters{&dbName, &port, &mgmtPort, &connTimeOut,
			&ipAddString, &ipMgmtAddString, &forwarder, &loadBalance}

		patch5 := gomonkey.ApplyFunc(registerInputParameters, func(inParam *InputParameters) {
			inParam.dbName = parameters.dbName
			inParam.port = parameters.port
			inParam.mgmtPort = parameters.mgmtPort
			inParam.connTimeOut = parameters.connTimeOut
			inParam.ipAddString = parameters.ipAddString
			inParam.ipMgmtAddString = parameters.ipMgmtAddString
			inParam.forwarder = parameters.forwarder
			inParam.loadBalance = parameters.loadBalance
			return
		})
		defer patch5.Reset()

		main()
	})

	t.Run("MaxLengthDbName", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s", epanic)
			}
			if r != panicProblem {
				t.Errorf("%s %v", panicSting, r)
			}
		}()

		var invalidDbName = "qwertyuiopqwertyuiopqwertyuiopqwertyuiopqwertyuiopqwertyuiopqwertyuiopqwertyuiopqwertyuiop" +
			"qwertyuiopqwertyuiopqwertyuiopqwertyuiopqwertyuiopqwertyuiopqwertyuiopqwertyuiopqwertyuiopqwertyuiop" +
			"qwertyuiopqwertyuiopqwertyuiopqwertyuiopqwertyuiopqwertyuiopqwertyuiop"

		parameters := InputParameters{&invalidDbName, &port, &mgmtPort, &connTimeOut,
			&ipAddString, &ipMgmtAddString, &forwarder, &loadBalance}

		patch5 := gomonkey.ApplyFunc(registerInputParameters, func(inParam *InputParameters) {
			inParam.dbName = parameters.dbName
			inParam.port = parameters.port
			inParam.mgmtPort = parameters.mgmtPort
			inParam.connTimeOut = parameters.connTimeOut
			inParam.ipAddString = parameters.ipAddString
			inParam.ipMgmtAddString = parameters.ipMgmtAddString
			inParam.forwarder = parameters.forwarder
			inParam.loadBalance = parameters.loadBalance
			return
		})
		defer patch5.Reset()

		main()
	})
}

func TestMainDnsServer6(t *testing.T) {
	var panicSting = " Panic: "
	defer func() {
		_ = os.RemoveAll(datastore.DBPath)
	}()

	var s *Server
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Run", func(*Server) error {
		go func() {
			time.Sleep(100 * time.Millisecond)
			log.Fatal(finish)
		}()
		return nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Stop", func(*Server) {
		return
	})
	defer patch2.Reset()

	server := Server{}
	patch3 := gomonkey.ApplyFunc(NewServer, func(config *Config, dataStore datastore.DataStore,
		mgmtCtl mgmt.ManagementCtrl) *Server {
		return &server
	})
	defer patch3.Reset()

	patch4 := gomonkey.ApplyFunc(waitForSignal, func() { // Empty Impl
	})
	defer patch4.Reset()

	patch6 := gomonkey.ApplyFunc(os.Exit, func(code int) { // Empty Impl
		assert.Equal(t, 1, code, eerror)
		panic(panicProblem)
	})
	defer patch6.Reset()

	t.Run("PortNumber0", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s", epanic)
			}
			if r != panicProblem {
				t.Errorf("%s %v", panicSting, r)
			}
		}()
		var invalidPortNo uint = 0
		parameters := InputParameters{&dbName, &invalidPortNo, &mgmtPort, &connTimeOut,
			&ipAddString, &ipMgmtAddString, &forwarder, &loadBalance}

		patch5 := gomonkey.ApplyFunc(registerInputParameters, func(inParam *InputParameters) {
			inParam.dbName = parameters.dbName
			inParam.port = parameters.port
			inParam.mgmtPort = parameters.mgmtPort
			inParam.connTimeOut = parameters.connTimeOut
			inParam.ipAddString = parameters.ipAddString
			inParam.ipMgmtAddString = parameters.ipMgmtAddString
			inParam.forwarder = parameters.forwarder
			inParam.loadBalance = parameters.loadBalance
			return
		})
		defer patch5.Reset()

		main()
	})

	t.Run("PortNumberMaxCheck", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s", epanic)
			}
			if r != panicProblem {
				t.Errorf("%s %v", panicSting, r)
			}
		}()
		var invalidPortNo uint = 65536
		parameters := InputParameters{&dbName, &invalidPortNo, &mgmtPort, &connTimeOut,
			&ipAddString, &ipMgmtAddString, &forwarder, &loadBalance}

		patch5 := gomonkey.ApplyFunc(registerInputParameters, func(inParam *InputParameters) {
			inParam.dbName = parameters.dbName
			inParam.port = parameters.port
			inParam.mgmtPort = parameters.mgmtPort
			inParam.connTimeOut = parameters.connTimeOut
			inParam.ipAddString = parameters.ipAddString
			inParam.ipMgmtAddString = parameters.ipMgmtAddString
			inParam.forwarder = parameters.forwarder
			inParam.loadBalance = parameters.loadBalance
			return
		})
		defer patch5.Reset()

		main()
	})

}

func TestMainDnsServer7(t *testing.T) {
	var panicSting = " Panic : "
	defer func() {
		_ = os.RemoveAll(datastore.DBPath)
	}()

	var s *Server
	patch1 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Run", func(*Server) error {
		go func() {
			time.Sleep(100 * time.Millisecond)
			log.Fatal(finish)
		}()
		return nil
	})
	defer patch1.Reset()

	patch2 := gomonkey.ApplyMethod(reflect.TypeOf(s), "Stop", func(*Server) {
		return
	})
	defer patch2.Reset()

	server := Server{}
	patch3 := gomonkey.ApplyFunc(NewServer, func(config *Config, dataStore datastore.DataStore,
		mgmtCtl mgmt.ManagementCtrl) *Server {
		return &server
	})
	defer patch3.Reset()

	patch4 := gomonkey.ApplyFunc(waitForSignal, func() { // Empty Impl
	})
	defer patch4.Reset()

	patch6 := gomonkey.ApplyFunc(os.Exit, func(code int) { // Empty Impl
		assert.Equal(t, 1, code, eerror)
		panic(panicProblem)
	})
	defer patch6.Reset()

	t.Run("InvalidConnectionTimeoutValue", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s", epanic)
			}
			if r != panicProblem {
				t.Errorf("%s %v", panicSting, r)
			}
		}()
		var invalidConnT uint = 0
		parameters := InputParameters{&dbName, &port, &mgmtPort, &invalidConnT,
			&ipAddString, &ipMgmtAddString, &forwarder, &loadBalance}

		patch5 := gomonkey.ApplyFunc(registerInputParameters, func(inParam *InputParameters) {
			inParam.dbName = parameters.dbName
			inParam.port = parameters.port
			inParam.mgmtPort = parameters.mgmtPort
			inParam.connTimeOut = parameters.connTimeOut
			inParam.ipAddString = parameters.ipAddString
			inParam.ipMgmtAddString = parameters.ipMgmtAddString
			inParam.forwarder = parameters.forwarder
			inParam.loadBalance = parameters.loadBalance
			return
		})
		defer patch5.Reset()

		main()
	})
}
