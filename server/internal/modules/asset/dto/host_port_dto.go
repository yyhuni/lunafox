package dto

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type HostPortListQuery struct {
	httpdto.PaginationQuery
	Filter  string `form:"filter"`
	OrderBy string `form:"orderBy" binding:"omitempty"`
}

func (q *HostPortListQuery) ValidatePageToken() error {
	return nil
}

type HostPortResponse struct {
	Name      string    `json:"name"`
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

type BatchUpsertHostPortsRequest struct {
	Mappings []HostPortItem `json:"mappings" binding:"required,min=1,max=5000,dive"`
}

type BatchUpsertHostPortsResponse struct {
	UpsertedCount int `json:"upsertedCount"`
}

type BatchDeleteHostPortsRequest struct {
	IPs []string `json:"ips" binding:"required,min=1,max=5000"`
}

type BatchDeleteHostPortsResponse struct {
	DeletedCount int64 `json:"deletedCount"`
}
