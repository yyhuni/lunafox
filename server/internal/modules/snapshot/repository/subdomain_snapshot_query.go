package repository

import (
	"database/sql"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// FindByScanID finds subdomain snapshots by scan ID with pagination and filter
func (r *SubdomainSnapshotRepository) FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.SubdomainSnapshot, int64, error) {
	var snapshots []model.SubdomainSnapshot
	var total int64

	baseQuery := r.db.Model(&model.SubdomainSnapshot{}).Where("scan_id = ?", scanID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, subdomainSnapshotFilterMappingNormalized, "name"))

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.Scopes(
		scope.WithPagination(page, pageSize),
		scope.OrderByCreatedAtDesc(),
	).Find(&snapshots).Error
	if err != nil {
		return nil, 0, err
	}

	return subdomainSnapshotModelListToDomain(snapshots), total, nil
}

// StreamByScanID returns a sql.Rows cursor for streaming export
func (r *SubdomainSnapshotRepository) StreamByScanID(scanID int) (*sql.Rows, error) {
	return r.db.Model(&model.SubdomainSnapshot{}).
		Where("scan_id = ?", scanID).
		Order("created_at DESC").
		Rows()
}

// CountByScanID returns the count of subdomain snapshots for a scan
func (r *SubdomainSnapshotRepository) CountByScanID(scanID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.SubdomainSnapshot{}).Where("scan_id = ?", scanID).Count(&count).Error
	return count, err
}

// ScanRow scans a single row into SubdomainSnapshot domain object
func (r *SubdomainSnapshotRepository) ScanRow(rows *sql.Rows) (*snapshotdomain.SubdomainSnapshot, error) {
	var snapshot model.SubdomainSnapshot
	if err := r.db.ScanRows(rows, &snapshot); err != nil {
		return nil, err
	}
	return subdomainSnapshotModelToDomain(&snapshot), nil
}
