package application

import scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"

type CreateQuickInput struct {
	Targets       []string
	ScanWorkflow  string
	Configuration map[string]any
	AgentID       *int
	InputSource   InputSource
	TriggerType   ScanTriggerType
}

type CreateQuickRequest struct {
	Targets       []string
	ScanWorkflow  string
	Configuration map[string]any
	AgentID       *int
	InputSource   InputSource
	TriggerType   ScanTriggerType
}

type CreateBatchItem struct {
	TargetID       int
	OrganizationID int
}

type CreateBatchInput struct {
	Requests      []CreateBatchItem
	ScanWorkflow  string
	Configuration map[string]any
	AgentID       *int
	InputSource   InputSource
	TriggerType   ScanTriggerType
}

type CreateBatchRequest struct {
	Requests      []CreateBatchItem
	ScanWorkflow  string
	Configuration map[string]any
	AgentID       *int
	InputSource   InputSource
	TriggerType   ScanTriggerType
}

type CreateBatchItemOutcome struct {
	Index          int
	TargetID       int
	OrganizationID int
	Reason         string
	Message        string
}

type CreateBatchResult struct {
	Scans        []CreateScan
	CreatedCount int
	Skipped      []CreateBatchItemOutcome
	Failed       []CreateBatchItemOutcome
}

type BatchScanResult struct {
	Scans        []QueryScan
	CreatedCount int
	Skipped      []CreateBatchItemOutcome
	Failed       []CreateBatchItemOutcome
}

type QuickTargetStats struct {
	Created int
	Skipped int
	Failed  int
}

type QuickTargetError struct {
	Input string
	Error string
}

type QuickTargetResolution struct {
	Targets     []TargetRef
	TargetStats QuickTargetStats
	Errors      []QuickTargetError
}

type CreateQuickResult struct {
	Scans       []CreateScan
	TargetStats QuickTargetStats
	Errors      []QuickTargetError
}

type QuickScanResult struct {
	Scans       []QueryScan
	TargetStats QuickTargetStats
	Errors      []QuickTargetError
}

func validateScanTriggerType(triggerType ScanTriggerType) error {
	if !scandomain.ScanTriggerType(triggerType).Valid() {
		return ErrCreateInvalidTriggerType
	}
	return nil
}

func validateInputSource(inputSource InputSource) error {
	if !scandomain.InputSource(inputSource).Valid() {
		return ErrCreateInvalidInputSource
	}
	return nil
}
