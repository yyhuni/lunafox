package application

import (
	"context"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

func (service *ScanTaskBridgeService) failClaimedTask(
	ctx context.Context,
	task *ScanTaskRecord,
	scan *ScanTaskRuntimeScanRecord,
	rejectReason string,
	originalErr error,
) error {
	if task == nil {
		return originalErr
	}

	failure := failureFromReason(rejectReason)
	if err := service.taskStore.FailClaimedTask(ctx, task.ID, failure); err != nil {
		pkg.Error(
			"Fail claimed task after scheduler rejection failed",
			zap.Int("task.id", task.ID),
			zap.Int("scan.id", scanIDOf(scan)),
			zap.String("failure.kind", failure.Kind),
			zap.String("error.reason", "task_failure_persistence_failed"),
		)
		return err
	}
	if scan != nil {
		if recalcErr := service.recalculateScanStatus(ctx, scan.ID); recalcErr != nil {
			pkg.Error(
				"Recalculate scan status after claimed task failure failed",
				zap.Int("scan.id", scan.ID),
				zap.String("error.reason", "scan_status_recalculation_failed"),
			)
			return recalcErr
		}
	}

	return originalErr
}

func (service *ScanTaskBridgeService) handleClaimedTaskRejection(
	ctx context.Context,
	task *ScanTaskRecord,
	scan *ScanTaskRuntimeScanRecord,
	agentID int,
	err error,
) error {
	reason := rejectionFailureKindForTaskExecutionError(err)
	pkg.Warn(
		"Rejecting claimed task during task assignment",
		zap.Int("task.id", taskIDOf(task)),
		zap.Int("scan.id", scanIDOf(scan)),
		zap.Int("agent.id", agentID),
		zap.String("execution.phase", "assignment"),
		zap.String("failure.kind", reason),
	)
	return service.failClaimedTask(ctx, task, scan, reason, err)
}

func taskIDOf(task *ScanTaskRecord) int {
	if task == nil {
		return 0
	}
	return task.ID
}

func scanIDOf(scan *ScanTaskRuntimeScanRecord) int {
	if scan == nil {
		return 0
	}
	return scan.ID
}

func rejectionFailureKindForTaskExecutionError(err error) string {
	if taskExecutionErr, ok := AsTaskExecutionError(err); ok {
		return taskExecutionErrorCodeToFailureKind(taskExecutionErr.Code)
	}
	return "scheduler_rejected"
}

func failureFromReason(reason string) *FailureDetail {
	kind := canonicalFailureKind(strings.TrimSpace(reason))
	if kind == "" {
		kind = "scheduler_rejected"
	}
	return &FailureDetail{Kind: kind, Message: schedulerRejectionFailureMessage(kind)}
}

func schedulerRejectionFailureMessage(kind string) string {
	switch canonicalFailureKind(kind) {
	case "schema_invalid":
		return "The task execution schema is invalid."
	case "task_execution_config_invalid":
		return "The task execution configuration is invalid."
	case "operation_runtime_prereq_missing":
		return "A required task execution prerequisite is unavailable."
	default:
		return "The Server rejected the task assignment."
	}
}

func taskExecutionErrorCodeToFailureKind(code string) string {
	switch strings.TrimSpace(code) {
	case TaskExecutionErrorCodeSchemaInvalid:
		return "schema_invalid"
	case TaskExecutionErrorCodeTaskExecutionConfigInvalid:
		return "task_execution_config_invalid"
	case TaskExecutionErrorCodeOperationRuntimePrereqMissing:
		return "operation_runtime_prereq_missing"
	default:
		return canonicalFailureKind(code)
	}
}

func canonicalFailureKind(value string) string {
	return strings.TrimSpace(value)
}
