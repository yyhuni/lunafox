package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/dto"
)

// ScanHandler handles scan HTTP requests.
type ScanHandler struct {
	svc *service.ScanFacade
}

// NewScanHandler creates a new scan handler.
func NewScanHandler(svc *service.ScanFacade) *ScanHandler {
	return &ScanHandler{svc: svc}
}

// List returns paginated scans.
// GET /api/scans
func (h *ScanHandler) List(c *gin.Context) {
	var query dto.ScanListQuery
	if !dto.BindQuery(c, &query) {
		return
	}

	scans, total, err := h.svc.List(toScanQueryInput(&query))
	if err != nil {
		dto.InternalError(c, "Failed to list scans")
		return
	}

	items := make([]dto.ScanResponse, len(scans))
	for i, scan := range scans {
		items[i] = toScanOutput(&scan)
	}

	dto.Paginated(c, items, total, query.GetPage(), query.GetPageSize())
}

// GetByID returns a scan by ID.
// GET /api/scans/:id
func (h *ScanHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	scan, err := h.svc.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFound) {
			dto.NotFound(c, "Scan not found")
			return
		}
		dto.InternalError(c, "Failed to get scan")
		return
	}

	dto.Success(c, toScanDetailOutput(scan))
}

// Statistics returns scan statistics.
// GET /api/scans/stats
func (h *ScanHandler) Statistics(c *gin.Context) {
	stats, err := h.svc.GetStatistics()
	if err != nil {
		dto.InternalError(c, "Failed to get scan statistics")
		return
	}

	dto.Success(c, toScanStatisticsOutput(stats))
}
