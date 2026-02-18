package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	"github.com/yyhuni/lunafox/server/internal/modules/snapshot/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

// ScreenshotSnapshotHandler handles screenshot snapshot endpoints
type ScreenshotSnapshotHandler struct {
	svc *service.ScreenshotSnapshotFacade
}

// NewScreenshotSnapshotHandler creates a new screenshot snapshot handler
func NewScreenshotSnapshotHandler(svc *service.ScreenshotSnapshotFacade) *ScreenshotSnapshotHandler {
	return &ScreenshotSnapshotHandler{svc: svc}
}

// BulkUpsert creates screenshot snapshots and syncs to asset table
// POST /api/scans/:id/screenshots/bulk-upsert
func (h *ScreenshotSnapshotHandler) BulkUpsert(c *gin.Context) {
	scanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	var req dto.BulkUpsertScreenshotSnapshotsRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	snapshotCount, assetCount, err := h.svc.SaveAndSync(scanID, req.TargetID, toScreenshotSnapshotItemsInput(req.Screenshots))
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			dto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrTargetMismatch) {
			dto.BadRequest(c, "targetId does not match scan's target")
			return
		}
		dto.InternalError(c, "Failed to save screenshot snapshots")
		return
	}

	dto.Success(c, dto.BulkUpsertScreenshotSnapshotsResponse{
		SnapshotCount: int(snapshotCount),
		AssetCount:    int(assetCount),
	})
}

// List returns paginated screenshot snapshots for a scan
// GET /api/scans/:id/screenshots
func (h *ScreenshotSnapshotHandler) List(c *gin.Context) {
	scanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	var query dto.ScreenshotSnapshotListQuery
	if !dto.BindQuery(c, &query) {
		return
	}

	snapshots, total, err := h.svc.ListByScan(scanID, toSnapshotListQueryInput(query.GetPage(), query.GetPageSize(), query.Filter))
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			dto.NotFound(c, "Scan not found")
			return
		}
		dto.InternalError(c, "Failed to list screenshot snapshots")
		return
	}

	// Convert to response (exclude image data)
	resp := make([]dto.ScreenshotSnapshotResponse, 0, len(snapshots))
	for _, s := range snapshots {
		resp = append(resp, toScreenshotSnapshotOutput(&s))
	}

	dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}

// GetImage returns screenshot snapshot image binary data
// GET /api/scans/:id/screenshots/:snapshotId/image
func (h *ScreenshotSnapshotHandler) GetImage(c *gin.Context) {
	scanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	snapshotID, err := strconv.Atoi(c.Param("snapshotId"))
	if err != nil {
		dto.BadRequest(c, "Invalid screenshot snapshot ID")
		return
	}

	snapshot, err := h.svc.GetByID(scanID, snapshotID)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			dto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrScreenshotSnapshotNotFound) {
			dto.NotFound(c, "Screenshot snapshot not found")
			return
		}
		dto.InternalError(c, "Failed to get screenshot snapshot")
		return
	}

	if len(snapshot.Image) == 0 {
		dto.NotFound(c, "Screenshot image not found")
		return
	}

	// Return WebP image
	c.Header("Content-Type", "image/webp")
	c.Header("Content-Disposition", "inline; filename=\"screenshot_snapshot_"+strconv.Itoa(snapshotID)+".webp\"")
	c.Data(200, "image/webp", snapshot.Image)
}

func toScreenshotSnapshotOutput(s *service.ScreenshotSnapshot) dto.ScreenshotSnapshotResponse {
	return dto.ScreenshotSnapshotResponse{
		ID:         s.ID,
		ScanID:     s.ScanID,
		URL:        s.URL,
		StatusCode: s.StatusCode,
		CreatedAt:  timeutil.ToUTC(s.CreatedAt),
	}
}
