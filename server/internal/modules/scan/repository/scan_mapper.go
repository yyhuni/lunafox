package repository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

type ScanTargetRecord = scandomain.QueryTargetRef

type ScanRecord = scandomain.QueryScan

type ScanCreateRecord = scandomain.CreateScan

type CreateScanTaskRecord = scandomain.CreateScanTask

type ScanTaskRecord = scandomain.ScanTaskRecord

type RuntimeTaskRecord = scandomain.QueryRuntimeTask

type ScanTaskRuntimeTargetRecord = scandomain.ScanTaskTargetRef

type ScanTaskRuntimeScanRecord = scandomain.ScanTaskRuntimeScanRecord

type TaskProgressLogRecord = scandomain.TaskProgressLogEntry

type TaskProgressLogScanRefRecord = scandomain.TaskProgressLogScanRef

func scanTargetModelToRecord(item *model.ScanTargetRef) *ScanTargetRecord {
	if item == nil {
		return nil
	}

	return &ScanTargetRecord{
		ID:        item.ID,
		Name:      item.Name,
		Type:      item.Type,
		CreatedAt: timeutil.ToUTC(item.CreatedAt),
	}
}

func scanTargetModelToScanTaskRuntimeTargetRecord(item *model.ScanTargetRef) *ScanTaskRuntimeTargetRecord {
	if item == nil {
		return nil
	}

	return &ScanTaskRuntimeTargetRecord{
		ID:   item.ID,
		Name: item.Name,
		Type: item.Type,
	}
}

func scanModelToScanTaskRuntimeScanRecord(item *model.Scan) *ScanTaskRuntimeScanRecord {
	if item == nil {
		return nil
	}

	return &ScanTaskRuntimeScanRecord{
		ID:       item.ID,
		TargetID: item.TargetID,
		Status:   item.Status,
		Failure:  failureDetailFromColumns(item.FailureKind, item.ErrorMessage, ""),
		Target:   scanTargetModelToScanTaskRuntimeTargetRecord(item.Target),
	}
}

func scanModelToTaskProgressLogScanRefRecord(item *model.Scan) *TaskProgressLogScanRefRecord {
	if item == nil {
		return nil
	}

	return &TaskProgressLogScanRefRecord{
		ID:     item.ID,
		Status: item.Status,
	}
}

func scanModelToRecord(item *model.Scan) (*ScanRecord, error) {
	if item == nil {
		return nil, nil
	}

	triggerType, ok := scandomain.ParseScanTriggerType(item.TriggerType)
	if !ok {
		return nil, fmt.Errorf("%w: %q", scandomain.ErrInvalidScanTriggerType, item.TriggerType)
	}
	inputSource, ok := scandomain.ParseDatabaseInputSource(item.InputSource)
	if !ok {
		return nil, fmt.Errorf("%w: persisted value %q", scandomain.ErrInvalidInputSource, item.InputSource)
	}

	return &ScanRecord{
		ID:                     item.ID,
		TargetID:               item.TargetID,
		ScanWorkflowID:         item.ScanWorkflowID,
		PlannedEngineIDs:       []string{},
		Configuration:          decodeJSONMap(item.Configuration),
		InputSource:            inputSource,
		TriggerType:            triggerType,
		Status:                 item.Status,
		ResultsDir:             item.ResultsDir,
		AgentID:                item.AgentID,
		AgentName:              item.AgentName,
		AgentStatus:            item.AgentStatus,
		AgentHealthState:       item.AgentHealthState,
		AgentDeleted:           item.AgentDeleted,
		AssignmentMode:         item.AssignmentMode,
		ErrorMessage:           item.ErrorMessage,
		Failure:                failureDetailFromColumns(item.FailureKind, item.ErrorMessage, ""),
		Progress:               item.Progress,
		CurrentStage:           item.CurrentStage,
		CreatedAt:              timeutil.ToUTC(item.CreatedAt),
		StoppedAt:              timeutil.ToUTCPtr(item.StoppedAt),
		CachedSubdomainsCount:  item.CachedSubdomainsCount,
		CachedWebsitesCount:    item.CachedWebsitesCount,
		CachedEndpointsCount:   item.CachedEndpointsCount,
		CachedIPsCount:         item.CachedIPsCount,
		CachedDirectoriesCount: item.CachedDirectoriesCount,
		CachedScreenshotsCount: item.CachedScreenshotsCount,
		CachedVulnsTotal:       item.CachedVulnsTotal,
		CachedVulnsCritical:    item.CachedVulnsCritical,
		CachedVulnsHigh:        item.CachedVulnsHigh,
		CachedVulnsMedium:      item.CachedVulnsMedium,
		CachedVulnsLow:         item.CachedVulnsLow,
		Target:                 scanTargetModelToRecord(item.Target),
	}, nil
}

func scanModelListToRecord(items []model.Scan) ([]ScanRecord, error) {
	results := make([]ScanRecord, 0, len(items))
	for index := range items {
		record, err := scanModelToRecord(&items[index])
		if err != nil {
			return nil, err
		}
		results = append(results, *record)
	}
	return results, nil
}

func scanCreateRecordToModel(item *ScanCreateRecord) (*model.Scan, error) {
	if item == nil {
		return nil, nil
	}
	if !item.TriggerType.Valid() {
		return nil, fmt.Errorf("%w: %q", scandomain.ErrInvalidScanTriggerType, item.TriggerType)
	}
	inputSource, ok := item.InputSource.DatabaseValue()
	if !ok {
		return nil, fmt.Errorf("%w: %q", scandomain.ErrInvalidInputSource, item.InputSource)
	}

	return &model.Scan{
		ID:             item.ID,
		TargetID:       item.TargetID,
		ScanWorkflowID: item.ScanWorkflowID,
		Configuration:  encodeJSONMap(item.Configuration),
		InputSource:    inputSource,
		TriggerType:    string(item.TriggerType),
		AssignmentMode: item.AssignmentMode,
		AgentID:        item.AgentID,
		Status:         item.Status,
		CreatedAt:      timeutil.ToUTC(item.CreatedAt),
	}, nil
}

func scanTaskCreateRecordToModel(item *CreateScanTaskRecord, scanID int) *model.ScanTask {
	if item == nil {
		return nil
	}

	return &model.ScanTask{
		ScanID:                scanID,
		StageOrder:            item.StageOrder,
		StageID:               item.StageID,
		StepOrder:             item.StepOrder,
		StepID:                item.StepID,
		EngineID:              item.EngineID,
		EngineConfig:          encodeJSONMap(item.EngineConfig),
		TaskExecutionConfig:   encodeJSONMap(item.TaskExecutionConfig),
		ResolvedExecutionPlan: append([]byte(nil), item.ResolvedExecutionPlan...),
		Status:                item.Status,
		SkipReason:            strings.TrimSpace(item.SkipReason),
	}
}

func scanTaskRuntimeRowsToRuntimeTaskRecord(items []scanTaskRuntimeRow) ([]RuntimeTaskRecord, error) {
	results := make([]RuntimeTaskRecord, 0, len(items))
	for index := range items {
		item := items[index]
		diagnostics, err := decodeEngineExecutionDiagnostics(item.EngineDiagnostics)
		if err != nil {
			return nil, err
		}
		if !requiresEngineDiagnostics(item.Status, item.HasResolvedExecutionPlan) {
			if diagnostics != nil {
				return nil, fmt.Errorf("scan task without Engine execution must not contain Engine diagnostics")
			}
		} else if diagnostics == nil {
			return nil, fmt.Errorf("terminal scan task Engine diagnostics are required")
		}
		results = append(results, RuntimeTaskRecord{
			ID:            item.ID,
			StepID:        item.ScanWorkflowStepID,
			StageID:       item.ScanWorkflowStageID,
			EngineID:      item.EngineID,
			Status:        item.Status,
			SkipReason:    strings.TrimSpace(item.SkipReason),
			Order:         item.ScanWorkflowStageOrder,
			StartedAt:     timeutil.ToUTCPtr(item.StartedAt),
			CompletedAt:   timeutil.ToUTCPtr(item.CompletedAt),
			Duration:      runtimeTaskDurationSeconds(item.StartedAt, item.CompletedAt),
			Error:         item.ErrorMessage,
			FailureKind:   item.FailureKind,
			FailureDetail: item.FailureDetail,
			Diagnostics:   diagnostics,
		})
	}
	return results, nil
}

// requiresEngineDiagnostics keys the terminal snapshot requirement to the
// immutable saved plan, not started_at. A container can exit before start time
// is recorded, but it is still an Engine execution whose unavailable evidence
// must be retained. Planning-time skipped tasks have no saved plan.
func requiresEngineDiagnostics(status string, hasResolvedExecutionPlan bool) bool {
	parsed, ok := scandomain.ParseTaskStatus(status)
	return ok && hasResolvedExecutionPlan && scandomain.IsTerminalTaskStatus(parsed)
}

func runtimeTaskDurationSeconds(startedAt *time.Time, completedAt *time.Time) *float64 {
	if startedAt == nil || completedAt == nil {
		return nil
	}
	started := startedAt.UTC()
	completed := completedAt.UTC()
	if completed.Before(started) {
		return nil
	}
	duration := completed.Sub(started).Seconds()
	return &duration
}

func failureDetailFromColumns(kind string, message string, displayMessage string) *scandomain.FailureDetail {
	kind = strings.TrimSpace(kind)
	message = strings.TrimSpace(message)
	if kind == "" || message == "" {
		return nil
	}
	return &scandomain.FailureDetail{Kind: kind, Message: message, DisplayMessage: strings.TrimSpace(displayMessage)}
}

func taskProgressLogModelToRecord(item *model.TaskProgressLog) *TaskProgressLogRecord {
	if item == nil {
		return nil
	}

	return &TaskProgressLogRecord{
		ID:        item.ID,
		ScanID:    item.ScanID,
		TaskID:    item.TaskID,
		RequestID: item.RequestID,
		Sequence:  item.Sequence,
		Level:     item.Level,
		Content:   item.Content,
		EmittedAt: timeutil.ToUTCPtr(item.EmittedAt),
		CreatedAt: timeutil.ToUTC(item.CreatedAt),
	}
}

func taskProgressLogModelListToRecord(items []model.TaskProgressLog) []TaskProgressLogRecord {
	results := make([]TaskProgressLogRecord, 0, len(items))
	for index := range items {
		results = append(results, *taskProgressLogModelToRecord(&items[index]))
	}
	return results
}

func taskProgressLogRecordToModel(item *TaskProgressLogRecord) *model.TaskProgressLog {
	if item == nil {
		return nil
	}

	return &model.TaskProgressLog{
		ID:        item.ID,
		ScanID:    item.ScanID,
		TaskID:    item.TaskID,
		RequestID: item.RequestID,
		Sequence:  item.Sequence,
		Level:     item.Level,
		Content:   item.Content,
		EmittedAt: timeutil.ToUTCPtr(item.EmittedAt),
		CreatedAt: timeutil.ToUTC(item.CreatedAt),
	}
}

func taskProgressLogRecordListToModel(items []TaskProgressLogRecord) []model.TaskProgressLog {
	results := make([]model.TaskProgressLog, 0, len(items))
	for index := range items {
		results = append(results, *taskProgressLogRecordToModel(&items[index]))
	}
	return results
}

func encodeJSONMap(value map[string]any) datatypes.JSON {
	// Scan persistence stores absent configuration as an empty JSON object so
	// task identity allocation never violates the non-null JSON column contract.
	if len(value) == 0 {
		return datatypes.JSON([]byte("{}"))
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return datatypes.JSON([]byte("{}"))
	}

	return datatypes.JSON(payload)
}

func decodeJSONMap(value datatypes.JSON) map[string]any {
	if len(value) == 0 {
		return nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return nil
	}

	return decoded
}

func encodeEngineExecutionDiagnostics(value *scandomain.EngineExecutionDiagnostics) (datatypes.JSON, error) {
	if value == nil {
		return nil, nil
	}
	if err := scandomain.ValidateEngineExecutionDiagnostics(value); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode Engine diagnostics: %w", err)
	}
	return datatypes.JSON(payload), nil
}

// unavailableEngineExecutionDiagnosticsJSON is the fixed Server-owned terminal
// observation for paths that close an executable task before the Engine can
// upload a snapshot. It prevents lifecycle callers from inventing counters or
// leaving a terminal saved plan without explicit evidence state.
func unavailableEngineExecutionDiagnosticsJSON() (datatypes.JSON, error) {
	return encodeEngineExecutionDiagnostics(scandomain.UnavailableEngineExecutionDiagnostics())
}

// unavailableEngineExecutionDiagnosticsForSavedPlan preserves the execution
// boundary in bulk lifecycle updates. Planning-time skipped rows have no saved
// plan and therefore must not acquire Engine evidence merely because their Scan
// is later cancelled or deleted.
func unavailableEngineExecutionDiagnosticsForSavedPlan(dialectName string) (clause.Expr, error) {
	diagnostics, err := unavailableEngineExecutionDiagnosticsJSON()
	if err != nil {
		return clause.Expr{}, err
	}
	sql := "CASE WHEN COALESCE(LENGTH(resolved_execution_plan), 0) > 0 THEN ? ELSE NULL END"
	if dialectName == "postgres" {
		// PostgreSQL resolves an untyped parameter in CASE as text unless the
		// JSONB target type is explicit; assignment context is not enough.
		sql = "CASE WHEN COALESCE(LENGTH(resolved_execution_plan), 0) > 0 THEN CAST(? AS jsonb) ELSE CAST(NULL AS jsonb) END"
	}
	return clause.Expr{
		SQL:  sql,
		Vars: []any{diagnostics},
	}, nil
}

func decodeEngineExecutionDiagnostics(value datatypes.JSON) (*scandomain.EngineExecutionDiagnostics, error) {
	payload := bytes.TrimSpace(value)
	if len(payload) == 0 || bytes.Equal(payload, []byte("null")) {
		return nil, nil
	}
	var diagnostics scandomain.EngineExecutionDiagnostics
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&diagnostics); err != nil {
		return nil, fmt.Errorf("decode persisted Engine diagnostics: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("decode persisted Engine diagnostics: multiple JSON values")
		}
		return nil, fmt.Errorf("decode persisted Engine diagnostics: %w", err)
	}
	if err := scandomain.ValidateEngineExecutionDiagnostics(&diagnostics); err != nil {
		return nil, fmt.Errorf("persisted Engine diagnostics are invalid: %w", err)
	}
	return scandomain.CloneEngineExecutionDiagnostics(&diagnostics), nil
}
