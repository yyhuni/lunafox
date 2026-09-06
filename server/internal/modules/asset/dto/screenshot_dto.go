package dto

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type ScreenshotListQuery struct {
	httpdto.PaginationQuery
	Filter  string `form:"filter"`
	OrderBy string `form:"orderBy" binding:"omitempty"`
}

func (q *ScreenshotListQuery) ValidatePageToken() error {
	return nil
}

type ScreenshotResponse struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
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

type BatchUpsertScreenshotRequest struct {
	Screenshots []ScreenshotItem `json:"screenshots" binding:"required,min=1,max=5000,dive"`
}

type BatchUpsertScreenshotResponse struct {
	UpsertedCount int64 `json:"upsertedCount"`
}
