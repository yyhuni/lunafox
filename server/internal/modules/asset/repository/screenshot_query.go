package repository

import (
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// FindByTargetID finds screenshots by target ID with pagination and filter
func (r *ScreenshotRepository) FindByTargetID(targetID int, page, pageSize int, filter string) ([]assetdomain.Screenshot, int64, error) {
	var screenshots []model.Screenshot
	var total int64

	baseQuery := r.db.Model(&model.Screenshot{}).Where("target_id = ?", targetID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, screenshotFilterMappingNormalized, "url"))

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.
		Scopes(
			scope.WithPagination(page, pageSize),
			scope.OrderByCreatedAtDesc(),
		).
		Find(&screenshots).Error
	if err != nil {
		return nil, 0, err
	}

	return screenshotModelListToDomain(screenshots), total, nil
}

// GetByID finds a screenshot by ID
func (r *ScreenshotRepository) GetByID(id int) (*assetdomain.Screenshot, error) {
	var screenshot model.Screenshot
	err := r.db.Where("id = ?", id).First(&screenshot).Error
	if err != nil {
		return nil, err
	}
	return screenshotModelToDomain(&screenshot), nil
}
