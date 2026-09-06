package application

type ScreenshotSnapshotItem struct {
	URL        string
	StatusCode *int16
	Image      []byte
}

type ScreenshotAssetItem struct {
	URL        string
	StatusCode *int16
	Image      []byte
}

type ScreenshotAssetUpsertRequest struct {
	Screenshots []ScreenshotAssetItem
}
