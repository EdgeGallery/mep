package plans

import (
	"encoding/json"
	"fmt"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"mepserver/common/arch/workspace"
	"mepserver/common/extif/backend"
	"mepserver/common/models"
	mputil "mepserver/common/util"
)

// Transports to get timing capabilities
type Transports struct {
	workspace.TaskBase
	HttpRsp interface{} `json:"httpRsp,out"`
}

func fillTransportInfo(tpInfos *models.TransportInfo) {
	log.Info("In fillTransportInfo")
	tpInfos.ID = util.GenerateUuid()
	tpInfos.Name = "REST" //key
	tpInfos.Description = "REST API"
	tpInfos.TransType = "REST_HTTP"
	tpInfos.Protocol = "HTTP"
	tpInfos.Version = "2.0"
}

func InitTransportInfo() error {

	var transportInfos models.TransportInfo
	fillTransportInfo(&transportInfos)
	log.Infof("In InitTransportInfo %v", transportInfos.ID)

	updateJSON, err := json.Marshal(transportInfos)
	if err != nil {
		log.Errorf(err, "Can not marshal the input transport info.")
		return fmt.Errorf("error: Can not marshal the input transport info")
	}

	resultErr := backend.PutRecord(mputil.TransportInfoPath+transportInfos.ID, updateJSON)
	if resultErr != 0 {
		log.Errorf(nil, "Transport info update on etcd failed.")
		return fmt.Errorf("error: Transport info update on etcd failed")
	}

	return nil
}

// OnRequest handles to get timing capabilities query
func (t *Transports) OnRequest(data string) workspace.TaskCode {
	InitTransportInfo()

	respLists, err := backend.GetRecords(mputil.TransportInfoPath)
	if err != 0 {
		log.Errorf(nil, "Get transport info from data-store failed.")
		t.SetFirstErrorCode(workspace.ErrCode(err), "transport info retrieval failed")
		return workspace.TaskFinish
	}
	var transportRecords []*models.TransportInfo
	for _, value := range respLists {
		var transportInfo *models.TransportInfo
		err := json.Unmarshal(value, &transportInfo)
		if err != nil {
			log.Errorf(nil, "Transport Info decode failed.")
			t.SetFirstErrorCode(mputil.ParseInfoErr, err.Error())
			return workspace.TaskFinish
		}
		log.Infof("out Id  %v", transportInfo.ID)
		transportRecords = append(transportRecords, transportInfo)
	}

	t.HttpRsp = transportRecords
	return workspace.TaskFinish
}
