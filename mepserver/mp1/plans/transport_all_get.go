package plans

import (
	"github.com/apache/servicecomb-service-center/pkg/util"
	"mepserver/common/arch/workspace"
	"mepserver/common/models"
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

// OnRequest handles to get timing capabilities query
func (t *Transports) OnRequest(data string) workspace.TaskCode {
	transportRecords := t.getTransportInfo()
	t.HttpRsp = transportRecords
	return workspace.TaskFinish
}
