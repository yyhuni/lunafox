package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	scheduledapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
	"github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/dto"
)

type ScheduledScanHandler struct {
	service scheduledScanService
}

type scheduledScanService interface {
	List(context.Context, scheduledapp.ScheduledScanListQuery) ([]scheduledapp.ScheduledScan, int64, error)
	GetOverviewSummary(context.Context, *scheduledapp.ScheduledScanOverviewInput) (*scheduledapp.ScheduledScanOverview, error)
	GetByID(context.Context, int) (*scheduledapp.ScheduledScan, error)
	Create(context.Context, *scheduledapp.CreateScheduledScanInput) (*scheduledapp.ScheduledScan, error)
	Update(context.Context, int, *scheduledapp.UpdateScheduledScanInput) (*scheduledapp.ScheduledScan, error)
	BatchUpdateStatus(context.Context, []scheduledapp.ScheduledScanStatusUpdate) (int, error)
	Delete(context.Context, int) error
}

func (handler *ScheduledScanHandler) Summarize(c *gin.Context) {
	var query dto.ScheduledScanOverviewQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}
	if c.Request.URL.Query().Get("timeZone") != "" {
		httpdto.BadRequest(c, "timeZone query parameter is no longer supported; schedules use UTC")
		return
	}
	overview, err := handler.service.GetOverviewSummary(c.Request.Context(), &scheduledapp.ScheduledScanOverviewInput{})
	if err != nil {
		handleScheduledScanError(c, err)
		return
	}
	httpdto.Success(c, toScheduledScanOverviewOutput(overview))
}

func NewScheduledScanHandler(service scheduledScanService) *ScheduledScanHandler {
	return &ScheduledScanHandler{service: service}
}

func (handler *ScheduledScanHandler) List(c *gin.Context) {
	var query dto.ScheduledScanListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}
	items, total, err := handler.service.List(c.Request.Context(), toScheduledScanListQuery(&query))
	if err != nil {
		httpdto.InternalError(c, "Failed to list scheduled scans")
		return
	}
	httpdto.Success(c, dto.NewScheduledScanListResponse(toScheduledScanListOutput(items), total, query.GetPage(), query.GetPageSize()))
}

func (handler *ScheduledScanHandler) GetByID(c *gin.Context) {
	id, ok := parseScheduledScanID(c)
	if !ok {
		return
	}
	item, err := handler.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, scheduledapp.ErrScheduledScanNotFound) {
			httpdto.NotFound(c, "Scheduled scan not found")
			return
		}
		httpdto.InternalError(c, "Failed to get scheduled scan")
		return
	}
	httpdto.Success(c, toScheduledScanOutput(item))
}

func (handler *ScheduledScanHandler) Create(c *gin.Context) {
	var request dto.CreateScheduledScanRequest
	if !httpdto.BindJSON(c, &request) {
		return
	}
	input, err := toCreateInput(&request)
	if err != nil {
		httpdto.BadRequest(c, err.Error())
		return
	}
	item, err := handler.service.Create(c.Request.Context(), input)
	if err != nil {
		if httpdto.WriteContextError(c, err) {
			return
		}
		handleScheduledScanError(c, err)
		return
	}
	httpdto.Created(c, toScheduledScanOutput(item))
}

func (handler *ScheduledScanHandler) Update(c *gin.Context) {
	id, ok := parseScheduledScanID(c)
	if !ok {
		return
	}
	var request dto.UpdateScheduledScanRequest
	if !httpdto.BindJSON(c, &request) {
		return
	}
	if !validateScheduledScanUpdateMask(c, id, &request) {
		return
	}
	input, err := toUpdateInput(&request)
	if err != nil {
		httpdto.BadRequest(c, err.Error())
		return
	}
	item, err := handler.service.Update(c.Request.Context(), id, input)
	if err != nil {
		handleScheduledScanError(c, err)
		return
	}
	httpdto.Success(c, toScheduledScanOutput(item))
}

// BatchUpdate updates the enabled state of multiple Scheduled Scans atomically.
func (handler *ScheduledScanHandler) BatchUpdate(c *gin.Context) {
	var request dto.BatchUpdateScheduledScansRequest
	if !httpdto.BindJSON(c, &request) {
		return
	}
	updates, err := toBatchUpdateStatusInput(&request)
	if err != nil {
		httpdto.BadRequest(c, err.Error())
		return
	}
	updatedCount, err := handler.service.BatchUpdateStatus(c.Request.Context(), updates)
	if err != nil {
		handleScheduledScanError(c, err)
		return
	}
	httpdto.Success(c, dto.BatchUpdateScheduledScansResponse{UpdatedCount: updatedCount})
}

func (handler *ScheduledScanHandler) Delete(c *gin.Context) {
	id, ok := parseScheduledScanID(c)
	if !ok {
		return
	}
	if err := handler.service.Delete(c.Request.Context(), id); err != nil {
		handleScheduledScanError(c, err)
		return
	}
	httpdto.NoContent(c)
}

func parseScheduledScanID(c *gin.Context) (int, bool) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("scheduled_scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scheduled scan ID")
		return 0, false
	}
	return id, true
}

func validateScheduledScanUpdateMask(c *gin.Context, id int, request *dto.UpdateScheduledScanRequest) bool {
	if request == nil {
		httpdto.BadRequest(c, "Invalid request body")
		return false
	}
	if strings.TrimSpace(request.Name) != httpdto.ScheduledScanName(id) {
		httpdto.BadRequest(c, "name must match scheduled scan resource")
		return false
	}
	fields := map[string]bool{}
	for _, part := range strings.Split(request.UpdateMask, ",") {
		field := strings.TrimSpace(part)
		if field != "" {
			fields[field] = true
		}
	}
	if len(fields) == 0 {
		httpdto.BadRequest(c, "updateMask is required")
		return false
	}
	allowed := map[string]bool{
		"displayName":    request.DisplayName != nil,
		"scanWorkflow":   request.ScanWorkflow != nil,
		"configuration":  request.Configuration != nil,
		"inputSource":    request.InputSource != nil,
		"organization":   request.Organization != nil,
		"target":         request.Target != nil,
		"agent":          request.Agent != nil,
		"cronExpression": request.CronExpression != nil,
		"isEnabled":      request.IsEnabled != nil,
	}
	for field := range fields {
		provided, ok := allowed[field]
		if !ok {
			httpdto.BadRequest(c, "updateMask contains unsupported field")
			return false
		}
		if !provided {
			httpdto.BadRequest(c, "updateMask must match provided fields")
			return false
		}
	}
	for field, provided := range allowed {
		if provided && !fields[field] {
			httpdto.BadRequest(c, field+" must be declared in updateMask")
			return false
		}
	}
	return true
}

func handleScheduledScanError(c *gin.Context, err error) {
	if httpdto.WriteConfigResourceValidationError(c, err) {
		return
	}
	if httpdto.WriteWorkflowConfigurationError(c, err) {
		return
	}
	switch {
	case errors.Is(err, scheduledapp.ErrScheduledScanNotFound):
		httpdto.NotFound(c, "Scheduled scan not found")
	case errors.Is(err, scheduledapp.ErrScheduledScanAgentNotFound):
		httpdto.NotFound(c, "Selected Agent not found")
	case errors.Is(err, scheduledapp.ErrScheduledScanInvalidArgument):
		httpdto.BadRequest(c, err.Error())
	default:
		httpdto.Error(c, http.StatusInternalServerError, "INTERNAL", "Scheduled scan operation failed")
	}
}
