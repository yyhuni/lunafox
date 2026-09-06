package repository

import (
	"context"
	"fmt"
	"strings"

	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// SearchGlobalWebsites searches the Website current-state table across active
// Targets. It deliberately does not reuse the target-scoped list query, whose
// offset/count contract and permissive scope grammar are not safe here.
func (r *WebsiteRepository) SearchGlobalWebsites(ctx context.Context, query assetapp.GlobalAssetSearchStoreQuery) ([]assetdomain.Website, error) {
	var rows []model.Website
	if err := executeGlobalAssetSearchQuery(r.db, ctx, "website", query, &rows); err != nil {
		return nil, err
	}
	return websiteModelListToDomain(rows), nil
}

// FindByTargetID finds websites by target ID with pagination and filter
func (r *WebsiteRepository) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Website, int64, error) {
	return r.ListByTargetIDContext(context.Background(), targetID, page, pageSize, filter, orderBy)
}

// ListByTargetIDContext preserves a caller-owned cancellation/deadline through website list queries.
func (r *WebsiteRepository) ListByTargetIDContext(ctx context.Context, targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Website, int64, error) {
	var websites []model.Website
	var total int64

	baseQuery := dbtx.Resolve(ctx, r.db).WithContext(ctx).Model(&model.Website{}).Where("target_id = ?", targetID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, websiteFilterMappingNormalized, "url"))

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.Scopes(
		scope.WithPagination(page, pageSize),
		func(db *gorm.DB) *gorm.DB { return applyWebsiteOrder(db, orderBy) },
	).Find(&websites).Error
	if err != nil {
		return nil, 0, err
	}

	return websiteModelListToDomain(websites), total, nil
}

// ListFilterOptionsByTargetID returns parent-scoped website filter options.
// Performance is backed by target-prefixed btree indexes for scalar fields and the website.tech GIN index.
func (r *WebsiteRepository) ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error) {
	type optionRow struct {
		Value string
		Count int64
	}

	var rows []optionRow
	var err error
	switch field {
	case "statusCode":
		err = r.db.Model(&model.Website{}).
			Select("status_code::text AS value, COUNT(*) AS count").
			Where("target_id = ? AND status_code IS NOT NULL", targetID).
			Group("status_code").
			Order("status_code ASC").
			Scan(&rows).Error
	case "tech":
		err = r.db.Raw(`
			SELECT tech_value AS value, COUNT(*) AS count
			FROM website, unnest(tech) AS tech_value
			WHERE target_id = ? AND tech_value <> ''
			GROUP BY tech_value
			ORDER BY tech_value ASC
		`, targetID).Scan(&rows).Error
	case "webserver":
		err = r.db.Model(&model.Website{}).
			Select("webserver AS value, COUNT(*) AS count").
			Where("target_id = ? AND webserver <> ''", targetID).
			Group("webserver").
			Order("webserver ASC").
			Scan(&rows).Error
	case "contentType":
		err = r.db.Model(&model.Website{}).
			Select("content_type AS value, COUNT(*) AS count").
			Where("target_id = ? AND content_type <> ''", targetID).
			Group("content_type").
			Order("content_type ASC").
			Scan(&rows).Error
	case "vhost":
		err = r.db.Model(&model.Website{}).
			Select("vhost::text AS value, COUNT(*) AS count").
			Where("target_id = ? AND vhost IS NOT NULL", targetID).
			Group("vhost").
			Order("vhost ASC").
			Scan(&rows).Error
	default:
		return nil, fmt.Errorf("unsupported website filter option field: %s", field)
	}
	if err != nil {
		return nil, err
	}

	options := make([]assetdomain.FilterOption, 0, len(rows))
	for _, row := range rows {
		options = append(options, assetdomain.FilterOption{Value: row.Value, Label: row.Value, Count: row.Count})
	}
	return options, nil
}

func applyWebsiteOrder(db *gorm.DB, orderBy string) *gorm.DB {
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

// GetByID finds a website by ID
func (r *WebsiteRepository) GetByID(id int) (*assetdomain.Website, error) {
	var website model.Website
	err := r.db.First(&website, id).Error
	if err != nil {
		return nil, err
	}
	return websiteModelToDomain(&website), nil
}

// ForEachByTargetID streams websites for a target without exposing SQL cursor lifecycle to callers.
func (r *WebsiteRepository) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Website) error) error {
	rows, err := r.db.WithContext(ctx).Model(&model.Website{}).
		Where("target_id = ?", targetID).
		Order("created_at DESC").
		Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var website model.Website
		if err := r.db.ScanRows(rows, &website); err != nil {
			return err
		}
		if err := visit(*websiteModelToDomain(&website)); err != nil {
			return err
		}
	}
	return rows.Err()
}

// CountByTargetID returns the count of websites for a target
func (r *WebsiteRepository) CountByTargetID(targetID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.Website{}).Where("target_id = ?", targetID).Count(&count).Error
	return count, err
}
