package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/contracts/resourcenames"
	"github.com/yyhuni/lunafox/server/internal/mcp/idempotency"
	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm"
)

// MCPOperationRecord is the repository read projection for a Scan-backed
// operation. It intentionally carries Scan-derived state rather than a second
// persisted lifecycle state machine.
type MCPOperationRecord struct {
	ID          string
	ScanID      int
	TargetID    int
	ScanStatus  string
	Progress    int
	Phase       string
	CurrentTask string
	FailureKind string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CreateWithScanTasksAndPlansAndMCPOperation atomically persists one Scan,
// every frozen Task/plan, its polling operation, and optional replay result.
// A caller cancellation before commit aborts all of them together.
func (r *ScanRepository) CreateWithScanTasksAndPlansAndMCPOperation(
	ctx context.Context,
	scan *ScanCreateRecord,
	operationID string,
	requestID string,
	requestFingerprint string,
	replayExpiresAt time.Time,
	resolveEffectivePatterns func(context.Context, int) ([]string, error),
	finalize func(scanID, taskID int, task *CreateScanTaskRecord) error,
) error {
	operationID = strings.TrimSpace(operationID)
	requestID = strings.TrimSpace(requestID)
	requestFingerprint = strings.TrimSpace(requestFingerprint)
	if operationID == "" || requestFingerprint == "" {
		return fmt.Errorf("MCP operation identity and request fingerprint are required")
	}
	return r.createWithScanTasksAndPlans(ctx, scan, resolveEffectivePatterns, finalize, func(tx *gorm.DB, modelScan *model.Scan) error {
		operation := &model.ScanOperation{
			ID:                 operationID,
			ScanID:             modelScan.ID,
			TargetID:           modelScan.TargetID,
			RequestFingerprint: requestFingerprint,
			CreatedAt:          modelScan.CreatedAt,
		}
		if requestID != "" {
			requestIDCopy := requestID
			operation.RequestID = &requestIDCopy
		}
		if err := tx.Create(operation).Error; err != nil {
			return err
		}
		if requestID == "" {
			return nil
		}
		if replayExpiresAt.IsZero() {
			return fmt.Errorf("MCP replay expiration is required")
		}
		if err := pruneExpiredMCPRequestReplays(tx, modelScan.CreatedAt.UTC()); err != nil {
			return err
		}
		// The shared replay key is reusable once its bounded lifetime has ended.
		// Delete this exact expired row inside the create transaction so a key
		// cannot remain blocked merely because bounded background pruning has not
		// reached it yet. A concurrent winner still makes Create fail and rolls
		// back this whole Scan/Task/operation transaction.
		if err := deleteExpiredMCPRequestReplay(tx, requestID, modelScan.CreatedAt.UTC()); err != nil {
			return err
		}
		response, err := json.Marshal(struct {
			Operation string `json:"operation"`
			Scan      string `json:"scan"`
		}{Operation: "operations/" + operationID, Scan: resourcenames.Scan(modelScan.ID)})
		if err != nil {
			return fmt.Errorf("marshal MCP scan replay response: %w", err)
		}
		return tx.Create(&idempotency.ReplayRecord{
			RequestID:          requestID,
			Action:             "start_scan",
			RequestFingerprint: requestFingerprint,
			Response:           response,
			CreatedAt:          modelScan.CreatedAt.UTC(),
			ExpiresAt:          replayExpiresAt.UTC(),
		}).Error
	})
}

// FindMCPRequestReplay finds an unexpired replay record. Expired records are
// deliberately not replayed; the following write transaction reclaims its key.
func (r *ScanRepository) FindMCPRequestReplay(ctx context.Context, requestID string) (*idempotency.ReplayRecord, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("scan repository is not configured")
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return nil, nil
	}
	var replay idempotency.ReplayRecord
	err := dbtx.Resolve(ctx, r.db).WithContext(ctx).
		Where("request_id = ? AND expires_at > ?", requestID, time.Now().UTC()).
		Take(&replay).Error
	if err == nil {
		return &replay, nil
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return nil, err
}

// GetMCPOperation projects the linked Scan and current Task facts without
// filtering on target visibility, preserving historical operation context when
// a target was tombstoned after Scan creation.
func (r *ScanRepository) GetMCPOperation(ctx context.Context, operationID string) (*MCPOperationRecord, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("scan repository is not configured")
	}
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	db := dbtx.Resolve(ctx, r.db).WithContext(ctx)
	var base struct {
		ID            string     `gorm:"column:id"`
		ScanID        int        `gorm:"column:scan_id"`
		TargetID      int        `gorm:"column:target_id"`
		ScanStatus    string     `gorm:"column:scan_status"`
		Progress      int        `gorm:"column:progress"`
		Phase         string     `gorm:"column:phase"`
		FailureKind   string     `gorm:"column:failure_kind"`
		CreatedAt     time.Time  `gorm:"column:created_at"`
		ScanCreatedAt time.Time  `gorm:"column:scan_created_at"`
		StoppedAt     *time.Time `gorm:"column:stopped_at"`
	}
	if err := db.Table("scan_operation AS operation").
		Select(`operation.id, operation.scan_id, operation.target_id,
			scan.status AS scan_status, scan.progress, scan.current_stage AS phase,
			scan.failure_kind, operation.created_at, scan.created_at AS scan_created_at,
			scan.stopped_at`).
		Joins("JOIN scan ON scan.id = operation.scan_id").
		Where("operation.id = ?", operationID).
		Take(&base).Error; err != nil {
		return nil, err
	}
	if !validMCPOperationScanStatus(base.ScanStatus) {
		return nil, fmt.Errorf("scan operation has unsupported Scan status")
	}

	var currentTask struct {
		StepID      string     `gorm:"column:step_id"`
		CreatedAt   time.Time  `gorm:"column:created_at"`
		StartedAt   *time.Time `gorm:"column:started_at"`
		CompletedAt *time.Time `gorm:"column:completed_at"`
	}
	currentTaskErr := db.Table("scan_task").
		Select("step_id, created_at, started_at, completed_at").
		Where("scan_id = ? AND status IN ?", base.ScanID, []string{"running", "pending", "blocked"}).
		Order("CASE status WHEN 'running' THEN 0 WHEN 'pending' THEN 1 ELSE 2 END").
		Order("stage_order ASC").
		Order("step_order ASC").
		Order("id ASC").
		Take(&currentTask).Error
	if currentTaskErr != nil && currentTaskErr != gorm.ErrRecordNotFound {
		return nil, currentTaskErr
	}

	record := &MCPOperationRecord{
		ID:          base.ID,
		ScanID:      base.ScanID,
		TargetID:    base.TargetID,
		ScanStatus:  base.ScanStatus,
		Progress:    clampMCPOperationProgress(base.ScanStatus, base.Progress),
		Phase:       base.Phase,
		FailureKind: base.FailureKind,
		CreatedAt:   base.CreatedAt.UTC(),
		UpdatedAt:   latestMCPOperationTime(base.CreatedAt, base.ScanCreatedAt),
	}
	if base.StoppedAt != nil {
		record.UpdatedAt = latestMCPOperationTime(record.UpdatedAt, *base.StoppedAt)
	}
	if currentTaskErr == nil {
		record.CurrentTask = currentTask.StepID
		record.UpdatedAt = latestMCPOperationTime(record.UpdatedAt, currentTask.CreatedAt)
		if currentTask.StartedAt != nil {
			record.UpdatedAt = latestMCPOperationTime(record.UpdatedAt, *currentTask.StartedAt)
		}
		if currentTask.CompletedAt != nil {
			record.UpdatedAt = latestMCPOperationTime(record.UpdatedAt, *currentTask.CompletedAt)
		}
	}
	return record, nil
}

func pruneExpiredMCPRequestReplays(tx *gorm.DB, now time.Time) error {
	if tx == nil {
		return fmt.Errorf("MCP replay transaction is required")
	}
	keys := tx.Model(&idempotency.ReplayRecord{}).
		Select("request_id").
		Where("expires_at <= ?", now.UTC()).
		Order("expires_at ASC, request_id ASC").
		Limit(idempotency.ReplayPruneLimit)
	result := tx.Where("request_id IN (?)", keys).Delete(&idempotency.ReplayRecord{})
	return result.Error
}

func deleteExpiredMCPRequestReplay(tx *gorm.DB, requestID string, now time.Time) error {
	if tx == nil {
		return fmt.Errorf("MCP replay transaction is required")
	}
	return tx.Where("request_id = ? AND expires_at <= ?", requestID, now.UTC()).Delete(&idempotency.ReplayRecord{}).Error
}

func clampMCPOperationProgress(status string, progress int) int {
	if status == "succeeded" {
		return 100
	}
	if progress < 0 {
		return 0
	}
	if progress > 100 {
		return 100
	}
	return progress
}

func validMCPOperationScanStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "running", "succeeded", "failed", "cancelled":
		return true
	default:
		return false
	}
}

func latestMCPOperationTime(initial time.Time, values ...time.Time) time.Time {
	latest := initial.UTC()
	for _, candidate := range values {
		candidate = candidate.UTC()
		if candidate.After(latest) {
			latest = candidate
		}
	}
	return latest
}
