package repository

import (
	"context"
	"fmt"
	"strings"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// ListSummariesByTargetAndURLs returns metadata-only Screenshot projections for
// exact Website URL evidence. The image column is intentionally excluded.
func (r *ScreenshotRepository) ListSummariesByTargetAndURLs(targetID int, urls []string) ([]assetdomain.Screenshot, error) {
	return r.ListSummariesByTargetAndURLsContext(context.Background(), targetID, urls)
}

// ListSummariesByTargetAndURLsContext preserves a caller-owned cancellation/deadline.
func (r *ScreenshotRepository) ListSummariesByTargetAndURLsContext(ctx context.Context, targetID int, urls []string) ([]assetdomain.Screenshot, error) {
	if len(urls) == 0 {
		return []assetdomain.Screenshot{}, nil
	}

	var screenshots []model.Screenshot
	err := dbtx.Resolve(ctx, r.db).WithContext(ctx).Model(&model.Screenshot{}).
		Select("id, target_id, url, status_code, created_at, updated_at").
		Where("target_id = ? AND url IN ?", targetID, urls).
		Find(&screenshots).Error
	if err != nil {
		return nil, err
	}
	return screenshotModelListToDomain(screenshots), nil
}

// FindByTargetID finds screenshots by target ID with pagination and filter
func (r *ScreenshotRepository) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Screenshot, int64, error) {
	return r.ListByTargetIDContext(context.Background(), targetID, page, pageSize, filter, orderBy)
}

// ListByTargetIDContext preserves cancellation for request-scoped readers.
func (r *ScreenshotRepository) ListByTargetIDContext(ctx context.Context, targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Screenshot, int64, error) {
	var screenshots []model.Screenshot
	var total int64

	baseQuery := r.db.WithContext(ctx).Model(&model.Screenshot{}).Where("target_id = ?", targetID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, screenshotFilterMappingNormalized, "url"))

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.
		Select("id, target_id, url, status_code, created_at, updated_at").
		Scopes(
			scope.WithPagination(page, pageSize),
			func(db *gorm.DB) *gorm.DB { return applyScreenshotOrder(db, orderBy) },
		).
		Find(&screenshots).Error
	if err != nil {
		return nil, 0, err
	}

	return screenshotModelListToDomain(screenshots), total, nil
}

// ListFilterOptionsByTargetID returns target-scoped screenshot filter options.
// Performance is backed by target-prefixed btree indexes for exposed scalar facets.
func (r *ScreenshotRepository) ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error) {
	type optionRow struct {
		Value string
		Count int64
	}

	var rows []optionRow
	var err error
	switch field {
	case "statusCode":
		err = r.db.Model(&model.Screenshot{}).
			Select("status_code::text AS value, COUNT(*) AS count").
			Where("target_id = ? AND status_code IS NOT NULL", targetID).
			Group("status_code").
			Order("status_code ASC").
			Scan(&rows).Error
	default:
		return nil, fmt.Errorf("unsupported screenshot filter option field: %s", field)
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

func applyScreenshotOrder(db *gorm.DB, orderBy string) *gorm.DB {
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

// GetByID finds a screenshot by ID
func (r *ScreenshotRepository) GetByID(id int) (*assetdomain.Screenshot, error) {
	return r.GetByIDContext(context.Background(), id)
}

// GetByIDContext preserves cancellation for image reads.
func (r *ScreenshotRepository) GetByIDContext(ctx context.Context, id int) (*assetdomain.Screenshot, error) {
	var screenshot model.Screenshot
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&screenshot).Error
	if err != nil {
		return nil, err
	}
	return screenshotModelToDomain(&screenshot), nil
}
