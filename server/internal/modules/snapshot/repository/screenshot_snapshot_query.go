package repository

import (
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// FindByScanID finds screenshot snapshots by scan ID with pagination and filter.
// This method intentionally excludes the image blob to avoid large payloads.
func (r *ScreenshotSnapshotRepository) FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.ScreenshotSnapshot, int64, error) {
	var snapshots []model.ScreenshotSnapshot
	var total int64

	baseQuery := r.db.Model(&model.ScreenshotSnapshot{}).Where("scan_id = ?", scanID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, screenshotSnapshotFilterMappingNormalized, "url"))

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.
		Select("id, scan_id, url, status_code, created_at").
		Scopes(
			scope.WithPagination(page, pageSize),
			scope.OrderByCreatedAtDesc(),
		).
		Find(&snapshots).Error
	if err != nil {
		return nil, 0, err
	}

	return screenshotSnapshotModelListToDomain(snapshots), total, nil
}

// FindByIDAndScanID finds a screenshot snapshot by ID under a scan (includes image data)
func (r *ScreenshotSnapshotRepository) FindByIDAndScanID(id int, scanID int) (*snapshotdomain.ScreenshotSnapshot, error) {
	var snapshot model.ScreenshotSnapshot
	err := r.db.Where("id = ? AND scan_id = ?", id, scanID).First(&snapshot).Error
	if err != nil {
		return nil, err
	}
	return screenshotSnapshotModelToDomain(&snapshot), nil
}

// CountByScanID returns the count of screenshot snapshots for a scan
func (r *ScreenshotSnapshotRepository) CountByScanID(scanID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.ScreenshotSnapshot{}).Where("scan_id = ?", scanID).Count(&count).Error
	return count, err
}
