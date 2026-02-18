package handler

import (
	service "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	"github.com/yyhuni/lunafox/server/internal/modules/snapshot/dto"
)

func toWebsiteSnapshotItemsInput(items []dto.WebsiteSnapshotItem) []service.WebsiteSnapshotItem {
	results := make([]service.WebsiteSnapshotItem, 0, len(items))
	for index := range items {
		item := items[index]
		results = append(results, service.WebsiteSnapshotItem{
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

func toEndpointSnapshotItemsInput(items []dto.EndpointSnapshotItem) []service.EndpointSnapshotItem {
	results := make([]service.EndpointSnapshotItem, 0, len(items))
	for index := range items {
		item := items[index]
		results = append(results, service.EndpointSnapshotItem{
			URL:             item.URL,
			Host:            item.Host,
			Title:           item.Title,
			StatusCode:      item.StatusCode,
			ContentLength:   item.ContentLength,
			Location:        item.Location,
			Webserver:       item.Webserver,
			ContentType:     item.ContentType,
			Tech:            item.Tech,
			ResponseBody:    item.ResponseBody,
			Vhost:           item.Vhost,
			ResponseHeaders: item.ResponseHeaders,
		})
	}
	return results
}

func toDirectorySnapshotItemsInput(items []dto.DirectorySnapshotItem) []service.DirectorySnapshotItem {
	results := make([]service.DirectorySnapshotItem, 0, len(items))
	for index := range items {
		item := items[index]
		results = append(results, service.DirectorySnapshotItem{
			URL:           item.URL,
			Status:        item.Status,
			ContentLength: item.ContentLength,
			ContentType:   item.ContentType,
			Duration:      item.Duration,
		})
	}
	return results
}

func toSubdomainSnapshotItemsInput(items []dto.SubdomainSnapshotItem) []service.SubdomainSnapshotItem {
	results := make([]service.SubdomainSnapshotItem, 0, len(items))
	for index := range items {
		results = append(results, service.SubdomainSnapshotItem{Name: items[index].Name})
	}
	return results
}

func toHostPortSnapshotItemsInput(items []dto.HostPortSnapshotItem) []service.HostPortSnapshotItem {
	results := make([]service.HostPortSnapshotItem, 0, len(items))
	for index := range items {
		item := items[index]
		results = append(results, service.HostPortSnapshotItem{
			Host: item.Host,
			IP:   item.IP,
			Port: item.Port,
		})
	}
	return results
}

func toScreenshotSnapshotItemsInput(items []dto.ScreenshotSnapshotItem) []service.ScreenshotSnapshotItem {
	results := make([]service.ScreenshotSnapshotItem, 0, len(items))
	for index := range items {
		item := items[index]
		results = append(results, service.ScreenshotSnapshotItem{
			URL:        item.URL,
			StatusCode: item.StatusCode,
			Image:      item.Image,
		})
	}
	return results
}

func toVulnerabilitySnapshotItemsInput(items []dto.VulnerabilitySnapshotItem) []service.VulnerabilitySnapshotItem {
	results := make([]service.VulnerabilitySnapshotItem, 0, len(items))
	for index := range items {
		item := items[index]
		results = append(results, service.VulnerabilitySnapshotItem{
			URL:         item.URL,
			VulnType:    item.VulnType,
			Severity:    item.Severity,
			Source:      item.Source,
			CVSSScore:   item.CVSSScore,
			Description: item.Description,
			RawOutput:   item.RawOutput,
		})
	}
	return results
}

func toSnapshotListQueryInput(page, pageSize int, filter string) *service.SnapshotListQuery {
	return &service.SnapshotListQuery{
		Page:     page,
		PageSize: pageSize,
		Filter:   filter,
	}
}

func toVulnerabilitySnapshotListQueryInput(page, pageSize int, filter, severity, ordering string) *service.VulnerabilitySnapshotListQuery {
	return &service.VulnerabilitySnapshotListQuery{
		Page:     page,
		PageSize: pageSize,
		Filter:   filter,
		Severity: severity,
		Ordering: ordering,
	}
}
