package repository

import (
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func endpointModelToDomain(item *model.Endpoint) *assetdomain.Endpoint {
	if item == nil {
		return nil
	}
	return &assetdomain.Endpoint{
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

func endpointDomainToModel(item *assetdomain.Endpoint) *model.Endpoint {
	if item == nil {
		return nil
	}
	return &model.Endpoint{
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

func endpointDomainListToModel(items []assetdomain.Endpoint) []model.Endpoint {
	results := make([]model.Endpoint, 0, len(items))
	for index := range items {
		results = append(results, *endpointDomainToModel(&items[index]))
	}
	return results
}

func endpointModelListToDomain(items []model.Endpoint) []assetdomain.Endpoint {
	results := make([]assetdomain.Endpoint, 0, len(items))
	for index := range items {
		results = append(results, *endpointModelToDomain(&items[index]))
	}
	return results
}
