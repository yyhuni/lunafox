package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/dto"
)

// BulkCreate creates multiple logs for a scan (for worker to write logs)
// POST /api/scans/:id/logs
func (h *ScanLogHandler) BulkCreate(c *gin.Context) {
	scanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	var req dto.BulkCreateScanLogsRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	createdCount, err := h.svc.BulkCreate(c.Request.Context(), toScanLogCreateInput(scanID, &req))
	if err != nil {
		if errors.Is(err, service.ErrScanNotFound) {
			dto.NotFound(c, "Scan not found")
			return
		}
		dto.InternalError(c, "Failed to create scan logs")
		return
	}

	dto.Created(c, dto.BulkCreateScanLogsResponse{CreatedCount: createdCount})
}
