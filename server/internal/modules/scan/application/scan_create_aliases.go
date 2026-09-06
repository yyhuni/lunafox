package application

import scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"

const (
	ScanTriggerTypeManual    = scandomain.ScanTriggerTypeManual
	ScanTriggerTypeScheduled = scandomain.ScanTriggerTypeScheduled
	ScanTriggerTypeAI        = scandomain.ScanTriggerTypeAI

	InputSourceScanSnapshot    = scandomain.InputSourceScanSnapshot
	InputSourceTargetInventory = scandomain.InputSourceTargetInventory

	CreateScanStatusPending   = string(scandomain.ScanStatusPending)
	CreateScanStatusSucceeded = string(scandomain.ScanStatusSucceeded)
	CreateTaskStatusPending   = string(scandomain.TaskStatusPending)
	CreateTaskStatusBlocked   = string(scandomain.TaskStatusBlocked)
	CreateTaskStatusSkipped   = string(scandomain.TaskStatusSkipped)
)

type CreateScan = scandomain.CreateScan

type ScanTriggerType = scandomain.ScanTriggerType

type InputSource = scandomain.InputSource

func ParseInputSource(value string) (InputSource, bool) {
	return scandomain.ParseInputSource(value)
}

type CreateScanTask = scandomain.CreateScanTask

type TargetRef = scandomain.CreateTargetRef

type ScanCreateTaskFinalizer = scandomain.CreateScanTaskFinalizer
