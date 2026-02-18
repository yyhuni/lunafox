package application

import (
	"context"
	"errors"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

var ErrEndpointNotFound = errors.New("endpoint not found")

type EndpointUpsertItem struct {
	URL             string
	Host            string
	Location        string
	Title           string
	Webserver       string
	ContentType     string
	StatusCode      *int
	ContentLength   *int
	ResponseBody    string
	Tech            []string
	Vhost           *bool
	ResponseHeaders string
}

type EndpointCommandService struct {
	store        EndpointCommandStore
	targetLookup EndpointTargetLookup
}

func NewEndpointCommandService(store EndpointCommandStore, targetLookup EndpointTargetLookup) *EndpointCommandService {
	return &EndpointCommandService{store: store, targetLookup: targetLookup}
}

func (service *EndpointCommandService) BulkCreate(ctx context.Context, targetID int, urls []string) (int, error) {
	_ = ctx

	target, err := service.targetLookup.GetActiveByID(targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	endpoints := make([]assetdomain.Endpoint, 0, len(urls))
	for _, rawURL := range urls {
		if assetdomain.IsURLMatchTarget(rawURL, *target) {
			endpoints = append(endpoints, assetdomain.Endpoint{
				TargetID: targetID,
				URL:      rawURL,
				Host:     assetdomain.ExtractHostFromURL(rawURL),
			})
		}
	}

	if len(endpoints) == 0 {
		return 0, nil
	}

	return service.store.BulkCreate(endpoints)
}

func (service *EndpointCommandService) Delete(ctx context.Context, id int) error {
	_ = ctx

	if _, err := service.store.GetByID(id); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrEndpointNotFound
		}
		return err
	}

	return service.store.Delete(id)
}

func (service *EndpointCommandService) BulkDelete(ctx context.Context, ids []int) (int64, error) {
	_ = ctx

	if len(ids) == 0 {
		return 0, nil
	}

	return service.store.BulkDelete(ids)
}

func (service *EndpointCommandService) BulkUpsert(ctx context.Context, targetID int, items []EndpointUpsertItem) (int64, error) {
	_ = ctx

	target, err := service.targetLookup.GetActiveByID(targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	endpoints := make([]assetdomain.Endpoint, 0, len(items))
	for _, item := range items {
		if !assetdomain.IsURLMatchTarget(item.URL, *target) {
			continue
		}

		host := item.Host
		if host == "" {
			host = assetdomain.ExtractHostFromURL(item.URL)
		}

		endpoints = append(endpoints, assetdomain.Endpoint{
			TargetID:        targetID,
			URL:             item.URL,
			Host:            host,
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

	if len(endpoints) == 0 {
		return 0, nil
	}

	return service.store.BulkUpsert(endpoints)
}
