package application

import (
	"context"
	"errors"
	"fmt"

	contractresults "github.com/yyhuni/lunafox/contracts/results"
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

// WebsiteTechnologyMaterializationSummary describes the current-only
// projection without pretending that a fingerprint result is scan evidence.
type WebsiteTechnologyMaterializationSummary struct {
	ReceivedItems      int
	AssetCount         int64
	ScopeFilteredItems int
	DuplicateItems     int
}

type WebsiteTechnologyUpsertItem = assetdomain.WebsiteTechnology

type WebsiteCommandService struct {
	store        WebsiteCommandStore
	targetLookup AssetCommandTargetLookup
}

func NewWebsiteCommandService(store WebsiteCommandStore, targetLookup AssetCommandTargetLookup) *WebsiteCommandService {
	return &WebsiteCommandService{store: store, targetLookup: targetLookup}
}

func (service *WebsiteCommandService) BatchCreate(ctx context.Context, targetID int, urls []string) (int, error) {
	if ctx == nil {
		return 0, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	target, err := service.targetLookup.GetActiveByIDContext(ctx, targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	websites := make([]assetdomain.Website, 0, len(urls))
	for index, rawURL := range urls {
		host, err := observedAssetURLHostForBatchCreate(index, rawURL)
		if err != nil {
			return 0, err
		}
		if assetdomain.IsURLMatchTarget(rawURL, *target) {
			websites = append(websites, assetdomain.Website{
				TargetID: targetID,
				URL:      rawURL,
				Host:     host,
			})
		}
	}

	if len(websites) == 0 {
		return 0, nil
	}

	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return service.store.BatchCreateContext(ctx, websites)
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

func (service *WebsiteCommandService) BatchDelete(ctx context.Context, ids []int) (int64, error) {
	_ = ctx
	if len(ids) == 0 {
		return 0, nil
	}
	return service.store.BatchDelete(ids)
}

func (service *WebsiteCommandService) BatchUpsert(ctx context.Context, targetID int, items []WebsiteUpsertItem) (int64, error) {
	if ctx == nil {
		return 0, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	target, err := service.targetLookup.GetActiveByIDContext(ctx, targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	websites := make([]assetdomain.Website, 0, len(items))
	for index, item := range items {
		if _, err := contractresults.ValidateObservedAssetURL(item.URL); err != nil {
			return 0, fmt.Errorf("items[%d]: %w", index, err)
		}
		urlHost, err := contractresults.DeriveObservedAssetURLHost(item.URL)
		if err != nil {
			return 0, fmt.Errorf("items[%d]: %w", index, err)
		}
		if item.Host != urlHost {
			return 0, fmt.Errorf("items[%d]: website host conflicts with url authority", index)
		}
		if !assetdomain.IsURLMatchTarget(item.URL, *target) {
			continue
		}

		websites = append(websites, assetdomain.Website{
			TargetID:        targetID,
			URL:             item.URL,
			Host:            item.Host,
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

	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return service.store.BatchUpsertContext(ctx, websites)
}

// BatchUpsertTechnologyContext persists only current Website.tech values. The
// complete batch is already contract-validated before this method is called;
// this layer owns Target scope and last-item winner selection before one atomic
// repository statement. Technology values remain in their submitted order,
// including duplicates and the nil/explicit-empty distinction.
func (service *WebsiteCommandService) BatchUpsertTechnologyContext(ctx context.Context, targetID int, items []assetdomain.WebsiteTechnology) (WebsiteTechnologyMaterializationSummary, error) {
	summary := WebsiteTechnologyMaterializationSummary{ReceivedItems: len(items)}
	if ctx == nil {
		return summary, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return summary, err
	}
	target, err := service.targetLookup.GetActiveByIDContext(ctx, targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return summary, ErrTargetNotFound
		}
		return summary, err
	}

	// The result contract guarantees accepted raw URLs. Filtering before the
	// winner map keeps request order semantics explicit and excludes anything
	// outside the authenticated Target without creating a guessed Website.
	accepted := make([]assetdomain.WebsiteTechnology, 0, len(items))
	for _, item := range items {
		if !assetdomain.IsURLMatchTarget(item.URL, *target) {
			summary.ScopeFilteredItems++
			continue
		}
		accepted = append(accepted, item)
	}
	accepted, summary.DuplicateItems = lastWebsiteTechnologyWinners(accepted)
	for index := range accepted {
		// Host is a storage projection derived from the raw URL because the
		// technology result intentionally carries no Host field of its own.
		accepted[index].Host = assetdomain.ExtractHostFromURL(accepted[index].URL)
	}
	if len(accepted) == 0 {
		return summary, nil
	}
	if err := ctx.Err(); err != nil {
		return summary, err
	}
	summary.AssetCount, err = service.store.BatchUpsertTechnologyContext(ctx, targetID, accepted)
	return summary, err
}

func lastWebsiteTechnologyWinners(items []assetdomain.WebsiteTechnology) ([]assetdomain.WebsiteTechnology, int) {
	if len(items) < 2 {
		return items, 0
	}
	last := make(map[string]int, len(items))
	for index, item := range items {
		last[item.URL] = index
	}
	winners := make([]assetdomain.WebsiteTechnology, 0, len(last))
	for index, item := range items {
		if last[item.URL] == index {
			winners = append(winners, item)
		}
	}
	return winners, len(items) - len(winners)
}

func (service *WebsiteCommandService) ResolveTarget(ctx context.Context, targetID int) (*assetdomain.TargetRef, error) {
	if ctx == nil {
		return nil, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	target, err := service.targetLookup.GetActiveByIDContext(ctx, targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}

	return target, nil
}
