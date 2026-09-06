package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/identity/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Create creates a new organization.
func (r *OrganizationRepository) Create(org *model.Organization) error {
	return r.CreateContext(context.Background(), org)
}

// CreateContext preserves cancellation and maps only the named active-name
// uniqueness conflict to a domain error. Driver diagnostics stay repository-local.
func (r *OrganizationRepository) CreateContext(ctx context.Context, org *model.Organization) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return mapOrganizationWriteError(dbtx.Resolve(ctx, r.db).WithContext(ctx).Create(org).Error)
}

// Update updates an organization.
func (r *OrganizationRepository) Update(org *model.Organization) error {
	return r.UpdateContext(context.Background(), org)
}

// UpdateContext preserves cancellation and keeps duplicate-name mapping
// consistent with create operations.
func (r *OrganizationRepository) UpdateContext(ctx context.Context, org *model.Organization) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return mapOrganizationWriteError(dbtx.Resolve(ctx, r.db).WithContext(ctx).Save(org).Error)
}

// SoftDelete soft deletes an organization.
func (r *OrganizationRepository) SoftDelete(id int) error {
	return r.SoftDeleteContext(context.Background(), id)
}

// SoftDeleteContext soft deletes an organization with caller cancellation.
func (r *OrganizationRepository) SoftDeleteContext(ctx context.Context, id int) error {
	now := time.Now().UTC()
	return dbtx.Resolve(ctx, r.db).WithContext(ctx).Model(&model.Organization{}).Where("id = ?", id).Update("deleted_at", now).Error
}

// BatchSoftDelete soft deletes multiple organizations.
func (r *OrganizationRepository) BatchSoftDelete(ids []int) (int64, error) {
	return r.BatchSoftDeleteContext(context.Background(), ids)
}

// BatchSoftDeleteContext soft deletes multiple organizations with caller cancellation.
func (r *OrganizationRepository) BatchSoftDeleteContext(ctx context.Context, ids []int) (int64, error) {
	now := time.Now().UTC()
	result := dbtx.Resolve(ctx, r.db).WithContext(ctx).Model(&model.Organization{}).
		Scopes(scope.WithNotDeleted()).
		Where("id IN ?", ids).
		Update("deleted_at", now)
	return result.RowsAffected, result.Error
}

// BatchAddTargets adds multiple active targets to an organization while holding
// Target-first locks. A tombstone that commits first is rejected rather than
// leaving cleanup to repair a relationship created after deletion.
func (r *OrganizationRepository) BatchAddTargets(organizationID int, targetIDs []int) error {
	return r.batchAddTargetsInTransaction(context.Background(), organizationID, targetIDs)
}

// BatchAddTargetsContext adds relationships using the transaction attached to
// ctx when present. Callers that need target insertion and association atomicity
// must supply that transaction through the shared catalog coordinator.
func (r *OrganizationRepository) BatchAddTargetsContext(ctx context.Context, organizationID int, targetIDs []int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return batchAddTargets(ctx, dbtx.Resolve(ctx, r.db).WithContext(ctx), organizationID, targetIDs)
}

func (r *OrganizationRepository) batchAddTargetsInTransaction(ctx context.Context, organizationID int, targetIDs []int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := dbtx.WithTransaction(ctx, tx)
		return batchAddTargets(txCtx, tx.WithContext(ctx), organizationID, targetIDs)
	})
}

func batchAddTargets(ctx context.Context, db *gorm.DB, organizationID int, targetIDs []int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(targetIDs) == 0 {
		return nil
	}
	if db == nil || organizationID <= 0 {
		return ErrTargetNotFound
	}
	uniqueTargetIDs := stableOrganizationTargetIDs(targetIDs)
	if len(uniqueTargetIDs) == 0 {
		return ErrTargetNotFound
	}

	var targets []model.OrganizationTargetRef
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ? AND deleted_at IS NULL", uniqueTargetIDs).
		Order("id ASC").
		Find(&targets).Error; err != nil {
		return err
	}
	if len(targets) != len(uniqueTargetIDs) {
		return ErrTargetNotFound
	}

	values := make([]any, 0, len(uniqueTargetIDs)*2)
	placeholders := make([]string, 0, len(uniqueTargetIDs))

	for _, targetID := range uniqueTargetIDs {
		placeholders = append(placeholders, "(?, ?)")
		values = append(values, organizationID, targetID)
	}

	query := "INSERT INTO organization_target (organization_id, target_id) VALUES " +
		strings.Join(placeholders, ", ") +
		" ON CONFLICT DO NOTHING"

	if err := ctx.Err(); err != nil {
		return err
	}
	if err := db.Exec(query, values...).Error; err != nil {
		return fmt.Errorf("insert organization target relationships: %w", err)
	}
	return nil
}

func stableOrganizationTargetIDs(targetIDs []int) []int {
	seen := make(map[int]struct{}, len(targetIDs))
	unique := make([]int, 0, len(targetIDs))
	for _, targetID := range targetIDs {
		if targetID <= 0 {
			continue
		}
		if _, exists := seen[targetID]; exists {
			continue
		}
		seen[targetID] = struct{}{}
		unique = append(unique, targetID)
	}
	sort.Ints(unique)
	return unique
}

// UnlinkTargets removes targets from an organization.
func (r *OrganizationRepository) UnlinkTargets(organizationID int, targetIDs []int) (int64, error) {
	return r.UnlinkTargetsContext(context.Background(), organizationID, targetIDs)
}

// UnlinkTargetsContext removes relationships with caller cancellation.
func (r *OrganizationRepository) UnlinkTargetsContext(ctx context.Context, organizationID int, targetIDs []int) (int64, error) {
	if len(targetIDs) == 0 {
		return 0, nil
	}

	result := dbtx.Resolve(ctx, r.db).WithContext(ctx).Exec(
		"DELETE FROM organization_target WHERE organization_id = ? AND target_id IN ?",
		organizationID, targetIDs,
	)
	return result.RowsAffected, result.Error
}

func mapOrganizationWriteError(err error) error {
	if err == nil {
		return nil
	}
	if isActiveOrganizationNameConflict(err) {
		return identitydomain.ErrOrganizationExists
	}
	return err
}

func isActiveOrganizationNameConflict(err error) bool {
	var pgxErr *pgconn.PgError
	if errors.As(err, &pgxErr) {
		return pgxErr.Code == "23505" && pgxErr.ConstraintName == "idx_org_active_name_unique"
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == "23505" && pqErr.Constraint == "idx_org_active_name_unique"
	}
	return false
}
