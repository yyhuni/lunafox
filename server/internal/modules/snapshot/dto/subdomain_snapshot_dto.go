package dto

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type SubdomainSnapshotItem struct {
	DNSName string `json:"dnsName" binding:"required"`
}

type BatchUpsertSubdomainSnapshotsRequest struct {
	Target     string                  `json:"target" binding:"required"`
	Subdomains []SubdomainSnapshotItem `json:"subdomains" binding:"required,min=1,max=5000,dive"`
}

type BatchUpsertSubdomainSnapshotsResponse struct {
	SnapshotCount int `json:"snapshotCount"`
	AssetCount    int `json:"assetCount"`
}

type SubdomainSnapshotListQuery struct {
	httpdto.PaginationQuery
	Filter  string `form:"filter"`
	OrderBy string `form:"orderBy" binding:"omitempty"`
}

func (q *SubdomainSnapshotListQuery) ValidatePageToken() error {
	return nil
}

type SubdomainSnapshotResponse struct {
	ID        int       `json:"id"`
	ScanID    int       `json:"scanId"`
	Name      string    `json:"name"`
	DNSName   string    `json:"dnsName"`
	CreatedAt time.Time `json:"createdAt"`
}
