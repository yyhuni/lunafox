package repository

import (
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func directoryModelToDomain(item *model.Directory) *assetdomain.Directory {
	if item == nil {
		return nil
	}
	return &assetdomain.Directory{
		ID:            item.ID,
		TargetID:      item.TargetID,
		URL:           item.URL,
		Status:        item.Status,
		ContentLength: item.ContentLength,
		ContentType:   item.ContentType,
		Duration:      item.Duration,
		CreatedAt:     timeutil.ToUTC(item.CreatedAt),
	}
}

func directoryDomainToModel(item *assetdomain.Directory) *model.Directory {
	if item == nil {
		return nil
	}
	return &model.Directory{
		ID:            item.ID,
		TargetID:      item.TargetID,
		URL:           item.URL,
		Status:        item.Status,
		ContentLength: item.ContentLength,
		ContentType:   item.ContentType,
		Duration:      item.Duration,
		CreatedAt:     timeutil.ToUTC(item.CreatedAt),
	}
}

func directoryDomainListToModel(items []assetdomain.Directory) []model.Directory {
	results := make([]model.Directory, 0, len(items))
	for index := range items {
		results = append(results, *directoryDomainToModel(&items[index]))
	}
	return results
}

func directoryModelListToDomain(items []model.Directory) []assetdomain.Directory {
	results := make([]assetdomain.Directory, 0, len(items))
	for index := range items {
		results = append(results, *directoryModelToDomain(&items[index]))
	}
	return results
}
