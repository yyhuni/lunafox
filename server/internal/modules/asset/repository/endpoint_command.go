package repository

import (
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BulkCreate creates multiple endpoints, ignoring duplicates
func (r *EndpointRepository) BulkCreate(endpoints []assetdomain.Endpoint) (int, error) {
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

// BulkDelete deletes multiple endpoints by IDs
func (r *EndpointRepository) BulkDelete(ids []int) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.Where("id IN ?", ids).Delete(&model.Endpoint{})
	return result.RowsAffected, result.Error
}

// BulkUpsert creates or updates multiple endpoints
func (r *EndpointRepository) BulkUpsert(endpoints []assetdomain.Endpoint) (int64, error) {
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

		affected, err := r.upsertBatch(batch)
		if err != nil {
			return totalAffected, err
		}
		totalAffected += affected
	}

	return totalAffected, nil
}

// upsertBatch upserts a single batch of endpoints
func (r *EndpointRepository) upsertBatch(endpoints []model.Endpoint) (int64, error) {
	if len(endpoints) == 0 {
		return 0, nil
	}

	result := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "url"}, {Name: "target_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"host":             gorm.Expr("COALESCE(NULLIF(EXCLUDED.host, ''), endpoint.host)"),
			"location":         gorm.Expr("COALESCE(NULLIF(EXCLUDED.location, ''), endpoint.location)"),
			"title":            gorm.Expr("COALESCE(NULLIF(EXCLUDED.title, ''), endpoint.title)"),
			"webserver":        gorm.Expr("COALESCE(NULLIF(EXCLUDED.webserver, ''), endpoint.webserver)"),
			"response_body":    gorm.Expr("COALESCE(NULLIF(EXCLUDED.response_body, ''), endpoint.response_body)"),
			"content_type":     gorm.Expr("COALESCE(NULLIF(EXCLUDED.content_type, ''), endpoint.content_type)"),
			"status_code":      gorm.Expr("COALESCE(EXCLUDED.status_code, endpoint.status_code)"),
			"content_length":   gorm.Expr("COALESCE(EXCLUDED.content_length, endpoint.content_length)"),
			"vhost":            gorm.Expr("COALESCE(EXCLUDED.vhost, endpoint.vhost)"),
			"response_headers": gorm.Expr("COALESCE(NULLIF(EXCLUDED.response_headers, ''), endpoint.response_headers)"),
			"tech": gorm.Expr(`(
				SELECT ARRAY(SELECT DISTINCT unnest FROM unnest(
					COALESCE(endpoint.tech, ARRAY[]::varchar(100)[]) ||
					COALESCE(EXCLUDED.tech, ARRAY[]::varchar(100)[])
				) ORDER BY unnest)
			)`),
		}),
	}).Create(&endpoints)

	return result.RowsAffected, result.Error
}
