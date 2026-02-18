package application

import assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"

type AssetTargetLookup interface {
	GetActiveByID(id int) (*assetdomain.TargetRef, error)
}

type WebsiteTargetLookup = AssetTargetLookup
type EndpointTargetLookup = AssetTargetLookup
type DirectoryTargetLookup = AssetTargetLookup
type SubdomainTargetLookup = AssetTargetLookup
type HostPortTargetLookup = AssetTargetLookup
type ScreenshotTargetLookup = AssetTargetLookup
