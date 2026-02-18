package dto

import "time"

type CreateOrganizationRequest struct {
	Name        string `json:"name" binding:"required,max=300"`
	Description string `json:"description" binding:"max=1000"`
}

type UpdateOrganizationRequest struct {
	Name        string `json:"name" binding:"required,max=300"`
	Description string `json:"description" binding:"max=1000"`
}

type OrganizationListQuery struct {
	PaginationQuery
	Filter string `form:"filter"`
}

type OrganizationResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	TargetCount int64     `json:"targetCount"`
}

type LinkTargetsRequest struct {
	TargetIDs []int `json:"targetIds" binding:"required,min=1"`
}
