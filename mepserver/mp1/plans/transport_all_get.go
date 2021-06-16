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
	tpInfo.Name = meputil.TransportName
	tpInfo.Description = meputil.TransportDescription
	tpInfo.TransType = meputil.TransportTransType
	tpInfo.Protocol = meputil.TransportProtocol
	tpInfo.Version = meputil.TransportVersion
	var theArray = make([]string, 1)
	theArray[0] = meputil.TransportGrantTypes
	tpInfo.Security.OAuth2Info.GrantTypes = theArray
	tpInfo.Security.OAuth2Info.TokenEndpoint = meputil.TransportTokenEndpoint
	tpInfos = append(tpInfos, tpInfo)
	return tpInfos
}

func (t *Transports) addTransportInfoToDb(tpInfo *models.TransportInfo) int {
	updateJSON, jsonErr := json.Marshal(tpInfo)
	if jsonErr != nil {
		log.Errorf(jsonErr, "Can not marshal the input transport info.")
		return 1
	}

	resultErr := backend.PutRecord(meputil.TransportInfoPath+tpInfo.ID, updateJSON)
	if resultErr != 0 {
		log.Errorf(nil, "Transport info update on etcd failed.")
		return 1
	}

	log.Infof("Transport info added successfully for  %v", tpInfo.Name)
	return 0
}

func (t *Transports) checkAndUpdateTransportsInfo() ([]models.TransportInfo, int) {

	respLists, err := backend.GetRecords(meputil.TransportInfoPath)
	if err != 0 {
		log.Errorf(nil, "Get transport info from data-store failed.")
		return nil, err
	}

	if len(respLists) != 0 {
		tpInfoRecords := make([]models.TransportInfo, 0)
		for _, value := range respLists {
			var transportInfo models.TransportInfo
			tpInfo := &transportInfo
			err := json.Unmarshal(value, tpInfo)
			if err != nil {
				log.Errorf(nil, "Transport Info decode failed.")
				return nil, meputil.ParseInfoErr
			}
			tpInfoRecords = append(tpInfoRecords, transportInfo)
		}
		return tpInfoRecords, 0
	} else {
		tpInfos := t.getTransportInfo()
		for _, tpInfo := range tpInfos {
			ret := t.addTransportInfoToDb(&tpInfo)
			if ret != 0 {
				return nil, ret
			}
		}
		return tpInfos, 0
	}
}

// OnRequest handles to get timing capabilities query
func (t *Transports) OnRequest(data string) workspace.TaskCode {
	ts, err := t.checkAndUpdateTransportsInfo()
	if err != 0 {
		log.Errorf(nil, "Get transport info failed.")
		t.SetFirstErrorCode(workspace.ErrCode(err), "Get transport info failed")
		return workspace.TaskFinish
	}
	t.HttpRsp = ts
	return workspace.TaskFinish
}
