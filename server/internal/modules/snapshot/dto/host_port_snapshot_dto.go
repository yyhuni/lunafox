package dto

import "time"

type HostPortSnapshotItem struct {
	Host string `json:"host" binding:"required"`
	IP   string `json:"ip" binding:"required,ip"`
	Port int    `json:"port" binding:"required,min=1,max=65535"`
}

type BulkUpsertHostPortSnapshotsRequest struct {
	TargetID  int                    `json:"targetId" binding:"required"`
	HostPorts []HostPortSnapshotItem `json:"hostPorts" binding:"required,min=1,max=5000,dive"`
}

type BulkUpsertHostPortSnapshotsResponse struct {
	SnapshotCount int `json:"snapshotCount"`
	AssetCount    int `json:"assetCount"`
}

type HostPortSnapshotListQuery struct {
	PaginationQuery
	Filter string `form:"filter"`
}

type HostPortSnapshotResponse struct {
	ID        int       `json:"id"`
	ScanID    int       `json:"scanId"`
	Host      string    `json:"host"`
	IP        string    `json:"ip"`
	Port      int       `json:"port"`
	CreatedAt time.Time `json:"createdAt"`
}
