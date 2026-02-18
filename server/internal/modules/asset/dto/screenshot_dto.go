package dto

import "time"

type ScreenshotListQuery struct {
	PaginationQuery
	Filter string `form:"filter"`
}

type ScreenshotResponse struct {
	ID         int       `json:"id"`
	URL        string    `json:"url"`
	StatusCode *int16    `json:"statusCode"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type ScreenshotItem struct {
	URL        string `json:"url" binding:"required"`
	StatusCode *int16 `json:"statusCode"`
	Image      []byte `json:"image"`
}

type BulkUpsertScreenshotRequest struct {
	Screenshots []ScreenshotItem `json:"screenshots" binding:"required,min=1,max=5000,dive"`
}

type BulkUpsertScreenshotResponse struct {
	UpsertedCount int64 `json:"upsertedCount"`
}
