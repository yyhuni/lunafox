package repository

import (
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"gorm.io/gorm/clause"
)

// BulkCreate creates multiple subdomains, ignoring duplicates
func (r *SubdomainRepository) BulkCreate(subdomains []assetdomain.Subdomain) (int, error) {
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

		result := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&batch)
		if result.Error != nil {
			return totalAffected, result.Error
		}
		totalAffected += int(result.RowsAffected)
	}

	return totalAffected, nil
}

// BulkDelete deletes multiple subdomains by IDs
func (r *SubdomainRepository) BulkDelete(ids []int) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.Where("id IN ?", ids).Delete(&model.Subdomain{})
	return result.RowsAffected, result.Error
}
