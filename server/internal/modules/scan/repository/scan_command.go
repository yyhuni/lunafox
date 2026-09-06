package repository

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/yyhuni/lunafox/contracts/agentexecution"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateWithScanTasksAndPlans is the v2 additive creation path. IDs are
// allocated inside the same transaction after the effective blacklist is
// locked and frozen, so no executable task can commit without either snapshot.
func (r *ScanRepository) CreateWithScanTasksAndPlans(
	ctx context.Context,
	scan *ScanCreateRecord,
	resolveEffectivePatterns func(context.Context, int) ([]string, error),
	finalize scandomain.CreateScanTaskFinalizer,
) error {
	return r.createWithScanTasksAndPlans(ctx, scan, resolveEffectivePatterns, finalize, nil)
}

// createWithScanTasksAndPlans is the one Scan creation transaction. Optional
// callers can append a durable record only after every Task/plan is frozen but
// before the transaction commits, preserving all-or-nothing creation semantics.
func (r *ScanRepository) createWithScanTasksAndPlans(
	ctx context.Context,
	scan *ScanCreateRecord,
	resolveEffectivePatterns func(context.Context, int) ([]string, error),
	finalize scandomain.CreateScanTaskFinalizer,
	afterCreate func(*gorm.DB, *model.Scan) error,
) error {
	if ctx == nil {
		return fmt.Errorf("scan create context is required")
	}
	if scan == nil {
		return fmt.Errorf("scan is required")
	}
	if resolveEffectivePatterns == nil {
		return fmt.Errorf("effective blacklist policy resolver is required")
	}
	if finalize == nil {
		return fmt.Errorf("scan task plan finalizer is required")
	}
	db, err := r.resolveScanDatabase(ctx)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the active Target before resolving any dependent input so DELETE
		// and Scan creation serialize on the Target lifecycle boundary.
		if err := lockActiveTargetForScanCreate(tx, scan.TargetID); err != nil {
			return err
		}
		txContext := dbtx.WithTransaction(ctx, tx)
		effectivePatterns, err := resolveEffectivePatterns(txContext, scan.TargetID)
		if err != nil {
			return fmt.Errorf("resolve effective blacklist policy: %w", err)
		}
		if err := blacklistdomain.ValidateCanonicalEffectivePatterns(effectivePatterns); err != nil {
			return fmt.Errorf("effective blacklist policy is invalid: %w", err)
		}
		modelScan, err := scanCreateRecordToModel(scan)
		if err != nil {
			return err
		}
		if err := tx.Create(modelScan).Error; err != nil {
			return err
		}
		scan.ID = modelScan.ID
		scan.CreatedAt = modelScan.CreatedAt
		if err := r.createBlacklistSnapshot(tx, modelScan.ID, effectivePatterns); err != nil {
			return fmt.Errorf("create scan blacklist snapshot: %w", err)
		}
		for index := range scan.ScanTasks {
			task := &scan.ScanTasks[index]
			modelTask := scanTaskCreateRecordToModel(task, modelScan.ID)
			createTask := tx.Omit("resolved_execution_plan")
			if err := createTask.Create(modelTask).Error; err != nil {
				return err
			}
			task.ID = modelTask.ID
			if err := finalize(modelScan.ID, modelTask.ID, task); err != nil {
				return err
			}
			if err := validateFinalizedCreateTask(scan, modelTask.ID, task); err != nil {
				return err
			}
			updates := map[string]any{
				"engine_config":           encodeJSONMap(task.EngineConfig),
				"task_execution_config":   encodeJSONMap(task.TaskExecutionConfig),
				"status":                  task.Status,
				"skip_reason":             strings.TrimSpace(task.SkipReason),
				"resolved_execution_plan": append([]byte{}, task.ResolvedExecutionPlan...),
			}
			if err := tx.Model(&model.ScanTask{}).Where("id = ?", modelTask.ID).Updates(updates).Error; err != nil {
				return err
			}
		}
		if scan.Status != modelScan.Status {
			if err := tx.Model(&model.Scan{}).Where("id = ?", modelScan.ID).Update("status", scan.Status).Error; err != nil {
				return err
			}
			modelScan.Status = scan.Status
		}
		if afterCreate != nil {
			return afterCreate(tx, modelScan)
		}
		return nil
	})
}

func lockActiveTargetForScanCreate(tx *gorm.DB, targetID int) error {
	if tx == nil {
		return fmt.Errorf("scan create transaction is required")
	}
	var target struct {
		ID int `gorm:"column:id"`
	}
	return tx.Table("target").
		Clauses(clause.Locking{Strength: "SHARE"}).
		Select("id").
		Where("id = ? AND deleted_at IS NULL", targetID).
		Take(&target).Error
}

func validateFinalizedCreateTask(scan *ScanCreateRecord, taskID int, task *CreateScanTaskRecord) error {
	if scan == nil {
		return fmt.Errorf("scan is required")
	}
	if task == nil {
		return fmt.Errorf("scan task is required")
	}
	if task.Status == taskStatusSkipped {
		if len(task.ResolvedExecutionPlan) != 0 {
			return fmt.Errorf("skipped task must not have a saved execution plan")
		}
		if !validCanonicalSkipReason(task.SkipReason) {
			return fmt.Errorf("skipped task requires a canonical skip reason")
		}
		return nil
	}
	if task.Status != taskStatusPending && task.Status != taskStatusBlocked {
		return fmt.Errorf("executable task must start as pending or blocked")
	}
	if task.SkipReason != "" {
		return fmt.Errorf("executable task must not have a skip reason")
	}
	if len(task.ResolvedExecutionPlan) == 0 {
		return fmt.Errorf("executable task must have a saved execution plan")
	}
	plan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(task.ResolvedExecutionPlan)
	if err != nil {
		return fmt.Errorf("executable task saved execution plan is invalid: %w", err)
	}
	if plan.GetTask() != resourcenames.Task(scan.ID, taskID) {
		return fmt.Errorf("saved execution plan task scope does not match allocated task")
	}
	if plan.GetWorkflowStep().GetScan() != resourcenames.Scan(scan.ID) {
		return fmt.Errorf("saved execution plan scan scope does not match allocated scan")
	}
	if plan.GetWorkflowStep().GetWorkflow() != resourcenames.ScanWorkflow(scan.ScanWorkflowID) {
		return fmt.Errorf("saved execution plan workflow does not match allocated scan")
	}
	if plan.GetTarget().GetResource() != resourcenames.Target(scan.TargetID) {
		return fmt.Errorf("saved execution plan target does not match allocated scan")
	}
	if plan.GetEngineRelease().GetEngine() != task.EngineID {
		return fmt.Errorf("saved execution plan engine does not match allocated task")
	}
	if plan.GetWorkflowStep().GetStageId() != task.StageID || plan.GetWorkflowStep().GetStepId() != task.StepID {
		return fmt.Errorf("saved execution plan workflow scope does not match allocated task")
	}
	return nil
}

func validCanonicalSkipReason(reason string) bool {
	return reason != "" && len(reason) <= 1000 && utf8.ValidString(reason) && reason == strings.TrimSpace(reason) && strings.IndexFunc(reason, unicode.IsControl) < 0
}

// SoftDelete soft deletes a scan.
func (r *ScanRepository) SoftDelete(id int) error {
	now := time.Now().UTC()
	return r.db.Model(&model.Scan{}).Where("id = ?", id).Update("deleted_at", now).Error
}

// BatchSoftDelete soft deletes multiple scans by IDs.
func (r *ScanRepository) BatchSoftDelete(ids []int) (int64, []string, error) {
	if len(ids) == 0 {
		return 0, nil, nil
	}

	var scans []model.Scan
	if err := r.db.Select("id, target_id").
		Where("id IN ? AND deleted_at IS NULL", ids).
		Preload("Target", "deleted_at IS NULL").
		Find(&scans).Error; err != nil {
		return 0, nil, err
	}

	names := make([]string, 0, len(scans))
	for _, scan := range scans {
		if scan.Target != nil {
			names = append(names, scan.Target.Name)
		}
	}

	now := time.Now().UTC()
	result := r.db.Model(&model.Scan{}).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Update("deleted_at", now)

	return result.RowsAffected, names, result.Error
}

// UpdateScanStatus updates scan status with structured failure detail.
func (r *ScanRepository) UpdateScanStatus(id int, status string, failure *scandomain.FailureDetail) error {
	failure, err := normalizeScanFailureDetail(status, failure)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{"status": status}
	if failure != nil {
		updates["error_message"] = failure.Message
		updates["failure_kind"] = failure.Kind
	} else {
		updates["error_message"] = ""
		updates["failure_kind"] = ""
	}
	now := time.Now().UTC()
	isTerminal := status == scanStatusSucceeded || status == scanStatusFailed || status == scanStatusCancelled
	if isTerminal {
		updates["stopped_at"] = &now
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		// A terminal Scan is immutable. Restrict all status writes to the active
		// states so a late reconciliation cannot replace one terminal outcome with
		// another and emit a second terminal notification candidate.
		result := tx.Model(&model.Scan{}).
			Where("id = ? AND status IN ? AND status <> ?", id, []string{scanStatusPending, scanStatusRunning}, status).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		if (status == scanStatusSucceeded || status == scanStatusFailed) && r.terminalNotificationSink != nil {
			failureKind, failureMessage := "", ""
			if failure != nil {
				failureKind = failure.Kind
				failureMessage = failure.Message
			}
			// The conditional Scan transition and its immutable outbox envelope
			// share this transaction so a producer failure cannot leave a terminal
			// Scan with no durable notification candidate.
			if err := r.terminalNotificationSink.WriteScanTerminal(tx, id, status, failureKind, failureMessage, now); err != nil {
				return err
			}
		}
		if status != scanStatusSucceeded {
			return nil
		}
		// `last_scanned_at` represents the latest successful scan for a target,
		// not merely the newest terminal attempt; keep the update monotonic.
		return tx.Exec(`
			UPDATE target
			SET last_scanned_at = (SELECT stopped_at FROM scan WHERE id = ?)
			WHERE id = (SELECT target_id FROM scan WHERE id = ?)
				AND (SELECT stopped_at FROM scan WHERE id = ?) IS NOT NULL
				AND (
					last_scanned_at IS NULL
					OR last_scanned_at < (SELECT stopped_at FROM scan WHERE id = ?)
				)
		`, id, id, id, id).Error
	})
}

// RefreshScanResultSummary refreshes scan read-model counters from persisted scan evidence.
func (r *ScanRepository) RefreshScanResultSummary(ctx context.Context, scanID int, targetID int) error {
	return dbtx.Resolve(ctx, r.db).WithContext(ctx).Exec(`
		UPDATE scan
		SET cached_subdomains_count = (
				SELECT COUNT(*)
				FROM subdomain_snapshot
				WHERE subdomain_snapshot.scan_id = scan.id
			),
			cached_websites_count = (
				SELECT COUNT(*)
				FROM website_snapshot
				WHERE website_snapshot.scan_id = scan.id
			),
			cached_endpoints_count = (
				SELECT COUNT(*)
				FROM endpoint_snapshot
				WHERE endpoint_snapshot.scan_id = scan.id
			),
			cached_ips_count = (
				SELECT COUNT(DISTINCT ip)
				FROM host_port_mapping_snapshot
				WHERE host_port_mapping_snapshot.scan_id = scan.id
			),
			cached_directories_count = (
				SELECT COUNT(*)
				FROM directory_snapshot
				WHERE directory_snapshot.scan_id = scan.id
			),
			cached_screenshots_count = (
				SELECT COUNT(*)
				FROM screenshot_snapshot
				WHERE screenshot_snapshot.scan_id = scan.id
			),
			cached_vulns_total = (
				SELECT COUNT(*)
				FROM vulnerability_snapshot
				WHERE vulnerability_snapshot.scan_id = scan.id
			),
			cached_vulns_critical = (
				SELECT COUNT(*)
				FROM vulnerability_snapshot
				WHERE vulnerability_snapshot.scan_id = scan.id
					AND vulnerability_snapshot.severity = 'critical'
			),
			cached_vulns_high = (
				SELECT COUNT(*)
				FROM vulnerability_snapshot
				WHERE vulnerability_snapshot.scan_id = scan.id
					AND vulnerability_snapshot.severity = 'high'
			),
			cached_vulns_medium = (
				SELECT COUNT(*)
				FROM vulnerability_snapshot
				WHERE vulnerability_snapshot.scan_id = scan.id
					AND vulnerability_snapshot.severity = 'medium'
			),
			cached_vulns_low = (
				SELECT COUNT(*)
				FROM vulnerability_snapshot
				WHERE vulnerability_snapshot.scan_id = scan.id
					AND vulnerability_snapshot.severity = 'low'
			),
			stats_updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
			AND target_id = ?
			AND deleted_at IS NULL
	`, scanID, targetID).Error
}

// RefreshSubdomainResultSummary is retained for older callers; new ingest paths
// should refresh the whole scan summary so asset tabs stay internally consistent.
func (r *ScanRepository) RefreshSubdomainResultSummary(ctx context.Context, scanID int, targetID int) error {
	return r.RefreshScanResultSummary(ctx, scanID, targetID)
}

func normalizeScanFailureDetail(status string, failure *scandomain.FailureDetail) (*scandomain.FailureDetail, error) {
	if strings.TrimSpace(status) != scanStatusFailed {
		return nil, nil
	}
	if failure == nil {
		return nil, fmt.Errorf("failed scan requires failure detail")
	}
	message := strings.TrimSpace(failure.Message)
	if message == "" {
		return nil, fmt.Errorf("failed scan requires non-empty failure message")
	}
	kind := strings.TrimSpace(failure.Kind)
	if kind == "" {
		return nil, fmt.Errorf("failed scan requires non-empty failure kind")
	}
	return &scandomain.FailureDetail{Kind: kind, Message: message}, nil
}
