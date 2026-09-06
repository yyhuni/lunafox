package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type scanWorkflowManagementService interface {
	ListScanWorkflows(context.Context, catalogapp.ListManagedScanWorkflowsInput) (*catalogapp.ListManagedScanWorkflowsResult, error)
	GetScanWorkflow(context.Context, string) (*catalogapp.ManagedScanWorkflow, error)
	CreateScanWorkflow(context.Context, catalogapp.CreateManagedScanWorkflowInput) (*catalogapp.ManagedScanWorkflow, error)
	UpdateScanWorkflow(context.Context, catalogapp.UpdateManagedScanWorkflowInput) (*catalogapp.ManagedScanWorkflow, error)
}

type scanWorkflowProfileService interface {
	GetScanWorkflowProfile(context.Context, string) (*catalogapp.ManagedScanWorkflowProfile, error)
}

type ScanWorkflowManagementHandler struct {
	service        scanWorkflowManagementService
	profileService scanWorkflowProfileService
}

func NewScanWorkflowManagementHandler(service scanWorkflowManagementService, profileServices ...scanWorkflowProfileService) *ScanWorkflowManagementHandler {
	handler := &ScanWorkflowManagementHandler{service: service}
	if len(profileServices) > 0 {
		handler.profileService = profileServices[0]
	}
	return handler
}

func (handler *ScanWorkflowManagementHandler) List(c *gin.Context) {
	pageSize, err := optionalPositiveQueryInt(c, "pageSize")
	if err != nil {
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error())
		return
	}
	result, err := handler.service.ListScanWorkflows(c.Request.Context(), catalogapp.ListManagedScanWorkflowsInput{PageSize: pageSize, PageToken: c.Query("pageToken"), Filter: c.Query("filter")})
	if err != nil {
		handler.writeError(c, err)
		return
	}
	responses := make([]dto.ManagedScanWorkflowResponse, 0, len(result.Results))
	for index := range result.Results {
		responses = append(responses, dto.NewManagedScanWorkflowResponse(&result.Results[index].ManagedScanWorkflow, result.Results[index].ETag, result.Results[index].IsExecutable))
	}
	httpdto.Success(c, dto.NewManagedScanWorkflowListResponse(responses, result.NextPageToken, result.TotalSize))
}

func (handler *ScanWorkflowManagementHandler) Get(c *gin.Context) {
	workflow, err := handler.service.GetScanWorkflow(c.Request.Context(), c.Param("scanWorkflow"))
	if err != nil {
		handler.writeError(c, err)
		return
	}
	httpdto.Success(c, dto.NewManagedScanWorkflowResponse(&workflow.ManagedScanWorkflow, workflow.ETag, workflow.IsExecutable))
}

func (handler *ScanWorkflowManagementHandler) GetProfile(c *gin.Context) {
	if handler.profileService == nil {
		httpdto.InternalError(c, "Scan workflow Profile service is unavailable")
		return
	}
	profile, err := handler.profileService.GetScanWorkflowProfile(c.Request.Context(), c.Param("scanWorkflow"))
	if err != nil {
		handler.writeError(c, err)
		return
	}
	httpdto.Success(c, dto.NewManagedScanWorkflowProfileResponse(profile.ScanWorkflowID, profile.Configuration))
}

func (handler *ScanWorkflowManagementHandler) Create(c *gin.Context) {
	var request dto.CreateScanWorkflowRequest
	if !httpdto.BindJSON(c, &request) {
		return
	}
	workflow, err := handler.service.CreateScanWorkflow(c.Request.Context(), catalogapp.CreateManagedScanWorkflowInput{ScanWorkflowID: request.ScanWorkflowID, RequestID: request.RequestID, DisplayName: request.ScanWorkflow.DisplayName, Description: request.ScanWorkflow.Description, Stages: request.ScanWorkflow.Stages})
	if err != nil {
		handler.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.NewManagedScanWorkflowResponse(&workflow.ManagedScanWorkflow, workflow.ETag, workflow.IsExecutable))
}

func (handler *ScanWorkflowManagementHandler) Update(c *gin.Context) {
	var request dto.UpdateScanWorkflowRequest
	if !httpdto.BindJSON(c, &request) {
		return
	}
	pathID := c.Param("scanWorkflow")
	if request.ScanWorkflow.Name != httpdto.ScanWorkflowName(pathID) {
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", "scanWorkflow.name must match the resource path")
		return
	}
	workflow, err := handler.service.UpdateScanWorkflow(c.Request.Context(), catalogapp.UpdateManagedScanWorkflowInput{ScanWorkflowID: pathID, ETag: request.ScanWorkflow.ETag, UpdateMask: request.UpdateMask, DisplayName: request.ScanWorkflow.DisplayName, Description: request.ScanWorkflow.Description, Stages: request.ScanWorkflow.Stages})
	if err != nil {
		handler.writeError(c, err)
		return
	}
	httpdto.Success(c, dto.NewManagedScanWorkflowResponse(&workflow.ManagedScanWorkflow, workflow.ETag, workflow.IsExecutable))
}

func (handler *ScanWorkflowManagementHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, catalogapp.ErrInvalidScanWorkflowPageToken), errors.Is(err, catalogapp.ErrInvalidScanWorkflowUpdate):
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error())
	case errors.Is(err, catalogapp.ErrScanWorkflowAlreadyExists):
		httpdto.Error(c, http.StatusConflict, "ALREADY_EXISTS", "scan workflow already exists")
	case errors.Is(err, catalogapp.ErrScanWorkflowImmutable):
		httpdto.Error(c, http.StatusBadRequest, "BUILTIN_SCAN_WORKFLOW_IMMUTABLE", "built-in scan workflows are read-only")
	case errors.Is(err, catalogapp.ErrScanWorkflowConflict):
		httpdto.ErrorWithStatus(c, http.StatusConflict, "ABORTED", "ABORTED", "scan workflow was modified; reload before saving")
	case errors.Is(err, catalogdomain.ErrScanWorkflowNotFound):
		httpdto.NotFound(c, "Scan workflow not found")
	case errors.Is(err, catalogapp.ErrScanWorkflowEngineUnavailable):
		httpdto.Error(c, http.StatusBadRequest, "SCAN_WORKFLOW_ENGINE_UNAVAILABLE", "scan workflow references an unavailable Engine")
	default:
		httpdto.InternalError(c, "Failed to manage scan workflow")
	}
}

func optionalPositiveQueryInt(c *gin.Context, key string) (int, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, errors.New(key + " must be a positive integer")
	}
	return value, nil
}
