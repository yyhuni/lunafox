package dto

import "time"

type ScreenshotSnapshotItem struct {
	URL        string `json:"url" binding:"required,url"`
	StatusCode *int16 `json:"statusCode"`
	Image      []byte `json:"image"`
}

type BulkUpsertScreenshotSnapshotsRequest struct {
	TargetID    int                      `json:"targetId" binding:"required"`
	Screenshots []ScreenshotSnapshotItem `json:"screenshots" binding:"required,min=1,max=5000,dive"`
}

type BulkUpsertScreenshotSnapshotsResponse struct {
	SnapshotCount int `json:"snapshotCount"`
	AssetCount    int `json:"assetCount"`
}

type ScreenshotSnapshotListQuery struct {
	PaginationQuery
	Filter string `form:"filter"`
}

type ScreenshotSnapshotResponse struct {
	ID         int       `json:"id"`
	ScanID     int       `json:"scanId"`
	URL        string    `json:"url"`
	StatusCode *int16    `json:"statusCode"`
	CreatedAt  time.Time `json:"createdAt"`
}
