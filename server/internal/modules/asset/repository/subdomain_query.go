package repository

import (
	"context"
	"strings"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// FindByTargetID finds subdomains by target ID with pagination and filter
func (r *SubdomainRepository) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Subdomain, int64, error) {
	return r.ListByTargetIDContext(context.Background(), targetID, page, pageSize, filter, orderBy)
}

// ListByTargetIDContext preserves a caller-owned cancellation/deadline through subdomain list queries.
func (r *SubdomainRepository) ListByTargetIDContext(ctx context.Context, targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Subdomain, int64, error) {
	var subdomains []model.Subdomain
	var total int64

	baseQuery := dbtx.Resolve(ctx, r.db).WithContext(ctx).Model(&model.Subdomain{}).Where("target_id = ?", targetID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, subdomainFilterMappingNormalized, "dnsName"))

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.Scopes(
		scope.WithPagination(page, pageSize),
		func(db *gorm.DB) *gorm.DB { return applySubdomainOrder(db, orderBy) },
	).Find(&subdomains).Error
	if err != nil {
		return nil, 0, err
	}

	return subdomainModelListToDomain(subdomains), total, nil
}

func applySubdomainOrder(db *gorm.DB, orderBy string) *gorm.DB {
	switch strings.TrimSpace(orderBy) {
	case "dnsName", "dnsName asc":
		return db.Order("dns_name ASC").Order("id ASC")
	case "dnsName desc":
		return db.Order("dns_name DESC").Order("id DESC")
	case "createdAt", "createdAt asc":
		return db.Order("created_at ASC").Order("id ASC")
	case "createdAt desc", "":
		return db.Order("created_at DESC").Order("id DESC")
	default:
		return db.Where("1 = 0")
	}
}

// ForEachByTargetID streams subdomains for a target without exposing SQL cursor lifecycle to callers.
func (r *SubdomainRepository) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Subdomain) error) error {
	rows, err := r.db.WithContext(ctx).Model(&model.Subdomain{}).
		Where("target_id = ?", targetID).
		Order("created_at DESC").
		Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var subdomain model.Subdomain
		if err := r.db.ScanRows(rows, &subdomain); err != nil {
			return err
		}
		if err := visit(*subdomainModelToDomain(&subdomain)); err != nil {
			return err
		}
	}
	return rows.Err()
}

// CountByTargetID returns the count of subdomains for a target
func (r *SubdomainRepository) CountByTargetID(targetID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.Subdomain{}).Where("target_id = ?", targetID).Count(&count).Error
	return count, err
}
