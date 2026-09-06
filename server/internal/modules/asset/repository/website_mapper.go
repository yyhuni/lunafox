package repository

import (
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func websiteModelToDomain(item *model.Website) *assetdomain.Website {
	if item == nil {
		return nil
	}
	return &assetdomain.Website{
		ID:              item.ID,
		TargetID:        item.TargetID,
		URL:             item.URL,
		Host:            item.Host,
		Location:        item.Location,
		CreatedAt:       timeutil.ToUTC(item.CreatedAt),
		Title:           item.Title,
		Webserver:       item.Webserver,
		ResponseBody:    item.ResponseBody,
		ContentType:     item.ContentType,
		Tech:            item.Tech,
		StatusCode:      item.StatusCode,
		ContentLength:   item.ContentLength,
		Vhost:           item.Vhost,
		ResponseHeaders: item.ResponseHeaders,
	}
}

func websiteDomainToModel(item *assetdomain.Website) *model.Website {
	if item == nil {
		return nil
	}
	return &model.Website{
		ID:              item.ID,
		TargetID:        item.TargetID,
		URL:             item.URL,
		Host:            item.Host,
		Location:        item.Location,
		CreatedAt:       timeutil.ToUTC(item.CreatedAt),
		Title:           item.Title,
		Webserver:       item.Webserver,
		ResponseBody:    item.ResponseBody,
		ContentType:     item.ContentType,
		Tech:            item.Tech,
		StatusCode:      item.StatusCode,
		ContentLength:   item.ContentLength,
		Vhost:           item.Vhost,
		ResponseHeaders: item.ResponseHeaders,
	}
}

func websiteDomainListToModel(items []assetdomain.Website) []model.Website {
	results := make([]model.Website, 0, len(items))
	for index := range items {
		results = append(results, *websiteDomainToModel(&items[index]))
	}
	return results
}

func websiteModelListToDomain(items []model.Website) []assetdomain.Website {
	results := make([]assetdomain.Website, 0, len(items))
	for index := range items {
		results = append(results, *websiteModelToDomain(&items[index]))
	}
	return results
}
