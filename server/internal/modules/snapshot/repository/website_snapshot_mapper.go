package repository

import (
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func websiteSnapshotModelToDomain(item *model.WebsiteSnapshot) *snapshotdomain.WebsiteSnapshot {
	if item == nil {
		return nil
	}
	return &snapshotdomain.WebsiteSnapshot{
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

func websiteSnapshotDomainToModel(item *snapshotdomain.WebsiteSnapshot) *model.WebsiteSnapshot {
	if item == nil {
		return nil
	}
	return &model.WebsiteSnapshot{
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

func websiteSnapshotDomainListToModel(items []snapshotdomain.WebsiteSnapshot) []model.WebsiteSnapshot {
	results := make([]model.WebsiteSnapshot, 0, len(items))
	for index := range items {
		results = append(results, *websiteSnapshotDomainToModel(&items[index]))
	}
	return results
}

func websiteSnapshotModelListToDomain(items []model.WebsiteSnapshot) []snapshotdomain.WebsiteSnapshot {
	results := make([]snapshotdomain.WebsiteSnapshot, 0, len(items))
	for index := range items {
		results = append(results, *websiteSnapshotModelToDomain(&items[index]))
	}
	return results
}
