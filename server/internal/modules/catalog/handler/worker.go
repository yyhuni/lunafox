package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

// WorkerHandler handles worker API endpoints
type WorkerHandler struct {
	svc *service.WorkerProviderConfigService
}

// NewWorkerHandler creates a new worker handler
func NewWorkerHandler(svc *service.WorkerProviderConfigService) *WorkerHandler {
	return &WorkerHandler{svc: svc}
}

// GetProviderConfig returns provider config for a tool
// GET /api/worker/scans/:id/provider-config?tool=subfinder
func (h *WorkerHandler) GetProviderConfig(c *gin.Context) {
	scanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	toolName := c.Query("tool")
	config, err := h.svc.GetProviderConfig(scanID, toolName)
	if err != nil {
		if errors.Is(err, service.ErrWorkerScanNotFound) {
			dto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrWorkerToolRequired) {
			dto.BadRequest(c, "Tool parameter required")
			return
		}
		dto.InternalError(c, "Failed to get provider config")
		return
	}

	dto.Success(c, config)
}
