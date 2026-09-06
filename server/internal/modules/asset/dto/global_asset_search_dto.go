package dto

// GlobalAssetSearchQuery is the HTTP query shape for the cross-Target search
// custom method. Application validation owns the required-field, parser, and
// opaque-token checks so malformed inputs share one INVALID_ARGUMENT contract.
type GlobalAssetSearchQuery struct {
	Q         string `form:"q"`
	AssetType string `form:"assetType"`
	PageSize  *int   `form:"pageSize"`
	PageToken string `form:"pageToken"`
}
