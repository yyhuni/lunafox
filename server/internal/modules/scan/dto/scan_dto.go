package dto

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type ScanListQuery struct {
	httpdto.PaginationQuery
	TargetID int    `form:"target" binding:"omitempty"`
	Status   string `form:"status" binding:"omitempty"`
	Filter   string `form:"filter" binding:"omitempty"`
	OrderBy  string `form:"orderBy" binding:"omitempty"`
}

type FailureResponse struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type ScanResponse struct {
	ID               int              `json:"id"`
	Name             string           `json:"name"`
	TargetID         int              `json:"targetId"`
	ScanWorkflow     string           `json:"scanWorkflow"`
	PlannedEngineIDs []string         `json:"plannedEngineIds"`
	InputSource      string           `json:"inputSource"`
	TriggerType      string           `json:"triggerType"`
	Status           string           `json:"status"`
	Progress         int              `json:"progress"`
	CurrentStage     string           `json:"currentStage"`
	ErrorMessage     string           `json:"errorMessage,omitempty"`
	Failure          *FailureResponse `json:"failure,omitempty"`
	CreatedAt        time.Time        `json:"createdAt"`
	StoppedAt        *time.Time       `json:"stoppedAt,omitempty"`
	Target           *TargetBrief     `json:"target,omitempty"`
	CachedStats      *ScanCachedStats `json:"cachedStats,omitempty"`
}

type TargetBrief struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Type        string `json:"type"`
}

type ScanCachedStats struct {
	SubdomainsCount  int `json:"subdomainsCount"`
	WebsitesCount    int `json:"websitesCount"`
	EndpointsCount   int `json:"endpointsCount"`
	IPsCount         int `json:"ipsCount"`
	DirectoriesCount int `json:"directoriesCount"`
	ScreenshotsCount int `json:"screenshotsCount"`
	VulnsTotal       int `json:"vulnsTotal"`
	VulnsCritical    int `json:"vulnsCritical"`
	VulnsHigh        int `json:"vulnsHigh"`
	VulnsMedium      int `json:"vulnsMedium"`
	VulnsLow         int `json:"vulnsLow"`
}

type ScanDetailResponse struct {
	ScanResponse
	AgentID          *int                  `json:"agentId,omitempty"`
	Agent            string                `json:"agent,omitempty"`
	AgentName        string                `json:"agentName,omitempty"`
	AgentStatus      string                `json:"agentStatus,omitempty"`
	AgentHealthState string                `json:"agentHealthState,omitempty"`
	AgentDeleted     *bool                 `json:"agentDeleted,omitempty"`
	AssignmentMode   string                `json:"assignmentMode"`
	Configuration    map[string]any        `json:"configuration,omitempty"`
	ResultsDir       string                `json:"resultsDir,omitempty"`
	RuntimeTasks     []RuntimeTaskResponse `json:"runtimeTasks,omitempty"`
}

type RuntimeTaskResponse struct {
	ID            int                                 `json:"id"`
	Name          string                              `json:"name"`
	StepID        string                              `json:"stepId"`
	StageID       string                              `json:"stageId"`
	EngineID      string                              `json:"engineId"`
	Status        string                              `json:"status"`
	SkipReason    string                              `json:"skipReason,omitempty"`
	Order         int                                 `json:"order"`
	StartedAt     *time.Time                          `json:"startedAt,omitempty"`
	CompletedAt   *time.Time                          `json:"completedAt,omitempty"`
	Duration      *float64                            `json:"duration,omitempty"`
	Error         string                              `json:"error,omitempty"`
	FailureKind   string                              `json:"failureKind,omitempty"`
	FailureDetail string                              `json:"failureDetail,omitempty"`
	Diagnostics   *EngineExecutionDiagnosticsResponse `json:"diagnostics,omitempty"`
}

// EngineExecutionDiagnosticsResponse is the fixed terminal diagnostic
// projection. It is intentionally task-detail-only and carries no logs, raw
// error text, or per-result records.
type EngineExecutionDiagnosticsResponse struct {
	CompatibilityRevision string                        `json:"compatibilityRevision"`
	Availability          string                        `json:"availability"`
	ResultState           string                        `json:"resultState"`
	FailedStage           string                        `json:"failedStage,omitempty"`
	ErrorType             string                        `json:"errorType,omitempty"`
	ResultTypeWatermarks  []ResultTypeWatermarkResponse `json:"resultTypeWatermarks,omitempty"`
}

type ResultTypeWatermarkResponse struct {
	ResultType          string `json:"resultType"`
	ReceivedItems       uint64 `json:"receivedItems"`
	EncodedItems        uint64 `json:"encodedItems"`
	SubmittedItems      uint64 `json:"submittedItems"`
	AcknowledgedItems   uint64 `json:"acknowledgedItems"`
	SubmittedBatches    uint64 `json:"submittedBatches"`
	AcknowledgedBatches uint64 `json:"acknowledgedBatches"`
}

func decodeScanCreateRequestFields(data []byte, allowed map[string]struct{}, target any) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for key := range fields {
		if _, ok := allowed[key]; ok {
			continue
		}
		return fmt.Errorf("scan create request field %q is not supported; use scanWorkflow: scanWorkflows/{workflow}", key)
	}
	return json.Unmarshal(data, target)
}

type CreateQuickScanRequest struct {
	Targets       []string       `json:"targets" binding:"required,min=1,max=5000"`
	ScanWorkflow  string         `json:"scanWorkflow" binding:"required"`
	Configuration map[string]any `json:"configuration"`
	Agent         string         `json:"agent,omitempty"`
	InputSource   string         `json:"inputSource" binding:"required"`
}

func (req *CreateQuickScanRequest) UnmarshalJSON(data []byte) error {
	type rawRequest CreateQuickScanRequest
	var raw rawRequest
	if err := decodeScanCreateRequestFields(data, map[string]struct{}{
		"targets":       {},
		"scanWorkflow":  {},
		"configuration": {},
		"agent":         {},
		"inputSource":   {},
	}, &raw); err != nil {
		return err
	}
	*req = CreateQuickScanRequest(raw)
	return nil
}

type BatchCreateScanItem struct {
	Target       string `json:"target,omitempty"`
	Organization string `json:"organization,omitempty"`
}

func (item *BatchCreateScanItem) UnmarshalJSON(data []byte) error {
	type rawItem BatchCreateScanItem
	var raw rawItem
	if err := decodeScanCreateRequestFields(data, map[string]struct{}{
		"target":       {},
		"organization": {},
	}, &raw); err != nil {
		return err
	}
	*item = BatchCreateScanItem(raw)
	return nil
}

type BatchCreateScanRequest struct {
	Requests      []BatchCreateScanItem `json:"requests" binding:"required,min=1,max=5000"`
	ScanWorkflow  string                `json:"scanWorkflow" binding:"required"`
	Configuration map[string]any        `json:"configuration"`
	Agent         string                `json:"agent,omitempty"`
	InputSource   string                `json:"inputSource" binding:"required"`
}

func (req *BatchCreateScanRequest) UnmarshalJSON(data []byte) error {
	type rawRequest BatchCreateScanRequest
	var raw rawRequest
	if err := decodeScanCreateRequestFields(data, map[string]struct{}{
		"requests":      {},
		"scanWorkflow":  {},
		"configuration": {},
		"agent":         {},
		"inputSource":   {},
	}, &raw); err != nil {
		return err
	}
	*req = BatchCreateScanRequest(raw)
	return nil
}

type QuickScanResponse struct {
	Count       int                `json:"count"`
	TargetStats QuickTargetStats   `json:"targetStats"`
	AssetStats  QuickAssetStats    `json:"assetStats"`
	Errors      []QuickTargetError `json:"errors,omitempty"`
	Scans       []ScanResponse     `json:"scans"`
}

type BatchCreateScanResponse struct {
	Count        int                    `json:"count"`
	CreatedCount int                    `json:"createdCount"`
	Skipped      []BatchScanItemOutcome `json:"skipped"`
	Failed       []BatchScanItemOutcome `json:"failed"`
	Scans        []ScanResponse         `json:"scans"`
}

type BatchScanItemOutcome struct {
	Index        int    `json:"index"`
	Target       string `json:"target,omitempty"`
	Organization string `json:"organization,omitempty"`
	Reason       string `json:"reason"`
	Message      string `json:"message"`
}

type QuickTargetStats struct {
	Created int `json:"created"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

type QuickAssetStats struct {
	Websites  int `json:"websites"`
	Endpoints int `json:"endpoints"`
}

type QuickTargetError struct {
	Input string `json:"input"`
	Error string `json:"error"`
}

type StopScanResponse struct {
	RevokedTaskCount int `json:"revokedTaskCount"`
}

// BatchStopRequest identifies the Scan resources to stop. The bounded size
// keeps one synchronous lifecycle transaction from holding an unbounded lock
// set while preserving canonical resource-name semantics at the HTTP edge.
type BatchStopRequest struct {
	Names []string `json:"names" binding:"required,min=1,max=100"`
}

type BatchStopResponse struct {
	StoppedCount     int `json:"stoppedCount"`
	SkippedCount     int `json:"skippedCount"`
	RevokedTaskCount int `json:"revokedTaskCount"`
}

type ScanStatisticsResponse struct {
	Total           int64                              `json:"total"`
	Pending         int64                              `json:"pending"`
	Running         int64                              `json:"running"`
	Completed       int64                              `json:"completed"`
	Failed          int64                              `json:"failed"`
	Cancelled       int64                              `json:"cancelled"`
	TotalVulns      int64                              `json:"totalVulns"`
	TotalSubdomains int64                              `json:"totalSubdomains"`
	TotalEndpoints  int64                              `json:"totalEndpoints"`
	TotalWebsites   int64                              `json:"totalWebsites"`
	TotalAssets     int64                              `json:"totalAssets"`
	RetentionPolicy ScanHistoryRetentionPolicyResponse `json:"retentionPolicy"`
}

// ScanHistoryRetentionPolicyResponse is the public read-only retention state.
// It intentionally excludes raw environment variables and operational controls.
type ScanHistoryRetentionPolicyResponse struct {
	MinimumRetentionSeconds int64 `json:"minimumRetentionSeconds"`
	AutomaticCleanupEnabled bool  `json:"automaticCleanupEnabled"`
}

type BatchDeleteRequest struct {
	Names []string `json:"names" binding:"required,min=1,max=5000"`
}
