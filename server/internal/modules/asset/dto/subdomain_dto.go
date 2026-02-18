package dto

import "time"

type SubdomainListQuery struct {
	PaginationQuery
	Filter string `form:"filter"`
}

type SubdomainResponse struct {
	ID        int       `json:"id"`
	TargetID  int       `json:"targetId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type BulkCreateSubdomainsRequest struct {
	Names []string `json:"names" binding:"required,min=1,max=5000"`
}

type BulkCreateSubdomainsResponse struct {
	CreatedCount int `json:"createdCount"`
}
