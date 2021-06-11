package plans

import (
	"encoding/json"
	"github.com/apache/servicecomb-service-center/pkg/log"
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

// OnRequest handles to get timing capabilities query
func (t *Transports) OnRequest(data string) workspace.TaskCode {
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
