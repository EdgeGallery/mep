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

// Package path implements mep server api plans
package plans

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mepserver/common/models"
	"net"
	"net/http"
	"net/url"

	"github.com/apache/servicecomb-service-center/pkg/log"
	scutil "github.com/apache/servicecomb-service-center/pkg/util"
	"github.com/apache/servicecomb-service-center/server/core"
	"github.com/apache/servicecomb-service-center/server/core/backend"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	scerr "github.com/apache/servicecomb-service-center/server/error"
	"github.com/apache/servicecomb-service-center/server/plugin/pkg/registry"
	"github.com/satori/go.uuid"

	"mepserver/common/arch/workspace"
	"mepserver/common/util"
)

type SubscribeIst struct {
	workspace.TaskBase
	R             *http.Request       `json:"r,in"`
	HttpErrInf    *proto.Response     `json:"httpErrInf,out"`
	Ctx           context.Context     `json:"ctx,in"`
	W             http.ResponseWriter `json:"w,in"`
	RestBody      interface{}         `json:"restBody,in"`
	AppInstanceId string              `json:"appInstanceId,in"`
	SubscribeId   string              `json:"subscribeId,in"`
	HttpRsp       interface{}         `json:"httpRsp,out"`
	SubscribeType string              `json:"subscribeType,out"`
}

// set type and return SubscribeIst
func (t *SubscribeIst) WithType(subType string) *SubscribeIst {
	t.SubscribeType = subType
	return t
}

// OnRequest
func (t *SubscribeIst) OnRequest(data string) workspace.TaskCode {
	mp1SubscribeInfo := t.getMp1SubscribeInfo()
	if mp1SubscribeInfo == nil {
		return workspace.TaskFinish
	}
	errCheck := t.checkSubscribeSerInstanceExist(mp1SubscribeInfo)
	if errCheck != nil {
		log.Error("subscriber instance not exist", nil)
		t.SetFirstErrorCode(util.SerErrServiceNotFound, "subscriber instance not exist")
		return workspace.TaskFinish
	}
	subscribeJSON, err := json.Marshal(mp1SubscribeInfo)
	if err != nil {
		log.Errorf(nil, "can not marshal subscribe info")
		t.SetFirstErrorCode(util.ParseInfoErr, "marshal subscribe info error")
		return workspace.TaskFinish
	}
	callbackUriNotValid := t.ValidateCallbackUri(subscribeJSON)
	if callbackUriNotValid {
		log.Error("url validation failed", nil)
		t.SetFirstErrorCode(util.RequestParamErr, util.ErrorRequestBodyMessage)
		return workspace.TaskFinish
	}
	log.Debugf("request received for app subscription with appId %s", t.AppInstanceId)
	t.SubscribeId = uuid.NewV4().String()
	err = t.insertOrUpdateData(subscribeJSON)
	if err != nil {
		return workspace.TaskFinish
	}
	t.buildResponse(mp1SubscribeInfo)

	_, err = json.Marshal(mp1SubscribeInfo)
	if err != nil {
		return t.marshalError(t.AppInstanceId)
	}
	log.Debugf("response sent for app subscription with appId %s ", t.AppInstanceId)

	return workspace.TaskFinish
}

// Validate Callback Uri
func (t *SubscribeIst) ValidateCallbackUri(subscribeJSON []byte) bool {
	var callBack string
	if t.SubscribeType == util.SerAvailabilityNotificationSubscription {
		var serAvl models.SerAvailabilityNotificationSubscription
		errJson := json.Unmarshal(subscribeJSON, &serAvl)
		if errJson != nil {
			log.Error(util.ErrorRequestBodyMessage, nil)
			t.SetFirstErrorCode(util.RequestParamErr, util.ErrorRequestBodyMessage)
			return true
		}
		callBack = serAvl.CallbackReference
	} else {
		var appTermAvl models.AppTerminationNotificationSubscription
		errJson := json.Unmarshal(subscribeJSON, &appTermAvl)
		if errJson != nil {
			log.Error(util.ErrorRequestBodyMessage, nil)
			t.SetFirstErrorCode(util.RequestParamErr, util.ErrorRequestBodyMessage)
			return true
		}
		callBack = appTermAvl.CallbackReference
	}
	if callBack != "" {
		isInSameNet := isValidCallbackURI(callBack)
		if isInSameNet {
			log.Error("Invalid CallbackReference uri containing invalid host", nil)
			t.SetFirstErrorCode(util.RequestParamErr, "Invalid CallbackReference uri containing invalid host")
			return true
		}
	}
	return false
}

func isValidCallbackURI(reference string) bool {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Info("Interface address not able to find")
		return false
	}
	urlInfo, err1 := url.Parse(reference)
	if err1 != nil {
		log.Info("url parse is failed.")
		return false
	}
	urlIP := net.ParseIP(urlInfo.Hostname())
	if urlIP == nil {
		return true
	}
	for _, address := range addrs {
		// check the address type
		if ipnet, ok := address.(*net.IPNet); ok {
			_, ipnetA, parseErr := net.ParseCIDR(ipnet.String())
			if parseErr != nil {
				continue
			}
			// Check whether given IP is in the network
			if ipnetA.Contains(urlIP) {
				return true
			}
		}
	}
	return false
}

func (t *SubscribeIst) marshalError(appInstanceId string) workspace.TaskCode {
	subKeyPath := util.GetSubscribeKeyPath(t.SubscribeType)
	opts := []registry.PluginOp{
		registry.OpDel(registry.WithStrKey(subKeyPath + appInstanceId + "/" + t.SubscribeId)),
	}
	_, err := backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if err != nil {
		log.Errorf(errors.New("delete opertaion failed"), "deleting app subscription from etcd failed on error. "+
			"This might lead to data inconsistency!")
		t.SetFirstErrorCode(util.OperateDataWithEtcdErr, "delete subscription from etcd failed on marshal error")
		return workspace.TaskFinish
	}
	log.Error("marshal subscription info failed", nil)
	t.SetFirstErrorCode(util.ParseInfoErr, "marshal subscription info failed")
	return workspace.TaskFinish
}

func (t *SubscribeIst) buildResponse(sub interface{}) {

	switch sub := sub.(type) {
	case *models.SerAvailabilityNotificationSubscription:
		location := fmt.Sprintf("%s/applications/%s/subscriptions/%s", util.MecServicePath, t.AppInstanceId,
			t.SubscribeId)
		sub.Links = models.Links{Self: models.Self{Href: location}}
		sub.SubscriptionId = t.SubscribeId
		t.W.Header().Set("Location", location)
		t.HttpRsp = sub
	case *models.AppTerminationNotificationSubscription:
		location := fmt.Sprintf("%s/applications/%s/subscriptions/%s", util.MecAppSupportPath, t.AppInstanceId,
			t.SubscribeId)
		sub.Links = models.Links{Self: models.Self{Href: location}}
		sub.SubscriptionId = t.SubscribeId
		t.W.Header().Set("Location", location)
		t.HttpRsp = sub
	default:
		log.Warn("sub type not match")
	}

}

func (t *SubscribeIst) checkSubscribeSerInstanceExist(sub interface{}) error {

	switch sub := sub.(type) {
	case *models.SerAvailabilityNotificationSubscription:
		for _, serInstanceId := range sub.FilteringCriteria.SerInstanceIds {
			err := checkSerInstanceExist(t.R, serInstanceId)
			if err != nil {
				return err
			}
		}
	case *models.AppTerminationNotificationSubscription:
		return nil
	default:
		return nil
	}
	return nil
}

func checkSerInstanceExist(r *http.Request, serInstanceId string) error {
	query, ids := util.GetHTTPTags(r)
	serviceId := serInstanceId[:len(serInstanceId)/2]
	instanceId := serInstanceId[len(serInstanceId)/2:]
	req := &proto.GetOneInstanceRequest{
		ConsumerServiceId:  r.Header.Get("X-ConsumerId"),
		ProviderServiceId:  serviceId,
		ProviderInstanceId: instanceId,
		Tags:               ids,
	}
	ctx := scutil.SetTargetDomainProject(r.Context(), r.Header.Get("X-Domain-Name"), query.Get(":project"))
	resp, errGetOneInstance := core.InstanceAPI.GetOneInstance(ctx, req)
	if errGetOneInstance != nil {
		return errGetOneInstance
	}
	if resp != nil {
		respCode := resp.Response.GetCode()
		if respCode == proto.Response_SUCCESS {
			return nil
		} else if respCode == scerr.ErrInstanceNotExists || respCode == scerr.ErrServiceNotExists {
			return fmt.Errorf("subscribe service instance id no exist")
		} else {
			return fmt.Errorf("unexpected error")
		}
	}
	return nil
}

type AppSubscribeLimit struct {
	workspace.TaskBase
	Ctx           context.Context `json:"ctx,in"`
	RestBody      interface{}     `json:"restBody,in"`
	SubscribeType string          `json:"subscribeType,out"`
	AppInstanceId string          `json:"appInstanceId,in"`
}

// set type and return AppSubscribeLimit
func (t *AppSubscribeLimit) WithType(subType string) *AppSubscribeLimit {
	t.SubscribeType = subType
	return t
}

// OnRequest
func (t *AppSubscribeLimit) OnRequest(data string) workspace.TaskCode {
	subscribeKeyPath := util.GetSubscribeKeyPath(t.SubscribeType)
	appInstanceId := t.AppInstanceId

	opts := []registry.PluginOp{
		registry.OpGet(registry.WithStrKey(subscribeKeyPath+appInstanceId), registry.WithPrefix()),
	}
	resp, err := backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if err != nil {
		log.Errorf(nil, "get subscription from etcd failed")
		t.SetFirstErrorCode(util.OperateDataWithEtcdErr, "get subscription from etcd failed")
		return workspace.TaskFinish
	}
	if resp.Count >= util.AppSubscriptionCount {
		log.Errorf(nil, "subscription limit has been reached")
		t.SetFirstErrorCode(util.SubscriptionErr, "subscription has over the limit")
	}
	return workspace.TaskFinish
}

func (t *SubscribeIst) getMp1SubscribeInfo() interface{} {
	var mp1SubscribeInfo interface{}
	var ok bool
	if t.SubscribeType == util.SerAvailabilityNotificationSubscription {
		mp1SubscribeInfo, ok = t.RestBody.(*models.SerAvailabilityNotificationSubscription)
	} else {
		mp1SubscribeInfo, ok = t.RestBody.(*models.AppTerminationNotificationSubscription)
	}
	if !ok {
		log.Error(util.ErrorRequestBodyMessage, nil)
		t.SetFirstErrorCode(util.RequestParamErr, util.ErrorRequestBodyMessage)
		return nil
	}
	return mp1SubscribeInfo
}

func (t *SubscribeIst) insertOrUpdateData(subscribeJSON []byte) error {
	opts := []registry.PluginOp{
		registry.OpPut(registry.WithStrKey(util.GetSubscribeKeyPath(t.SubscribeType)+t.AppInstanceId+"/"+
			t.SubscribeId),
			registry.WithValue(subscribeJSON)),
	}
	_, resultErr := backend.Registry().TxnWithCmp(context.Background(), opts, nil, nil)
	if resultErr != nil {
		log.Errorf(nil, "subscription to etcd failed")
		t.SetFirstErrorCode(util.OperateDataWithEtcdErr, "put subscription to etcd failed")
		return resultErr
	}
	return nil
}
