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

// DirectorySnapshotHandler handles directory snapshot endpoints
type DirectorySnapshotHandler struct {
	svc *service.DirectorySnapshotFacade
}

// NewDirectorySnapshotHandler creates a new directory snapshot handler
func NewDirectorySnapshotHandler(svc *service.DirectorySnapshotFacade) *DirectorySnapshotHandler {
	return &DirectorySnapshotHandler{svc: svc}
}

// BulkUpsert creates directory snapshots and syncs to asset table
// POST /api/scans/:id/directories/bulk-upsert
func (h *DirectorySnapshotHandler) BulkUpsert(c *gin.Context) {
	scanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	var req dto.BulkUpsertDirectorySnapshotsRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	snapshotCount, assetCount, err := h.svc.SaveAndSync(scanID, req.TargetID, toDirectorySnapshotItemsInput(req.Directories))
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			dto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrTargetMismatch) {
			dto.BadRequest(c, "targetId does not match scan's target")
			return
		}
		dto.InternalError(c, "Failed to save directory snapshots")
		return
	}

	dto.Success(c, dto.BulkUpsertDirectorySnapshotsResponse{
		SnapshotCount: int(snapshotCount),
		AssetCount:    int(assetCount),
	})
}

// List returns paginated directory snapshots for a scan
// GET /api/scans/:id/directories
func (h *DirectorySnapshotHandler) List(c *gin.Context) {
	scanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	var query dto.DirectorySnapshotListQuery
	if !dto.BindQuery(c, &query) {
		return
	}

	snapshots, total, err := h.svc.ListByScan(scanID, toSnapshotListQueryInput(query.GetPage(), query.GetPageSize(), query.Filter))
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			dto.NotFound(c, "Scan not found")
			return
		}
		dto.InternalError(c, "Failed to list directory snapshots")
		return
	}

	var resp []dto.DirectorySnapshotResponse
	for _, s := range snapshots {
		resp = append(resp, toDirectorySnapshotOutput(&s))
	}

	dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}

// Export exports directory snapshots as CSV
// GET /api/scans/:id/directories/export
func (h *DirectorySnapshotHandler) Export(c *gin.Context) {
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
		dto.InternalError(c, "Failed to export directory snapshots")
		return
	}

	rows, err := h.svc.StreamByScan(scanID)
	if err != nil {
		dto.InternalError(c, "Failed to export directory snapshots")
		return
	}

	headers := []string{
		"id", "scan_id", "url", "status", "content_length",
		"content_type", "duration", "created_at",
	}
	filename := fmt.Sprintf("scan-%d-directories.csv", scanID)

	mapper := func(rows *sql.Rows) ([]string, error) {
		snapshot, err := h.svc.ScanRow(rows)
		if err != nil {
			return nil, err
		}

		status := ""
		if snapshot.Status != nil {
			status = strconv.Itoa(*snapshot.Status)
		}

		contentLength := ""
		if snapshot.ContentLength != nil {
			contentLength = strconv.Itoa(*snapshot.ContentLength)
		}

		duration := ""
		if snapshot.Duration != nil {
			duration = strconv.Itoa(*snapshot.Duration)
		}

		return []string{
			strconv.Itoa(snapshot.ID),
			strconv.Itoa(snapshot.ScanID),
			snapshot.URL,
			status,
			contentLength,
			snapshot.ContentType,
			duration,
			timeutil.FormatRFC3339NanoUTC(snapshot.CreatedAt),
		}, nil
	}

	if err := csv.StreamCSV(c, rows, headers, filename, mapper, count); err != nil {
		return
	}
}

func toDirectorySnapshotOutput(s *service.DirectorySnapshot) dto.DirectorySnapshotResponse {
	return dto.DirectorySnapshotResponse{
		ID:            s.ID,
		ScanID:        s.ScanID,
		URL:           s.URL,
		Status:        s.Status,
		ContentLength: s.ContentLength,
		ContentType:   s.ContentType,
		Duration:      s.Duration,
		CreatedAt:     timeutil.ToUTC(s.CreatedAt),
	}
}
