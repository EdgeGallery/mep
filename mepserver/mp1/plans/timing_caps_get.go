package plans

import (
	"github.com/apache/servicecomb-service-center/pkg/log"
	"mepserver/common/arch/workspace"
	"mepserver/common/extif/ntp"
	"mepserver/common/models"
	"mepserver/common/util"
)

// TimingCaps to get timing capabilities
type TimingCaps struct {
	workspace.TaskBase
	HttpRsp interface{} `json:"httpRsp,out"`
}

func (t *TimingCaps) GetNtpServer(tc *models.TimingCaps) {
	tc.NtpServers = make([]models.NtpServers, 0)
	var NtpServer models.NtpServers
	NtpServer.NtpServerAddr = util.NtpHost
	NtpServer.NtpServerAddrType = util.NtpDnsName
	NtpServer.AuthenticationOption = util.NtpAuthType //Authentication not supported now
	NtpServer.AuthenticationKeyNum = 0                // Invalid key number
	NtpServer.LocalPriority = 1
	NtpServer.MaxPollingInterval = util.MaxPoll
	NtpServer.MinPollingInterval = util.MinPoll
	tc.NtpServers = append(tc.NtpServers, NtpServer)
}

// OnRequest handles to get timing capabilities query
func (t *TimingCaps) OnRequest(data string) workspace.TaskCode {

	// Call external if api to get current time from NTP server
	timeStamp, errCode := ntp.GetTimeStamp()
	if errCode != 0 {
		log.Errorf(nil, "Get timing caps from NTP server failed")
		t.SetFirstErrorCode(workspace.ErrCode(errCode), "Timing caps get failed")
		return workspace.TaskFinish
	}

	tc := models.TimingCaps{}
	tc.TimeStamp.Seconds = timeStamp.Seconds
	tc.TimeStamp.NanoSeconds = timeStamp.NanoSeconds
	t.GetNtpServer(&tc)
	t.HttpRsp = tc
	return workspace.TaskFinish
}
