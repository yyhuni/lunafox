package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/dto"
)

// ScanLogHandler handles scan log HTTP requests
type ScanLogHandler struct {
	svc service.ScanLogApplicationService
}

// NewScanLogHandler creates a new scan log handler
func NewScanLogHandler(svc service.ScanLogApplicationService) *ScanLogHandler {
	return &ScanLogHandler{svc: svc}
}

// List returns logs for a scan with afterId pagination
// GET /api/scans/:id/logs?afterId=123&limit=200
func (h *ScanLogHandler) List(c *gin.Context) {
	scanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}
	if c.Query("cursor") != "" {
		dto.Error(c, 400, "CURSOR_UNSUPPORTED_PARAM", "cursor is unsupported, use afterId")
		return
	}

	var query dto.ScanLogListQuery
	if !dto.BindQuery(c, &query) {
		return
	}

	logs, hasMore, err := h.svc.ListByScanID(c.Request.Context(), scanID, toScanLogQueryInput(&query))
	if err != nil {
		if errors.Is(err, service.ErrScanNotFound) {
			dto.NotFound(c, "Scan not found")
			return
		}
		dto.InternalError(c, "Failed to get scan logs")
		return
	}

	dto.Success(c, toScanLogListOutput(logs, hasMore))
}
