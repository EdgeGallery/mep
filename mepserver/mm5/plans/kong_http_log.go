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

package plans

import (
	"context"
	"io/ioutil"
	"mepserver/common/arch/workspace"
	meputil "mepserver/common/util"
	"mepserver/mp1"
	"net/http"
	"net/url"
	"strconv"

	"github.com/apache/servicecomb-service-center/pkg/log"
	es "github.com/olivere/elastic/v7"
)

var EsClient *es.Client

func init() {
	EsClient = createEsClient()
}

func createEsClient() *es.Client {
	esHost := "http://114.116.17.54:9200"
	esClient, err := es.NewClient(es.SetSniff(false), es.SetURL(esHost))
	if err != nil {
		log.Error("Connect to es fail.", err)
		return EsClient
	}
	log.Info("Connect to es success")
	return esClient
}

type CreateKongHttpLog struct {
	workspace.TaskBase
	R       *http.Request `json:"r,in"`
	HttpRsp interface{}   `json:"httpRsp,out"`
}

func (t *CreateKongHttpLog) OnRequest(data string) workspace.TaskCode {
	msg, err := ioutil.ReadAll(t.R.Body)
	if err != nil {
		log.Error("read failed", nil)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "read request body error")
		return workspace.TaskFinish
	}
	resp, err := EsClient.Index().Index(meputil.KongHttpLogIndex).BodyString(string(msg)).Do(context.Background())
	if err != nil {
		log.Error("Create doc fail.", err)
	}
	t.HttpRsp = resp

	return workspace.TaskFinish
}

type GetKongHttpLog struct {
	workspace.TaskBase
	R       *http.Request `json:"r,in"`
	HttpRsp interface{}   `json:"httpRsp,out"`
}

func (t *GetKongHttpLog) OnRequest(data string) workspace.TaskCode {
	// 注册服务的调用
	appServices := make(map[string]interface{})
	// 获取MEP上注册的服务列表
	serviceNames := getAllServiceNames()
	statisticAppServices(appServices, serviceNames)

	// MEP自身能力的调用
	mepServices := make(map[string]interface{})
	statisticMepServices(mepServices)

	res := make(map[string]interface{})
	res["appServices"] = appServices
	res["mepServices"] = mepServices
	t.HttpRsp = res
	return workspace.TaskFinish
}

func statisticMepServices(services map[string]interface{}) {
	// 服务注册
	services["serviceRegister"] = statisticRegisterServices()
	// 服务发现
	services["serviceDiscovery"] = statisticDiscoveryServices()
}

func statisticDiscoveryServices() interface{} {
	dayCount := make(map[string]int)
	for i := 0; i < 7; i++ {
		boolQuery := es.NewBoolQuery()
		serviceNameQuery := es.NewTermsQuery("service.name", "mepserver")
		boolQuery.Filter(serviceNameQuery)

		upstreamUriQuery := es.NewRegexpQuery("upstream_uri.keyword",
			"/mep/mec_service_mgmt/v1/services*")
		boolQuery.Filter(upstreamUriQuery)

		requestMethodQuery := es.NewMatchQuery("request.method", "GET")
		boolQuery.Filter(requestMethodQuery)

		timeRangeQuery := es.NewRangeQuery("started_at").Gte("now-" + strconv.Itoa(i) + "d/d").Lt("now-" + strconv.Itoa(
			i-1) + "d/d")
		boolQuery.Filter(timeRangeQuery)

		resp, err := EsClient.Count(meputil.KongHttpLogIndex).Query(boolQuery).Do(context.Background())
		if err != nil {
			dayCount["day"+strconv.Itoa(i)] = 0
		} else {
			dayCount["day"+strconv.Itoa(i)] = int(resp)
		}
	}
	return dayCount
}

func statisticRegisterServices() interface{} {
	dayCount := make(map[string]int)
	for i := 0; i < 7; i++ {
		boolQuery := es.NewBoolQuery()
		serviceNameQuery := es.NewTermsQuery("service.name", "mepserver")
		boolQuery.Filter(serviceNameQuery)

		upstreamUriQuery := es.NewRegexpQuery("upstream_uri.keyword",
			"/mep/mec_service_mgmt/v1/applications/[-A-Za-z0-9]+/services")
		boolQuery.Filter(upstreamUriQuery)

		requestMethodQuery := es.NewMatchQuery("request.method", "POST")
		boolQuery.Filter(requestMethodQuery)

		timeRangeQuery := es.NewRangeQuery("started_at").Gte("now-" + strconv.Itoa(i) + "d/d").Lt("now-" + strconv.Itoa(
			i-1) + "d/d")
		boolQuery.Filter(timeRangeQuery)

		resp, err := EsClient.Count(meputil.KongHttpLogIndex).Query(boolQuery).Do(context.Background())
		if err != nil {
			dayCount["day"+strconv.Itoa(i)] = 0
		} else {
			dayCount["day"+strconv.Itoa(i)] = int(resp)
		}
	}
	return dayCount
}

func statisticAppServices(res map[string]interface{}, names []string) {
	for _, serviceName := range names {
		dayCount := make(map[string]int)
		for i := 0; i < 7; i++ {
			boolQuery := es.NewBoolQuery()
			serviceNameQuery := es.NewTermQuery("service.name.keyword", serviceName)
			boolQuery.Filter(serviceNameQuery)

			timeRangeQuery := es.NewRangeQuery("started_at").Gte("now-" + strconv.Itoa(i) + "d/d").Lt("now-" + strconv.Itoa(
				i-1) + "d/d")
			boolQuery.Filter(timeRangeQuery)

			resp, err := EsClient.Count(meputil.KongHttpLogIndex).Query(boolQuery).Do(context.Background())
			if err != nil {
				dayCount["day"+strconv.Itoa(i)] = 0
			} else {
				dayCount["day"+strconv.Itoa(i)] = int(resp)
			}
		}
		res[serviceName] = dayCount
	}
}

func getAllServiceNames() []string {
	serviceNames := make([]string, 0)
	findInstancesResponse, err := meputil.FindInstanceByKey(url.Values{})
	if err != nil {
		log.Errorf(nil, "FindInstanceByKey failed.")
		return serviceNames
	}
	_, serviceInfos := mp1.Mp1CvtSrvDiscover(findInstancesResponse)
	if serviceInfos == nil {
		log.Errorf(nil, "Mp1CvtSrvDiscover failed.")
		return serviceNames
	}
	for _, service := range serviceInfos {
		serviceNames = append(serviceNames, service.SerName)
	}
	return serviceNames
}