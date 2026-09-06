package dto

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type ScreenshotSnapshotItem struct {
	URL        string `json:"url" binding:"required"`
	StatusCode *int16 `json:"statusCode"`
	Image      []byte `json:"image"`
}

type BatchUpsertScreenshotSnapshotsRequest struct {
	Target      string                   `json:"target" binding:"required"`
	Screenshots []ScreenshotSnapshotItem `json:"screenshots" binding:"required,min=1,max=5000,dive"`
}

type BatchUpsertScreenshotSnapshotsResponse struct {
	SnapshotCount int `json:"snapshotCount"`
	AssetCount    int `json:"assetCount"`
}

type ScreenshotSnapshotListQuery struct {
	httpdto.PaginationQuery
	Filter  string `form:"filter"`
	OrderBy string `form:"orderBy" binding:"omitempty"`
}

func (q *ScreenshotSnapshotListQuery) ValidatePageToken() error {
	return nil
}

type ScreenshotSnapshotResponse struct {
	ID         int       `json:"id"`
	ScanID     int       `json:"scanId"`
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	StatusCode *int16    `json:"statusCode"`
	CreatedAt  time.Time `json:"createdAt"`
}
