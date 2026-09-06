package repository

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm/clause"
)

// BatchCreate creates multiple websites, ignoring duplicates
func (r *WebsiteRepository) BatchCreate(websites []assetdomain.Website) (int, error) {
	return r.BatchCreateContext(context.Background(), websites)
}

// BatchCreateContext persists website assets under the caller deadline.
func (r *WebsiteRepository) BatchCreateContext(ctx context.Context, websites []assetdomain.Website) (int, error) {
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

		result := dbtx.Resolve(ctx, r.db).WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&batch)
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

// BatchDelete deletes multiple websites by IDs
func (r *WebsiteRepository) BatchDelete(ids []int) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.Where("id IN ?", ids).Delete(&model.Website{})
	return result.RowsAffected, result.Error
}

// BatchUpsert creates or updates multiple websites
func (r *WebsiteRepository) BatchUpsert(websites []assetdomain.Website) (int64, error) {
	return r.BatchUpsertContext(context.Background(), websites)
}

// BatchUpsertContext persists website asset projections under the caller deadline.
func (r *WebsiteRepository) BatchUpsertContext(ctx context.Context, websites []assetdomain.Website) (int64, error) {
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

		affected, err := r.upsertBatchContext(ctx, batch)
		if err != nil {
			return totalAffected, err
		}
		totalAffected += affected
	}

	return totalAffected, nil
}

// BatchUpsertTechnologyContext creates missing Website rows or replaces only
// tech on an existing (target_id, url) row. The conflict update list is
// intentionally limited to tech so a fingerprint-only result cannot erase
// complete Website observations.
func (r *WebsiteRepository) BatchUpsertTechnologyContext(ctx context.Context, targetID int, websites []assetdomain.WebsiteTechnology) (int64, error) {
	if len(websites) == 0 {
		return 0, nil
	}
	models := make([]model.Website, 0, len(websites))
	for _, website := range websites {
		tech := append([]string(nil), website.Tech...)
		if tech == nil {
			tech = []string{}
		}
		models = append(models, model.Website{
			TargetID: targetID,
			URL:      website.URL,
			Host:     website.Host,
			Tech:     tech,
		})
	}
	result := dbtx.Resolve(ctx, r.db).WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "url"}, {Name: "target_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"tech"}),
	}).Create(&models)
	return result.RowsAffected, result.Error
}

// upsertBatch upserts a single batch of websites
func (r *WebsiteRepository) upsertBatch(websites []model.Website) (int64, error) {
	return r.upsertBatchContext(context.Background(), websites)
}

func (r *WebsiteRepository) upsertBatchContext(ctx context.Context, websites []model.Website) (int64, error) {
	if len(websites) == 0 {
		return 0, nil
	}

	result := dbtx.Resolve(ctx, r.db).WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "url"}, {Name: "target_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"host", "location", "title", "webserver", "response_body", "content_type",
			"status_code", "content_length", "vhost", "response_headers", "tech",
		}),
	}).Create(&websites)

	return result.RowsAffected, result.Error
}
