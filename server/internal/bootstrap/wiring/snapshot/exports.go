package snapshotwiring

import (
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	securityapp "github.com/yyhuni/lunafox/server/internal/modules/security/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	snapshotinfra "github.com/yyhuni/lunafox/server/internal/modules/snapshot/infrastructure"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
	"gorm.io/gorm"
)

func NewSnapshotMaterializationCoordinator(db *gorm.DB, repo *scanrepo.ScanRepository) snapshotapp.MutableObservationMaterializationCoordinator {
	return newSnapshotMaterializationCoordinator(db, repo)
}

func NewSnapshotScanRefLookupAdapter(repo *scanrepo.ScanRepository) snapshotapp.SnapshotApplicationScanRefLookup {
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
	scanLookup snapshotapp.SnapshotApplicationScanRefLookup,
	assetSync snapshotapp.WebsiteAssetSync,
) *snapshotapp.WebsiteSnapshotFacade {
	return snapshotapp.NewWebsiteSnapshotFacade(
		snapshotapp.NewWebsiteSnapshotQueryService(queryStore, scanLookup),
		snapshotapp.NewWebsiteSnapshotCommandService(commandStore, scanLookup, assetSync),
	)
}

func NewSnapshotSubdomainApplicationService(
	queryStore snapshotapp.SubdomainSnapshotQueryStore,
	commandStore snapshotapp.SubdomainSnapshotCommandStore,
	scanLookup snapshotapp.SnapshotApplicationScanRefLookup,
	assetSync snapshotapp.SubdomainAssetSync,
) *snapshotapp.SubdomainSnapshotFacade {
	return snapshotapp.NewSubdomainSnapshotFacade(
		snapshotapp.NewSubdomainSnapshotQueryService(queryStore, scanLookup),
		snapshotapp.NewSubdomainSnapshotCommandService(commandStore, scanLookup, assetSync),
	)
}

func NewSnapshotEndpointApplicationService(
	queryStore snapshotapp.EndpointSnapshotQueryStore,
	commandStore snapshotapp.EndpointSnapshotCommandStore,
	scanLookup snapshotapp.SnapshotApplicationScanRefLookup,
	assetSync snapshotapp.EndpointAssetSync,
) *snapshotapp.EndpointSnapshotFacade {
	return snapshotapp.NewEndpointSnapshotFacade(
		snapshotapp.NewEndpointSnapshotQueryService(queryStore, scanLookup),
		snapshotapp.NewEndpointSnapshotCommandService(commandStore, scanLookup, assetSync),
	)
}

func NewSnapshotDirectoryApplicationService(
	queryStore snapshotapp.DirectorySnapshotQueryStore,
	commandStore snapshotapp.DirectorySnapshotCommandStore,
	scanLookup snapshotapp.SnapshotApplicationScanRefLookup,
	assetSync snapshotapp.DirectoryAssetSync,
	coordinator snapshotapp.MutableObservationMaterializationCoordinator,
) *snapshotapp.DirectorySnapshotFacade {
	return snapshotapp.NewDirectorySnapshotFacade(
		snapshotapp.NewDirectorySnapshotQueryService(queryStore, scanLookup),
		snapshotapp.NewDirectorySnapshotCommandService(commandStore, scanLookup, assetSync, coordinator),
	)
}

func NewSnapshotHostPortApplicationService(
	queryStore snapshotapp.HostPortSnapshotQueryStore,
	commandStore snapshotapp.HostPortSnapshotCommandStore,
	scanLookup snapshotapp.SnapshotApplicationScanRefLookup,
	assetSync snapshotapp.HostPortAssetSync,
) *snapshotapp.HostPortSnapshotFacade {
	return snapshotapp.NewHostPortSnapshotFacade(
		snapshotapp.NewHostPortSnapshotQueryService(queryStore, scanLookup),
		snapshotapp.NewHostPortSnapshotCommandService(commandStore, scanLookup, assetSync),
	)
}

func NewSnapshotScreenshotApplicationService(
	queryStore snapshotapp.ScreenshotSnapshotQueryStore,
	commandStore snapshotapp.ScreenshotSnapshotCommandStore,
	scanLookup snapshotapp.SnapshotApplicationScanRefLookup,
	assetSync snapshotapp.ScreenshotAssetSync,
	coordinator snapshotapp.MutableObservationMaterializationCoordinator,
) *snapshotapp.ScreenshotSnapshotFacade {
	return snapshotapp.NewScreenshotSnapshotFacade(
		snapshotapp.NewScreenshotSnapshotQueryService(queryStore, scanLookup),
		snapshotapp.NewScreenshotSnapshotCommandService(commandStore, scanLookup, assetSync, coordinator),
	)
}

func NewSnapshotVulnerabilityApplicationService(
	queryStore snapshotapp.VulnerabilitySnapshotQueryStore,
	commandStore snapshotapp.VulnerabilitySnapshotCommandStore,
	scanLookup snapshotapp.SnapshotApplicationScanRefLookup,
	assetSync snapshotapp.VulnerabilityAssetSync,
	rawOutputCodec snapshotapp.VulnerabilityRawOutputCodec,
	coordinator snapshotapp.MutableObservationMaterializationCoordinator,
	occurrenceWriter snapshotapp.VulnerabilityNotificationOccurrenceWriter,
) *snapshotapp.VulnerabilitySnapshotFacade {
	return snapshotapp.NewVulnerabilitySnapshotFacade(
		snapshotapp.NewVulnerabilitySnapshotQueryService(queryStore, scanLookup),
		snapshotapp.NewVulnerabilitySnapshotCommandService(commandStore, scanLookup, assetSync, rawOutputCodec).
			WithMaterializationCoordinator(coordinator).
			WithNotificationOccurrenceWriter(occurrenceWriter),
	)
}
