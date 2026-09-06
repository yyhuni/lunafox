package assetwiring

import (
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

var _ assetdomain.TargetLookup = (*assetTargetLookupAdapter)(nil)

var _ assetdomain.WebsiteQueryStore = (*assetWebsiteStoreAdapter)(nil)
var _ assetdomain.WebsiteCommandStore = (*assetWebsiteStoreAdapter)(nil)
var _ assetapp.GlobalWebsiteSearchStore = (*assetWebsiteStoreAdapter)(nil)

var _ assetdomain.EndpointQueryStore = (*assetEndpointStoreAdapter)(nil)
var _ assetdomain.EndpointCommandStore = (*assetEndpointStoreAdapter)(nil)
var _ assetapp.GlobalEndpointSearchStore = (*assetEndpointStoreAdapter)(nil)

var _ assetdomain.DirectoryQueryStore = (*assetDirectoryStoreAdapter)(nil)
var _ assetdomain.DirectoryCommandStore = (*assetDirectoryStoreAdapter)(nil)

var _ assetdomain.SubdomainQueryStore = (*assetSubdomainStoreAdapter)(nil)
var _ assetdomain.SubdomainCommandStore = (*assetSubdomainStoreAdapter)(nil)

var _ assetdomain.HostPortQueryStore = (*assetHostPortStoreAdapter)(nil)
var _ assetdomain.HostPortCommandStore = (*assetHostPortStoreAdapter)(nil)

var _ assetdomain.ScreenshotQueryStore = (*assetScreenshotStoreAdapter)(nil)
var _ assetdomain.ScreenshotCommandStore = (*assetScreenshotStoreAdapter)(nil)
var _ assetapp.WebsiteScreenshotStore = (*assetScreenshotStoreAdapter)(nil)
var _ assetapp.AssetStatisticsStore = (*assetStatisticsStoreAdapter)(nil)
