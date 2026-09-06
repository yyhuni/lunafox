package dto

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type DirectoryListQuery struct {
	httpdto.PaginationQuery
	Filter  string `form:"filter"`
	OrderBy string `form:"orderBy" binding:"omitempty"`
}

func (q *DirectoryListQuery) ValidatePageToken() error {
	return nil
}

type DirectoryResponse struct {
	ID            int       `json:"id"`
	TargetID      int       `json:"targetId"`
	Name          string    `json:"name"`
	URL           string    `json:"url"`
	Status        *int      `json:"status"`
	ContentLength *string   `json:"contentLength"`
	ContentType   string    `json:"contentType"`
	Duration      *string   `json:"duration"`
	CreatedAt     time.Time `json:"createdAt"`
}

type BatchCreateDirectoriesRequest struct {
	URLs []string `json:"urls" binding:"required,min=1,max=5000"`
}

type BatchCreateDirectoriesResponse struct {
	CreatedCount int `json:"createdCount"`
}

type DirectoryUpsertItem struct {
	URL           string `json:"url" binding:"required"`
	Status        *int   `json:"status"`
	ContentLength *int64 `json:"contentLength"`
	ContentType   string `json:"contentType"`
	Duration      *int64 `json:"duration"`
}

type BatchUpsertDirectoriesRequest struct {
	Directories []DirectoryUpsertItem `json:"directories" binding:"required,min=1,max=5000,dive"`
}

type BatchUpsertDirectoriesResponse struct {
	AffectedCount int64 `json:"affectedCount"`
}
