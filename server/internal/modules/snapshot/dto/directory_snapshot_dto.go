package dto

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type DirectorySnapshotItem struct {
	URL           string `json:"url" binding:"required"`
	Status        *int   `json:"status"`
	ContentLength *int64 `json:"contentLength"`
	ContentType   string `json:"contentType"`
	Duration      *int64 `json:"duration"`
}

type BatchUpsertDirectorySnapshotsRequest struct {
	Target      string                  `json:"target" binding:"required"`
	Directories []DirectorySnapshotItem `json:"directories" binding:"required,min=1,max=5000,dive"`
}

type BatchUpsertDirectorySnapshotsResponse struct {
	SnapshotCount int `json:"snapshotCount"`
	AssetCount    int `json:"assetCount"`
}

type DirectorySnapshotListQuery struct {
	httpdto.PaginationQuery
	Filter  string `form:"filter"`
	OrderBy string `form:"orderBy" binding:"omitempty"`
}

func (q *DirectorySnapshotListQuery) ValidatePageToken() error {
	return nil
}

type DirectorySnapshotResponse struct {
	ID            int       `json:"id"`
	ScanID        int       `json:"scanId"`
	Name          string    `json:"name"`
	URL           string    `json:"url"`
	Status        *int      `json:"status"`
	ContentLength *string   `json:"contentLength"`
	ContentType   string    `json:"contentType"`
	Duration      *string   `json:"duration"`
	CreatedAt     time.Time `json:"createdAt"`
}
