package repository

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm/clause"
)

// BatchCreate creates multiple subdomains, ignoring duplicates
func (r *SubdomainRepository) BatchCreate(subdomains []assetdomain.Subdomain) (int, error) {
	return r.BatchCreateContext(context.Background(), subdomains)
}

// BatchCreateContext persists subdomain assets under the caller deadline.
func (r *SubdomainRepository) BatchCreateContext(ctx context.Context, subdomains []assetdomain.Subdomain) (int, error) {
	if len(subdomains) == 0 {
		return 0, nil
	}

	modelSubdomains := subdomainDomainListToModel(subdomains)
	var totalAffected int

	batchSize := 500
	for i := 0; i < len(modelSubdomains); i += batchSize {
		end := i + batchSize
		if end > len(modelSubdomains) {
			end = len(modelSubdomains)
		}
		batch := modelSubdomains[i:end]

		// Asset projection is part of ResultIngest's atomic Snapshot/Asset/summary
		// commit, so it must not escape to a root connection.
		result := dbtx.Resolve(ctx, r.db).WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&batch)
		if result.Error != nil {
			return totalAffected, result.Error
		}
		totalAffected += int(result.RowsAffected)
	}

	return totalAffected, nil
}

// BatchDelete deletes multiple subdomains by IDs
func (r *SubdomainRepository) BatchDelete(ids []int) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.Where("id IN ?", ids).Delete(&model.Subdomain{})
	return result.RowsAffected, result.Error
}
