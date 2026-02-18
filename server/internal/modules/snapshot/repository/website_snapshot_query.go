package repository

import (
	"database/sql"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// FindByScanID finds website snapshots by scan ID with pagination and filter
func (r *WebsiteSnapshotRepository) FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.WebsiteSnapshot, int64, error) {
	var snapshots []model.WebsiteSnapshot
	var total int64

	baseQuery := r.db.Model(&model.WebsiteSnapshot{}).Where("scan_id = ?", scanID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, websiteSnapshotFilterMappingNormalized, "url"))

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

	return websiteSnapshotModelListToDomain(snapshots), total, nil
}

// StreamByScanID returns a sql.Rows cursor for streaming export
func (r *WebsiteSnapshotRepository) StreamByScanID(scanID int) (*sql.Rows, error) {
	return r.db.Model(&model.WebsiteSnapshot{}).
		Where("scan_id = ?", scanID).
		Order("created_at DESC").
		Rows()
}

// CountByScanID returns the count of website snapshots for a scan
func (r *WebsiteSnapshotRepository) CountByScanID(scanID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.WebsiteSnapshot{}).Where("scan_id = ?", scanID).Count(&count).Error
	return count, err
}

// ScanRow scans a single row into WebsiteSnapshot domain object
func (r *WebsiteSnapshotRepository) ScanRow(rows *sql.Rows) (*snapshotdomain.WebsiteSnapshot, error) {
	var snapshot model.WebsiteSnapshot
	if err := r.db.ScanRows(rows, &snapshot); err != nil {
		return nil, err
	}
	return websiteSnapshotModelToDomain(&snapshot), nil
}
