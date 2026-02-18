package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/dto"
)

// WorkerScanHandler handles worker-facing scan endpoints.
type WorkerScanHandler struct {
	svc *scanapp.ScanFacade
}

func NewWorkerScanHandler(svc *scanapp.ScanFacade) *WorkerScanHandler {
	return &WorkerScanHandler{svc: svc}
}

// GetTargetName returns target name for a scan.
// GET /api/worker/scans/:id/target
func (handler *WorkerScanHandler) GetTargetName(c *gin.Context) {
	scanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	target, err := handler.svc.GetTargetName(scanID)
	if err != nil {
		if errors.Is(err, scanapp.ErrScanNotFound) {
			dto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, scanapp.ErrScanTargetNotFound) {
			dto.NotFound(c, "Scan target not found")
			return
		}
		dto.InternalError(c, "Failed to get target name")
		return
	}

	dto.Success(c, toWorkerTargetNameOutput(target))
}
