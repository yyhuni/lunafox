package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type assetQueryTargetLookupContext interface {
	GetActiveByIDContext(context.Context, int) (*assetdomain.TargetRef, error)
}

type websiteListQueryStoreContext interface {
	ListByTargetIDContext(context.Context, int, int, int, string, string) ([]assetdomain.Website, int64, error)
}

type websiteScreenshotQueryStoreContext interface {
	ListSummariesByTargetAndURLsContext(context.Context, int, []string) ([]assetdomain.Screenshot, error)
}

type subdomainListQueryStoreContext interface {
	ListByTargetIDContext(context.Context, int, int, int, string, string) ([]assetdomain.Subdomain, int64, error)
}

type endpointListQueryStoreContext interface {
	ListByTargetIDContext(context.Context, int, int, int, string, string) ([]assetdomain.Endpoint, int64, error)
}

type directoryListQueryStoreContext interface {
	ListByTargetIDContext(context.Context, int, int, int, string, string) ([]assetdomain.Directory, int64, error)
}

type hostPortQueryStoreContext interface {
	GetIPAggregationContext(context.Context, int, int, int, string, string) ([]assetdomain.IPAggregationRow, int64, error)
	GetHostsAndPortsByIPContext(context.Context, int, string, string) ([]string, []int, error)
}

func getAssetTargetForQuery(ctx context.Context, lookup AssetTargetLookup, targetID int) (*assetdomain.TargetRef, error) {
	if contextual, ok := lookup.(assetQueryTargetLookupContext); ok {
		return contextual.GetActiveByIDContext(ctx, targetID)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return lookup.GetActiveByID(targetID)
}

func listWebsitesForQuery(ctx context.Context, store WebsiteQueryStore, targetID, page, pageSize int, filter, orderBy string) ([]assetdomain.Website, int64, error) {
	if contextual, ok := store.(websiteListQueryStoreContext); ok {
		return contextual.ListByTargetIDContext(ctx, targetID, page, pageSize, filter, orderBy)
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	return store.ListByTargetID(targetID, page, pageSize, filter, orderBy)
}

func listWebsiteScreenshotSummaries(ctx context.Context, store WebsiteScreenshotQueryStore, targetID int, urls []string) ([]assetdomain.Screenshot, error) {
	if contextual, ok := store.(websiteScreenshotQueryStoreContext); ok {
		return contextual.ListSummariesByTargetAndURLsContext(ctx, targetID, urls)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return store.ListSummariesByTargetAndURLs(targetID, urls)
}

func listSubdomainsForQuery(ctx context.Context, store SubdomainQueryStore, targetID, page, pageSize int, filter, orderBy string) ([]assetdomain.Subdomain, int64, error) {
	if contextual, ok := store.(subdomainListQueryStoreContext); ok {
		return contextual.ListByTargetIDContext(ctx, targetID, page, pageSize, filter, orderBy)
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	return store.ListByTargetID(targetID, page, pageSize, filter, orderBy)
}

func listEndpointsForQuery(ctx context.Context, store EndpointQueryStore, targetID, page, pageSize int, filter, orderBy string) ([]assetdomain.Endpoint, int64, error) {
	if contextual, ok := store.(endpointListQueryStoreContext); ok {
		return contextual.ListByTargetIDContext(ctx, targetID, page, pageSize, filter, orderBy)
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	return store.ListByTargetID(targetID, page, pageSize, filter, orderBy)
}

func listDirectoriesForQuery(ctx context.Context, store DirectoryQueryStore, targetID, page, pageSize int, filter, orderBy string) ([]assetdomain.Directory, int64, error) {
	if contextual, ok := store.(directoryListQueryStoreContext); ok {
		return contextual.ListByTargetIDContext(ctx, targetID, page, pageSize, filter, orderBy)
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	return store.ListByTargetID(targetID, page, pageSize, filter, orderBy)
}

func listHostPortIPsForQuery(ctx context.Context, store HostPortQueryStore, targetID, page, pageSize int, filter, orderBy string) ([]assetdomain.IPAggregationRow, int64, error) {
	if contextual, ok := store.(hostPortQueryStoreContext); ok {
		return contextual.GetIPAggregationContext(ctx, targetID, page, pageSize, filter, orderBy)
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	return store.GetIPAggregation(targetID, page, pageSize, filter, orderBy)
}

func listHostPortDetailsForQuery(ctx context.Context, store HostPortQueryStore, targetID int, ip, filter string) ([]string, []int, error) {
	if contextual, ok := store.(hostPortQueryStoreContext); ok {
		return contextual.GetHostsAndPortsByIPContext(ctx, targetID, ip, filter)
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	return store.GetHostsAndPortsByIP(targetID, ip, filter)
}
