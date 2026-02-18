package dto

import "time"

type SubdomainSnapshotItem struct {
	Name string `json:"name" binding:"required"`
}

type BulkUpsertSubdomainSnapshotsRequest struct {
	TargetID   int                     `json:"targetId" binding:"required"`
	Subdomains []SubdomainSnapshotItem `json:"subdomains" binding:"required,min=1,max=5000,dive"`
}

type BulkUpsertSubdomainSnapshotsResponse struct {
	SnapshotCount int `json:"snapshotCount"`
	AssetCount    int `json:"assetCount"`
}

type SubdomainSnapshotListQuery struct {
	PaginationQuery
	Filter string `form:"filter"`
}

type SubdomainSnapshotResponse struct {
	ID        int       `json:"id"`
	ScanID    int       `json:"scanId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}
