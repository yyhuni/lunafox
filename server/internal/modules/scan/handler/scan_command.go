package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/dto"
)

// Delete soft deletes a scan.
// DELETE /api/scans/:id
func (h *ScanHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	deletedCount, deletedNames, err := h.svc.Delete(id)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFound) {
			dto.NotFound(c, "Scan not found")
			return
		}
		dto.InternalError(c, "Failed to delete scan")
		return
	}

	dto.Success(c, gin.H{
		"scanId":       id,
		"deletedCount": deletedCount,
		"deletedScans": deletedNames,
	})
}

// BulkDelete soft deletes multiple scans.
// POST /api/scans/deletions
func (h *ScanHandler) BulkDelete(c *gin.Context) {
	var req dto.BulkDeleteRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	deletedCount, deletedNames, err := h.svc.BulkDelete(req.IDs)
	if err != nil {
		dto.InternalError(c, "Failed to bulk delete scans")
		return
	}

	dto.Success(c, gin.H{
		"deletedCount": deletedCount,
		"deletedScans": deletedNames,
	})
}

// HardDelete permanently deletes a scan (placeholder).
// DELETE /api/scans/:id/permanent
func (h *ScanHandler) HardDelete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	err = h.svc.HardDelete(id)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFound) {
			dto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrScanHardDeleteNotReady) {
			dto.Error(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Scan hard delete is not implemented yet")
			return
		}
		dto.InternalError(c, "Failed to hard delete scan")
		return
	}

	dto.Success(c, gin.H{"scanId": id})
}

// Stop stops a running scan.
// POST /api/scans/:id/stoppages
func (h *ScanHandler) Stop(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	revokedCount, err := h.svc.Stop(id)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFound) {
			dto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrScanCannotStop) {
			dto.BadRequest(c, "Cannot stop scan: scan is not running")
			return
		}
		dto.InternalError(c, "Failed to stop scan")
		return
	}

	dto.Success(c, dto.StopScanResponse{RevokedTaskCount: revokedCount})
}

// Create starts a new scan.
// POST /api/scans
func (h *ScanHandler) Create(c *gin.Context) {
	var req dto.CreateScanRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	if req.Mode == "" {
		req.Mode = "normal"
	}

	switch req.Mode {
	case "normal":
		if req.TargetID == 0 {
			dto.BadRequest(c, "targetId is required for normal mode")
			return
		}

		scan, err := h.svc.CreateNormal(toScanCreateNormalInput(&req))
		if err != nil {
			if workflowErr, ok := service.AsWorkflowError(err); ok {
				dto.ErrorWithContract(c, http.StatusBadRequest, workflowErr.Code, workflowErr.Stage, workflowErr.Field, workflowErr.Message)
				return
			}
			if errors.Is(err, service.ErrTargetNotFound) {
				dto.NotFound(c, "Target not found")
				return
			}
			if errors.Is(err, service.ErrScanInvalidConfig) ||
				errors.Is(err, service.ErrScanInvalidEngineNames) ||
				errors.Is(err, service.ErrScanNoWorkflows) {
				dto.BadRequest(c, err.Error())
				return
			}
			dto.InternalError(c, "Failed to create scan")
			return
		}

		dto.Created(c, toScanDetailOutput(scan))

	case "quick":
		if len(req.Targets) == 0 {
			dto.BadRequest(c, "targets is required for quick mode")
			return
		}
		dto.Error(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Quick scan is not yet implemented")

	default:
		dto.BadRequest(c, "Invalid mode, must be 'normal' or 'quick'")
	}
}
