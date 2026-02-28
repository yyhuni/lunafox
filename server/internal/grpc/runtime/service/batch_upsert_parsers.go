package service

import (
	"encoding/json"
	"fmt"
	"strings"

	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	snapshotdto "github.com/yyhuni/lunafox/server/internal/modules/snapshot/dto"
)

func parseSubdomainItems(itemsJSON []string) ([]snapshotapp.SubdomainSnapshotItem, error) {
	decoded, err := parseJSONItems[snapshotdto.SubdomainSnapshotItem](itemsJSON)
	if err != nil {
		return nil, err
	}
	items := make([]snapshotapp.SubdomainSnapshotItem, 0, len(decoded))
	for index := range decoded {
		name := strings.TrimSpace(decoded[index].Name)
		if name == "" {
			return nil, fmt.Errorf("items_json[%d].name is required", index)
		}
		items = append(items, snapshotapp.SubdomainSnapshotItem{Name: name})
	}
	return items, nil
}

func parseWebsiteItems(itemsJSON []string) ([]snapshotapp.WebsiteSnapshotItem, error) {
	decoded, err := parseJSONItems[snapshotdto.WebsiteSnapshotItem](itemsJSON)
	if err != nil {
		return nil, err
	}
	items := make([]snapshotapp.WebsiteSnapshotItem, 0, len(decoded))
	for index := range decoded {
		item := decoded[index]
		if strings.TrimSpace(item.URL) == "" {
			return nil, fmt.Errorf("items_json[%d].url is required", index)
		}
		items = append(items, snapshotapp.WebsiteSnapshotItem{
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
	return items, nil
}

func parseEndpointItems(itemsJSON []string) ([]snapshotapp.EndpointSnapshotItem, error) {
	decoded, err := parseJSONItems[snapshotdto.EndpointSnapshotItem](itemsJSON)
	if err != nil {
		return nil, err
	}
	items := make([]snapshotapp.EndpointSnapshotItem, 0, len(decoded))
	for index := range decoded {
		item := decoded[index]
		if strings.TrimSpace(item.URL) == "" {
			return nil, fmt.Errorf("items_json[%d].url is required", index)
		}
		items = append(items, snapshotapp.EndpointSnapshotItem{
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
	return items, nil
}

func parseHostPortItems(itemsJSON []string) ([]snapshotapp.HostPortSnapshotItem, error) {
	decoded, err := parseJSONItems[snapshotdto.HostPortSnapshotItem](itemsJSON)
	if err != nil {
		return nil, err
	}
	items := make([]snapshotapp.HostPortSnapshotItem, 0, len(decoded))
	for index := range decoded {
		item := decoded[index]
		if strings.TrimSpace(item.Host) == "" {
			return nil, fmt.Errorf("items_json[%d].host is required", index)
		}
		if strings.TrimSpace(item.IP) == "" {
			return nil, fmt.Errorf("items_json[%d].ip is required", index)
		}
		if item.Port <= 0 || item.Port > 65535 {
			return nil, fmt.Errorf("items_json[%d].port must be between 1 and 65535", index)
		}
		items = append(items, snapshotapp.HostPortSnapshotItem{
			Host: item.Host,
			IP:   item.IP,
			Port: item.Port,
		})
	}
	return items, nil
}

func parseJSONItems[T any](itemsJSON []string) ([]T, error) {
	items := make([]T, 0, len(itemsJSON))
	for index := range itemsJSON {
		var item T
		if err := json.Unmarshal([]byte(itemsJSON[index]), &item); err != nil {
			return nil, fmt.Errorf("items_json[%d] is invalid JSON: %w", index, err)
		}
		items = append(items, item)
	}
	return items, nil
}
