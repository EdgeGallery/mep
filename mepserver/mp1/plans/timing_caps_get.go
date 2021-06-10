package plans

import (
	"github.com/apache/servicecomb-service-center/pkg/log"
	"mepserver/common/arch/workspace"
	"mepserver/common/extif/ntp"
	"mepserver/common/models"
)

// TimingCaps to get timing capabilities
type TimingCaps struct {
	workspace.TaskBase
	HttpRsp interface{} `json:"httpRsp,out"`
}

func (t *TimingCaps) fillTimingCapsRsp(tcOut *models.TimingCaps, tcIn *ntp.NtpTimingCaps) {

	tcOut.TimeStamp.Seconds = tcIn.TimeStamp.Seconds
	tcOut.TimeStamp.NanoSeconds = tcIn.TimeStamp.NanoSeconds
	tcOut.NtpServers = make([]models.NtpServers, 0)

	for _, NtpServerIn := range tcIn.NtpServers {
		NtpServer := models.NtpServers(NtpServerIn)
		tcOut.NtpServers = append(tcOut.NtpServers, NtpServer)
	}
}

// OnRequest handles to get timing capabilities query
func (t *TimingCaps) OnRequest(data string) workspace.TaskCode {

	// Call external if api to get current time from NTP server
	timingCapsRsp, errCode := ntp.GetTimingCaps()
	if errCode != 0 {
		log.Errorf(nil, "Get timing caps from NTP server failed")
		t.SetFirstErrorCode(workspace.ErrCode(errCode), "timing caps get failed")
		return workspace.TaskFinish
	}

	tc := models.TimingCaps{}
	t.fillTimingCapsRsp(&tc, timingCapsRsp)

	t.HttpRsp = tc
	return workspace.TaskFinish
}
