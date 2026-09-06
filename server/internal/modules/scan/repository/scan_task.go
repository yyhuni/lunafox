package repository

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ScanTaskRepository interface {
	GetByID(ctx context.Context, id int) (*ScanTaskRecord, error)
	GetSavedExecutionPlanLease(ctx context.Context, taskID int) (*scandomain.SavedExecutionPlanLease, error)
	ClaimNextCompatibleSavedExecutionPlan(ctx context.Context, agentID int, sessionID string, sessionEpoch int64, requestID string, supportedEngineAPIMajors []uint32) (*agentexecutionv1.ResolvedEngineExecutionPlan, error)
	RequireCurrentAgentExecutionSession(ctx context.Context, agentID int, sessionID string, sessionEpoch int64) error
	CommitScanTaskTerminalStatusForSession(ctx context.Context, id, agentID int, sessionID string, sessionEpoch int64, status string, failure *scandomain.FailureDetail) (bool, error)
	CommitScanTaskTerminalStatusWithDiagnosticsForSession(ctx context.Context, id, agentID int, sessionID string, sessionEpoch int64, status string, failure *scandomain.FailureDetail, diagnostics *scandomain.EngineExecutionDiagnostics) (bool, error)
	ListTerminalTasksPendingReconciliation(ctx context.Context, afterTaskID, limit int) ([]ScanTaskRecord, error)
	ListSupersededSessionTerminalTasksPendingReconciliation(ctx context.Context, agentID int, currentSessionEpoch int64, limit int) ([]ScanTaskRecord, error)
	ClearTerminalTaskReconciliationPending(ctx context.Context, taskID int) error
	FailClaimedTask(ctx context.Context, id int, failure *scandomain.FailureDetail) error
	SkipUnstartedTasksByScanID(ctx context.Context, scanID int, reason string) (int64, error)
	CancelUnstartedTasksByScanID(ctx context.Context, scanID int) (int64, error)
	ListFailedByScanID(ctx context.Context, scanID int) ([]ScanTaskRecord, error)
	CountByStatusForScanID(ctx context.Context, scanID int) (pending, running, completed, failed, cancelled, skipped int, err error)
	CountActiveByScanAndStageOrder(ctx context.Context, scanID, scanWorkflowStageOrder int) (int, error)
	UnlockNextStageOrder(ctx context.Context, scanID, scanWorkflowStageOrder int) (int64, error)
	CancelTasksByScanID(ctx context.Context, scanID int) ([]CancelledTaskInfo, error)
	FailTasksForOfflineAgent(ctx context.Context, agentID int) ([]int, error)
	FailTasksForSupersededAgentSession(ctx context.Context, agentID int, currentSessionEpoch int64) ([]int, error)
}

type scanTaskRepository struct{ db *gorm.DB }

type CancelledTaskInfo struct {
	TaskID  int  `gorm:"column:id"`
	AgentID *int `gorm:"column:agent_id"`
}

const (
	taskStatusBlocked   = string(scandomain.TaskStatusBlocked)
	taskStatusPending   = string(scandomain.TaskStatusPending)
	taskStatusRunning   = string(scandomain.TaskStatusRunning)
	taskStatusSucceeded = string(scandomain.TaskStatusSucceeded)
	taskStatusSkipped   = string(scandomain.TaskStatusSkipped)
	taskStatusFailed    = string(scandomain.TaskStatusFailed)
	taskStatusCancelled = string(scandomain.TaskStatusCancelled)
)

const (
	scanStatusPending   = string(scandomain.ScanStatusPending)
	scanStatusRunning   = string(scandomain.ScanStatusRunning)
	scanStatusSucceeded = string(scandomain.ScanStatusSucceeded)
	scanStatusFailed    = string(scandomain.ScanStatusFailed)
	scanStatusCancelled = string(scandomain.ScanStatusCancelled)
)

func NewScanTaskRepository(db *gorm.DB) ScanTaskRepository {
	return &scanTaskRepository{db: db}
}

func (r *scanTaskRepository) GetByID(ctx context.Context, id int) (*ScanTaskRecord, error) {
	var task scanTaskRuntimeRow
	if err := r.scanTaskRuntimeQuery(ctx).Where("st.id = ?", id).First(&task).Error; err != nil {
		return nil, err
	}
	return scanTaskRuntimeRowToRecord(&task)
}

func (r *scanTaskRepository) RequireCurrentAgentExecutionSession(ctx context.Context, agentID int, sessionID string, sessionEpoch int64) error {
	if r == nil || r.db == nil || ctx == nil {
		return scandomain.ErrAgentExecutionSessionFenced
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return requirePersistedAgentExecutionSession(tx, agentID, sessionID, sessionEpoch)
	})
}

func (r *scanTaskRepository) CommitScanTaskTerminalStatusForSession(ctx context.Context, id, agentID int, sessionID string, sessionEpoch int64, status string, failure *scandomain.FailureDetail) (bool, error) {
	return r.CommitScanTaskTerminalStatusWithDiagnosticsForSession(ctx, id, agentID, sessionID, sessionEpoch, status, failure, scandomain.UnavailableEngineExecutionDiagnostics())
}

func (r *scanTaskRepository) CommitScanTaskTerminalStatusWithDiagnosticsForSession(ctx context.Context, id, agentID int, sessionID string, sessionEpoch int64, status string, failure *scandomain.FailureDetail, diagnostics *scandomain.EngineExecutionDiagnostics) (bool, error) {
	return r.commitScanTaskTerminalStatus(ctx, id, agentID, strings.TrimSpace(sessionID), sessionEpoch, status, failure, diagnostics)
}

func (r *scanTaskRepository) commitScanTaskTerminalStatus(ctx context.Context, id, agentID int, sessionID string, sessionEpoch int64, status string, failure *scandomain.FailureDetail, diagnostics *scandomain.EngineExecutionDiagnostics) (bool, error) {
	if r == nil || r.db == nil || ctx == nil || id <= 0 || agentID <= 0 || sessionEpoch <= 0 {
		return false, fmt.Errorf("terminal scan task lease scope is invalid")
	}
	if sessionID == "" {
		return false, fmt.Errorf("terminal scan task session ID is required")
	}
	if status != taskStatusSucceeded && status != taskStatusFailed && status != taskStatusCancelled {
		return false, fmt.Errorf("terminal scan task status is invalid: %q", status)
	}
	failure, err := normalizeTaskFailureDetail(status, failure)
	if err != nil {
		return false, err
	}
	encodedDiagnostics, err := encodeEngineExecutionDiagnostics(diagnostics)
	if err != nil {
		return false, err
	}
	if encodedDiagnostics == nil {
		return false, fmt.Errorf("terminal Engine diagnostics are required")
	}
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"status":                          status,
		"completed_at":                    now,
		"terminal_reconciliation_pending": true,
		"error_message":                   "",
		"failure_kind":                    "",
		"failure_detail":                  "",
		"engine_diagnostics":              encodedDiagnostics,
	}
	if failure != nil {
		updates["error_message"] = failure.Message
		updates["failure_kind"] = failure.Kind
		updates["failure_detail"] = failure.DisplayMessage
	}

	var committed bool
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := requirePersistedAgentExecutionSession(tx, agentID, sessionID, sessionEpoch); err != nil {
			return err
		}
		query := tx.Model(&model.ScanTask{}).
			Where(`id = ? AND status = ? AND assigned_session_epoch = ?
				AND scan_id IN (SELECT id FROM scan WHERE agent_id = ?)`, id, taskStatusRunning, sessionEpoch, agentID)
		query = query.
			Where("assigned_agent_id = ?", agentID).
			Where("assigned_session_id = ?", sessionID)
		result := query.Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		committed = result.RowsAffected == 1
		return nil
	})
	return committed, err
}

func (r *scanTaskRepository) ListTerminalTasksPendingReconciliation(ctx context.Context, afterTaskID, limit int) ([]ScanTaskRecord, error) {
	if r == nil || r.db == nil || ctx == nil || afterTaskID < 0 || limit <= 0 || limit > 1000 {
		return nil, fmt.Errorf("terminal reconciliation cursor is invalid")
	}
	var rows []scanTaskRuntimeRow
	if err := r.scanTaskRuntimeQuery(ctx).
		Where("st.terminal_reconciliation_pending = ?", true).
		Where("st.status IN ?", []string{taskStatusSucceeded, taskStatusFailed, taskStatusCancelled}).
		Where("st.id > ?", afterTaskID).
		Order("st.id ASC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	records := make([]ScanTaskRecord, 0, len(rows))
	for index := range rows {
		record, err := scanTaskRuntimeRowToRecord(&rows[index])
		if err != nil {
			return nil, err
		}
		records = append(records, *record)
	}
	return records, nil
}

func (r *scanTaskRepository) ListSupersededSessionTerminalTasksPendingReconciliation(ctx context.Context, agentID int, currentSessionEpoch int64, limit int) ([]ScanTaskRecord, error) {
	if r == nil || r.db == nil || ctx == nil || agentID <= 0 || currentSessionEpoch <= 0 || limit <= 0 || limit > 1000 {
		return nil, fmt.Errorf("superseded terminal reconciliation scope is invalid")
	}
	var rows []scanTaskRuntimeRow
	if err := r.scanTaskRuntimeQuery(ctx).
		Where("s.agent_id = ?", agentID).
		Where("st.assigned_session_epoch < ?", currentSessionEpoch).
		Where("st.terminal_reconciliation_pending = ?", true).
		Where("st.status IN ?", []string{taskStatusSucceeded, taskStatusFailed, taskStatusCancelled}).
		Order("st.id ASC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	records := make([]ScanTaskRecord, 0, len(rows))
	for index := range rows {
		record, err := scanTaskRuntimeRowToRecord(&rows[index])
		if err != nil {
			return nil, err
		}
		records = append(records, *record)
	}
	return records, nil
}

func (r *scanTaskRepository) ClearTerminalTaskReconciliationPending(ctx context.Context, taskID int) error {
	if r == nil || r.db == nil || ctx == nil || taskID <= 0 {
		return fmt.Errorf("terminal reconciliation task is invalid")
	}
	return r.db.WithContext(ctx).Model(&model.ScanTask{}).
		Where("id = ? AND terminal_reconciliation_pending = ?", taskID, true).
		Update("terminal_reconciliation_pending", false).Error
}

func (r *scanTaskRepository) FailClaimedTask(ctx context.Context, id int, failure *scandomain.FailureDetail) error {
	failure, err := normalizeFailedFailureDetail(failure)
	if err != nil {
		return err
	}
	unavailableDiagnostics, err := unavailableEngineExecutionDiagnosticsJSON()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Model(&model.ScanTask{}).
			Where("id = ? AND status = ?", id, taskStatusRunning).
			Updates(map[string]interface{}{
				"status":                          taskStatusFailed,
				"started_at":                      nil,
				"completed_at":                    &now,
				"terminal_reconciliation_pending": true,
				"error_message":                   failure.Message,
				"failure_kind":                    failure.Kind,
				"failure_detail":                  failure.DisplayMessage,
				"engine_diagnostics":              unavailableDiagnostics,
			}).Error
	})
}

func (r *scanTaskRepository) SkipUnstartedTasksByScanID(ctx context.Context, scanID int, reason string) (int64, error) {
	diagnosticsUpdate, err := unavailableEngineExecutionDiagnosticsForSavedPlan(r.db.Dialector.Name())
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&model.ScanTask{}).
		Where("scan_id = ? AND status IN ?", scanID, []string{taskStatusPending, taskStatusBlocked}).
		Updates(map[string]interface{}{
			"status":             taskStatusSkipped,
			"skip_reason":        strings.TrimSpace(reason),
			"completed_at":       now,
			"engine_diagnostics": diagnosticsUpdate,
		})
	return result.RowsAffected, result.Error
}

func (r *scanTaskRepository) CancelUnstartedTasksByScanID(ctx context.Context, scanID int) (int64, error) {
	diagnosticsUpdate, err := unavailableEngineExecutionDiagnosticsForSavedPlan(r.db.Dialector.Name())
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&model.ScanTask{}).
		Where("scan_id = ? AND status IN ?", scanID, []string{taskStatusPending, taskStatusBlocked}).
		Updates(map[string]interface{}{
			"status":             taskStatusCancelled,
			"completed_at":       now,
			"engine_diagnostics": diagnosticsUpdate,
		})
	return result.RowsAffected, result.Error
}

func (r *scanTaskRepository) ListFailedByScanID(ctx context.Context, scanID int) ([]ScanTaskRecord, error) {
	var rows []scanTaskRuntimeRow
	err := r.scanTaskRuntimeQuery(ctx).Where("st.scan_id = ? AND st.status = ?", scanID, taskStatusFailed).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	results := make([]ScanTaskRecord, 0, len(rows))
	for index := range rows {
		record, err := scanTaskRuntimeRowToRecord(&rows[index])
		if err != nil {
			return nil, err
		}
		results = append(results, *record)
	}
	return results, nil
}

func (r *scanTaskRepository) CountByStatusForScanID(ctx context.Context, scanID int) (pending, running, completed, failed, cancelled, skipped int, err error) {
	var results []struct {
		Status string
		Count  int
	}
	err = r.db.WithContext(ctx).
		Model(&model.ScanTask{}).
		Select("status, COUNT(*) as count").
		Where("scan_id = ?", scanID).
		Group("status").
		Scan(&results).Error
	if err != nil {
		return
	}
	for _, row := range results {
		switch row.Status {
		case taskStatusPending:
			pending += row.Count
		case taskStatusBlocked:
			// Blocked tasks are dependency-gated work, not active pending work for scan status resolution.
		case taskStatusRunning:
			running = row.Count
		case taskStatusSucceeded:
			completed = row.Count
		case taskStatusSkipped:
			skipped = row.Count
		case taskStatusFailed:
			failed = row.Count
		case taskStatusCancelled:
			cancelled = row.Count
		}
	}
	return
}

func (r *scanTaskRepository) CountActiveByScanAndStageOrder(ctx context.Context, scanID, scanWorkflowStageOrder int) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("scan_task AS st").
		Where("st.scan_id = ? AND st.stage_order = ? AND st.status IN ?", scanID, scanWorkflowStageOrder, []string{taskStatusPending, taskStatusRunning}).
		Count(&count).Error
	return int(count), err
}

func (r *scanTaskRepository) UnlockNextStageOrder(ctx context.Context, scanID, scanWorkflowStageOrder int) (int64, error) {
	var rowsAffected int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		nextStageOrder, err := r.findNextUnlockableStageOrder(ctx, tx, scanID, scanWorkflowStageOrder)
		if err != nil {
			return err
		}
		if nextStageOrder == 0 {
			return nil
		}

		affected, err := r.unlockStageOrder(ctx, tx, scanID, nextStageOrder)
		if err != nil {
			return err
		}
		rowsAffected = affected
		return nil
	})
	return rowsAffected, err
}

func (r *scanTaskRepository) findNextUnlockableStageOrder(ctx context.Context, tx *gorm.DB, scanID, scanWorkflowStageOrder int) (int, error) {
	var stageOrder int
	err := tx.WithContext(ctx).Raw(`
		SELECT COALESCE(MIN(candidate.stage_order), 0)
		FROM scan_task AS candidate
		WHERE candidate.scan_id = ?
		  AND candidate.status = ?
		  AND candidate.stage_order > ?
		  AND NOT EXISTS (
			SELECT 1
			FROM scan_task AS active
			WHERE active.scan_id = candidate.scan_id
			  AND active.stage_order > ?
			  AND active.stage_order < candidate.stage_order
			  AND active.status IN (?, ?, ?, ?)
		)
	`, scanID, taskStatusBlocked, scanWorkflowStageOrder, scanWorkflowStageOrder, taskStatusPending, taskStatusRunning, taskStatusFailed, taskStatusCancelled).Scan(&stageOrder).Error
	return stageOrder, err
}

func (r *scanTaskRepository) unlockStageOrder(ctx context.Context, tx *gorm.DB, scanID, stageOrder int) (int64, error) {
	result := tx.WithContext(ctx).Exec(`
		UPDATE scan_task
		SET status = ?
		WHERE scan_id = ?
		  AND status = ?
		  AND stage_order = ?
	`, taskStatusPending, scanID, taskStatusBlocked, stageOrder)
	return result.RowsAffected, result.Error
}

func (r *scanTaskRepository) CancelTasksByScanID(ctx context.Context, scanID int) ([]CancelledTaskInfo, error) {
	var rows []CancelledTaskInfo
	diagnosticsUpdate, err := unavailableEngineExecutionDiagnosticsForSavedPlan(r.db.Dialector.Name())
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("scan_task AS st").Joins("JOIN scan AS s ON s.id = st.scan_id").Select("st.id AS id, CASE WHEN st.status = ? THEN s.agent_id ELSE NULL END AS agent_id", taskStatusRunning).Where("st.scan_id = ? AND st.status IN ?", scanID, []string{taskStatusPending, taskStatusRunning, taskStatusBlocked}).Scan(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Model(&model.ScanTask{}).Where("scan_id = ? AND status IN ?", scanID, []string{taskStatusPending, taskStatusRunning, taskStatusBlocked}).Updates(map[string]interface{}{
			"status":             taskStatusCancelled,
			"completed_at":       now,
			"engine_diagnostics": diagnosticsUpdate,
		}).Error
	})
	return rows, err
}

func (r *scanTaskRepository) FailTasksForOfflineAgent(ctx context.Context, agentID int) ([]int, error) {
	return r.failRunningTasksForAgent(ctx, agentID, nil)
}

func (r *scanTaskRepository) FailTasksForSupersededAgentSession(ctx context.Context, agentID int, currentSessionEpoch int64) ([]int, error) {
	return r.failRunningTasksForAgent(ctx, agentID, &currentSessionEpoch)
}

func (r *scanTaskRepository) failRunningTasksForAgent(ctx context.Context, agentID int, currentSessionEpoch *int64) ([]int, error) {
	var scanIDs []int
	unavailableDiagnostics, err := unavailableEngineExecutionDiagnosticsJSON()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Table("scan_task AS st").
			Joins("JOIN scan AS s ON s.id = st.scan_id").
			Where("s.agent_id = ?", agentID)
		if currentSessionEpoch != nil {
			query = query.
				Where("st.assigned_session_epoch < ?", *currentSessionEpoch).
				Where("s.status NOT IN ?", []string{scanStatusFailed, scanStatusCancelled}).
				Where("(st.status = ? OR (st.status = ? AND st.failure_kind = ?))", taskStatusRunning, taskStatusFailed, "agent_disconnected")
		} else {
			query = query.Where("st.status = ?", taskStatusRunning)
		}
		if err := query.Distinct("st.scan_id").Pluck("st.scan_id", &scanIDs).Error; err != nil {
			return err
		}
		scanIDs = uniqueInts(scanIDs)
		if len(scanIDs) == 0 {
			return nil
		}
		updates := map[string]interface{}{
			"status":                          taskStatusFailed,
			"completed_at":                    now,
			"terminal_reconciliation_pending": true,
			"error_message":                   "Agent disconnected",
			"failure_kind":                    "agent_disconnected",
			"failure_detail":                  "",
			"engine_diagnostics":              unavailableDiagnostics,
		}
		updateQuery := tx.Model(&model.ScanTask{}).Where("scan_id IN ? AND status = ?", scanIDs, taskStatusRunning)
		if currentSessionEpoch != nil {
			updateQuery = updateQuery.Where("assigned_session_epoch < ?", *currentSessionEpoch)
		}
		return updateQuery.Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return uniqueInts(scanIDs), nil
}

type scanTaskRuntimeRow struct {
	ID                       int
	ScanID                   int
	ScanWorkflowStageOrder   int
	ScanWorkflowID           string
	ScanWorkflowStageID      string
	ScanWorkflowStepID       string
	EngineID                 string
	Status                   string
	SkipReason               string
	AgentID                  *int
	AssignedAgentID          *int
	AssignedSessionID        *string
	AssignedSessionEpoch     *int64
	AssignedRequestID        *string
	HasResolvedExecutionPlan bool
	TaskExecutionConfig      []byte
	ErrorMessage             string
	FailureKind              string
	FailureDetail            string
	EngineDiagnostics        datatypes.JSON
	StartedAt                *time.Time
	CompletedAt              *time.Time
}

func (r *scanTaskRepository) scanTaskRuntimeQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("scan_task AS st").
		Joins("JOIN scan AS s ON s.id = st.scan_id").
		Select(strings.Join([]string{
			"st.id AS id",
			"st.scan_id AS scan_id",
			"st.stage_order AS scan_workflow_stage_order",
			"s.scan_workflow_id AS scan_workflow_id",
			"st.stage_id AS scan_workflow_stage_id",
			"st.step_id AS scan_workflow_step_id",
			"st.engine_id AS engine_id",
			"st.status AS status",
			"st.skip_reason AS skip_reason",
			"s.agent_id AS agent_id",
			"st.assigned_agent_id AS assigned_agent_id",
			"st.assigned_session_id AS assigned_session_id",
			"st.assigned_session_epoch AS assigned_session_epoch",
			"st.assigned_request_id AS assigned_request_id",
			"(LENGTH(st.resolved_execution_plan) > 0) AS has_resolved_execution_plan",
			"st.task_execution_config AS task_execution_config",
			"st.error_message AS error_message",
			"st.failure_kind AS failure_kind",
			"st.failure_detail AS failure_detail",
			"st.engine_diagnostics AS engine_diagnostics",
			"st.started_at AS started_at",
			"st.completed_at AS completed_at",
		}, ", "))
}

func scanTaskRuntimeRowToRecord(row *scanTaskRuntimeRow) (*ScanTaskRecord, error) {
	if row == nil {
		return nil, nil
	}
	diagnostics, err := decodeEngineExecutionDiagnostics(row.EngineDiagnostics)
	if err != nil {
		return nil, err
	}
	if requiresEngineDiagnostics(row.Status, row.HasResolvedExecutionPlan) {
		if diagnostics == nil {
			return nil, fmt.Errorf("terminal scan task Engine diagnostics are required")
		}
	} else if diagnostics != nil {
		return nil, fmt.Errorf("scan task without Engine execution must not contain Engine diagnostics")
	}
	return &ScanTaskRecord{
		ID:                       row.ID,
		ScanID:                   row.ScanID,
		ScanWorkflowStageOrder:   row.ScanWorkflowStageOrder,
		ScanWorkflowID:           row.ScanWorkflowID,
		ScanWorkflowStageID:      row.ScanWorkflowStageID,
		ScanWorkflowStepID:       row.ScanWorkflowStepID,
		EngineID:                 row.EngineID,
		Status:                   row.Status,
		SkipReason:               strings.TrimSpace(row.SkipReason),
		AgentID:                  row.AgentID,
		AssignedAgentID:          row.AssignedAgentID,
		AssignedSessionID:        row.AssignedSessionID,
		AssignedSessionEpoch:     row.AssignedSessionEpoch,
		AssignedRequestID:        row.AssignedRequestID,
		HasResolvedExecutionPlan: row.HasResolvedExecutionPlan,
		TaskExecutionConfig:      decodeJSONMap(row.TaskExecutionConfig),
		Failure:                  failureDetailFromColumns(row.FailureKind, row.ErrorMessage, row.FailureDetail),
		Diagnostics:              diagnostics,
		CompletedAt:              timeutilToUTCPtr(row.CompletedAt),
	}, nil
}

func timeutilToUTCPtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

func uniqueInts(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[int]struct{}, len(values))
	results := make([]int, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		results = append(results, value)
	}
	return results
}

func normalizeTaskFailureDetail(status string, failure *scandomain.FailureDetail) (*scandomain.FailureDetail, error) {
	if strings.TrimSpace(status) != taskStatusFailed {
		return nil, nil
	}
	return normalizeFailedFailureDetail(failure)
}

func normalizeFailedFailureDetail(failure *scandomain.FailureDetail) (*scandomain.FailureDetail, error) {
	if failure == nil {
		return nil, fmt.Errorf("failed task requires failure detail")
	}
	message := strings.TrimSpace(failure.Message)
	if message == "" {
		return nil, fmt.Errorf("failed task requires non-empty failure message")
	}
	kind := canonicalTaskFailureKind(strings.TrimSpace(failure.Kind))
	if kind == "" {
		return nil, fmt.Errorf("failed task requires non-empty failure kind")
	}
	displayMessage := failure.DisplayMessage
	if displayMessage != "" && !validFailureDetail(displayMessage) {
		return nil, fmt.Errorf("failed task requires valid failure detail")
	}
	return &scandomain.FailureDetail{Kind: kind, Message: message, DisplayMessage: displayMessage}, nil
}

func validFailureDetail(value string) bool {
	return len(value) <= 500 && utf8.ValidString(value) && value == strings.TrimSpace(value) && strings.IndexFunc(value, unicode.IsControl) < 0
}

func canonicalTaskFailureKind(value string) string {
	return strings.TrimSpace(value)
}
