package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	cleanupdomain "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/repository/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (repo *TargetCleanupRepository) ConfirmTombstone(ctx context.Context, targetID int) error {
	if repo == nil || repo.db == nil || targetID <= 0 {
		return cleanupdomain.ErrTargetTombstoneUnavailable
	}
	return repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := applyTargetCleanupTimeouts(tx); err != nil {
			return err
		}
		return lockTargetTombstone(tx, targetID)
	})
}

func (repo *TargetCleanupRepository) DeleteTargetControlPlane(ctx context.Context, targetID int) (int64, int64, error) {
	if repo == nil || repo.db == nil || targetID <= 0 {
		return 0, 0, cleanupdomain.ErrTargetTombstoneUnavailable
	}
	var relationships, policies int64
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := applyTargetCleanupTimeouts(tx); err != nil {
			return err
		}
		if err := lockTargetTombstone(tx, targetID); err != nil {
			return err
		}
		result := tx.Exec("DELETE FROM organization_target WHERE target_id = ?", targetID)
		if result.Error != nil {
			return result.Error
		}
		relationships = result.RowsAffected
		result = tx.Exec("DELETE FROM blacklist_policy WHERE target_id = ? AND scope = 'target'", targetID)
		if result.Error != nil {
			return result.Error
		}
		policies = result.RowsAffected
		return nil
	})
	return relationships, policies, err
}

func (repo *TargetCleanupRepository) DeleteCurrentAssetBatch(
	ctx context.Context,
	targetID int,
	resource cleanupdomain.AssetResource,
	batchSize int,
) (int64, error) {
	table, ok := targetCleanupAssetTable(resource)
	if repo == nil || repo.db == nil || targetID <= 0 || batchSize <= 0 || !ok {
		return 0, fmt.Errorf("target cleanup asset batch scope is invalid")
	}
	var deleted int64
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := applyTargetCleanupTimeouts(tx); err != nil {
			return err
		}
		if err := lockTargetTombstone(tx, targetID); err != nil {
			return err
		}
		var candidateIDs []int
		if err := tx.Table(table).
			Select("id").
			Where("target_id = ?", targetID).
			Order("id ASC").
			Limit(batchSize).
			Pluck("id", &candidateIDs).Error; err != nil {
			return err
		}
		if len(candidateIDs) == 0 {
			return nil
		}
		result := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE target_id = ? AND id IN ?", table), targetID, candidateIDs)
		if result.Error != nil {
			return result.Error
		}
		deleted = result.RowsAffected
		return nil
	})
	return deleted, err
}

func (repo *TargetCleanupRepository) MarkCompletedIfClear(ctx context.Context, jobID, targetID int, completedAt time.Time) (bool, error) {
	if repo == nil || repo.db == nil || jobID <= 0 || targetID <= 0 {
		return false, cleanupdomain.ErrTargetTombstoneUnavailable
	}
	completed := false
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := applyTargetCleanupTimeouts(tx); err != nil {
			return err
		}
		if err := lockTargetTombstone(tx, targetID); err != nil {
			return err
		}
		for _, condition := range targetCleanupCompletionConditions {
			var remaining int
			result := tx.Raw(condition, targetID).Scan(&remaining)
			if result.Error != nil {
				return result.Error
			}
			if remaining != 0 {
				return nil
			}
		}
		result := tx.Model(&model.TargetCleanupJob{}).
			Where("id = ? AND target_id = ? AND status = ?", jobID, targetID, string(cleanupdomain.CleanupJobPending)).
			Updates(map[string]any{
				"status":       string(cleanupdomain.CleanupJobCompleted),
				"completed_at": completedAt.UTC(),
				"last_error":   "",
			})
		if result.Error != nil {
			return result.Error
		}
		completed = result.RowsAffected == 1
		return nil
	})
	return completed, err
}

func lockTargetTombstone(tx *gorm.DB, targetID int) error {
	var target struct {
		ID int `gorm:"column:id"`
	}
	if err := tx.Table("target").Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").
		Where("id = ? AND deleted_at IS NOT NULL", targetID).
		Take(&target).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return cleanupdomain.ErrTargetTombstoneUnavailable
		}
		return err
	}
	return nil
}

func applyTargetCleanupTimeouts(tx *gorm.DB) error {
	if tx == nil || tx.Dialector.Name() != "postgres" {
		return nil
	}
	if err := tx.Exec("SELECT set_config('lock_timeout', '5s', true)").Error; err != nil {
		return err
	}
	return tx.Exec("SELECT set_config('statement_timeout', '30s', true)").Error
}

func targetCleanupAssetTable(resource cleanupdomain.AssetResource) (string, bool) {
	switch resource {
	case cleanupdomain.AssetResourceSubdomain:
		return "subdomain", true
	case cleanupdomain.AssetResourceHostPortMapping:
		return "host_port_mapping", true
	case cleanupdomain.AssetResourceWebsite:
		return "website", true
	case cleanupdomain.AssetResourceEndpoint:
		return "endpoint", true
	case cleanupdomain.AssetResourceDirectory:
		return "directory", true
	case cleanupdomain.AssetResourceScreenshot:
		return "screenshot", true
	case cleanupdomain.AssetResourceVulnerability:
		return "vulnerability", true
	default:
		return "", false
	}
}

var targetCleanupCompletionConditions = []string{
	"SELECT 1 FROM scheduled_scan WHERE target_id = ? LIMIT 1",
	`SELECT 1
		FROM scheduled_scan_occurrence AS occurrence
		JOIN scheduled_scan AS schedule ON schedule.id = occurrence.scheduled_scan_id
		WHERE schedule.target_id = ?
		LIMIT 1`,
	"SELECT 1 FROM organization_target WHERE target_id = ? LIMIT 1",
	"SELECT 1 FROM blacklist_policy WHERE target_id = ? AND scope = 'target' LIMIT 1",
	"SELECT 1 FROM scan WHERE target_id = ? AND status IN ('pending', 'running') LIMIT 1",
	"SELECT 1 FROM subdomain WHERE target_id = ? LIMIT 1",
	"SELECT 1 FROM host_port_mapping WHERE target_id = ? LIMIT 1",
	"SELECT 1 FROM website WHERE target_id = ? LIMIT 1",
	"SELECT 1 FROM endpoint WHERE target_id = ? LIMIT 1",
	"SELECT 1 FROM directory WHERE target_id = ? LIMIT 1",
	"SELECT 1 FROM screenshot WHERE target_id = ? LIMIT 1",
	"SELECT 1 FROM vulnerability WHERE target_id = ? LIMIT 1",
}
