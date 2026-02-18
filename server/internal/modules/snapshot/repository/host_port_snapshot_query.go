package repository

import (
	"database/sql"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// FindByScanID finds host-port snapshots by scan ID with pagination and filter
func (r *HostPortSnapshotRepository) FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.HostPortSnapshot, int64, error) {
	var snapshots []model.HostPortSnapshot
	var total int64

	baseQuery := r.db.Model(&model.HostPortSnapshot{}).Where("scan_id = ?", scanID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, hostPortSnapshotFilterMappingNormalized, "ip"))

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

	return hostPortSnapshotModelListToDomain(snapshots), total, nil
}

// StreamByScanID returns a sql.Rows cursor for streaming export
func (r *HostPortSnapshotRepository) StreamByScanID(scanID int) (*sql.Rows, error) {
	return r.db.Model(&model.HostPortSnapshot{}).
		Where("scan_id = ?", scanID).
		Order("created_at DESC").
		Rows()
}

// CountByScanID returns the count of host-port snapshots for a scan
func (r *HostPortSnapshotRepository) CountByScanID(scanID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.HostPortSnapshot{}).Where("scan_id = ?", scanID).Count(&count).Error
	return count, err
}

// ScanRow scans a single row into HostPortSnapshot domain object
func (r *HostPortSnapshotRepository) ScanRow(rows *sql.Rows) (*snapshotdomain.HostPortSnapshot, error) {
	var snapshot model.HostPortSnapshot
	if err := r.db.ScanRows(rows, &snapshot); err != nil {
		return nil, err
	}
	return hostPortSnapshotModelToDomain(&snapshot), nil
}
