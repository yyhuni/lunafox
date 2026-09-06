package repository

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm/clause"
)

// BatchCreate creates multiple endpoints, ignoring duplicates
func (r *EndpointRepository) BatchCreate(endpoints []assetdomain.Endpoint) (int, error) {
	if len(endpoints) == 0 {
		return 0, nil
	}

	modelEndpoints := endpointDomainListToModel(endpoints)
	var totalAffected int

	batchSize := 500
	for i := 0; i < len(modelEndpoints); i += batchSize {
		end := i + batchSize
		if end > len(modelEndpoints) {
			end = len(modelEndpoints)
		}
		batch := modelEndpoints[i:end]

		result := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&batch)
		if result.Error != nil {
			return totalAffected, result.Error
		}
		totalAffected += int(result.RowsAffected)
	}

	return totalAffected, nil
}

// Delete deletes an endpoint by ID
func (r *EndpointRepository) Delete(id int) error {
	return r.db.Delete(&model.Endpoint{}, id).Error
}

// BatchDelete deletes multiple endpoints by IDs
func (r *EndpointRepository) BatchDelete(ids []int) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.Where("id IN ?", ids).Delete(&model.Endpoint{})
	return result.RowsAffected, result.Error
}

// BatchUpsert creates or updates multiple endpoints
func (r *EndpointRepository) BatchUpsert(endpoints []assetdomain.Endpoint) (int64, error) {
	return r.BatchUpsertContext(context.Background(), endpoints)
}

func (r *EndpointRepository) BatchUpsertContext(ctx context.Context, endpoints []assetdomain.Endpoint) (int64, error) {
	if len(endpoints) == 0 {
		return 0, nil
	}

	modelEndpoints := endpointDomainListToModel(endpoints)
	var totalAffected int64

	batchSize := 100
	for i := 0; i < len(modelEndpoints); i += batchSize {
		end := i + batchSize
		if end > len(modelEndpoints) {
			end = len(modelEndpoints)
		}
		batch := modelEndpoints[i:end]

		affected, err := r.upsertBatchContext(ctx, batch)
		if err != nil {
			return totalAffected, err
		}
		totalAffected += affected
	}

	return totalAffected, nil
}

// upsertBatch upserts a single batch of endpoints
func (r *EndpointRepository) upsertBatch(endpoints []model.Endpoint) (int64, error) {
	return r.upsertBatchContext(context.Background(), endpoints)
}

func (r *EndpointRepository) upsertBatchContext(ctx context.Context, endpoints []model.Endpoint) (int64, error) {
	if len(endpoints) == 0 {
		return 0, nil
	}

	result := dbtx.Resolve(ctx, r.db).WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "url"}, {Name: "target_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"host", "location", "title", "webserver", "response_body", "content_type",
			"response_body_truncated", "status_code", "content_length", "vhost", "response_headers", "response_headers_truncated", "tech",
		}),
	}).Create(&endpoints)

	return result.RowsAffected, result.Error
}
