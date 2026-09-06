package handler

import (
	"errors"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/dto"
)

// ScanHandler handles scan HTTP requests.
type ScanHandler struct {
	svc             *service.ScanFacade
	retentionPolicy ScanHistoryRetentionPolicy
}

// ScanHistoryRetentionPolicy is the safe, read-only subset of retention
// configuration that scan-history users need to understand data availability.
type ScanHistoryRetentionPolicy struct {
	MinimumRetentionSeconds int64
	AutomaticCleanupEnabled bool
}

// NewScanHandler creates a new scan handler.
func NewScanHandler(svc *service.ScanFacade, retentionPolicy ScanHistoryRetentionPolicy) *ScanHandler {
	return &ScanHandler{svc: svc, retentionPolicy: retentionPolicy}
}

// List returns paginated scans.
// GET /v1/scans
func (h *ScanHandler) List(c *gin.Context) {
	var query dto.ScanListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	scans, total, err := h.svc.List(toScanQueryInput(&query))
	if err != nil {
		if errors.Is(err, service.ErrUnsupportedScanFilter) || errors.Is(err, service.ErrUnsupportedScanOrderBy) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list scans")
		return
	}

	items := make([]dto.ScanResponse, len(scans))
	for i, scan := range scans {
		items[i] = toScanOutput(&scan)
	}

	httpdto.Paginated(c, items, total, query.GetPage(), query.GetPageSize())
}

// GetByID returns a scan by ID.
// GET /v1/scans/:scan
func (h *ScanHandler) GetByID(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	scan, err := h.svc.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFound) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		httpdto.InternalError(c, "Failed to get scan")
		return
	}

	httpdto.Success(c, toScanDetailOutput(scan))
}

// Statistics returns scan statistics.
// GET /v1/scanStatistics
func (h *ScanHandler) Statistics(c *gin.Context) {
	stats, err := h.svc.GetGlobalStatsSummary()
	if err != nil {
		httpdto.InternalError(c, "Failed to get scan statistics")
		return
	}

	httpdto.Success(c, toScanStatisticsOutput(stats, h.retentionPolicy))
}
