package domain

import "errors"

var (
	ErrInvalidScanID         = errors.New("invalid scan id")
	ErrInvalidTargetID       = errors.New("invalid target id")
	ErrInvalidScanMode       = errors.New("invalid scan mode")
	ErrInvalidStatusChange   = errors.New("invalid scan status transition")
	ErrScanCannotStop        = errors.New("scan cannot be stopped in current status")
	ErrFailureMessageMissing = errors.New("failure message is required")
	ErrNoEnabledWorkflows    = errors.New("no workflows enabled")
)
