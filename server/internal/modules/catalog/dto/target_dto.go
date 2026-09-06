package dto

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type CreateTargetRequest struct {
	Name string `json:"name" binding:"required,max=300"`
}

type UpdateTargetRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"displayName" binding:"required,max=300"`
	UpdateMask  string `json:"updateMask" binding:"required"`
}

type TargetListQuery struct {
	httpdto.PaginationQuery
	Filter  string `form:"filter"`
	OrderBy string `form:"orderBy" binding:"omitempty"`
}

func (q *TargetListQuery) ValidatePageToken() error {
	return nil
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

type OrganizationBrief struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type BatchCreateTargetRequest struct {
	Targets      []TargetItem `json:"targets" binding:"required,min=1,max=5000,dive"`
	Organization string       `json:"organization"`
}

type TargetItem struct {
	Name string `json:"name" binding:"required,max=300"`
}

type BatchCreateTargetResponse struct {
	CreatedCount  int            `json:"createdCount"`
	FailedCount   int            `json:"failedCount"`
	FailedTargets []FailedTarget `json:"failedTargets"`
	Message       string         `json:"message"`
}

type FailedTarget struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type BatchDeleteRequest struct {
	Names []string `json:"names" binding:"required,min=1,max=5000"`
}

type BatchDeleteResponse struct {
	DeletedCount int64 `json:"deletedCount"`
}

type TargetDetailResponse struct {
	ID            int                 `json:"id"`
	Name          string              `json:"name"`
	DisplayName   string              `json:"displayName"`
	Type          string              `json:"type"`
	CreatedAt     time.Time           `json:"createdAt"`
	LastScannedAt *time.Time          `json:"lastScannedAt"`
	Organizations []OrganizationBrief `json:"organizations,omitempty"`
	Summary       *TargetSummary      `json:"summary"`
}

type TargetSummary struct {
	Subdomains      int64                 `json:"subdomains"`
	Websites        int64                 `json:"websites"`
	Endpoints       int64                 `json:"endpoints"`
	IPs             int64                 `json:"ips"`
	Directories     int64                 `json:"directories"`
	Screenshots     int64                 `json:"screenshots"`
	Vulnerabilities *VulnerabilitySummary `json:"vulnerabilities"`
}

type VulnerabilitySummary struct {
	Total    int64 `json:"total"`
	Critical int64 `json:"critical"`
	High     int64 `json:"high"`
	Medium   int64 `json:"medium"`
	Low      int64 `json:"low"`
}
