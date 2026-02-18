package repository

import (
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BulkCreate creates multiple websites, ignoring duplicates
func (r *WebsiteRepository) BulkCreate(websites []assetdomain.Website) (int, error) {
	if len(websites) == 0 {
		return 0, nil
	}

	modelWebsites := websiteDomainListToModel(websites)
	var totalAffected int

	batchSize := 500
	for i := 0; i < len(modelWebsites); i += batchSize {
		end := i + batchSize
		if end > len(modelWebsites) {
			end = len(modelWebsites)
		}
		batch := modelWebsites[i:end]

		result := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&batch)
		if result.Error != nil {
			return totalAffected, result.Error
		}
		totalAffected += int(result.RowsAffected)
	}

	return totalAffected, nil
}

// Delete deletes a website by ID
func (r *WebsiteRepository) Delete(id int) error {
	return r.db.Delete(&model.Website{}, id).Error
}

// BulkDelete deletes multiple websites by IDs
func (r *WebsiteRepository) BulkDelete(ids []int) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.Where("id IN ?", ids).Delete(&model.Website{})
	return result.RowsAffected, result.Error
}

// BulkUpsert creates or updates multiple websites
func (r *WebsiteRepository) BulkUpsert(websites []assetdomain.Website) (int64, error) {
	if len(websites) == 0 {
		return 0, nil
	}

	modelWebsites := websiteDomainListToModel(websites)
	var totalAffected int64

	batchSize := 100
	for i := 0; i < len(modelWebsites); i += batchSize {
		end := i + batchSize
		if end > len(modelWebsites) {
			end = len(modelWebsites)
		}
		batch := modelWebsites[i:end]

		affected, err := r.upsertBatch(batch)
		if err != nil {
			return totalAffected, err
		}
		totalAffected += affected
	}

	return totalAffected, nil
}

// upsertBatch upserts a single batch of websites
func (r *WebsiteRepository) upsertBatch(websites []model.Website) (int64, error) {
	if len(websites) == 0 {
		return 0, nil
	}

	result := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "url"}, {Name: "target_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"host":             gorm.Expr("COALESCE(NULLIF(EXCLUDED.host, ''), website.host)"),
			"location":         gorm.Expr("COALESCE(NULLIF(EXCLUDED.location, ''), website.location)"),
			"title":            gorm.Expr("COALESCE(NULLIF(EXCLUDED.title, ''), website.title)"),
			"webserver":        gorm.Expr("COALESCE(NULLIF(EXCLUDED.webserver, ''), website.webserver)"),
			"response_body":    gorm.Expr("COALESCE(NULLIF(EXCLUDED.response_body, ''), website.response_body)"),
			"content_type":     gorm.Expr("COALESCE(NULLIF(EXCLUDED.content_type, ''), website.content_type)"),
			"status_code":      gorm.Expr("COALESCE(EXCLUDED.status_code, website.status_code)"),
			"content_length":   gorm.Expr("COALESCE(EXCLUDED.content_length, website.content_length)"),
			"vhost":            gorm.Expr("COALESCE(EXCLUDED.vhost, website.vhost)"),
			"response_headers": gorm.Expr("COALESCE(NULLIF(EXCLUDED.response_headers, ''), website.response_headers)"),
			"tech": gorm.Expr(`(
				SELECT ARRAY(SELECT DISTINCT unnest FROM unnest(
					COALESCE(website.tech, ARRAY[]::varchar(100)[]) ||
					COALESCE(EXCLUDED.tech, ARRAY[]::varchar(100)[])
				) ORDER BY unnest)
			)`),
		}),
	}).Create(&websites)

	return result.RowsAffected, result.Error
}
