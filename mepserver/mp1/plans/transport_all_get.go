package plans

import (
	"encoding/json"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"mepserver/common/arch/workspace"
	"mepserver/common/extif/backend"
	"mepserver/common/models"
	"mepserver/common/util"
)

// Transports to get timing capabilities
type Transports struct {
	workspace.TaskBase
	HttpRsp interface{} `json:"httpRsp,out"`
}

// OnRequest handles to get timing capabilities query
func (t *Transports) OnRequest(data string) workspace.TaskCode {

	transportsBytes, err := backend.GetRecord(util.TransportInfoPath)
	if err != 0 {
		log.Errorf(nil, "Get transport info from data-store failed.")
		t.SetFirstErrorCode(workspace.ErrCode(err), "transport info retrieval failed")
		return workspace.TaskFinish
	}

	transportInfo := &models.TransportInfo{}
	jsonErr := json.Unmarshal(transportsBytes, transportInfo)
	if jsonErr != nil {
		log.Errorf(nil, "Failed to parse the transport info from data-store.")
		t.SetFirstErrorCode(util.OperateDataWithEtcdErr, "parse transport info from data-store failed")
		return workspace.TaskFinish
	}

	t.HttpRsp = transportInfo
	return workspace.TaskFinish
}
