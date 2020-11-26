package models

import (
	"github.com/apache/servicecomb-service-center/server/core/proto"
	"github.com/apache/servicecomb-service-center/pkg/log"
	meputil "mepserver/common/util"
	"strconv"
)

type ServiceLivenessInfo struct {
	State     string    `json:"state"`
	TimeStamp TimeStamp `json:"timeStamp"`
	Interval  int       `json:"interval"`
}

type TimeStamp struct {
	Seconds     uint32 `json:"seconds"`
	Nanoseconds uint32 `json:"nanoSeconds"`
}

type ServiceLivenessUpdate struct {
	State string `json:"state" validate:"required,oneof=ACTIVE"`
}

// transform MicroServiceInstance to HeartbeatInfo
func (s *ServiceLivenessInfo) FromServiceInstance(inst *proto.MicroServiceInstance) {
	if inst == nil || inst.Properties == nil {
		return
	}
	var err error
	var interval int
	var seconds, nanoSeconds uint64
	interval, err = strconv.Atoi(inst.Properties["livenessInterval"])
	if err != nil {
		log.Warn("liveness Interval is fail")
	}
	if interval == 0{
		log.Warn("It is not subscribed for heartbeat")
	}
	s.State = inst.Properties["mecState"]
	seconds, err = strconv.ParseUint(inst.Properties["timestamp/seconds"], FormatIntBase, meputil.BitSize)
	if err != nil {
		log.Warn("timestamp seconds is fail")
	}
	s.TimeStamp.Seconds = uint32(seconds)
	nanoSeconds, err = strconv.ParseUint(inst.Properties["timestamp/nanoseconds"], FormatIntBase, meputil.BitSize)
	if err != nil {
		log.Warn("timestamp seconds is fail")
	}
	s.TimeStamp.Nanoseconds = uint32(nanoSeconds)
	s.Interval = interval
}

//Check the patched details
func (t *ServiceLivenessUpdate) UpdateHeartbeat() string{
	return t.State
}
