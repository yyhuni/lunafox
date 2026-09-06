package repository

import (
	"context"
	"fmt"
	"strings"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// FindByScanID finds directory snapshots by scan ID with pagination and filter
func (r *DirectorySnapshotRepository) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.DirectorySnapshot, int64, error) {
	var snapshots []model.DirectorySnapshot
	var total int64

	baseQuery := r.db.Model(&model.DirectorySnapshot{}).Where("scan_id = ?", scanID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, directorySnapshotFilterMappingNormalized, "url"))

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.Scopes(
		scope.WithPagination(page, pageSize),
		func(db *gorm.DB) *gorm.DB { return applyDirectorySnapshotOrder(db, orderBy) },
	).Find(&snapshots).Error
	if err != nil {
		return nil, 0, err
	}

	return directorySnapshotModelListToDomain(snapshots), total, nil
}

// ListFilterOptionsByScanID returns scan-scoped directory snapshot filter options.
// Performance is backed by scan-prefixed btree indexes for the exposed scalar facets.
func (r *DirectorySnapshotRepository) ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	type optionRow struct {
		Value string
		Count int64
	}

	var rows []optionRow
	var err error
	switch field {
	case "status":
		err = r.db.Model(&model.DirectorySnapshot{}).
			Select("status::text AS value, COUNT(*) AS count").
			Where("scan_id = ? AND status IS NOT NULL", scanID).
			Group("status").
			Order("status ASC").
			Scan(&rows).Error
	case "contentType":
		err = r.db.Model(&model.DirectorySnapshot{}).
			Select("content_type AS value, COUNT(*) AS count").
			Where("scan_id = ? AND content_type <> ''", scanID).
			Group("content_type").
			Order("content_type ASC").
			Scan(&rows).Error
	default:
		return nil, fmt.Errorf("unsupported directory snapshot filter option field: %s", field)
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

func applyDirectorySnapshotOrder(db *gorm.DB, orderBy string) *gorm.DB {
	switch strings.TrimSpace(orderBy) {
	case "status", "status asc":
		return db.Order("status ASC NULLS LAST").Order("id ASC")
	case "status desc":
		return db.Order("status DESC NULLS LAST").Order("id DESC")
	case "contentLength", "contentLength asc":
		return db.Order("content_length ASC NULLS LAST").Order("id ASC")
	case "contentLength desc":
		return db.Order("content_length DESC NULLS LAST").Order("id DESC")
	case "createdAt", "createdAt asc":
		return db.Order("created_at ASC").Order("id ASC")
	case "createdAt desc", "":
		return db.Order("created_at DESC").Order("id DESC")
	default:
		return db.Where("1 = 0")
	}
}

// ForEachByScanID streams directory snapshots for a scan without exposing SQL cursor lifecycle to callers.
func (r *DirectorySnapshotRepository) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.DirectorySnapshot) error) error {
	rows, err := r.db.WithContext(ctx).Model(&model.DirectorySnapshot{}).
		Where("scan_id = ?", scanID).
		Order("created_at DESC").
		Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var snapshot model.DirectorySnapshot
		if err := r.db.ScanRows(rows, &snapshot); err != nil {
			return err
		}
		if err := visit(*directorySnapshotModelToDomain(&snapshot)); err != nil {
			return err
		}
	}
	return rows.Err()
}

// CountByScanID returns the count of directory snapshots for a scan
func (r *DirectorySnapshotRepository) CountByScanID(scanID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.DirectorySnapshot{}).Where("scan_id = ?", scanID).Count(&count).Error
	return count, err
}
