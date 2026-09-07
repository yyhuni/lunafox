package dto

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type CreateOrganizationRequest struct {
	Name        string `json:"name" binding:"required,max=300"`
	Description string `json:"description" binding:"max=1000"`
}

type UpdateOrganizationRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"displayName" binding:"required,max=300"`
	Description string `json:"description" binding:"max=1000"`
	UpdateMask  string `json:"updateMask" binding:"required"`
}

type OrganizationListQuery struct {
	httpdto.PaginationQuery
	Filter  string `form:"filter"`
	OrderBy string `form:"orderBy" binding:"omitempty"`
}

func (q *OrganizationListQuery) ValidatePageToken() error {
	return nil
}

type OrganizationResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	TargetCount int64     `json:"targetCount"`
}

type LinkTargetsRequest struct {
	Targets []string `json:"targets" binding:"required,min=1,max=5000"`
}
