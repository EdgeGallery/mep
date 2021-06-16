package plans

import (
	"github.com/apache/servicecomb-service-center/pkg/log"
	"mepserver/common/arch/workspace"
	"mepserver/common/extif/ntp"
	"mepserver/common/models"
	"mepserver/common/util"
	"os"
	"strings"
)

// TimingCaps to get timing capabilities
type TimingCaps struct {
	workspace.TaskBase
	HttpRsp interface{} `json:"httpRsp,out"`
}

func (t *TimingCaps) UpdateNtpServer(tc *models.TimingCaps) {
	serverList := os.Getenv(util.NtpServers)
	servers := strings.Split(serverList, ",")
	priority := 1
	tc.NtpServers = make([]models.NtpServers, 0)
	for _, server := range servers {
		var NtpServer models.NtpServers
		NtpServer.NtpServerAddr = strings.TrimSpace(server)
		NtpServer.NtpServerAddrType = util.NtpDnsName
		NtpServer.AuthenticationOption = util.NtpAuthType //Authentication not supported now
		NtpServer.AuthenticationKeyNum = 0                // Invalid key number
		NtpServer.LocalPriority = priority
		NtpServer.MaxPollingInterval = util.MaxPoll
		NtpServer.MinPollingInterval = util.MinPoll
		tc.NtpServers = append(tc.NtpServers, NtpServer)
		priority++
	}
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
	t.UpdateNtpServer(&tc)

	t.HttpRsp = tc
	return workspace.TaskFinish
}
