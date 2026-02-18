package snapshotwiring

import (
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
)

func snapshotWebsiteAssetUpsertItemsToApplication(items []snapshotapp.WebsiteAssetUpsertItem) []assetapp.WebsiteUpsertItem {
	results := make([]assetapp.WebsiteUpsertItem, 0, len(items))
	for index := range items {
		item := items[index]
		results = append(results, assetapp.WebsiteUpsertItem{
			URL:             item.URL,
			Host:            item.Host,
			Location:        item.Location,
			Title:           item.Title,
			Webserver:       item.Webserver,
			ContentType:     item.ContentType,
			StatusCode:      item.StatusCode,
			ContentLength:   item.ContentLength,
			ResponseBody:    item.ResponseBody,
			Tech:            item.Tech,
			Vhost:           item.Vhost,
			ResponseHeaders: item.ResponseHeaders,
		})
	}
	return results
}

func snapshotEndpointAssetUpsertItemsToApplication(items []snapshotapp.EndpointAssetUpsertItem) []assetapp.EndpointUpsertItem {
	results := make([]assetapp.EndpointUpsertItem, 0, len(items))
	for index := range items {
		item := items[index]
		results = append(results, assetapp.EndpointUpsertItem{
			URL:             item.URL,
			Host:            item.Host,
			Location:        item.Location,
			Title:           item.Title,
			Webserver:       item.Webserver,
			ContentType:     item.ContentType,
			StatusCode:      item.StatusCode,
			ContentLength:   item.ContentLength,
			ResponseBody:    item.ResponseBody,
			Tech:            item.Tech,
			Vhost:           item.Vhost,
			ResponseHeaders: item.ResponseHeaders,
		})
	}
	return results
}

func snapshotDirectoryAssetUpsertItemsToApplication(items []snapshotapp.DirectoryAssetUpsertItem) []assetapp.DirectoryUpsertItem {
	results := make([]assetapp.DirectoryUpsertItem, 0, len(items))
	for index := range items {
		item := items[index]
		results = append(results, assetapp.DirectoryUpsertItem{
			URL:           item.URL,
			Status:        item.Status,
			ContentLength: item.ContentLength,
			ContentType:   item.ContentType,
			Duration:      item.Duration,
		})
	}
	return results
}

func snapshotHostPortAssetItemsToApplication(items []snapshotapp.HostPortAssetItem) []assetapp.HostPortItem {
	results := make([]assetapp.HostPortItem, 0, len(items))
	for index := range items {
		item := items[index]
		results = append(results, assetapp.HostPortItem{Host: item.Host, IP: item.IP, Port: item.Port})
	}
	return results
}

func snapshotScreenshotAssetRequestToApplication(req *snapshotapp.ScreenshotAssetUpsertRequest) *assetapp.BulkUpsertScreenshotRequest {
	if req == nil {
		return nil
	}
	items := make([]assetapp.ScreenshotItem, 0, len(req.Screenshots))
	for index := range req.Screenshots {
		item := req.Screenshots[index]
		items = append(items, assetapp.ScreenshotItem{URL: item.URL, StatusCode: item.StatusCode, Image: item.Image})
	}
	return &assetapp.BulkUpsertScreenshotRequest{Screenshots: items}
}
