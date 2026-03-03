package application

import (
	"errors"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

var (
	ErrCreateTargetNotFound       = errors.New("target not found")
	ErrCreateInvalidConfig        = errors.New("invalid scan configuration")
	ErrCreateInvalidEngineNames   = errors.New("invalid engines: engineNames must be non-empty and align with engineIDs by position")
	ErrCreateNoWorkflows          = errors.New("no workflows enabled for scan")
	ErrCreateTargetLookupNotReady = errors.New("target lookup is not configured")
)

const (
	CreateScanModeFull  = string(scandomain.ScanModeFull)
	CreateScanModeQuick = string(scandomain.ScanModeQuick)

	CreateScanStatusPending = string(scandomain.ScanStatusPending)
	CreateTaskStatusPending = string(scandomain.TaskStatusPending)
	CreateTaskStatusBlocked = string(scandomain.TaskStatusBlocked)
)

type CreateScan = scandomain.CreateScan

type CreateScanInputTarget = scandomain.CreateScanInputTarget

type CreateScanTask = scandomain.CreateScanTask

type TargetRef = scandomain.CreateTargetRef

type TargetLookupFunc func(id int) (*TargetRef, error)
