package domain

import "errors"

var (
	ErrInvalidScanID                   = errors.New("invalid scan id")
	ErrInvalidTargetID                 = errors.New("invalid target id")
	ErrInvalidScanTriggerType          = errors.New("invalid scan trigger type")
	ErrInvalidStatusChange             = errors.New("invalid scan status transition")
	ErrScanCannotStop                  = errors.New("scan cannot be stopped in current status")
	ErrFailureMessageMissing           = errors.New("failure message is required")
	ErrNoEnabledWorkflows              = errors.New("no workflows enabled")
	ErrSavedExecutionPlanLeaseNotFound = errors.New("saved execution plan lease not found")
	ErrAgentExecutionSessionFenced     = errors.New("agent execution session is fenced")
)
