package dto

import (
	"time"

	"github.com/yyhuni/lunafox/contracts/scanworkflow"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type ManagedScanWorkflowResponse struct {
	Name         string               `json:"name"`
	DisplayName  string               `json:"displayName"`
	Description  string               `json:"description"`
	Stages       []scanworkflow.Stage `json:"stages"`
	IsBuiltin    bool                 `json:"isBuiltin"`
	IsExecutable bool                 `json:"isExecutable"`
	CreateTime   time.Time            `json:"createTime"`
	UpdateTime   time.Time            `json:"updateTime"`
	ETag         string               `json:"etag"`
}

type ListManagedScanWorkflowsResponse struct {
	Results       []ManagedScanWorkflowResponse `json:"results"`
	NextPageToken string                        `json:"nextPageToken,omitempty"`
	TotalSize     int64                         `json:"totalSize"`
}

type ManagedScanWorkflowProfileResponse struct {
	Name          string         `json:"name"`
	ScanWorkflow  string         `json:"scanWorkflow"`
	Configuration map[string]any `json:"configuration"`
}

func NewManagedScanWorkflowProfileResponse(scanWorkflowID string, configuration map[string]any) ManagedScanWorkflowProfileResponse {
	return ManagedScanWorkflowProfileResponse{Name: httpdto.ScanWorkflowName(scanWorkflowID) + "/profile", ScanWorkflow: httpdto.ScanWorkflowName(scanWorkflowID), Configuration: configuration}
}

type CreateScanWorkflowRequest struct {
	ScanWorkflowID string                     `json:"scanWorkflowId,omitempty"`
	RequestID      string                     `json:"requestId"`
	ScanWorkflow   CreateScanWorkflowResource `json:"scanWorkflow"`
}

type CreateScanWorkflowResource struct {
	DisplayName string               `json:"displayName"`
	Description string               `json:"description"`
	Stages      []scanworkflow.Stage `json:"stages"`
}

type UpdateScanWorkflowRequest struct {
	ScanWorkflow UpdateScanWorkflowResource `json:"scanWorkflow"`
	UpdateMask   []string                   `json:"updateMask"`
}

type UpdateScanWorkflowResource struct {
	Name        string                `json:"name"`
	DisplayName *string               `json:"displayName,omitempty"`
	Description *string               `json:"description,omitempty"`
	Stages      *[]scanworkflow.Stage `json:"stages,omitempty"`
	ETag        string                `json:"etag"`
}

func NewManagedScanWorkflowResponse(workflow *catalogdomain.ManagedScanWorkflow, etag string, isExecutable bool) ManagedScanWorkflowResponse {
	return ManagedScanWorkflowResponse{Name: httpdto.ScanWorkflowName(workflow.ScanWorkflowID), DisplayName: workflow.DisplayName, Description: workflow.Description, Stages: workflow.Stages, IsBuiltin: workflow.IsBuiltin, IsExecutable: isExecutable, CreateTime: workflow.CreateTime, UpdateTime: workflow.UpdateTime, ETag: etag}
}

func NewManagedScanWorkflowListResponse(items []ManagedScanWorkflowResponse, nextPageToken string, totalSize int64) ListManagedScanWorkflowsResponse {
	return ListManagedScanWorkflowsResponse{Results: items, NextPageToken: nextPageToken, TotalSize: totalSize}
}
