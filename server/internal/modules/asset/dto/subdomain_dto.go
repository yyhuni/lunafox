package dto

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type SubdomainListQuery struct {
	httpdto.PaginationQuery
	Filter  string `form:"filter"`
	OrderBy string `form:"orderBy" binding:"omitempty"`
}

func (q *SubdomainListQuery) ValidatePageToken() error {
	return nil
}

type SubdomainResponse struct {
	ID        int       `json:"id"`
	TargetID  int       `json:"targetId"`
	Name      string    `json:"name"`
	DNSName   string    `json:"dnsName"`
	CreatedAt time.Time `json:"createdAt"`
}

type BatchCreateSubdomainsRequest struct {
	DNSNames []string `json:"dnsNames" binding:"required,min=1,max=5000"`
}

type BatchCreateSubdomainsResponse struct {
	CreatedCount int `json:"createdCount"`
}
