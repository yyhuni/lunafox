package repository

import (
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// Create creates a new organization.
func (r *OrganizationRepository) Create(org *model.Organization) error {
	return r.db.Create(org).Error
}

// Update updates an organization.
func (r *OrganizationRepository) Update(org *model.Organization) error {
	return r.db.Save(org).Error
}

// SoftDelete soft deletes an organization.
func (r *OrganizationRepository) SoftDelete(id int) error {
	now := time.Now().UTC()
	return r.db.Model(&model.Organization{}).Where("id = ?", id).Update("deleted_at", now).Error
}

// BulkSoftDelete soft deletes multiple organizations.
func (r *OrganizationRepository) BulkSoftDelete(ids []int) (int64, error) {
	now := time.Now().UTC()
	result := r.db.Model(&model.Organization{}).
		Scopes(scope.WithNotDeleted()).
		Where("id IN ?", ids).
		Update("deleted_at", now)
	return result.RowsAffected, result.Error
}

// BulkAddTargets adds multiple targets to an organization (ignore duplicates).
// Returns ErrTargetNotFound if any target ID does not exist.
func (r *OrganizationRepository) BulkAddTargets(organizationID int, targetIDs []int) error {
	if len(targetIDs) == 0 {
		return nil
	}

	values := make([]any, 0, len(targetIDs)*2)
	placeholders := make([]string, 0, len(targetIDs))

	for _, targetID := range targetIDs {
		placeholders = append(placeholders, "(?, ?)")
		values = append(values, organizationID, targetID)
	}

	sql := "INSERT INTO organization_target (organization_id, target_id) VALUES " +
		strings.Join(placeholders, ", ") +
		" ON CONFLICT DO NOTHING"

	err := r.db.Exec(sql, values...).Error
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
			return ErrTargetNotFound
		}
		return err
	}
	return nil
}

// UnlinkTargets removes targets from an organization.
func (r *OrganizationRepository) UnlinkTargets(organizationID int, targetIDs []int) (int64, error) {
	if len(targetIDs) == 0 {
		return 0, nil
	}

	result := r.db.Exec(
		"DELETE FROM organization_target WHERE organization_id = ? AND target_id IN ?",
		organizationID, targetIDs,
	)
	return result.RowsAffected, result.Error
}
