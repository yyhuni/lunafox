package dto

import "time"

type TargetListQuery struct {
	PaginationQuery
	Type   string `form:"type" binding:"omitempty,oneof=domain ip cidr"`
	Filter string `form:"filter"`
}

type OrganizationBrief struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TargetResponse struct {
	ID            int                 `json:"id"`
	Name          string              `json:"name"`
	Type          string              `json:"type"`
	CreatedAt     time.Time           `json:"createdAt"`
	LastScannedAt *time.Time          `json:"lastScannedAt"`
	Organizations []OrganizationBrief `json:"organizations,omitempty"`
}
