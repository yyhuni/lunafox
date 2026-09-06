package application

import (
	"context"

	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
)

type subdomainResultMaterializer interface {
	SaveAndSyncContext(context.Context, int, int, []snapshotapp.SubdomainSnapshotItem) (snapshotapp.MaterializationSummary, error)
}

type hostPortResultMaterializer interface {
	SaveAndSyncContext(context.Context, int, int, []snapshotapp.HostPortSnapshotItem) (snapshotapp.MaterializationSummary, error)
}

type websiteResultMaterializer interface {
	SaveAndSyncContext(context.Context, int, int, []snapshotapp.WebsiteSnapshotItem) (snapshotapp.MaterializationSummary, error)
}

type websiteTechnologyResultMaterializer interface {
	BatchUpsertTechnologyContext(context.Context, int, []assetapp.WebsiteTechnologyUpsertItem) (assetapp.WebsiteTechnologyMaterializationSummary, error)
}

type endpointResultMaterializer interface {
	SaveAndSyncContext(context.Context, int, int, []snapshotapp.EndpointSnapshotItem) (snapshotapp.MaterializationSummary, error)
}

type directoryResultMaterializer interface {
	SaveResultBatchContext(context.Context, int, int, []snapshotapp.DirectorySnapshotItem) (snapshotapp.MaterializationSummary, error)
}

type screenshotResultMaterializer interface {
	SaveResultBatchContext(context.Context, int, int, []snapshotapp.ScreenshotSnapshotItem) (snapshotapp.MaterializationSummary, error)
}

type vulnerabilityResultMaterializer interface {
	SaveResultBatchContext(context.Context, int, int, []snapshotapp.VulnerabilitySnapshotItem) (snapshotapp.MaterializationSummary, error)
}

// ScanResultSummaryUpdater refreshes backend-owned read-model counters after result materialization.
type ScanResultSummaryUpdater interface {
	RefreshScanResultSummary(ctx context.Context, scanID int, targetID int) error
}

// ResultMaterializationCoordinator atomically commits database projections for
// one accepted result batch. It owns the final Target, Scan, and Task lease
// fence without exposing infrastructure transaction types.
type ResultMaterializationCoordinator interface {
	Materialize(context.Context, ResultMaterializationScope, func(context.Context) error) error
}
