package resultingestwiring

import (
	"context"
	"errors"
	"strings"

	resultingestapp "github.com/yyhuni/lunafox/server/internal/modules/resultingest/application"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// resultIngestMaterializationCoordinator owns the final durable authorization
// for Agent-provided result writes. The transport's earlier scope check is only
// advisory once concurrent Target deletion, cancellation, or reassignment is
// possible.
type resultIngestMaterializationCoordinator struct {
	db *gorm.DB
}

func newResultIngestMaterializationCoordinator(db *gorm.DB) *resultIngestMaterializationCoordinator {
	return &resultIngestMaterializationCoordinator{db: db}
}

func (coordinator *resultIngestMaterializationCoordinator) Materialize(ctx context.Context, scope resultingestapp.ResultMaterializationScope, persist func(context.Context) error) error {
	if ctx == nil {
		return context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if coordinator == nil || coordinator.db == nil || persist == nil {
		return resultingestapp.ErrResultMaterializerUnavailable
	}
	if !validResultMaterializationScope(scope) {
		return resultingestapp.ErrResultExecutionFenceRejected
	}

	return coordinator.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)
		// Snapshot, Asset, and summary work must resolve this context rather than
		// a root DB handle, otherwise the locks below can block the same request.
		txContext := dbtx.WithTransaction(ctx, tx)
		if err := lockActiveResultExecution(tx, scope); err != nil {
			return err
		}
		return persist(txContext)
	})
}

func validResultMaterializationScope(scope resultingestapp.ResultMaterializationScope) bool {
	return scope.TaskID > 0 &&
		scope.ScanID > 0 &&
		scope.TargetID > 0 &&
		scope.AgentID > 0 &&
		scope.SessionEpoch > 0 &&
		scope.SessionID != "" &&
		scope.SessionID == strings.TrimSpace(scope.SessionID)
}

func lockActiveResultExecution(tx *gorm.DB, scope resultingestapp.ResultMaterializationScope) error {
	var target struct {
		ID int `gorm:"column:id"`
	}
	if err := tx.Table("target").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").
		Where("id = ? AND deleted_at IS NULL", scope.TargetID).
		Take(&target).Error; err != nil {
		return mapResultExecutionFenceError(err)
	}

	var scan struct {
		ID int `gorm:"column:id"`
	}
	if err := tx.Table("scan").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").
		Where(
			"id = ? AND target_id = ? AND deleted_at IS NULL AND status IN ?",
			scope.ScanID,
			scope.TargetID,
			[]string{string(scandomain.ScanStatusPending), string(scandomain.ScanStatusRunning)},
		).
		Take(&scan).Error; err != nil {
		return mapResultExecutionFenceError(err)
	}

	// An RPC may have passed the transport admission check immediately before a
	// replacement process session is persisted. Lock the durable session authority
	// here so that either this complete batch commits first or the old lease is
	// rejected; the later Task-fencing pass alone cannot close that gap.
	var agentRuntimeStatus struct {
		AgentID int `gorm:"column:agent_id"`
	}
	if err := tx.Table("agent_runtime_status").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("agent_id").
		Where(
			"agent_id = ? AND session_id = ? AND session_epoch = ?",
			scope.AgentID,
			scope.SessionID,
			scope.SessionEpoch,
		).
		Take(&agentRuntimeStatus).Error; err != nil {
		return mapResultExecutionFenceError(err)
	}

	var task struct {
		ID int `gorm:"column:id"`
	}
	if err := tx.Table("scan_task").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").
		Where(
			`id = ? AND scan_id = ? AND status = ? AND assigned_agent_id = ? AND assigned_session_id = ? AND assigned_session_epoch = ?`,
			scope.TaskID,
			scope.ScanID,
			string(scandomain.TaskStatusRunning),
			scope.AgentID,
			scope.SessionID,
			scope.SessionEpoch,
		).
		Take(&task).Error; err != nil {
		return mapResultExecutionFenceError(err)
	}

	return nil
}

func mapResultExecutionFenceError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return resultingestapp.ErrResultExecutionFenceRejected
	}
	return err
}
