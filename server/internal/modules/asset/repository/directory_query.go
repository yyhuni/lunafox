package repository

import (
	"context"
	"fmt"
	"strings"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"github.com/yyhuni/lunafox/server/internal/pkg/webscope"
	"gorm.io/gorm"
)

// ListByTargetID finds directories by target ID with pagination and filter
func (r *DirectoryRepository) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Directory, int64, error) {
	return r.ListByTargetIDContext(context.Background(), targetID, page, pageSize, filter, orderBy)
}

// ListByTargetIDContext preserves a caller-owned cancellation/deadline through directory list queries.
func (r *DirectoryRepository) ListByTargetIDContext(ctx context.Context, targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Directory, int64, error) {
	var directories []model.Directory
	var total int64

	baseQuery := dbtx.Resolve(ctx, r.db).WithContext(ctx).Model(&model.Directory{}).Where("target_id = ?", targetID)
	baseQuery = baseQuery.Scopes(applyDirectoryListFilter(filter))

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.Scopes(
		scope.WithPagination(page, pageSize),
		func(db *gorm.DB) *gorm.DB { return applyDirectoryOrder(db, orderBy) },
	).Find(&directories).Error
	if err != nil {
		return nil, 0, err
	}

	return directoryModelListToDomain(directories), total, nil
}

func applyDirectoryListFilter(filter string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		remaining, websiteScope, err := webscope.ExtractFilterScope(filter)
		if err != nil {
			return db.Where("1 = 0")
		}
		if websiteScope != nil {
			condition, args := websiteScope.SQLPredicate("url")
			db = db.Where(condition, args...)
		}
		return scope.WithFilterDefault(remaining, directoryFilterMappingNormalized, "url")(db)
	}
}

// ListFilterOptionsByTargetID returns parent-scoped directory filter options.
// Performance is backed by target-prefixed btree indexes for the exposed scalar facets.
func (r *DirectoryRepository) ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error) {
	type optionRow struct {
		Value string
		Count int64
	}

	var rows []optionRow
	var err error
	switch field {
	case "status":
		err = r.db.Model(&model.Directory{}).
			Select("status::text AS value, COUNT(*) AS count").
			Where("target_id = ? AND status IS NOT NULL", targetID).
			Group("status").
			Order("status ASC").
			Scan(&rows).Error
	case "contentType":
		err = r.db.Model(&model.Directory{}).
			Select("content_type AS value, COUNT(*) AS count").
			Where("target_id = ? AND content_type <> ''", targetID).
			Group("content_type").
			Order("content_type ASC").
			Scan(&rows).Error
	default:
		return nil, fmt.Errorf("unsupported directory filter option field: %s", field)
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

func applyDirectoryOrder(db *gorm.DB, orderBy string) *gorm.DB {
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

// ForEachByTargetID streams directories for a target without exposing SQL cursor lifecycle to callers.
func (r *DirectoryRepository) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Directory) error) error {
	rows, err := r.db.WithContext(ctx).Model(&model.Directory{}).
		Where("target_id = ?", targetID).
		Order("created_at DESC").
		Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var directory model.Directory
		if err := r.db.ScanRows(rows, &directory); err != nil {
			return err
		}
		if err := visit(*directoryModelToDomain(&directory)); err != nil {
			return err
		}
	}
	return rows.Err()
}

// CountByTargetID returns the count of directories for a target
func (r *DirectoryRepository) CountByTargetID(targetID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.Directory{}).Where("target_id = ?", targetID).Count(&count).Error
	return count, err
}
