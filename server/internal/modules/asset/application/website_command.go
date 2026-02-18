package application

import (
	"context"
	"errors"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

var ErrWebsiteNotFound = errors.New("website not found")

type WebsiteUpsertItem struct {
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

type WebsiteCommandService struct {
	store        WebsiteCommandStore
	targetLookup WebsiteTargetLookup
}

func NewWebsiteCommandService(store WebsiteCommandStore, targetLookup WebsiteTargetLookup) *WebsiteCommandService {
	return &WebsiteCommandService{store: store, targetLookup: targetLookup}
}

func (service *WebsiteCommandService) BulkCreate(ctx context.Context, targetID int, urls []string) (int, error) {
	_ = ctx

	target, err := service.targetLookup.GetActiveByID(targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	websites := make([]assetdomain.Website, 0, len(urls))
	for _, rawURL := range urls {
		if assetdomain.IsURLMatchTarget(rawURL, *target) {
			websites = append(websites, assetdomain.Website{
				TargetID: targetID,
				URL:      rawURL,
				Host:     assetdomain.ExtractHostFromURL(rawURL),
			})
		}
	}

	if len(websites) == 0 {
		return 0, nil
	}

	return service.store.BulkCreate(websites)
}

func (service *WebsiteCommandService) Delete(ctx context.Context, id int) error {
	_ = ctx

	if _, err := service.store.GetByID(id); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrWebsiteNotFound
		}
		return err
	}

	return service.store.Delete(id)
}

func (service *WebsiteCommandService) BulkDelete(ctx context.Context, ids []int) (int64, error) {
	_ = ctx
	if len(ids) == 0 {
		return 0, nil
	}
	return service.store.BulkDelete(ids)
}

func (service *WebsiteCommandService) BulkUpsert(ctx context.Context, targetID int, items []WebsiteUpsertItem) (int64, error) {
	_ = ctx

	target, err := service.targetLookup.GetActiveByID(targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	websites := make([]assetdomain.Website, 0, len(items))
	for _, item := range items {
		if !assetdomain.IsURLMatchTarget(item.URL, *target) {
			continue
		}

		host := item.Host
		if host == "" {
			host = assetdomain.ExtractHostFromURL(item.URL)
		}

		websites = append(websites, assetdomain.Website{
			TargetID:        targetID,
			URL:             item.URL,
			Host:            host,
			Location:        item.Location,
			Title:           item.Title,
			Webserver:       item.Webserver,
			ResponseBody:    item.ResponseBody,
			ContentType:     item.ContentType,
			Tech:            item.Tech,
			StatusCode:      item.StatusCode,
			ContentLength:   item.ContentLength,
			Vhost:           item.Vhost,
			ResponseHeaders: item.ResponseHeaders,
		})
	}

	if len(websites) == 0 {
		return 0, nil
	}

	return service.store.BulkUpsert(websites)
}

func (service *WebsiteCommandService) ResolveTarget(ctx context.Context, targetID int) (*assetdomain.TargetRef, error) {
	_ = ctx

	target, err := service.targetLookup.GetActiveByID(targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}

	return target, nil
}
