package application

import (
	"errors"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanservice "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

var (
	ErrRegistrationTokenInvalid  = agentdomain.ErrRegistrationTokenInvalid
	ErrRegistrationTokenNotFound = agentdomain.ErrRegistrationTokenNotFound
	ErrAgentNotFound             = agentdomain.ErrAgentNotFound
	ErrUnsupportedAgentFilter    = errors.New("unsupported agent filter")
	ErrUnsupportedAgentOrderBy   = errors.New("unsupported agent orderBy")
	ErrInvalidAgentPageToken     = errors.New("invalid agent pageToken")

	ErrAgentTaskNotFound          = scanservice.ErrScanTaskNotFound
	ErrAgentTaskNotOwned          = scanservice.ErrScanTaskNotOwned
	ErrAgentTaskInvalidTransition = scanservice.ErrScanTaskInvalidTransition
	ErrAgentTaskInvalidUpdate     = scanservice.ErrScanTaskInvalidUpdate
)
