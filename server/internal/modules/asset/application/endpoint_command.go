package application

import (
	"context"
	"errors"
	"fmt"

	contractresults "github.com/yyhuni/lunafox/contracts/results"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

var ErrEndpointNotFound = errors.New("endpoint not found")

type EndpointUpsertItem struct {
	URL                      string
	Host                     string
	Location                 string
	Title                    string
	Webserver                string
	ContentType              string
	StatusCode               *int
	ContentLength            *int
	ResponseBody             string
	ResponseBodyTruncated    bool
	Tech                     []string
	Vhost                    *bool
	ResponseHeaders          string
	ResponseHeadersTruncated bool
}

type EndpointCommandService struct {
	store        EndpointCommandStore
	targetLookup EndpointTargetLookup
}

func NewEndpointCommandService(store EndpointCommandStore, targetLookup EndpointTargetLookup) *EndpointCommandService {
	return &EndpointCommandService{store: store, targetLookup: targetLookup}
}

func (service *EndpointCommandService) BatchCreate(ctx context.Context, targetID int, urls []string) (int, error) {
	_ = ctx

	target, err := service.targetLookup.GetActiveByID(targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	endpoints := make([]assetdomain.Endpoint, 0, len(urls))
	for index, rawURL := range urls {
		host, err := observedAssetURLHostForBatchCreate(index, rawURL)
		if err != nil {
			return 0, err
		}
		if assetdomain.IsURLMatchTarget(rawURL, *target) {
			endpoints = append(endpoints, assetdomain.Endpoint{
				TargetID: targetID,
				URL:      rawURL,
				Host:     host,
			})
		}
	}

	if len(endpoints) == 0 {
		return 0, nil
	}

	return service.store.BatchCreate(endpoints)
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

func (service *EndpointCommandService) BatchDelete(ctx context.Context, ids []int) (int64, error) {
	_ = ctx

	if len(ids) == 0 {
		return 0, nil
	}

	return service.store.BatchDelete(ids)
}

func (service *EndpointCommandService) BatchUpsert(ctx context.Context, targetID int, items []EndpointUpsertItem) (int64, error) {
	_ = ctx

	target, err := service.targetLookup.GetActiveByID(targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	endpoints := make([]assetdomain.Endpoint, 0, len(items))
	for index, item := range items {
		if _, err := contractresults.ValidateObservedAssetURL(item.URL); err != nil {
			return 0, fmt.Errorf("items[%d]: %w", index, err)
		}
		urlHost, err := contractresults.DeriveObservedAssetURLHost(item.URL)
		if err != nil {
			return 0, fmt.Errorf("items[%d]: %w", index, err)
		}
		if item.Host != urlHost {
			return 0, fmt.Errorf("items[%d]: endpoint host conflicts with url authority", index)
		}
		if !assetdomain.IsURLMatchTarget(item.URL, *target) {
			continue
		}

		endpoints = append(endpoints, assetdomain.Endpoint{
			TargetID:                 targetID,
			URL:                      item.URL,
			Host:                     item.Host,
			Location:                 item.Location,
			Title:                    item.Title,
			Webserver:                item.Webserver,
			ContentType:              item.ContentType,
			StatusCode:               item.StatusCode,
			ContentLength:            item.ContentLength,
			ResponseBody:             item.ResponseBody,
			ResponseBodyTruncated:    item.ResponseBodyTruncated,
			Tech:                     item.Tech,
			Vhost:                    item.Vhost,
			ResponseHeaders:          item.ResponseHeaders,
			ResponseHeadersTruncated: item.ResponseHeadersTruncated,
		})
	}

	if len(endpoints) == 0 {
		return 0, nil
	}

	if contextual, ok := service.store.(interface {
		BatchUpsertContext(context.Context, []assetdomain.Endpoint) (int64, error)
	}); ok {
		return contextual.BatchUpsertContext(ctx, endpoints)
	}
	return service.store.BatchUpsert(endpoints)
}
