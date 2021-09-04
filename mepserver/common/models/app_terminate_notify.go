package models

import meputil "mepserver/common/util"

type TerminationNotification struct {
	NotificationType   string                  `json:"notificationType"`
	OperationAction    meputil.OperationAction `json:"operationAction"`
	MaxGracefulTimeout uint32                  `json:"maxGracefulTimeout"`
	Links              _Links                  `json:"_links"`
}

type _Links struct {
	Subscription       string `json:"subscription"`
	ConfirmTermination string `json:"confirmTermination"`
}
