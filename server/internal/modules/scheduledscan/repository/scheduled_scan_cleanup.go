package repository

import (
	"context"
	"errors"
	"fmt"

	scheduledapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DeleteTargetScopedForCleanup reuses the normal Schedule serialization and
// cascade path one row at a time. Organization-owned Schedules never match the
// direct target_id predicate and therefore remain outside Target cleanup.
func (repo *ScheduledScanRepository) DeleteTargetScopedForCleanup(ctx context.Context, targetID int) (int, error) {
	if repo == nil || repo.db == nil || targetID <= 0 {
		return 0, scheduledapp.ErrScheduledScanInvalidArgument
	}

	deleted := 0
	for {
		deletedOne := false
		err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := applyScheduledScanCleanupTimeouts(tx); err != nil {
				return err
			}
			var target targetRefModel
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ? AND deleted_at IS NOT NULL", targetID).
				First(&target).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("Target %d is not a tombstone", targetID)
				}
				return err
			}

			var schedule scheduledScanModel
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("target_id = ?", targetID).
				Order("id ASC").
				First(&schedule).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}
			if err := deleteScheduledScanLocked(tx, schedule.ID); err != nil {
				return err
			}
			deletedOne = true
			return nil
		})
		if err != nil {
			return deleted, err
		}
		if !deletedOne {
			return deleted, nil
		}
		deleted++
	}
}

func applyScheduledScanCleanupTimeouts(tx *gorm.DB) error {
	if tx == nil || tx.Dialector.Name() != "postgres" {
		return nil
	}
	if err := tx.Exec("SELECT set_config('lock_timeout', '5s', true)").Error; err != nil {
		return err
	}
	return tx.Exec("SELECT set_config('statement_timeout', '30s', true)").Error
}
