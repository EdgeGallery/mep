package plans

import (
	"encoding/json"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"mepserver/common/arch/workspace"
	"mepserver/common/extif/backend"
	"mepserver/common/models"
	meputil "mepserver/common/util"
)

// Transports to get timing capabilities
type Transports struct {
	workspace.TaskBase
	HttpRsp interface{} `json:"httpRsp,out"`
}

func (t *Transports) getTransportInfo() []models.TransportInfo {
	var tpInfo models.TransportInfo
	tpInfos := make([]models.TransportInfo, 0)
	tpInfo.ID = util.GenerateUuid()
	tpInfo.Name = "REST"
	tpInfo.Description = "REST API"
	tpInfo.TransType = "REST_HTTP"
	tpInfo.Protocol = "HTTP"
	tpInfo.Version = "2.0"
	var theArray = make([]string, 1)
	theArray[0] = "OAUTH2_CLIENT_CREDENTIALS"
	tpInfo.Security.OAuth2Info.GrantTypes = theArray
	tpInfo.Security.OAuth2Info.TokenEndpoint = "/mep/token"
	tpInfos = append(tpInfos, tpInfo)
	return tpInfos
}

func (t *Transports) addTransportInfoToDb(tpInfo *models.TransportInfo) error {
	updateJSON, jsonErr := json.Marshal(tpInfo)
	if jsonErr != nil {
		log.Errorf(jsonErr, "Can not marshal the input transport info.")
		return nil
	}

	resultErr := backend.PutRecord(meputil.TransportInfoPath+tpInfo.ID, updateJSON)
	if resultErr != 0 {
		log.Errorf(nil, "Transport info update on etcd failed.")
		return nil
	}

	log.Infof("Transport info added for  %v", tpInfo.Name)
	return nil
}

func (t *Transports) checkAndUpdateTransportsInfo() []models.TransportInfo {

	tpInfos := t.getTransportInfo()

	respLists, err := backend.GetRecords(meputil.TransportInfoPath)
	if err != 0 {
		log.Errorf(nil, "Get transport info from data-store failed.")
		return nil
	}

	tpInfoRecords := make([]models.TransportInfo, 0)
	isExist := false
	for _, value := range respLists {
		var transportInfo *models.TransportInfo
		err := json.Unmarshal(value, &transportInfo)
		if err != nil {
			log.Errorf(nil, "Transport Info decode failed.")
			return nil
		}

		for _, tpInfo := range tpInfos {
			if transportInfo.Name == tpInfo.Name {
				tpInfo.ID = transportInfo.ID
				tpInfoRecords = append(tpInfoRecords, tpInfo)
				log.Infof("Transport info exists for  %v", transportInfo.Name)
				isExist = true
			}
		}
	}

	if isExist {
		return tpInfoRecords
	}
	// If not present then add to DB
	for _, tpInfo := range tpInfos {
		t.addTransportInfoToDb(&tpInfo)
	}

	return tpInfos
}

// OnRequest handles to get timing capabilities query
func (t *Transports) OnRequest(data string) workspace.TaskCode {
	ts := t.checkAndUpdateTransportsInfo()
	t.HttpRsp = ts
	return workspace.TaskFinish
}
