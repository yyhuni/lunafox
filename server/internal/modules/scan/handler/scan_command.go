package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/dto"
)

// Delete soft deletes a scan.
// DELETE /v1/scans/:scan
func (h *ScanHandler) Delete(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	deletedCount, deletedNames, err := h.svc.Delete(id)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFound) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		httpdto.InternalError(c, "Failed to delete scan")
		return
	}

	httpdto.Success(c, gin.H{
		"scanId":       id,
		"deletedCount": deletedCount,
		"deletedScans": deletedNames,
	})
}

// BatchDelete soft deletes multiple scans.
// POST /v1/scans:batchDelete
func (h *ScanHandler) BatchDelete(c *gin.Context) {
	var req dto.BatchDeleteRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	ids, err := httpdto.ParseResourceNameIDs(req.Names, "scans")
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan names")
		return
	}

	deletedCount, deletedNames, err := h.svc.BatchDelete(ids)
	if err != nil {
		httpdto.InternalError(c, "Failed to batch delete scans")
		return
	}

	httpdto.Success(c, gin.H{
		"deletedCount": deletedCount,
		"deletedScans": deletedNames,
	})
}

// BatchStop synchronously stops the selected active Scans.
// POST /v1/scans:batchStop
func (h *ScanHandler) BatchStop(c *gin.Context) {
	var req dto.BatchStopRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	ids, err := parseUniqueScanNames(req.Names)
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan names")
		return
	}

	result, err := h.svc.BatchStop(c.Request.Context(), ids)
	if err != nil {
		if httpdto.WriteContextError(c, err) {
			return
		}
		if errors.Is(err, service.ErrScanNotFound) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		httpdto.InternalError(c, "Failed to batch stop scans")
		return
	}

	httpdto.Success(c, dto.BatchStopResponse{
		StoppedCount:     result.StoppedCount,
		SkippedCount:     result.SkippedCount,
		RevokedTaskCount: result.RevokedTaskCount,
	})
}

func parseUniqueScanNames(names []string) ([]int, error) {
	if len(names) == 0 || len(names) > 100 {
		return nil, errors.New("scan batch stop requires between 1 and 100 names")
	}
	ids, err := httpdto.ParseResourceNameIDs(names, "scans")
	if err != nil {
		return nil, err
	}
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			return nil, errors.New("scan batch stop names must be unique")
		}
		seen[id] = struct{}{}
	}
	return ids, nil
}

// HardDelete permanently deletes a scan (placeholder).
// DELETE /v1/scans/:scan
func (h *ScanHandler) HardDelete(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	err = h.svc.HardDelete(id)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFound) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrScanHardDeleteNotReady) {
			httpdto.Error(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Scan hard delete is not implemented yet")
			return
		}
		httpdto.InternalError(c, "Failed to hard delete scan")
		return
	}

	httpdto.Success(c, gin.H{"scanId": id})
}

// Stop stops a running scan.
// POST /v1/scans/:scan:stop
func (h *ScanHandler) Stop(c *gin.Context) {
	id, err := parseScanStopSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	revokedCount, err := h.svc.Stop(c.Request.Context(), id)
	if err != nil {
		if httpdto.WriteContextError(c, err) {
			return
		}
		if errors.Is(err, service.ErrScanNotFound) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrScanCannotStop) {
			httpdto.BadRequest(c, "Cannot stop scan: scan is not running")
			return
		}
		httpdto.InternalError(c, "Failed to stop scan")
		return
	}

	httpdto.Success(c, dto.StopScanResponse{RevokedTaskCount: revokedCount})
}

func parseScanStopSegment(value string) (int, error) {
	idSegment, verb, ok := strings.Cut(strings.TrimSpace(value), ":")
	if !ok || verb != "stop" {
		return 0, errors.New("invalid scan custom method")
	}
	return httpdto.ParseResourceIDSegment(idSegment)
}

// CreateQuick starts a quick scan.
// POST /v1/scans:quickCreate
func (h *ScanHandler) CreateQuick(c *gin.Context) {
	var req dto.CreateQuickScanRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	if len(req.Targets) == 0 {
		httpdto.BadRequest(c, "targets is required for quick mode")
		return
	}

	input, err := toScanCreateQuickInput(&req)
	if err != nil {
		httpdto.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.CreateQuick(c.Request.Context(), input)
	if err != nil {
		if httpdto.WriteConfigResourceValidationError(c, err) {
			return
		}
		if httpdto.WriteWorkflowConfigurationError(c, err) {
			return
		}
		if taskExecutionErr, ok := service.AsTaskExecutionError(err); ok {
			httpdto.ErrorWithContract(c, http.StatusBadRequest, taskExecutionErr.Code, taskExecutionErr.Stage, taskExecutionErr.Field, taskExecutionErr.Message)
			return
		}
		if errors.Is(err, service.ErrNoTargetsForScan) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrCreateInvalidInputSource) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrScanAgentNotFound) {
			httpdto.NotFound(c, "Selected Agent not found")
			return
		}
		if errors.Is(err, service.ErrScanEngineConfigInvalid) {
			httpdto.Error(c, http.StatusBadRequest, "ENGINE_CONFIG_INVALID", "complete Engine configuration is invalid; reload the workflow Profile and review the configuration")
			return
		}
		if errors.Is(err, service.ErrScanWorkflowEngineUnavailable) {
			httpdto.Error(c, http.StatusBadRequest, "SCAN_WORKFLOW_ENGINE_UNAVAILABLE", "scan workflow references an unavailable Engine")
			return
		}
		if errors.Is(err, service.ErrScanInvalidConfig) ||
			errors.Is(err, service.ErrScanInvalidScanWorkflow) ||
			errors.Is(err, service.ErrScanNoScanWorkflows) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to create quick scan")
		return
	}

	httpdto.Created(c, toQuickScanOutput(result))
}

// BatchCreate starts normal scans for target or organization scopes.
// POST /v1/scans:batchCreate
func (h *ScanHandler) BatchCreate(c *gin.Context) {
	var req dto.BatchCreateScanRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	input, err := toScanBatchCreateInput(&req)
	if err != nil {
		httpdto.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.CreateBatch(c.Request.Context(), input)
	if err != nil {
		if httpdto.WriteConfigResourceValidationError(c, err) {
			return
		}
		if httpdto.WriteWorkflowConfigurationError(c, err) {
			return
		}
		if taskExecutionErr, ok := service.AsTaskExecutionError(err); ok {
			httpdto.ErrorWithContract(c, http.StatusBadRequest, taskExecutionErr.Code, taskExecutionErr.Stage, taskExecutionErr.Field, taskExecutionErr.Message)
			return
		}
		if errors.Is(err, service.ErrNoTargetsForScan) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrCreateInvalidInputSource) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrScanAgentNotFound) {
			httpdto.NotFound(c, "Selected Agent not found")
			return
		}
		if errors.Is(err, service.ErrScanEngineConfigInvalid) {
			httpdto.Error(c, http.StatusBadRequest, "ENGINE_CONFIG_INVALID", "complete Engine configuration is invalid; reload the workflow Profile and review the configuration")
			return
		}
		if errors.Is(err, service.ErrScanWorkflowEngineUnavailable) {
			httpdto.Error(c, http.StatusBadRequest, "SCAN_WORKFLOW_ENGINE_UNAVAILABLE", "scan workflow references an unavailable Engine")
			return
		}
		if errors.Is(err, service.ErrScanInvalidConfig) ||
			errors.Is(err, service.ErrScanInvalidScanWorkflow) ||
			errors.Is(err, service.ErrScanNoScanWorkflows) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to batch create scans")
		return
	}

	httpdto.Created(c, toBatchScanOutput(result))
}
