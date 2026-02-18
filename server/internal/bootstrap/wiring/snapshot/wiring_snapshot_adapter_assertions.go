package snapshotwiring

import snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"

var _ snapshotapp.SnapshotScanRefLookup = (*snapshotScanRefLookupAdapter)(nil)

var _ snapshotapp.WebsiteSnapshotQueryStore = (*snapshotWebsiteQueryStoreAdapter)(nil)
var _ snapshotapp.WebsiteSnapshotCommandStore = (*snapshotWebsiteCommandStoreAdapter)(nil)
var _ snapshotapp.EndpointSnapshotQueryStore = (*snapshotEndpointQueryStoreAdapter)(nil)
var _ snapshotapp.EndpointSnapshotCommandStore = (*snapshotEndpointCommandStoreAdapter)(nil)
var _ snapshotapp.DirectorySnapshotQueryStore = (*snapshotDirectoryQueryStoreAdapter)(nil)
var _ snapshotapp.DirectorySnapshotCommandStore = (*snapshotDirectoryCommandStoreAdapter)(nil)
var _ snapshotapp.SubdomainSnapshotQueryStore = (*snapshotSubdomainQueryStoreAdapter)(nil)
var _ snapshotapp.SubdomainSnapshotCommandStore = (*snapshotSubdomainCommandStoreAdapter)(nil)
var _ snapshotapp.HostPortSnapshotQueryStore = (*snapshotHostPortQueryStoreAdapter)(nil)
var _ snapshotapp.HostPortSnapshotCommandStore = (*snapshotHostPortCommandStoreAdapter)(nil)
var _ snapshotapp.ScreenshotSnapshotQueryStore = (*snapshotScreenshotQueryStoreAdapter)(nil)
var _ snapshotapp.ScreenshotSnapshotCommandStore = (*snapshotScreenshotCommandStoreAdapter)(nil)
var _ snapshotapp.VulnerabilitySnapshotQueryStore = (*snapshotVulnerabilityQueryStoreAdapter)(nil)
var _ snapshotapp.VulnerabilitySnapshotCommandStore = (*snapshotVulnerabilityCommandStoreAdapter)(nil)

var _ snapshotapp.WebsiteAssetSync = (*snapshotWebsiteAssetSyncAdapter)(nil)
var _ snapshotapp.EndpointAssetSync = (*snapshotEndpointAssetSyncAdapter)(nil)
var _ snapshotapp.DirectoryAssetSync = (*snapshotDirectoryAssetSyncAdapter)(nil)
var _ snapshotapp.SubdomainAssetSync = (*snapshotSubdomainAssetSyncAdapter)(nil)
var _ snapshotapp.HostPortAssetSync = (*snapshotHostPortAssetSyncAdapter)(nil)
var _ snapshotapp.ScreenshotAssetSync = (*snapshotScreenshotAssetSyncAdapter)(nil)
var _ snapshotapp.VulnerabilityAssetSync = (*snapshotVulnerabilityAssetSyncAdapter)(nil)
