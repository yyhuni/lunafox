package assetwiring

import (
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
)

func NewAssetTargetLookupAdapter(repo *catalogrepo.TargetRepository) assetapp.AssetTargetLookup {
	return newAssetTargetLookupAdapter(repo)
}

func NewAssetWebsiteStoreAdapter(repo *assetrepo.WebsiteRepository) assetapp.WebsiteStore {
	return newAssetWebsiteStoreAdapter(repo)
}

func NewAssetSubdomainStoreAdapter(repo *assetrepo.SubdomainRepository) assetapp.SubdomainStore {
	return newAssetSubdomainStoreAdapter(repo)
}

func NewAssetEndpointStoreAdapter(repo *assetrepo.EndpointRepository) assetapp.EndpointStore {
	return newAssetEndpointStoreAdapter(repo)
}

func NewAssetDirectoryStoreAdapter(repo *assetrepo.DirectoryRepository) assetapp.DirectoryStore {
	return newAssetDirectoryStoreAdapter(repo)
}

func NewAssetHostPortStoreAdapter(repo *assetrepo.HostPortRepository) assetapp.HostPortStore {
	return newAssetHostPortStoreAdapter(repo)
}

func NewAssetScreenshotStoreAdapter(repo *assetrepo.ScreenshotRepository) assetapp.ScreenshotStore {
	return newAssetScreenshotStoreAdapter(repo)
}
