package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type AssetTargetLookup interface {
	GetActiveByID(id int) (*assetdomain.TargetRef, error)
}

// AssetCommandTargetLookup keeps result-ingest cancellation and deadlines on
// the target authorization query that precedes asset projection writes.
type AssetCommandTargetLookup interface {
	GetActiveByIDContext(ctx context.Context, id int) (*assetdomain.TargetRef, error)
}

type AssetApplicationTargetLookup interface {
	AssetTargetLookup
	AssetCommandTargetLookup
}

type WebsiteTargetLookup = AssetTargetLookup
type EndpointTargetLookup = AssetTargetLookup
type DirectoryTargetLookup = AssetTargetLookup
type SubdomainTargetLookup = AssetTargetLookup
type HostPortTargetLookup = AssetTargetLookup
type ScreenshotTargetLookup = AssetTargetLookup
