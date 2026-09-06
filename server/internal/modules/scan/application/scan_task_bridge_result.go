package application

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

// ReportTerminalTaskResult is the session-fenced control-plane boundary.
func (service *ScanTaskBridgeService) ReportTerminalTaskResult(
	ctx context.Context,
	agentID int,
	sessionID string,
	sessionEpoch int64,
	taskID int,
	status string,
	failure *FailureDetail,
) error {
	return service.ReportTerminalTaskResultWithDiagnostics(
		ctx,
		agentID,
		sessionID,
		sessionEpoch,
		taskID,
		status,
		failure,
		scandomain.UnavailableEngineExecutionDiagnostics(),
	)
}

// ReportTerminalTaskResultWithDiagnostics is the strict Agent-control
// terminal boundary. Its diagnostic parameter is required and is persisted in
// the same fenced transaction as the task result.
func (service *ScanTaskBridgeService) ReportTerminalTaskResultWithDiagnostics(
	ctx context.Context,
	agentID int,
	sessionID string,
	sessionEpoch int64,
	taskID int,
	status string,
	failure *FailureDetail,
	diagnostics *EngineExecutionDiagnostics,
) error {
	return service.updateScanTaskStatus(ctx, agentID, strings.TrimSpace(sessionID), sessionEpoch, taskID, status, failure, diagnostics)
}

func (service *ScanTaskBridgeService) updateScanTaskStatus(
	ctx context.Context,
	agentID int,
	sessionID string,
	sessionEpoch int64,
	taskID int,
	status string,
	failure *FailureDetail,
	diagnostics *EngineExecutionDiagnostics,
) error {
	status = strings.TrimSpace(status)
	if status == "" {
		return ErrScanTaskInvalidUpdate
	}
	if sessionID == "" {
		return ErrScanTaskInvalidUpdate
	}
	if err := service.taskStore.RequireCurrentAgentExecutionSession(ctx, agentID, sessionID, sessionEpoch); err != nil {
		if errors.Is(err, scandomain.ErrAgentExecutionSessionFenced) {
			return ErrScanTaskNotOwned
		}
		return err
	}

	nextStatus, ok := scandomain.ParseTaskStatus(status)
	if !ok || !scandomain.IsTerminalTaskStatus(nextStatus) {
		return ErrScanTaskInvalidTransition
	}

	normalizedFailure, err := normalizeTaskFailure(nextStatus, failure)
	if err != nil {
		return err
	}
	if err := scandomain.ValidateTerminalEngineExecutionDiagnostics(nextStatus, diagnostics); err != nil {
		return ErrScanTaskInvalidUpdate
	}
	normalizedDiagnostics := scandomain.CloneEngineExecutionDiagnostics(diagnostics)

	task, err := service.taskStore.GetByID(ctx, taskID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrScanTaskNotFound
		}
		return err
	}
	if !terminalTaskLeaseMatches(task, agentID, sessionID, sessionEpoch) {
		return ErrScanTaskNotOwned
	}

	currentStatus, ok := scandomain.ParseTaskStatus(task.Status)
	if !ok {
		return ErrScanTaskInvalidTransition
	}
	if currentStatus == nextStatus {
		if !terminalTaskResultMatches(task, nextStatus, normalizedFailure, normalizedDiagnostics) {
			return ErrScanTaskInvalidTransition
		}
		return service.reconcileAndClearTerminalTaskResult(ctx, task, nextStatus)
	}

	domainTask := &scandomain.ScanTask{Status: currentStatus}
	if err := domainTask.ApplyAgentResult(nextStatus, failureMessage(normalizedFailure), time.Now().UTC()); err != nil {
		if errors.Is(err, scandomain.ErrFailureMessageMissing) {
			return ErrScanTaskInvalidUpdate
		}
		return ErrScanTaskInvalidTransition
	}

	var committed bool
	diagnosticStore, ok := service.taskStore.(EngineDiagnosticTerminalTaskStore)
	if ok {
		committed, err = diagnosticStore.CommitScanTaskTerminalStatusWithDiagnosticsForSession(ctx, taskID, agentID, sessionID, sessionEpoch, string(nextStatus), normalizedFailure, normalizedDiagnostics)
	} else if normalizedDiagnostics.Availability == scandomain.EngineDiagnosticAvailabilityUnavailable {
		// In-memory application fakes model Server-originated lifecycle paths.
		// Production storage implements the strict diagnostic write capability.
		committed, err = service.taskStore.CommitScanTaskTerminalStatusForSession(ctx, taskID, agentID, sessionID, sessionEpoch, string(nextStatus), normalizedFailure)
	} else {
		return ErrScanTaskInvalidUpdate
	}
	if err != nil {
		return err
	}
	if !committed {
		// Another delivery may have won between the read and terminal write. Only
		// an identical committed result is a replay; a different result is a
		// conflict and must not be acknowledged.
		task, err = service.taskStore.GetByID(ctx, taskID)
		if err != nil {
			if dberrors.IsRecordNotFound(err) {
				return ErrScanTaskNotFound
			}
			return err
		}
		if !terminalTaskLeaseMatches(task, agentID, sessionID, sessionEpoch) {
			return ErrScanTaskNotOwned
		}
		committedStatus, ok := scandomain.ParseTaskStatus(task.Status)
		if !ok || committedStatus != nextStatus || !terminalTaskResultMatches(task, nextStatus, normalizedFailure, normalizedDiagnostics) {
			return ErrScanTaskInvalidTransition
		}
	}
	return service.reconcileAndClearTerminalTaskResult(ctx, task, nextStatus)
}

func terminalTaskLeaseMatches(task *ScanTaskRecord, agentID int, sessionID string, sessionEpoch int64) bool {
	if task == nil || task.AgentID == nil || *task.AgentID != agentID || task.AssignedSessionEpoch == nil || *task.AssignedSessionEpoch != sessionEpoch {
		return false
	}
	if task.AssignedAgentID != nil && *task.AssignedAgentID != agentID {
		return false
	}
	return task.AssignedAgentID != nil && *task.AssignedAgentID == agentID &&
		task.AssignedSessionID != nil && *task.AssignedSessionID == sessionID
}

func terminalTaskResultMatches(task *ScanTaskRecord, status scandomain.TaskStatus, failure *FailureDetail, diagnostics *EngineExecutionDiagnostics) bool {
	if task == nil {
		return false
	}
	if !scandomain.EqualEngineExecutionDiagnostics(task.Diagnostics, diagnostics) &&
		!(task.Diagnostics == nil && diagnostics != nil && diagnostics.Availability == scandomain.EngineDiagnosticAvailabilityUnavailable) {
		return false
	}
	if status != scandomain.TaskStatusFailed {
		return failure == nil && task.Failure == nil
	}
	return failure != nil && task.Failure != nil &&
		task.Failure.Kind == failure.Kind && task.Failure.Message == failure.Message && task.Failure.DisplayMessage == failure.DisplayMessage
}

func (service *ScanTaskBridgeService) reconcileTerminalTaskResult(ctx context.Context, task *ScanTaskRecord, status scandomain.TaskStatus) error {
	// A prior delivery may have committed the task and then lost its response
	// while workflow reconciliation was failing. Every identical replay retries
	// this synchronous work before Server may acknowledge it.
	if status == scandomain.TaskStatusSucceeded {
		if err := service.unlockNextStageIfReady(ctx, task.ScanID, task.ScanWorkflowStageOrder); err != nil {
			return err
		}
	}
	return service.recalculateScanStatus(ctx, task.ScanID)
}

func (service *ScanTaskBridgeService) reconcileAndClearTerminalTaskResult(ctx context.Context, task *ScanTaskRecord, status scandomain.TaskStatus) error {
	if err := service.reconcileTerminalTaskResult(ctx, task, status); err != nil {
		return err
	}
	return service.taskStore.ClearTerminalTaskReconciliationPending(ctx, task.ID)
}

func normalizeTaskFailure(status scandomain.TaskStatus, failure *FailureDetail) (*FailureDetail, error) {
	if status != scandomain.TaskStatusFailed {
		if failure != nil && (strings.TrimSpace(failure.Kind) != "" || strings.TrimSpace(failure.Message) != "" || strings.TrimSpace(failure.DisplayMessage) != "") {
			return nil, ErrScanTaskInvalidUpdate
		}
		return nil, nil
	}
	if failure == nil {
		return nil, ErrScanTaskInvalidUpdate
	}
	message := strings.TrimSpace(failure.Message)
	if message == "" {
		return nil, ErrScanTaskInvalidUpdate
	}
	kind := canonicalFailureKind(strings.TrimSpace(failure.Kind))
	if kind == "" {
		return nil, ErrScanTaskInvalidUpdate
	}
	displayMessage := failure.DisplayMessage
	if displayMessage != "" && !validFailureDisplayMessage(displayMessage) {
		return nil, ErrScanTaskInvalidUpdate
	}
	return &FailureDetail{Kind: kind, Message: message, DisplayMessage: displayMessage}, nil
}

const maxFailureDisplayMessageBytes = 500

func validFailureDisplayMessage(value string) bool {
	return len(value) <= maxFailureDisplayMessageBytes && utf8.ValidString(value) && value == strings.TrimSpace(value) && strings.IndexFunc(value, unicode.IsControl) < 0
}

func failureMessage(failure *FailureDetail) string {
	if failure == nil {
		return ""
	}
	return failure.Message
}
