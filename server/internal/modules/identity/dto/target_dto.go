package dto

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type TargetListQuery struct {
	httpdto.PaginationQuery
	Type   string `form:"type" binding:"omitempty,oneof=domain ip cidr"`
	Filter string `form:"filter"`
}

type OrganizationBrief struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type TargetResponse struct {
	ID            int                 `json:"id"`
	Name          string              `json:"name"`
	DisplayName   string              `json:"displayName"`
	Type          string              `json:"type"`
	CreatedAt     time.Time           `json:"createdAt"`
	LastScannedAt *time.Time          `json:"lastScannedAt"`
	Organizations []OrganizationBrief `json:"organizations,omitempty"`
}
