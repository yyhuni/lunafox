package dto

import "time"

type DirectoryListQuery struct {
	PaginationQuery
	Filter string `form:"filter"`
}

type DirectoryResponse struct {
	ID            int       `json:"id"`
	TargetID      int       `json:"targetId"`
	URL           string    `json:"url"`
	Status        *int      `json:"status"`
	ContentLength *int      `json:"contentLength"`
	ContentType   string    `json:"contentType"`
	Duration      *int      `json:"duration"`
	CreatedAt     time.Time `json:"createdAt"`
}

type BulkCreateDirectoriesRequest struct {
	URLs []string `json:"urls" binding:"required,min=1,max=5000"`
}

type BulkCreateDirectoriesResponse struct {
	CreatedCount int `json:"createdCount"`
}

type DirectoryUpsertItem struct {
	URL           string `json:"url" binding:"required,url"`
	Status        *int   `json:"status"`
	ContentLength *int   `json:"contentLength"`
	ContentType   string `json:"contentType"`
	Duration      *int   `json:"duration"`
}

type BulkUpsertDirectoriesRequest struct {
	Directories []DirectoryUpsertItem `json:"directories" binding:"required,min=1,max=5000,dive"`
}

type BulkUpsertDirectoriesResponse struct {
	AffectedCount int64 `json:"affectedCount"`
}
