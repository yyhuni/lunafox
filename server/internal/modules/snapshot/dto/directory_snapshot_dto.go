package dto

import "time"

type DirectorySnapshotItem struct {
	URL           string `json:"url" binding:"required,url"`
	Status        *int   `json:"status"`
	ContentLength *int   `json:"contentLength"`
	ContentType   string `json:"contentType"`
	Duration      *int   `json:"duration"`
}

type BulkUpsertDirectorySnapshotsRequest struct {
	TargetID    int                     `json:"targetId" binding:"required"`
	Directories []DirectorySnapshotItem `json:"directories" binding:"required,min=1,max=5000,dive"`
}

type BulkUpsertDirectorySnapshotsResponse struct {
	SnapshotCount int `json:"snapshotCount"`
	AssetCount    int `json:"assetCount"`
}

type DirectorySnapshotListQuery struct {
	PaginationQuery
	Filter string `form:"filter"`
}

type DirectorySnapshotResponse struct {
	ID            int       `json:"id"`
	ScanID        int       `json:"scanId"`
	URL           string    `json:"url"`
	Status        *int      `json:"status"`
	ContentLength *int      `json:"contentLength"`
	ContentType   string    `json:"contentType"`
	Duration      *int      `json:"duration"`
	CreatedAt     time.Time `json:"createdAt"`
}
