package repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Create creates a new target.
func (r *TargetRepository) Create(target *catalogdomain.Target) error {
	if target == nil {
		return fmt.Errorf("target is required")
	}
	modelTarget := targetDomainToModel(target)
	if err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(modelTarget).Error; err != nil {
			return err
		}
		return createTargetBlacklistPolicy(tx, modelTarget.ID)
	}); err != nil {
		return err
	}
	*target = *targetModelToDomain(modelTarget)
	return nil
}

// Update updates a target.
func (r *TargetRepository) Update(target *catalogdomain.Target) error {
	if target == nil || target.ID <= 0 {
		return fmt.Errorf("target is required")
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		var current model.Target
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Scopes(scope.WithNotDeleted()).
			Where("id = ?", target.ID).
			First(&current).Error; err != nil {
			return err
		}
		result := tx.Model(&model.Target{}).
			Where("id = ? AND deleted_at IS NULL", target.ID).
			Updates(map[string]any{
				"name": target.Name,
				"type": target.Type,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		// Preserve persisted lifecycle fields even if a caller constructed a
		// partial domain object. A full-model save could otherwise clear a
		// concurrent deletion tombstone.
		target.CreatedAt = current.CreatedAt
		target.LastScannedAt = current.LastScannedAt
		target.DeletedAt = nil
		return nil
	})
}

// SoftDelete soft deletes a target.
func (r *TargetRepository) SoftDelete(id int) error {
	_, err := r.TombstoneAndEnsureCleanup(context.Background(), id)
	return err
}

// BatchSoftDelete soft deletes multiple targets by IDs.
func (r *TargetRepository) BatchSoftDelete(ids []int) (int64, error) {
	return r.BatchTombstoneAndEnsureCleanup(context.Background(), ids)
}

// TombstoneAndEnsureCleanup makes one Target immediately unavailable and
// atomically records its durable cleanup obligation. It must stay independent
// of all Target-owned row counts, so this transaction only touches target and
// target_cleanup_job.
func (r *TargetRepository) TombstoneAndEnsureCleanup(ctx context.Context, id int) (bool, error) {
	if r == nil || r.db == nil || id <= 0 {
		return false, gorm.ErrRecordNotFound
	}
	deletedNow := false
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var target model.Target
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&target).Error; err != nil {
			return err
		}
		if target.DeletedAt == nil {
			result := tx.Model(&model.Target{}).
				Where("id = ? AND deleted_at IS NULL", id).
				Update("deleted_at", now)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return gorm.ErrRecordNotFound
			}
			deletedNow = true
		}
		return ensureTargetCleanupJobs(tx, []int{id}, now)
	})
	return deletedNow, err
}

// BatchTombstoneAndEnsureCleanup atomically validates all requested real
// Targets, locks them in ascending ID order, and creates one cleanup record per
// Target. Existing tombstones are idempotent successes and do not add to the
// transition count.
func (r *TargetRepository) BatchTombstoneAndEnsureCleanup(ctx context.Context, ids []int) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("target repository is required")
	}
	uniqueIDs := stableUniqueTargetIDs(ids)
	if len(uniqueIDs) == 0 {
		return 0, fmt.Errorf("at least one target is required")
	}
	lockIDs := targetIDsInLockOrder(uniqueIDs)
	now := time.Now().UTC()
	var deletedCount int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var targets []model.Target
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id IN ?", lockIDs).
			Order("id ASC").
			Find(&targets).Error; err != nil {
			return err
		}
		if len(targets) != len(lockIDs) {
			return gorm.ErrRecordNotFound
		}
		result := tx.Model(&model.Target{}).
			Where("id IN ? AND deleted_at IS NULL", lockIDs).
			Update("deleted_at", now)
		if result.Error != nil {
			return result.Error
		}
		deletedCount = result.RowsAffected
		return ensureTargetCleanupJobs(tx, uniqueIDs, now)
	})
	return deletedCount, err
}

func ensureTargetCleanupJobs(tx *gorm.DB, targetIDs []int, now time.Time) error {
	jobs := make([]model.TargetCleanupJob, 0, len(targetIDs))
	for _, targetID := range targetIDs {
		jobs = append(jobs, model.TargetCleanupJob{
			TargetID:    targetID,
			Status:      "pending",
			NextRetryAt: now,
		})
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "target_id"}},
		DoNothing: true,
	}).Create(&jobs).Error
}

func stableUniqueTargetIDs(ids []int) []int {
	seen := make(map[int]struct{}, len(ids))
	unique := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

func targetIDsInLockOrder(ids []int) []int {
	ordered := append([]int(nil), ids...)
	sort.Ints(ordered)
	return ordered
}

// BatchCreateIgnoreConflicts creates multiple targets, ignoring duplicates.
func (r *TargetRepository) BatchCreateIgnoreConflicts(targets []catalogdomain.Target) (int, error) {
	return r.batchCreateIgnoreConflictsInTransaction(context.Background(), targets)
}

// BatchCreateIgnoreConflictsContext performs the insert and policy creation on
// the transaction attached to ctx. The catalog coordinator owns commit/rollback.
func (r *TargetRepository) BatchCreateIgnoreConflictsContext(ctx context.Context, targets []catalogdomain.Target) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if len(targets) == 0 {
		return 0, nil
	}

	modelTargets := targetDomainListToModel(targets)
	var created int
	db := dbtx.Resolve(ctx, r.db).WithContext(ctx)
	result := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&modelTargets)
	if result.Error != nil {
		return 0, result.Error
	}
	created = int(result.RowsAffected)
	for index := range modelTargets {
		// PostgreSQL/SQLite RETURNING only fills IDs for rows inserted by this
		// statement. Existing conflicting Targets are intentionally not repaired.
		if modelTargets[index].ID == 0 {
			continue
		}
		if err := createTargetBlacklistPolicy(db, modelTargets[index].ID); err != nil {
			return 0, err
		}
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return created, nil
}

func (r *TargetRepository) batchCreateIgnoreConflictsInTransaction(ctx context.Context, targets []catalogdomain.Target) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	var created int
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		created, err = r.BatchCreateIgnoreConflictsContext(dbtx.WithTransaction(ctx, tx), targets)
		return err
	})
	return created, err
}

func createTargetBlacklistPolicy(tx *gorm.DB, targetID int) error {
	if tx == nil || targetID <= 0 {
		return fmt.Errorf("target blacklist policy requires a persisted target")
	}
	return tx.Table("blacklist_policy").Create(map[string]any{
		"scope":     "target",
		"target_id": targetID,
		"patterns":  datatypes.JSON([]byte("[]")),
	}).Error
}
