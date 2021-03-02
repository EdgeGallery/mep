package plans

import (
	"mepserver/common/arch/workspace"
	"mepserver/mp1/event"
	"strings"

	log "github.com/sirupsen/logrus"
)

type SubscriptionInfoReq struct {
	workspace.TaskBase
	HttpRsp interface{} `json:"httpRsp,out"`
}

func (t *SubscriptionInfoReq) OnRequest(data string) workspace.TaskCode {
	subscriptionInfos := event.GetAllSubscriberInfoFromDB()
	log.Info("subscriptionInfos: ", subscriptionInfos)
	appSubscribeSet := make(map[string]bool)
	serviceSubscribedSet := make(map[string]bool)

	for key, value := range subscriptionInfos {
		log.Info("key: ", key)
		pos := strings.LastIndex(key, "/")
		appInstance := key[0:pos]
		appSubscribeSet[appInstance] = true

		serviceNames := value.FilteringCriteria.SerNames
		for _, name := range serviceNames {
			serviceSubscribedSet[name] = true
		}
	}
	appSubscribeNum := len(appSubscribeSet)
	serviceSubscribedNum := len(serviceSubscribedSet)
	log.Info("appSubscribeNum: ", appSubscribeNum)
	log.Info("serviceSubscribedNum: ", serviceSubscribedNum)

	result := make(map[string]int)
	result["appSubscribeNum"] = appSubscribeNum
	result["serviceSubscribedNum"] = serviceSubscribedNum
	t.HttpRsp = result
	return workspace.TaskFinish
}
