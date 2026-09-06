package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	service "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/snapshot/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

// ScreenshotSnapshotHandler handles screenshot snapshot endpoints
type ScreenshotSnapshotHandler struct {
	svc *service.ScreenshotSnapshotFacade
}

type screenshotSnapshotListResponse struct {
	Results       []dto.ScreenshotSnapshotResponse `json:"results"`
	NextPageToken string                           `json:"nextPageToken"`
	TotalSize     int64                            `json:"totalSize"`
}

type screenshotSnapshotFilterOptionsQuery struct {
	Field string `form:"field" binding:"required"`
}

// NewScreenshotSnapshotHandler creates a new screenshot snapshot handler
func NewScreenshotSnapshotHandler(svc *service.ScreenshotSnapshotFacade) *ScreenshotSnapshotHandler {
	return &ScreenshotSnapshotHandler{svc: svc}
}

// BatchIngest creates screenshot snapshots and syncs to asset table
// POST /v1/scans/:scan/screenshots:batchIngest
func (h *ScreenshotSnapshotHandler) BatchIngest(c *gin.Context) {
	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	var req dto.BatchUpsertScreenshotSnapshotsRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	targetID, err := httpdto.ParseResourceNameID(req.Target, "targets")
	if err != nil {
		httpdto.BadRequest(c, "Invalid target name")
		return
	}

	summary, err := h.svc.SaveAndSync(scanID, targetID, toScreenshotSnapshotItemsInput(req.Screenshots))
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrTargetMismatch) {
			httpdto.BadRequest(c, "target does not match scan's target")
			return
		}
		httpdto.InternalError(c, "Failed to save screenshot snapshots")
		return
	}

	httpdto.Success(c, dto.BatchUpsertScreenshotSnapshotsResponse{
		SnapshotCount: int(summary.SnapshotCount),
		AssetCount:    int(summary.AssetCount),
	})
}

// List returns paginated screenshot snapshots for a scan
// GET /v1/scans/:scan/screenshots
func (h *ScreenshotSnapshotHandler) List(c *gin.Context) {
	for _, legacyParam := range []string{"page", "sort", "sortBy", "sortOrder", "keyword"} {
		if _, ok := c.GetQuery(legacyParam); ok {
			httpdto.BadRequest(c, "Unsupported screenshot snapshot list query parameter: "+legacyParam)
			return
		}
	}

	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	var query dto.ScreenshotSnapshotListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	result, err := h.svc.ListByScan(scanID, service.ScreenshotSnapshotListQueryInput{
		PageSize:  query.GetPageSize(),
		PageToken: query.PageToken,
		Filter:    query.Filter,
		OrderBy:   query.OrderBy,
	})
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrUnsupportedScreenshotSnapshotFilter) || errors.Is(err, service.ErrUnsupportedScreenshotSnapshotOrderBy) || errors.Is(err, service.ErrInvalidScreenshotSnapshotPageToken) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list screenshot snapshots")
		return
	}

	// Convert to response (exclude image data)
	resp := make([]dto.ScreenshotSnapshotResponse, 0, len(result.Snapshots))
	for _, s := range result.Snapshots {
		resp = append(resp, toScreenshotSnapshotOutput(&s))
	}

	httpdto.Success(c, screenshotSnapshotListResponse{
		Results:       resp,
		NextPageToken: result.NextPageToken,
		TotalSize:     result.TotalSize,
	})
}

// FilterOptions returns scan-scoped screenshot snapshot filter options.
// GET /v1/scans/:scan/screenshots/filterOptions?field=statusCode
func (h *ScreenshotSnapshotHandler) FilterOptions(c *gin.Context) {
	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	var query screenshotSnapshotFilterOptionsQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	options, err := h.svc.ListFilterOptionsByScan(scanID, query.Field)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrUnsupportedScreenshotSnapshotFilter) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list screenshot snapshot filter options")
		return
	}

	httpdto.Success(c, httpdto.FilterOptionsResponse{Results: toScreenshotSnapshotFilterOptionDTOs(options)})
}

func toScreenshotSnapshotFilterOptionDTOs(options []snapshotdomain.FilterOption) []httpdto.FilterOption {
	results := make([]httpdto.FilterOption, 0, len(options))
	for _, option := range options {
		results = append(results, httpdto.FilterOption{Value: option.Value, Label: option.Label, Count: option.Count})
	}
	return results
}

// GetImage returns screenshot snapshot image binary data
// GET /v1/scans/:scan/screenshotSnapshots/:screenshot_snapshot/blob
func (h *ScreenshotSnapshotHandler) GetImage(c *gin.Context) {
	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	snapshotID, err := httpdto.ParseResourceIDSegment(c.Param("screenshot_snapshot"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid screenshot snapshot ID")
		return
	}

	snapshot, err := h.svc.GetByID(scanID, snapshotID)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrScreenshotSnapshotNotFound) {
			httpdto.NotFound(c, "Screenshot snapshot not found")
			return
		}
		httpdto.InternalError(c, "Failed to get screenshot snapshot")
		return
	}

	if len(snapshot.Image) == 0 {
		httpdto.NotFound(c, "Screenshot image not found")
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
		Name:       httpdto.ScreenshotSnapshotName(s.ScanID, s.ID),
		URL:        s.URL,
		StatusCode: s.StatusCode,
		CreatedAt:  timeutil.ToUTC(s.CreatedAt),
	}
}
