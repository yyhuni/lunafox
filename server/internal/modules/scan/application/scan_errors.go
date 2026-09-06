package application

import "errors"

var (
	ErrScanNotFound                  = errors.New("scan not found")
	ErrScanCannotStop                = errors.New("scan cannot be stopped in current status")
	ErrScanHardDeleteNotReady        = errors.New("scan hard delete not implemented")
	ErrNoTargetsForScan              = errors.New("no targets provided for scan")
	ErrScanInvalidConfig             = errors.New("invalid scan configuration")
	ErrScanEngineConfigInvalid       = errors.New("invalid complete Engine configuration")
	ErrScanWorkflowEngineUnavailable = errors.New("scan workflow references an unavailable Engine")
	ErrScanInvalidScanWorkflow       = errors.New("invalid scan workflow")
	ErrScanNoScanWorkflows           = errors.New("no scan workflows enabled for scan")
	ErrScanAgentNotFound             = errors.New("selected agent not found")
	ErrTargetNotFound                = errors.New("target not found")
	ErrUnsupportedScanFilter         = errors.New("unsupported scan filter")
	ErrUnsupportedScanOrderBy        = errors.New("unsupported scan orderBy")
)
