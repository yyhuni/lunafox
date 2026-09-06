package handler

import (
	"errors"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/dto"
)

// TaskProgressLogHandler handles task progress log HTTP requests.
type TaskProgressLogHandler struct {
	svc service.TaskProgressLogApplicationService
}

// NewTaskProgressLogHandler creates a new task progress log handler.
func NewTaskProgressLogHandler(svc service.TaskProgressLogApplicationService) *TaskProgressLogHandler {
	return &TaskProgressLogHandler{svc: svc}
}

// List returns task progress logs for a scan with AIP list pagination.
// GET /v1/scans/{scan}/taskProgressLogs?pageSize=200&pageToken=...
func (h *TaskProgressLogHandler) List(c *gin.Context) {
	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}
	if c.Query("cursor") != "" || c.Query("afterId") != "" || c.Query("limit") != "" {
		httpdto.Error(c, 400, "AIP_PAGING_REQUIRED", "use pageSize and pageToken for pagination")
		return
	}

	var query dto.TaskProgressLogListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	input, err := toTaskProgressLogQueryInput(&query)
	if err != nil {
		httpdto.BadRequest(c, err.Error())
		return
	}

	logs, hasMore, err := h.svc.ListByScanID(c.Request.Context(), scanID, input)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFound) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		httpdto.InternalError(c, "Failed to get task progress logs")
		return
	}

	httpdto.Success(c, toTaskProgressLogListOutput(scanID, logs, hasMore))
}
