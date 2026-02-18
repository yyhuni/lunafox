package repository

import (
	"database/sql"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// FindByScanID finds directory snapshots by scan ID with pagination and filter
func (r *DirectorySnapshotRepository) FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.DirectorySnapshot, int64, error) {
	var snapshots []model.DirectorySnapshot
	var total int64

	baseQuery := r.db.Model(&model.DirectorySnapshot{}).Where("scan_id = ?", scanID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, directorySnapshotFilterMappingNormalized, "url"))

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

	return directorySnapshotModelListToDomain(snapshots), total, nil
}

// StreamByScanID returns a sql.Rows cursor for streaming export
func (r *DirectorySnapshotRepository) StreamByScanID(scanID int) (*sql.Rows, error) {
	return r.db.Model(&model.DirectorySnapshot{}).
		Where("scan_id = ?", scanID).
		Order("created_at DESC").
		Rows()
}

// CountByScanID returns the count of directory snapshots for a scan
func (r *DirectorySnapshotRepository) CountByScanID(scanID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.DirectorySnapshot{}).Where("scan_id = ?", scanID).Count(&count).Error
	return count, err
}

// ScanRow scans a single row into DirectorySnapshot domain object
func (r *DirectorySnapshotRepository) ScanRow(rows *sql.Rows) (*snapshotdomain.DirectorySnapshot, error) {
	var snapshot model.DirectorySnapshot
	if err := r.db.ScanRows(rows, &snapshot); err != nil {
		return nil, err
	}
	return directorySnapshotModelToDomain(&snapshot), nil
}
