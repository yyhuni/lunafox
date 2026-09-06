package dto

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type HostPortSnapshotItem struct {
	Host string `json:"host" binding:"required"`
	IP   string `json:"ip" binding:"required,ip"`
	Port int    `json:"port" binding:"required,min=1,max=65535"`
}

type BatchUpsertHostPortSnapshotsRequest struct {
	Target    string                 `json:"target" binding:"required"`
	HostPorts []HostPortSnapshotItem `json:"hostPorts" binding:"required,min=1,max=5000,dive"`
}

type BatchUpsertHostPortSnapshotsResponse struct {
	SnapshotCount int `json:"snapshotCount"`
	AssetCount    int `json:"assetCount"`
}

type HostPortSnapshotListQuery struct {
	httpdto.PaginationQuery
	Filter  string `form:"filter"`
	OrderBy string `form:"orderBy" binding:"omitempty"`
}

func (q *HostPortSnapshotListQuery) ValidatePageToken() error {
	return nil
}

type HostPortSnapshotResponse struct {
	ID        int       `json:"id"`
	ScanID    int       `json:"scanId"`
	Name      string    `json:"name"`
	Host      string    `json:"host"`
	IP        string    `json:"ip"`
	Port      int       `json:"port"`
	CreatedAt time.Time `json:"createdAt"`
}

type HostPortSnapshotAggregateResponse struct {
	Name      string    `json:"name"`
	IP        string    `json:"ip"`
	Hosts     []string  `json:"hosts"`
	Ports     []int     `json:"ports"`
	CreatedAt time.Time `json:"createdAt"`
}
