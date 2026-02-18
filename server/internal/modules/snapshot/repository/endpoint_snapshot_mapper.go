package repository

import (
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func endpointSnapshotModelToDomain(item *model.EndpointSnapshot) *snapshotdomain.EndpointSnapshot {
	if item == nil {
		return nil
	}
	return &snapshotdomain.EndpointSnapshot{
		ID:              item.ID,
		ScanID:          item.ScanID,
		URL:             item.URL,
		Host:            item.Host,
		Title:           item.Title,
		StatusCode:      item.StatusCode,
		ContentLength:   item.ContentLength,
		Location:        item.Location,
		Webserver:       item.Webserver,
		ContentType:     item.ContentType,
		Tech:            append([]string(nil), item.Tech...),
		ResponseBody:    item.ResponseBody,
		Vhost:           item.Vhost,
		ResponseHeaders: item.ResponseHeaders,
		CreatedAt:       timeutil.ToUTC(item.CreatedAt),
	}
}

func endpointSnapshotDomainToModel(item *snapshotdomain.EndpointSnapshot) *model.EndpointSnapshot {
	if item == nil {
		return nil
	}
	return &model.EndpointSnapshot{
		ID:              item.ID,
		ScanID:          item.ScanID,
		URL:             item.URL,
		Host:            item.Host,
		Title:           item.Title,
		StatusCode:      item.StatusCode,
		ContentLength:   item.ContentLength,
		Location:        item.Location,
		Webserver:       item.Webserver,
		ContentType:     item.ContentType,
		Tech:            append([]string(nil), item.Tech...),
		ResponseBody:    item.ResponseBody,
		Vhost:           item.Vhost,
		ResponseHeaders: item.ResponseHeaders,
		CreatedAt:       timeutil.ToUTC(item.CreatedAt),
	}
}

func endpointSnapshotDomainListToModel(items []snapshotdomain.EndpointSnapshot) []model.EndpointSnapshot {
	results := make([]model.EndpointSnapshot, 0, len(items))
	for index := range items {
		results = append(results, *endpointSnapshotDomainToModel(&items[index]))
	}
	return results
}

func endpointSnapshotModelListToDomain(items []model.EndpointSnapshot) []snapshotdomain.EndpointSnapshot {
	results := make([]snapshotdomain.EndpointSnapshot, 0, len(items))
	for index := range items {
		results = append(results, *endpointSnapshotModelToDomain(&items[index]))
	}
	return results
}
