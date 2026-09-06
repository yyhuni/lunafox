package repository

import (
	"fmt"
	"strings"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// FindByScanID finds screenshot snapshots by scan ID with pagination and filter.
// This method intentionally excludes the image blob to avoid large payloads.
func (r *ScreenshotSnapshotRepository) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.ScreenshotSnapshot, int64, error) {
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
			func(db *gorm.DB) *gorm.DB { return applyScreenshotSnapshotOrder(db, orderBy) },
		).
		Find(&snapshots).Error
	if err != nil {
		return nil, 0, err
	}

	return screenshotSnapshotModelListToDomain(snapshots), total, nil
}

// ListFilterOptionsByScanID returns scan-scoped screenshot status-code filter options.
func (r *ScreenshotSnapshotRepository) ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	type optionRow struct {
		Value string
		Count int64
	}

	var rows []optionRow
	var err error
	switch field {
	case "statusCode":
		err = r.db.Model(&model.ScreenshotSnapshot{}).
			Select("status_code::text AS value, COUNT(*) AS count").
			Where("scan_id = ? AND status_code IS NOT NULL", scanID).
			Group("status_code").
			Order("status_code ASC").
			Scan(&rows).Error
	default:
		return nil, fmt.Errorf("unsupported screenshot snapshot filter option field: %s", field)
	}
	if err != nil {
		return nil, err
	}

	options := make([]snapshotdomain.FilterOption, 0, len(rows))
	for _, row := range rows {
		options = append(options, snapshotdomain.FilterOption{Value: row.Value, Label: row.Value, Count: row.Count})
	}
	return options, nil
}

func applyScreenshotSnapshotOrder(db *gorm.DB, orderBy string) *gorm.DB {
	switch strings.TrimSpace(orderBy) {
	case "statusCode", "statusCode asc":
		return db.Order("status_code ASC NULLS LAST").Order("id ASC")
	case "statusCode desc":
		return db.Order("status_code DESC NULLS LAST").Order("id DESC")
	case "createdAt", "createdAt asc":
		return db.Order("created_at ASC").Order("id ASC")
	case "createdAt desc", "":
		return db.Order("created_at DESC").Order("id DESC")
	default:
		return db.Where("1 = 0")
	}
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
