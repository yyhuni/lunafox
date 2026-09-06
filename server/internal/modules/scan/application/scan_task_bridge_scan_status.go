package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

func (service *ScanTaskBridgeService) unlockNextStageIfReady(ctx context.Context, scanID, scanWorkflowStageOrder int) error {
	active, err := service.taskStore.CountActiveByScanAndStageOrder(ctx, scanID, scanWorkflowStageOrder)
	if err != nil {
		return err
	}
	if active > 0 {
		return nil
	}

	_, err = service.taskStore.UnlockNextStageOrder(ctx, scanID, scanWorkflowStageOrder)
	return err
}

func (service *ScanTaskBridgeService) recalculateScanStatus(
	ctx context.Context,
	scanID int,
) error {
	pending, running, _, failed, cancelled, skipped, err := service.taskStore.CountByStatusForScanID(ctx, scanID)
	if err != nil {
		return err
	}

	nextStatus, shouldUpdate := scandomain.ResolveScanStatusFromTaskCounts(
		pending,
		running,
		failed,
		cancelled,
		skipped,
	)
	if !shouldUpdate {
		return nil
	}
	if nextStatus == scandomain.ScanStatusFailed {
		if _, err := service.taskStore.CancelUnstartedTasksByScanID(ctx, scanID); err != nil {
			return err
		}
		failedTasks, err := service.taskStore.ListFailedByScanID(ctx, scanID)
		if err != nil {
			return err
		}
		canonicalTask, err := selectCanonicalFailedTask(failedTasks)
		if err != nil {
			return err
		}
		return service.scanTaskRuntimeScanStore.UpdateScanStatus(scanID, string(nextStatus), cloneFailureDetail(canonicalTask.Failure))
	}
	if nextStatus == scandomain.ScanStatusCancelled {
		if _, err := service.taskStore.CancelUnstartedTasksByScanID(ctx, scanID); err != nil {
			return err
		}
	}

	return service.scanTaskRuntimeScanStore.UpdateScanStatus(scanID, string(nextStatus), nil)
}

func cloneFailureDetail(failure *FailureDetail) *FailureDetail {
	if failure == nil {
		return nil
	}
	cloned := *failure
	return &cloned
}

func selectCanonicalFailedTask(tasks []ScanTaskRecord) (*ScanTaskRecord, error) {
	var best *ScanTaskRecord
	for index := range tasks {
		candidate := &tasks[index]
		if candidate == nil || candidate.Failure == nil {
			continue
		}
		if strings.TrimSpace(candidate.Failure.Message) == "" {
			continue
		}
		if best == nil || canonicalTaskLess(candidate, best) {
			best = candidate
		}
	}
	if best == nil {
		return nil, fmt.Errorf("failed scan has no canonical failed task")
	}
	return best, nil
}

func canonicalTaskLess(left, right *ScanTaskRecord) bool {
	leftPriority := failurePriority(left.Failure.Kind)
	rightPriority := failurePriority(right.Failure.Kind)
	if leftPriority != rightPriority {
		return leftPriority < rightPriority
	}
	if left.ScanWorkflowStageOrder != right.ScanWorkflowStageOrder {
		return left.ScanWorkflowStageOrder < right.ScanWorkflowStageOrder
	}
	if compared := compareTimePtr(left.CompletedAt, right.CompletedAt); compared != 0 {
		return compared < 0
	}
	return left.ID < right.ID
}

func compareTimePtr(left, right *time.Time) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return 1
	}
	if right == nil {
		return -1
	}
	if left.Before(*right) {
		return -1
	}
	if left.After(*right) {
		return 1
	}
	return 0
}

func failurePriority(kind string) int {
	switch canonicalFailureKind(kind) {
	case "server_package_load_failed",
		"server_plan_compilation_request_failed",
		"server_plan_compilation_config_failed",
		"server_plan_compilation_resource_failed",
		"server_plan_compilation_protocol_failed",
		"schema_invalid", "task_execution_config_invalid", "operation_runtime_prereq_missing", "decode_config_failed":
		return 1
	case "agent_plan_validation_failed",
		"agent_plan_version_validation_failed",
		"agent_plan_scope_validation_failed",
		"agent_node_compatibility_validation_failed",
		"scheduler_rejected":
		return 2
	case "execution_input_derivation_failed",
		"execution_input_fetch_failed",
		"execution_input_content_type_invalid",
		"execution_input_frame_invalid",
		"execution_input_integrity_failed",
		"execution_input_write_failed",
		"config_resource_authorization_failed",
		"config_resource_fetch_failed",
		"config_resource_content_type_invalid",
		"config_resource_frame_invalid",
		"config_resource_hash_failed",
		"config_resource_write_failed",
		"platform_resource_authorization_failed",
		"platform_resource_fetch_failed",
		"platform_resource_content_type_invalid",
		"platform_resource_frame_invalid",
		"platform_resource_hash_failed",
		"platform_resource_write_failed",
		"shared_materialization_staging_failed",
		"shared_materialization_atomic_commit_failed",
		"context_encode_failed":
		return 3
	case "image_pull_failed",
		"platform_mismatch",
		"container_create_failed",
		"container_start_failed",
		"protocol_bootstrap_failed":
		return 4
	case "container_wait_failed",
		"container_cleanup_failed",
		"engine_exit_failed",
		"result_protocol_failed",
		"task_timeout",
		"agent_disconnected",
		"invalid_lifecycle_outcome",
		"execution_failed",
		"runtime_error", "unknown":
		return 5
	default:
		return 5
	}
}
