package handler

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	service "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	"github.com/yyhuni/lunafox/server/internal/modules/snapshot/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/csv"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

// DirectorySnapshotHandler handles directory snapshot endpoints
type DirectorySnapshotHandler struct {
	svc *service.DirectorySnapshotFacade
}

type directorySnapshotListResponse struct {
	Results       []dto.DirectorySnapshotResponse `json:"results"`
	NextPageToken string                          `json:"nextPageToken"`
	TotalSize     int64                           `json:"totalSize"`
}

type directorySnapshotFilterOptionsQuery struct {
	Field string `form:"field" binding:"required"`
}

// NewDirectorySnapshotHandler creates a new directory snapshot handler
func NewDirectorySnapshotHandler(svc *service.DirectorySnapshotFacade) *DirectorySnapshotHandler {
	return &DirectorySnapshotHandler{svc: svc}
}

// BatchIngest creates directory snapshots and syncs to asset table
// POST /v1/scans/:scan/directories:batchIngest
func (h *DirectorySnapshotHandler) BatchIngest(c *gin.Context) {
	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	var req dto.BatchUpsertDirectorySnapshotsRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	targetID, err := httpdto.ParseResourceNameID(req.Target, "targets")
	if err != nil {
		httpdto.BadRequest(c, "Invalid target name")
		return
	}

	summary, err := h.svc.SaveAndSync(scanID, targetID, toDirectorySnapshotItemsInput(req.Directories))
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrTargetMismatch) {
			httpdto.BadRequest(c, "target does not match scan's target")
			return
		}
		httpdto.InternalError(c, "Failed to save directory snapshots")
		return
	}

	httpdto.Success(c, dto.BatchUpsertDirectorySnapshotsResponse{
		SnapshotCount: int(summary.SnapshotCount),
		AssetCount:    int(summary.AssetCount),
	})
}

// List returns paginated directory snapshots for a scan
// GET /v1/scans/:scan/directories
func (h *DirectorySnapshotHandler) List(c *gin.Context) {
	for _, legacyParam := range []string{"page", "sort", "sortBy", "sortOrder", "keyword"} {
		if _, ok := c.GetQuery(legacyParam); ok {
			httpdto.BadRequest(c, "Unsupported directory snapshot list query parameter: "+legacyParam)
			return
		}
	}

	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	var query dto.DirectorySnapshotListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	result, err := h.svc.ListByScan(scanID, service.DirectorySnapshotListQueryInput{
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
		if errors.Is(err, service.ErrUnsupportedDirectorySnapshotFilter) || errors.Is(err, service.ErrUnsupportedDirectorySnapshotOrderBy) || errors.Is(err, service.ErrInvalidDirectorySnapshotPageToken) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list directory snapshots")
		return
	}

	var resp []dto.DirectorySnapshotResponse
	for _, s := range result.Snapshots {
		resp = append(resp, toDirectorySnapshotOutput(&s))
	}

	httpdto.Success(c, directorySnapshotListResponse{
		Results:       resp,
		NextPageToken: result.NextPageToken,
		TotalSize:     result.TotalSize,
	})
}

// FilterOptions returns scan-scoped directory snapshot filter options.
// GET /v1/scans/:scan/directories/filterOptions?field=status
func (h *DirectorySnapshotHandler) FilterOptions(c *gin.Context) {
	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	var query directorySnapshotFilterOptionsQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	options, err := h.svc.ListFilterOptionsByScan(scanID, query.Field)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrUnsupportedDirectorySnapshotFilter) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list directory snapshot filter options")
		return
	}

	httpdto.Success(c, httpdto.FilterOptionsResponse{Results: toSnapshotFilterOptionDTOs(options)})
}

// Export exports directory snapshots as CSV
// GET /v1/scans/:scan/directories/exportFiles/current
func (h *DirectorySnapshotHandler) Export(c *gin.Context) {
	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	count, err := h.svc.CountByScan(scanID)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		httpdto.InternalError(c, "Failed to export directory snapshots")
		return
	}

	headers := []string{
		"id", "scan_id", "url", "status", "content_length",
		"content_type", "duration", "created_at",
	}
	filename := fmt.Sprintf("scan-%d-directories.csv", scanID)

	producer := func(write csv.RowWriter) error {
		return h.svc.ForEachByScan(scanID, func(snapshot service.DirectorySnapshot) error {
			status := ""
			if snapshot.Status != nil {
				status = strconv.Itoa(*snapshot.Status)
			}

			contentLength := ""
			if snapshot.ContentLength != nil {
				contentLength = strconv.FormatInt(*snapshot.ContentLength, 10)
			}

			duration := ""
			if snapshot.Duration != nil {
				duration = strconv.FormatInt(*snapshot.Duration, 10)
			}

			return write([]string{
				strconv.Itoa(snapshot.ID),
				strconv.Itoa(snapshot.ScanID),
				snapshot.URL,
				status,
				contentLength,
				snapshot.ContentType,
				duration,
				timeutil.FormatRFC3339NanoUTC(snapshot.CreatedAt),
			})
		})
	}

	if err := csv.StreamCSV(c, headers, filename, producer, count); err != nil {
		return
	}
}

func toDirectorySnapshotOutput(s *service.DirectorySnapshot) dto.DirectorySnapshotResponse {
	return dto.DirectorySnapshotResponse{
		ID:            s.ID,
		ScanID:        s.ScanID,
		Name:          httpdto.DirectorySnapshotName(s.ScanID, s.ID),
		URL:           s.URL,
		Status:        s.Status,
		ContentLength: directorySnapshotInt64String(s.ContentLength),
		ContentType:   s.ContentType,
		Duration:      directorySnapshotInt64String(s.Duration),
		CreatedAt:     timeutil.ToUTC(s.CreatedAt),
	}
}

func directorySnapshotInt64String(value *int64) *string {
	if value == nil {
		return nil
	}
	formatted := strconv.FormatInt(*value, 10)
	return &formatted
}
