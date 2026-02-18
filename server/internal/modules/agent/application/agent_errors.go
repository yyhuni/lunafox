package application

import (
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanservice "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

var (
	ErrRegistrationTokenInvalid = agentdomain.ErrRegistrationTokenInvalid
	ErrAgentNotFound            = agentdomain.ErrAgentNotFound

	ErrAgentTaskNotFound          = scanservice.ErrScanTaskNotFound
	ErrAgentTaskNotOwned          = scanservice.ErrScanTaskNotOwned
	ErrAgentTaskInvalidTransition = scanservice.ErrScanTaskInvalidTransition
	ErrAgentTaskInvalidUpdate     = scanservice.ErrScanTaskInvalidUpdate
)
