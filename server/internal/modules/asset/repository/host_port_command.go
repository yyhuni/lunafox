package repository

import (
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"gorm.io/gorm/clause"
)

// BulkUpsert creates multiple mappings, ignoring duplicates (ON CONFLICT DO NOTHING)
func (r *HostPortRepository) BulkUpsert(mappings []assetdomain.HostPort) (int64, error) {
	if len(mappings) == 0 {
		return 0, nil
	}

	modelMappings := hostPortDomainListToModel(mappings)
	var totalAffected int64

	batchSize := 100
	for i := 0; i < len(modelMappings); i += batchSize {
		end := min(i+batchSize, len(modelMappings))
		batch := modelMappings[i:end]

		result := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&batch)
		if result.Error != nil {
			return totalAffected, result.Error
		}
		totalAffected += result.RowsAffected
	}

	return totalAffected, nil
}

// DeleteByIPs deletes all mappings for the given IPs
func (r *HostPortRepository) DeleteByIPs(ips []string) (int64, error) {
	if len(ips) == 0 {
		return 0, nil
	}
	result := r.db.Where("ip IN ?", ips).Delete(&model.HostPort{})
	return result.RowsAffected, result.Error
}
