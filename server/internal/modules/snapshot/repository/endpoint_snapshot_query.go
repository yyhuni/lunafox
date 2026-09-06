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

// FindByScanID finds endpoint snapshots by scan ID with pagination and filter
func (r *EndpointSnapshotRepository) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.EndpointSnapshot, int64, error) {
	var snapshots []model.EndpointSnapshot
	var total int64

	baseQuery := r.db.Model(&model.EndpointSnapshot{}).Where("scan_id = ?", scanID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, endpointSnapshotFilterMappingNormalized, "url"))

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.Scopes(
		scope.WithPagination(page, pageSize),
		func(db *gorm.DB) *gorm.DB { return applyEndpointSnapshotOrder(db, orderBy) },
	).Find(&snapshots).Error
	if err != nil {
		return nil, 0, err
	}

	return endpointSnapshotModelListToDomain(snapshots), total, nil
}

// ListFilterOptionsByScanID returns scan-scoped endpoint snapshot filter options.
// Performance is backed by scan-prefixed btree indexes for scalar fields and the endpoint_snapshot.tech GIN index.
func (r *EndpointSnapshotRepository) ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	type optionRow struct {
		Value string
		Count int64
	}

	var rows []optionRow
	var err error
	switch field {
	case "statusCode":
		err = r.db.Model(&model.EndpointSnapshot{}).
			Select("status_code::text AS value, COUNT(*) AS count").
			Where("scan_id = ? AND status_code IS NOT NULL", scanID).
			Group("status_code").
			Order("status_code ASC").
			Scan(&rows).Error
	case "tech":
		err = r.db.Raw(`
			SELECT tech_value AS value, COUNT(*) AS count
			FROM endpoint_snapshot, unnest(tech) AS tech_value
			WHERE scan_id = ? AND tech_value <> ''
			GROUP BY tech_value
			ORDER BY tech_value ASC
		`, scanID).Scan(&rows).Error
	case "webserver":
		err = r.db.Model(&model.EndpointSnapshot{}).
			Select("webserver AS value, COUNT(*) AS count").
			Where("scan_id = ? AND webserver <> ''", scanID).
			Group("webserver").
			Order("webserver ASC").
			Scan(&rows).Error
	case "contentType":
		err = r.db.Model(&model.EndpointSnapshot{}).
			Select("content_type AS value, COUNT(*) AS count").
			Where("scan_id = ? AND content_type <> ''", scanID).
			Group("content_type").
			Order("content_type ASC").
			Scan(&rows).Error
	case "vhost":
		err = r.db.Model(&model.EndpointSnapshot{}).
			Select("vhost::text AS value, COUNT(*) AS count").
			Where("scan_id = ? AND vhost IS NOT NULL", scanID).
			Group("vhost").
			Order("vhost ASC").
			Scan(&rows).Error
	default:
		return nil, fmt.Errorf("unsupported endpoint snapshot filter option field: %s", field)
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

func applyEndpointSnapshotOrder(db *gorm.DB, orderBy string) *gorm.DB {
	switch strings.TrimSpace(orderBy) {
	case "statusCode", "statusCode asc":
		return db.Order("status_code ASC NULLS LAST").Order("id ASC")
	case "statusCode desc":
		return db.Order("status_code DESC NULLS LAST").Order("id DESC")
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

// ForEachByScanID streams endpoint snapshots for a scan without exposing SQL cursor lifecycle to callers.
func (r *EndpointSnapshotRepository) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.EndpointSnapshot) error) error {
	rows, err := r.db.WithContext(ctx).Model(&model.EndpointSnapshot{}).
		Where("scan_id = ?", scanID).
		Order("created_at DESC").
		Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var snapshot model.EndpointSnapshot
		if err := r.db.ScanRows(rows, &snapshot); err != nil {
			return err
		}
		if err := visit(*endpointSnapshotModelToDomain(&snapshot)); err != nil {
			return err
		}
	}
	return rows.Err()
}

// CountByScanID returns the count of endpoint snapshots for a scan
func (r *EndpointSnapshotRepository) CountByScanID(scanID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.EndpointSnapshot{}).Where("scan_id = ?", scanID).Count(&count).Error
	return count, err
}
