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
	"github.com/yyhuni/lunafox/server/internal/pkg/webscope"
	"gorm.io/gorm"
)

// SearchGlobalEndpoints searches the Endpoint current-state table across
// active Targets without consulting snapshots or any other asset table.
func (r *EndpointRepository) SearchGlobalEndpoints(ctx context.Context, query assetapp.GlobalAssetSearchStoreQuery) ([]assetdomain.Endpoint, error) {
	var rows []model.Endpoint
	if err := executeGlobalAssetSearchQuery(r.db, ctx, "endpoint", query, &rows); err != nil {
		return nil, err
	}
	return endpointModelListToDomain(rows), nil
}

// FindByTargetID finds endpoints by target ID with pagination and filter
func (r *EndpointRepository) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Endpoint, int64, error) {
	return r.ListByTargetIDContext(context.Background(), targetID, page, pageSize, filter, orderBy)
}

// ListByTargetIDContext preserves a caller-owned cancellation/deadline through endpoint list queries.
func (r *EndpointRepository) ListByTargetIDContext(ctx context.Context, targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Endpoint, int64, error) {
	var endpoints []model.Endpoint
	var total int64

	baseQuery := dbtx.Resolve(ctx, r.db).WithContext(ctx).Model(&model.Endpoint{}).Where("target_id = ?", targetID)
	baseQuery = baseQuery.Scopes(applyEndpointListFilter(filter))

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.Scopes(
		scope.WithPagination(page, pageSize),
		func(db *gorm.DB) *gorm.DB { return applyEndpointOrder(db, orderBy) },
	).Find(&endpoints).Error
	if err != nil {
		return nil, 0, err
	}

	return endpointModelListToDomain(endpoints), total, nil
}

func applyEndpointListFilter(filter string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		remaining, websiteScope, err := webscope.ExtractFilterScope(filter)
		if err != nil {
			return db.Where("1 = 0")
		}
		if websiteScope != nil {
			condition, args := websiteScope.SQLPredicate("url")
			db = db.Where(condition, args...)
		}
		return scope.WithFilterDefault(remaining, endpointFilterMappingNormalized, "url")(db)
	}
}

// ListFilterOptionsByTargetID returns parent-scoped endpoint filter options.
// Performance is backed by target-prefixed btree indexes for scalar fields and the endpoint.tech GIN index.
func (r *EndpointRepository) ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error) {
	type optionRow struct {
		Value string
		Count int64
	}

	var rows []optionRow
	var err error
	switch field {
	case "statusCode":
		err = r.db.Model(&model.Endpoint{}).
			Select("status_code::text AS value, COUNT(*) AS count").
			Where("target_id = ? AND status_code IS NOT NULL", targetID).
			Group("status_code").
			Order("status_code ASC").
			Scan(&rows).Error
	case "tech":
		err = r.db.Raw(`
			SELECT tech_value AS value, COUNT(*) AS count
			FROM endpoint, unnest(tech) AS tech_value
			WHERE target_id = ? AND tech_value <> ''
			GROUP BY tech_value
			ORDER BY tech_value ASC
		`, targetID).Scan(&rows).Error
	case "webserver":
		err = r.db.Model(&model.Endpoint{}).
			Select("webserver AS value, COUNT(*) AS count").
			Where("target_id = ? AND webserver <> ''", targetID).
			Group("webserver").
			Order("webserver ASC").
			Scan(&rows).Error
	case "contentType":
		err = r.db.Model(&model.Endpoint{}).
			Select("content_type AS value, COUNT(*) AS count").
			Where("target_id = ? AND content_type <> ''", targetID).
			Group("content_type").
			Order("content_type ASC").
			Scan(&rows).Error
	case "vhost":
		err = r.db.Model(&model.Endpoint{}).
			Select("vhost::text AS value, COUNT(*) AS count").
			Where("target_id = ? AND vhost IS NOT NULL", targetID).
			Group("vhost").
			Order("vhost ASC").
			Scan(&rows).Error
	default:
		return nil, fmt.Errorf("unsupported endpoint filter option field: %s", field)
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

func applyEndpointOrder(db *gorm.DB, orderBy string) *gorm.DB {
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

// GetByID finds an endpoint by ID
func (r *EndpointRepository) GetByID(id int) (*assetdomain.Endpoint, error) {
	var endpoint model.Endpoint
	err := r.db.First(&endpoint, id).Error
	if err != nil {
		return nil, err
	}
	return endpointModelToDomain(&endpoint), nil
}

// ForEachByTargetID streams endpoints for a target without exposing SQL cursor lifecycle to callers.
func (r *EndpointRepository) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Endpoint) error) error {
	rows, err := r.db.WithContext(ctx).Model(&model.Endpoint{}).
		Where("target_id = ?", targetID).
		Order("created_at DESC").
		Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var endpoint model.Endpoint
		if err := r.db.ScanRows(rows, &endpoint); err != nil {
			return err
		}
		if err := visit(*endpointModelToDomain(&endpoint)); err != nil {
			return err
		}
	}
	return rows.Err()
}

// CountByTargetID returns the count of endpoints for a target
func (r *EndpointRepository) CountByTargetID(targetID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.Endpoint{}).Where("target_id = ?", targetID).Count(&count).Error
	return count, err
}
