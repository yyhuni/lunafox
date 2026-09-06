package repository

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm/clause"
)

// BatchUpsert creates multiple mappings, ignoring duplicates (ON CONFLICT DO NOTHING)
func (r *HostPortRepository) BatchUpsert(mappings []assetdomain.HostPort) (int64, error) {
	return r.BatchUpsertContext(context.Background(), mappings)
}

// BatchUpsertContext persists host-port assets under the caller deadline.
func (r *HostPortRepository) BatchUpsertContext(ctx context.Context, mappings []assetdomain.HostPort) (int64, error) {
	if len(mappings) == 0 {
		return 0, nil
	}

	modelMappings := hostPortDomainListToModel(mappings)
	var totalAffected int64

	batchSize := 100
	for i := 0; i < len(modelMappings); i += batchSize {
		end := min(i+batchSize, len(modelMappings))
		batch := modelMappings[i:end]

		// Host-port writes share ResultIngest's Scan/Target locks; resolving the
		// caller transaction prevents an intra-request lock wait on root DB.
		result := dbtx.Resolve(ctx, r.db).WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&batch)
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
