package repository

import (
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func screenshotModelToDomain(item *model.Screenshot) *assetdomain.Screenshot {
	if item == nil {
		return nil
	}
	return &assetdomain.Screenshot{
		ID:         item.ID,
		TargetID:   item.TargetID,
		URL:        item.URL,
		StatusCode: item.StatusCode,
		Image:      item.Image,
		CreatedAt:  timeutil.ToUTC(item.CreatedAt),
		UpdatedAt:  timeutil.ToUTC(item.UpdatedAt),
	}
}

func screenshotDomainToModel(item *assetdomain.Screenshot) *model.Screenshot {
	if item == nil {
		return nil
	}
	return &model.Screenshot{
		ID:         item.ID,
		TargetID:   item.TargetID,
		URL:        item.URL,
		StatusCode: item.StatusCode,
		Image:      item.Image,
		CreatedAt:  timeutil.ToUTC(item.CreatedAt),
		UpdatedAt:  timeutil.ToUTC(item.UpdatedAt),
	}
}

func screenshotDomainListToModel(items []assetdomain.Screenshot) []model.Screenshot {
	results := make([]model.Screenshot, 0, len(items))
	for index := range items {
		results = append(results, *screenshotDomainToModel(&items[index]))
	}
	return results
}

func screenshotModelListToDomain(items []model.Screenshot) []assetdomain.Screenshot {
	results := make([]assetdomain.Screenshot, 0, len(items))
	for index := range items {
		results = append(results, *screenshotModelToDomain(&items[index]))
	}
	return results
}
