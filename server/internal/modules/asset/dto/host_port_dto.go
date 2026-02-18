package dto

import "time"

type HostPortListQuery struct {
	PaginationQuery
	Filter string `form:"filter"`
}

type HostPortResponse struct {
	IP        string    `json:"ip"`
	Hosts     []string  `json:"hosts"`
	Ports     []int     `json:"ports"`
	CreatedAt time.Time `json:"createdAt"`
}

type HostPortItem struct {
	Host string `json:"host" binding:"required"`
	IP   string `json:"ip" binding:"required,ip"`
	Port int    `json:"port" binding:"required,min=1,max=65535"`
}

type BulkUpsertHostPortsRequest struct {
	Mappings []HostPortItem `json:"mappings" binding:"required,min=1,max=5000,dive"`
}

type BulkUpsertHostPortsResponse struct {
	UpsertedCount int `json:"upsertedCount"`
}

type BulkDeleteHostPortsRequest struct {
	IPs []string `json:"ips" binding:"required,min=1"`
}

type BulkDeleteHostPortsResponse struct {
	DeletedCount int64 `json:"deletedCount"`
}
