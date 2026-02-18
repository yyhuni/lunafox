package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	"github.com/yyhuni/lunafox/server/internal/modules/snapshot/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/csv"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

// SubdomainSnapshotHandler handles subdomain snapshot endpoints
type SubdomainSnapshotHandler struct {
	svc *service.SubdomainSnapshotFacade
}

// NewSubdomainSnapshotHandler creates a new subdomain snapshot handler
func NewSubdomainSnapshotHandler(svc *service.SubdomainSnapshotFacade) *SubdomainSnapshotHandler {
	return &SubdomainSnapshotHandler{svc: svc}
}

// BulkUpsert creates subdomain snapshots and syncs to asset table
// POST /api/scans/:id/subdomains/bulk-upsert
func (h *SubdomainSnapshotHandler) BulkUpsert(c *gin.Context) {
	scanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	var req dto.BulkUpsertSubdomainSnapshotsRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	snapshotCount, assetCount, err := h.svc.SaveAndSync(scanID, req.TargetID, toSubdomainSnapshotItemsInput(req.Subdomains))
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			dto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrTargetMismatch) {
			dto.BadRequest(c, "targetId does not match scan's target")
			return
		}
		if errors.Is(err, service.ErrInvalidTargetType) {
			dto.BadRequest(c, "Target type must be domain")
			return
		}
		dto.InternalError(c, "Failed to save snapshots")
		return
	}

	dto.Success(c, dto.BulkUpsertSubdomainSnapshotsResponse{
		SnapshotCount: int(snapshotCount),
		AssetCount:    int(assetCount),
	})
}

// List returns paginated subdomain snapshots for a scan
// GET /api/scans/:id/subdomains
func (h *SubdomainSnapshotHandler) List(c *gin.Context) {
	scanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	var query dto.SubdomainSnapshotListQuery
	if !dto.BindQuery(c, &query) {
		return
	}

	snapshots, total, err := h.svc.ListByScan(scanID, toSnapshotListQueryInput(query.GetPage(), query.GetPageSize(), query.Filter))
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			dto.NotFound(c, "Scan not found")
			return
		}
		dto.InternalError(c, "Failed to list snapshots")
		return
	}

	var resp []dto.SubdomainSnapshotResponse
	for _, s := range snapshots {
		resp = append(resp, toSubdomainSnapshotOutput(&s))
	}

	dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}

// Export exports subdomain snapshots as CSV
// GET /api/scans/:id/subdomains/export
func (h *SubdomainSnapshotHandler) Export(c *gin.Context) {
	scanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	count, err := h.svc.CountByScan(scanID)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			dto.NotFound(c, "Scan not found")
			return
		}
		dto.InternalError(c, "Failed to export snapshots")
		return
	}

	rows, err := h.svc.StreamByScan(scanID)
	if err != nil {
		dto.InternalError(c, "Failed to export snapshots")
		return
	}

	headers := []string{"id", "scan_id", "name", "created_at"}
	filename := fmt.Sprintf("scan-%d-subdomains.csv", scanID)

	mapper := func(rows *sql.Rows) ([]string, error) {
		snapshot, err := h.svc.ScanRow(rows)
		if err != nil {
			return nil, err
		}

		return []string{
			strconv.Itoa(snapshot.ID),
			strconv.Itoa(snapshot.ScanID),
			snapshot.Name,
			timeutil.FormatRFC3339NanoUTC(snapshot.CreatedAt),
		}, nil
	}

	if err := csv.StreamCSV(c, rows, headers, filename, mapper, count); err != nil {
		return
	}
}

// toSubdomainSnapshotOutput converts model to output DTO
func toSubdomainSnapshotOutput(s *service.SubdomainSnapshot) dto.SubdomainSnapshotResponse {
	return dto.SubdomainSnapshotResponse{
		ID:        s.ID,
		ScanID:    s.ScanID,
		Name:      s.Name,
		CreatedAt: timeutil.ToUTC(s.CreatedAt),
	}
}
