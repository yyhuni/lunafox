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

// FindByScanID finds website snapshots by scan ID with pagination and filter
func (r *WebsiteSnapshotRepository) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.WebsiteSnapshot, int64, error) {
	var snapshots []model.WebsiteSnapshot
	var total int64

	baseQuery := r.db.Model(&model.WebsiteSnapshot{}).Where("scan_id = ?", scanID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, websiteSnapshotFilterMappingNormalized, "url"))

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.Scopes(
		scope.WithPagination(page, pageSize),
		func(db *gorm.DB) *gorm.DB { return applyWebsiteSnapshotOrder(db, orderBy) },
	).Find(&snapshots).Error
	if err != nil {
		return nil, 0, err
	}

	return websiteSnapshotModelListToDomain(snapshots), total, nil
}

// ListFilterOptionsByScanID returns scan-scoped website snapshot filter options.
// Performance is backed by scan-prefixed btree indexes for scalar fields and the website_snapshot.tech GIN index.
func (r *WebsiteSnapshotRepository) ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	type optionRow struct {
		Value string
		Count int64
	}

	var rows []optionRow
	var err error
	switch field {
	case "statusCode":
		err = r.db.Model(&model.WebsiteSnapshot{}).
			Select("status_code::text AS value, COUNT(*) AS count").
			Where("scan_id = ? AND status_code IS NOT NULL", scanID).
			Group("status_code").
			Order("status_code ASC").
			Scan(&rows).Error
	case "tech":
		err = r.db.Raw(`
			SELECT tech_value AS value, COUNT(*) AS count
			FROM website_snapshot, unnest(tech) AS tech_value
			WHERE scan_id = ? AND tech_value <> ''
			GROUP BY tech_value
			ORDER BY tech_value ASC
		`, scanID).Scan(&rows).Error
	case "webserver":
		err = r.db.Model(&model.WebsiteSnapshot{}).
			Select("webserver AS value, COUNT(*) AS count").
			Where("scan_id = ? AND webserver <> ''", scanID).
			Group("webserver").
			Order("webserver ASC").
			Scan(&rows).Error
	case "contentType":
		err = r.db.Model(&model.WebsiteSnapshot{}).
			Select("content_type AS value, COUNT(*) AS count").
			Where("scan_id = ? AND content_type <> ''", scanID).
			Group("content_type").
			Order("content_type ASC").
			Scan(&rows).Error
	case "vhost":
		err = r.db.Model(&model.WebsiteSnapshot{}).
			Select("vhost::text AS value, COUNT(*) AS count").
			Where("scan_id = ? AND vhost IS NOT NULL", scanID).
			Group("vhost").
			Order("vhost ASC").
			Scan(&rows).Error
	default:
		return nil, fmt.Errorf("unsupported website snapshot filter option field: %s", field)
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

func applyWebsiteSnapshotOrder(db *gorm.DB, orderBy string) *gorm.DB {
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

// ForEachByScanID streams website snapshots for a scan without exposing SQL cursor lifecycle to callers.
func (r *WebsiteSnapshotRepository) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.WebsiteSnapshot) error) error {
	rows, err := r.db.WithContext(ctx).Model(&model.WebsiteSnapshot{}).
		Where("scan_id = ?", scanID).
		Order("created_at DESC").
		Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var snapshot model.WebsiteSnapshot
		if err := r.db.ScanRows(rows, &snapshot); err != nil {
			return err
		}
		if err := visit(*websiteSnapshotModelToDomain(&snapshot)); err != nil {
			return err
		}
	}
	return rows.Err()
}

// ForEachWebsiteURLByScanID streams only the stored URL projection for
// Server-produced execution inputs. The database query deliberately does not
// order, deduplicate, or rebuild URL values; the producer owns validation and
// the repository's existing row stream remains the source order.
func (r *WebsiteSnapshotRepository) ForEachWebsiteURLByScanID(ctx context.Context, scanID int, visit func(string) error) error {
	rows, err := r.db.WithContext(ctx).Model(&model.WebsiteSnapshot{}).Select("url").Where("scan_id = ?", scanID).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return err
		}
		if err := visit(url); err != nil {
			return err
		}
	}
	return rows.Err()
}

// CountByScanID returns the count of website snapshots for a scan
func (r *WebsiteSnapshotRepository) CountByScanID(scanID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.WebsiteSnapshot{}).Where("scan_id = ?", scanID).Count(&count).Error
	return count, err
}
