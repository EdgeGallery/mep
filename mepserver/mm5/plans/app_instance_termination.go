/*
 * Copyright 2021 Huawei Technologies Co., Ltd.
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

package plans

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"mepserver/common/arch/workspace"
	"mepserver/common/extif/backend"
	"mepserver/common/models"
	meputil "mepserver/common/util"
	"mepserver/mm5/task"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"github.com/apache/servicecomb-service-center/server/core"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	scerr "github.com/apache/servicecomb-service-center/server/error"
)

type DecodeAppTerminationReq struct {
	workspace.TaskBase
	R             *http.Request   `json:"r,in"`
	Ctx           context.Context `json:"ctx,out"`
	AppInstanceId string          `json:"appInstanceId,out"`
}

// OnRequest discover decode request
func (t *DecodeAppTerminationReq) OnRequest(data string) workspace.TaskCode {
	var err error
	log.Infof("Received message from ClientIP [%s] AppInstanceId [%s] Operation [%s] Resource [%s].",
		meputil.GetClientIp(t.R), meputil.GetAppInstanceId(t.R), meputil.GetMethod(t.R), meputil.GetHttpResourceInfo(t.R))
	t.Ctx, err = t.GetFindParam(t.R)
	if err != nil {
		log.Error("Parameters validation failed.", err)
		return workspace.TaskFinish
	}
	log.Debugf("Query request arrived to fetch the app Id.")
	return workspace.TaskFinish
}

// GetFindParam get find param by request
func (t *DecodeAppTerminationReq) GetFindParam(r *http.Request) (context.Context, error) {

	query, _ := meputil.GetHTTPTags(r)

	t.AppInstanceId = query.Get(":appInstanceId")
	if err := meputil.ValidateAppInstanceIdWithHeader(t.AppInstanceId, r); err != nil {
		log.Error("Validate X-AppInstanceId failed.", err)
		t.SetFirstErrorCode(meputil.AuthorizationValidateErr, err.Error())
		return nil, err
	}
	Ctx := util.SetTargetDomainProject(r.Context(), r.Header.Get("X-Domain-Name"), query.Get(":project"))
	return Ctx, nil
}

type DeleteService struct {
	workspace.TaskBase
	Ctx           context.Context `json:"ctx,in"`
	HttpErrInf    *proto.Response `json:"httpErrInf,out"`
	HttpRsp       interface{}     `json:"httpRsp,out"`
	AppInstanceId string          `json:"appInstanceId,in"`
}

// OnRequest handles the service deletion
func (t *DeleteService) OnRequest(data string) workspace.TaskCode {
	log.Info("Application termination request.")
	resp, errInt := backend.GetRecords("/cse-sr/inst/files///")
	if errInt != 0 {
		log.Errorf(nil, "Data store read failed.")
		t.SetFirstErrorCode(meputil.OperateDataWithEtcdErr, "query error from etcd")
		return workspace.TaskFinish
	}
	var findResp []*proto.MicroServiceInstance
	for _, value := range resp {
		var instances map[string]interface{}
		err := json.Unmarshal(value, &instances)
		if err != nil {
			log.Errorf(nil, "Instance unmarshall failed.")
			t.SetFirstErrorCode(meputil.ParseInfoErr, err.Error())
			return workspace.TaskFinish
		}
		dci := &proto.DataCenterInfo{Name: "", Region: "", AvailableZone: ""}
		instances[meputil.ServiceInfoDataCenter] = dci
		message, err := json.Marshal(&instances)
		if err != nil {
			log.Errorf(nil, "Instance marshall failed.")
			t.SetFirstErrorCode(meputil.ParseInfoErr, err.Error())
			return workspace.TaskFinish
		}
		var ins *proto.MicroServiceInstance
		err = json.Unmarshal(message, &ins)
		if err != nil {
			log.Errorf(nil, "Micro service instance unmarshall failed.")
			t.SetFirstErrorCode(meputil.ParseInfoErr, err.Error())
			return workspace.TaskFinish
		}
		instanceId := ins.InstanceId
		serviceId := ins.ServiceId
		property := ins.Properties
		if t.AppInstanceId == property["appInstanceId"] {
			findResp = append(findResp, ins)
			unRegSvc := &proto.UnregisterInstanceRequest{
				ServiceId:  serviceId,
				InstanceId: instanceId,
			}
			resp, err := core.InstanceAPI.Unregister(t.Ctx, unRegSvc)

			errorCode, errorString := checkErr(resp, err)
			if errorCode != 0 {
				t.SetFirstErrorCode(workspace.ErrCode(errorCode), errorString)
				return workspace.TaskFinish
			}

		}
	}
	if len(findResp) == 0 {
		log.Infof("Requested application instances not found.")
		t.HttpRsp = ""
		return workspace.TaskFinish
	}
	log.Info("Successfully terminated application services.")
	t.HttpRsp = ""
	return workspace.TaskFinish
}

func checkErr(response *proto.UnregisterInstanceResponse, err error) (int, string) {
	if err != nil {
		log.Error("Service delete failed.", nil)
		return meputil.SerErrServiceInstanceFailed, "service delete failed"
	}
	if response != nil && response.Response.Code == scerr.ErrInstanceNotExists {
		log.Errorf(nil, "Instance not found %s.", response.String())
		return meputil.SerInstanceNotFound, "instance not found"
	}
	return 0, ""
}

type DeleteFromMepauth struct {
	workspace.TaskBase
	AppInstanceId string `json:"appInstanceId,in"`
}

// OnRequest handles
func (t *DeleteFromMepauth) OnRequest(data string) workspace.TaskCode {
	log.Info("Deleting authentication key entry.")
	mepauthPort := os.Getenv("MEPAUTH_SERVICE_PORT")
	if len(mepauthPort) <= 0 || len(mepauthPort) > meputil.MaxPortLength {
		log.Error("Invalid mep-auth port.", nil)
		return workspace.TaskFinish
	} else if num, err := strconv.Atoi(mepauthPort); err == nil {
		if num <= 0 || num > meputil.MaxPortNumber {
			log.Error("Invalid mep-auth port.", nil)
			return workspace.TaskFinish
		}
	}
	mepauthIp := os.Getenv("MEPAUTH_PORT_10443_TCP_ADDR")
	if net.ParseIP(mepauthIp) == nil {
		log.Error("Mep-auth ip env is not set.", nil)
		return workspace.TaskFinish
	}

	deleteUrl := fmt.Sprintf("https://%s:%s/mepauth/v1/applications/%s/confs", mepauthIp, mepauthPort, t.AppInstanceId)

	// Create request
	req, err := http.NewRequest("DELETE", deleteUrl, nil)
	if err != nil {
		log.Errorf(nil, "Not able to send the request to mep-auth %s.", err.Error())
		return workspace.TaskFinish
	}
	config, err := t.TlsConfig()
	if err != nil {
		log.Errorf(nil, "Unable to set the cipher %s.", err.Error())
		return workspace.TaskFinish
	}
	tr := &http.Transport{
		TLSClientConfig: config,
	}

	client := &http.Client{Transport: tr}

	// Fetch Request
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf(nil, "Mep-auth request failed.", err.Error())
		return workspace.TaskFinish
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		log.Error("Mep-auth reported failure.", nil)
		return workspace.TaskFinish
	}

	// Read Response Body
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("Couldn't read the response body.", nil)
		return workspace.TaskFinish
	}
	log.Info("Successfully deleted the mep-auth key.")
	return workspace.TaskFinish
}

// TlsConfig Constructs tls configuration
func (t *DeleteFromMepauth) TlsConfig() (*tls.Config, error) {
	rootCAs := x509.NewCertPool()

	domainName := os.Getenv("MEPSERVER_CERT_DOMAIN_NAME")
	if meputil.ValidateDomainName(domainName) != nil {
		return nil, errors.New("domain name validation failed")
	}
	return &tls.Config{
		RootCAs:            rootCAs,
		ServerName:         domainName,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
	}, nil
}

type DeleteAppDConfigWithSync struct {
	workspace.TaskBase
	Ctx           context.Context `json:"ctx,in"`
	AppInstanceId string          `json:"appInstanceId,in"`
	HttpRsp       interface{}     `json:"httpRsp,out"`
	Worker        *task.Worker
}

func (t *DeleteAppDConfigWithSync) WithWorker(w *task.Worker) *DeleteAppDConfigWithSync {
	t.Worker = w
	return t
}

func (t *DeleteAppDConfigWithSync) OnRequest(data string) workspace.TaskCode {

	log.Info("Deleting the DNS and traffic rule.")
	/*
			1. Check if AppInstanceId already exists and return an error if not exist. (query from DB)
		    2. Check if any other ongoing operation for this AppInstance Id in the system.
			3. update this request to DB (job, task and task status)
			4. Check inside DB for an error
	*/
	if !IsAppInstanceIdAlreadyExists(t.AppInstanceId) {
		log.Errorf(nil, "App instance not found.")
		return workspace.TaskFinish
	}

	// Check if any other ongoing operation for this AppInstance Id in the system.
	if IsAnyOngoingOperationExist(t.AppInstanceId) {
		log.Errorf(nil, "App instance has other operation in progress.")
		t.SetFirstErrorCode(meputil.ForbiddenOperation, "app instance has other operation in progress")
		return workspace.TaskFinish
	}

	var appDConfig models.AppDConfig
	appDConfig.Operation = http.MethodDelete

	taskId := meputil.GenerateUniqueId()
	errCode, msg := UpdateProcessingDatabase(t.AppInstanceId, taskId, &appDConfig)
	if errCode != 0 {
		t.SetFirstErrorCode(errCode, msg)
		return workspace.TaskFinish
	}
	t.Worker.ProcessDataPlaneSync(appDConfig.AppName, t.AppInstanceId, taskId)

	err := task.CheckErrorInDB(t.AppInstanceId, taskId)
	if err != nil {
		log.Errorf(nil, err.Error())
		t.SetFirstErrorCode(meputil.OperateDataWithEtcdErr, err.Error())
		return workspace.TaskFinish
	}

	log.Info("Successfully deleted DNS and traffic rule.")

	return workspace.TaskFinish
}
