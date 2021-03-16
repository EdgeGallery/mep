package plans

import (
	"mepserver/common/arch/workspace"
	"mepserver/mp1/event"
	"net/http"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type SubscriptionInfoReq struct {
	workspace.TaskBase
	R       *http.Request `json:"r,in"`
	HttpRsp interface{}   `json:"httpRsp,out"`
}

type SubRelation struct {
	SubscribeAppId string   `json:"subscribeAppId"`
	ServiceList    []string `json:"serviceList"`
}

// This interface is query numbers of app subscribe other services and services subscribed by other app.
func (t *SubscriptionInfoReq) OnRequest(data string) workspace.TaskCode {
	// query subscription info from DB, all the subscription info stored in DB
	subscriptionInfos := event.GetAllSubscriberInfoFromDB()
	log.Info("subscriptionInfos: ", subscriptionInfos)

	// appInstance set for all the app who subscribe services
	appSubscribeSet := make(map[string]bool)
	// services set for all the services who subscribe by app
	serviceSubscribedSet := make(map[string]bool)
	// subscribe relations
	relations := make(map[string][]string)

	for key, value := range subscriptionInfos {
		log.Info("key: ", key)
		//pos := strings.LastIndex(key, "/")
		str := strings.Split(key, "/")
		appInstanceId := str[len(str)-2]
		appSubscribeSet[appInstanceId] = true

		serviceNames := value.FilteringCriteria.SerNames
		for _, name := range serviceNames {
			serviceSubscribedSet[name] = true
		}

		serInstanceIds := value.FilteringCriteria.SerInstanceIds
		if values, ok := relations[appInstanceId]; ok {
			relations[appInstanceId] = append(values, serInstanceIds...)
		} else {
			relations[appInstanceId] = serInstanceIds
		}
	}

	// app numbers who subscribe services
	appSubscribeNum := len(appSubscribeSet)
	// service numbers who subscribed by app
	serviceSubscribedNum := len(serviceSubscribedSet)
	log.Info("appSubscribeNum: " + strconv.Itoa(appSubscribeNum) + "; serviceSubscribedNum: " + strconv.Itoa(serviceSubscribedNum))

	result := make(map[string]int)
	result["appSubscribeNum"] = appSubscribeNum
	result["serviceSubscribedNum"] = serviceSubscribedNum

	subscribeRes := make(map[string]interface{})
	subscribeRes["subscribeNum"] = result

	relationsList := make([]SubRelation, 0)
	for k, v := range relations {
		rel := SubRelation{
			SubscribeAppId: k,
			ServiceList:    v,
		}
		relationsList = append(relationsList, rel)
	}
	subscribeRes["subscribeRelations"] = relationsList

	t.HttpRsp = subscribeRes
	return workspace.TaskFinish
}
