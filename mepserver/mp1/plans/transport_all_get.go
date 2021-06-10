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

func fillTransportInfo(tpInfos []models.TransportInfo) {
	log.Info("In fillTransportInfo")
	var transportInfo models.TransportInfo

	transportInfo.ID = util.GenerateUuid()
	transportInfo.Name = "REST"
	transportInfo.Description = "REST API"
	transportInfo.TransType = "REST_HTTP"
	transportInfo.Protocol = "HTTP"
	transportInfo.Version = "2.0"
	tpInfos = append(tpInfos, transportInfo)
}

func InitTransportInfo() error {
	log.Info("In InitTransportInfo")
	transportInfos := make([]models.TransportInfo, 0)
	fillTransportInfo(transportInfos)
	updateJSON, err := json.Marshal(transportInfos)
	if err != nil {
		log.Errorf(err, "Can not marshal the input transport info.")
		return fmt.Errorf("error: Can not marshal the input transport info")
	}

	resultErr := backend.PutRecord(mputil.TransportInfoPath, updateJSON)
	if resultErr != 0 {
		log.Errorf(nil, "Transport info update on etcd failed.")
		return fmt.Errorf("error: Transport info update on etcd failed")
	}

	return nil
}

// OnRequest handles to get timing capabilities query
func (t *Transports) OnRequest(data string) workspace.TaskCode {
	InitTransportInfo()

	transportsBytes, err := backend.GetRecord(mputil.TransportInfoPath)
	if err != 0 {
		log.Errorf(nil, "Get transport info from data-store failed.")
		t.SetFirstErrorCode(workspace.ErrCode(err), "transport info retrieval failed")
		return workspace.TaskFinish
	}

	transportInfo := make([]models.TransportInfo, 0)
	jsonErr := json.Unmarshal(transportsBytes, &transportInfo)
	if jsonErr != nil {
		log.Errorf(nil, "Failed to parse the transport info from data-store.")
		t.SetFirstErrorCode(mputil.OperateDataWithEtcdErr, "parse transport info from data-store failed")
		return workspace.TaskFinish
	}

	t.HttpRsp = transportInfo
	return workspace.TaskFinish
}
