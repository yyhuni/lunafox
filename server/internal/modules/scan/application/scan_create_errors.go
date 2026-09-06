package application

import (
	"errors"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

var (
	ErrCreateTargetNotFound                = errors.New("target not found")
	ErrCreateInvalidConfig                 = errors.New("invalid scan configuration")
	ErrCreateInvalidScanWorkflow           = errors.New("invalid scan workflow")
	ErrCreateScanWorkflowEngineUnavailable = errors.New("scan workflow engine unavailable")
	ErrCreateNoScanWorkflows               = errors.New("no scan workflows enabled for scan")
	ErrCreateTargetLookupNotReady          = errors.New("target lookup is not configured")
	ErrCreateAgentNotFound                 = errors.New("selected agent not found")
	ErrCreateInvalidTriggerType            = errors.New("scan trigger type is required and must be manual, scheduled, or ai")
	ErrCreateInvalidInputSource            = scandomain.ErrInvalidInputSource
)
