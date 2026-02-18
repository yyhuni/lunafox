package repository

import (
	"database/sql"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// FindByScanID finds endpoint snapshots by scan ID with pagination and filter
func (r *EndpointSnapshotRepository) FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.EndpointSnapshot, int64, error) {
	var snapshots []model.EndpointSnapshot
	var total int64

	baseQuery := r.db.Model(&model.EndpointSnapshot{}).Where("scan_id = ?", scanID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, endpointSnapshotFilterMappingNormalized, "url"))

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

	return endpointSnapshotModelListToDomain(snapshots), total, nil
}

// StreamByScanID returns a sql.Rows cursor for streaming export
func (r *EndpointSnapshotRepository) StreamByScanID(scanID int) (*sql.Rows, error) {
	return r.db.Model(&model.EndpointSnapshot{}).
		Where("scan_id = ?", scanID).
		Order("created_at DESC").
		Rows()
}

// CountByScanID returns the count of endpoint snapshots for a scan
func (r *EndpointSnapshotRepository) CountByScanID(scanID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.EndpointSnapshot{}).Where("scan_id = ?", scanID).Count(&count).Error
	return count, err
}

// ScanRow scans a single row into EndpointSnapshot domain object
func (r *EndpointSnapshotRepository) ScanRow(rows *sql.Rows) (*snapshotdomain.EndpointSnapshot, error) {
	var snapshot model.EndpointSnapshot
	if err := r.db.ScanRows(rows, &snapshot); err != nil {
		return nil, err
	}
	return endpointSnapshotModelToDomain(&snapshot), nil
}
