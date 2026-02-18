package snapshotwiring

import (
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	securityapp "github.com/yyhuni/lunafox/server/internal/modules/security/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	snapshotinfra "github.com/yyhuni/lunafox/server/internal/modules/snapshot/infrastructure"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

func NewSnapshotScanRefLookupAdapter(repo *scanrepo.ScanRepository) snapshotapp.SnapshotScanRefLookup {
	return newSnapshotScanRefLookupAdapter(repo)
}

func NewSnapshotWebsiteQueryStoreAdapter(repo *snapshotrepo.WebsiteSnapshotRepository) snapshotapp.WebsiteSnapshotQueryStore {
	return newSnapshotWebsiteQueryStoreAdapter(repo)
}

func NewSnapshotSubdomainQueryStoreAdapter(repo *snapshotrepo.SubdomainSnapshotRepository) snapshotapp.SubdomainSnapshotQueryStore {
	return newSnapshotSubdomainQueryStoreAdapter(repo)
}

func NewSnapshotEndpointQueryStoreAdapter(repo *snapshotrepo.EndpointSnapshotRepository) snapshotapp.EndpointSnapshotQueryStore {
	return newSnapshotEndpointQueryStoreAdapter(repo)
}

func NewSnapshotDirectoryQueryStoreAdapter(repo *snapshotrepo.DirectorySnapshotRepository) snapshotapp.DirectorySnapshotQueryStore {
	return newSnapshotDirectoryQueryStoreAdapter(repo)
}

func NewSnapshotHostPortQueryStoreAdapter(repo *snapshotrepo.HostPortSnapshotRepository) snapshotapp.HostPortSnapshotQueryStore {
	return newSnapshotHostPortQueryStoreAdapter(repo)
}

func NewSnapshotScreenshotQueryStoreAdapter(repo *snapshotrepo.ScreenshotSnapshotRepository) snapshotapp.ScreenshotSnapshotQueryStore {
	return newSnapshotScreenshotQueryStoreAdapter(repo)
}

func NewSnapshotVulnerabilityQueryStoreAdapter(repo *snapshotrepo.VulnerabilitySnapshotRepository) snapshotapp.VulnerabilitySnapshotQueryStore {
	return newSnapshotVulnerabilityQueryStoreAdapter(repo)
}

func NewSnapshotWebsiteCommandStoreAdapter(repo *snapshotrepo.WebsiteSnapshotRepository) snapshotapp.WebsiteSnapshotCommandStore {
	return newSnapshotWebsiteCommandStoreAdapter(repo)
}

func NewSnapshotSubdomainCommandStoreAdapter(repo *snapshotrepo.SubdomainSnapshotRepository) snapshotapp.SubdomainSnapshotCommandStore {
	return newSnapshotSubdomainCommandStoreAdapter(repo)
}

func NewSnapshotEndpointCommandStoreAdapter(repo *snapshotrepo.EndpointSnapshotRepository) snapshotapp.EndpointSnapshotCommandStore {
	return newSnapshotEndpointCommandStoreAdapter(repo)
}

func NewSnapshotDirectoryCommandStoreAdapter(repo *snapshotrepo.DirectorySnapshotRepository) snapshotapp.DirectorySnapshotCommandStore {
	return newSnapshotDirectoryCommandStoreAdapter(repo)
}

func NewSnapshotHostPortCommandStoreAdapter(repo *snapshotrepo.HostPortSnapshotRepository) snapshotapp.HostPortSnapshotCommandStore {
	return newSnapshotHostPortCommandStoreAdapter(repo)
}

func NewSnapshotScreenshotCommandStoreAdapter(repo *snapshotrepo.ScreenshotSnapshotRepository) snapshotapp.ScreenshotSnapshotCommandStore {
	return newSnapshotScreenshotCommandStoreAdapter(repo)
}

func NewSnapshotVulnerabilityCommandStoreAdapter(repo *snapshotrepo.VulnerabilitySnapshotRepository) snapshotapp.VulnerabilitySnapshotCommandStore {
	return newSnapshotVulnerabilityCommandStoreAdapter(repo)
}

func NewSnapshotWebsiteAssetSyncAdapter(service *assetapp.WebsiteFacade) snapshotapp.WebsiteAssetSync {
	return newSnapshotWebsiteAssetSyncAdapter(service)
}

func NewSnapshotSubdomainAssetSyncAdapter(service *assetapp.SubdomainFacade) snapshotapp.SubdomainAssetSync {
	return newSnapshotSubdomainAssetSyncAdapter(service)
}

func NewSnapshotEndpointAssetSyncAdapter(service *assetapp.EndpointFacade) snapshotapp.EndpointAssetSync {
	return newSnapshotEndpointAssetSyncAdapter(service)
}

func NewSnapshotDirectoryAssetSyncAdapter(service *assetapp.DirectoryFacade) snapshotapp.DirectoryAssetSync {
	return newSnapshotDirectoryAssetSyncAdapter(service)
}

func NewSnapshotHostPortAssetSyncAdapter(service *assetapp.HostPortFacade) snapshotapp.HostPortAssetSync {
	return newSnapshotHostPortAssetSyncAdapter(service)
}

func NewSnapshotScreenshotAssetSyncAdapter(service *assetapp.ScreenshotFacade) snapshotapp.ScreenshotAssetSync {
	return newSnapshotScreenshotAssetSyncAdapter(service)
}

func NewSnapshotVulnerabilityAssetSyncAdapter(service *securityapp.VulnerabilityFacade) snapshotapp.VulnerabilityAssetSync {
	return newSnapshotVulnerabilityAssetSyncAdapter(service)
}

func NewSnapshotVulnerabilityRawOutputCodec() snapshotapp.VulnerabilityRawOutputCodec {
	return snapshotinfra.NewVulnerabilityRawOutputCodec()
}

func NewSnapshotWebsiteApplicationService(
	queryStore snapshotapp.WebsiteSnapshotQueryStore,
	commandStore snapshotapp.WebsiteSnapshotCommandStore,
	scanLookup snapshotapp.SnapshotScanRefLookup,
	assetSync snapshotapp.WebsiteAssetSync,
) *snapshotapp.WebsiteSnapshotFacade {
	queryService := snapshotapp.NewWebsiteSnapshotQueryService(queryStore, scanLookup)
	commandService := snapshotapp.NewWebsiteSnapshotCommandService(commandStore, scanLookup, assetSync)
	return snapshotapp.NewWebsiteSnapshotFacade(queryService, commandService)
}

func NewSnapshotSubdomainApplicationService(
	queryStore snapshotapp.SubdomainSnapshotQueryStore,
	commandStore snapshotapp.SubdomainSnapshotCommandStore,
	scanLookup snapshotapp.SnapshotScanRefLookup,
	assetSync snapshotapp.SubdomainAssetSync,
) *snapshotapp.SubdomainSnapshotFacade {
	queryService := snapshotapp.NewSubdomainSnapshotQueryService(queryStore, scanLookup)
	commandService := snapshotapp.NewSubdomainSnapshotCommandService(commandStore, scanLookup, assetSync)
	return snapshotapp.NewSubdomainSnapshotFacade(queryService, commandService)
}

func NewSnapshotEndpointApplicationService(
	queryStore snapshotapp.EndpointSnapshotQueryStore,
	commandStore snapshotapp.EndpointSnapshotCommandStore,
	scanLookup snapshotapp.SnapshotScanRefLookup,
	assetSync snapshotapp.EndpointAssetSync,
) *snapshotapp.EndpointSnapshotFacade {
	queryService := snapshotapp.NewEndpointSnapshotQueryService(queryStore, scanLookup)
	commandService := snapshotapp.NewEndpointSnapshotCommandService(commandStore, scanLookup, assetSync)
	return snapshotapp.NewEndpointSnapshotFacade(queryService, commandService)
}

func NewSnapshotDirectoryApplicationService(
	queryStore snapshotapp.DirectorySnapshotQueryStore,
	commandStore snapshotapp.DirectorySnapshotCommandStore,
	scanLookup snapshotapp.SnapshotScanRefLookup,
	assetSync snapshotapp.DirectoryAssetSync,
) *snapshotapp.DirectorySnapshotFacade {
	queryService := snapshotapp.NewDirectorySnapshotQueryService(queryStore, scanLookup)
	commandService := snapshotapp.NewDirectorySnapshotCommandService(commandStore, scanLookup, assetSync)
	return snapshotapp.NewDirectorySnapshotFacade(queryService, commandService)
}

func NewSnapshotHostPortApplicationService(
	queryStore snapshotapp.HostPortSnapshotQueryStore,
	commandStore snapshotapp.HostPortSnapshotCommandStore,
	scanLookup snapshotapp.SnapshotScanRefLookup,
	assetSync snapshotapp.HostPortAssetSync,
) *snapshotapp.HostPortSnapshotFacade {
	queryService := snapshotapp.NewHostPortSnapshotQueryService(queryStore, scanLookup)
	commandService := snapshotapp.NewHostPortSnapshotCommandService(commandStore, scanLookup, assetSync)
	return snapshotapp.NewHostPortSnapshotFacade(queryService, commandService)
}

func NewSnapshotScreenshotApplicationService(
	queryStore snapshotapp.ScreenshotSnapshotQueryStore,
	commandStore snapshotapp.ScreenshotSnapshotCommandStore,
	scanLookup snapshotapp.SnapshotScanRefLookup,
	assetSync snapshotapp.ScreenshotAssetSync,
) *snapshotapp.ScreenshotSnapshotFacade {
	queryService := snapshotapp.NewScreenshotSnapshotQueryService(queryStore, scanLookup)
	commandService := snapshotapp.NewScreenshotSnapshotCommandService(commandStore, scanLookup, assetSync)
	return snapshotapp.NewScreenshotSnapshotFacade(queryService, commandService)
}

func NewSnapshotVulnerabilityApplicationService(
	queryStore snapshotapp.VulnerabilitySnapshotQueryStore,
	commandStore snapshotapp.VulnerabilitySnapshotCommandStore,
	scanLookup snapshotapp.SnapshotScanRefLookup,
	assetSync snapshotapp.VulnerabilityAssetSync,
	rawOutputCodec snapshotapp.VulnerabilityRawOutputCodec,
) *snapshotapp.VulnerabilitySnapshotFacade {
	queryService := snapshotapp.NewVulnerabilitySnapshotQueryService(queryStore, scanLookup)
	commandService := snapshotapp.NewVulnerabilitySnapshotCommandService(commandStore, scanLookup, assetSync, rawOutputCodec)
	return snapshotapp.NewVulnerabilitySnapshotFacade(queryService, commandService)
}
