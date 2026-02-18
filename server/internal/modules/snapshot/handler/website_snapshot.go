package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	"github.com/yyhuni/lunafox/server/internal/modules/snapshot/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/csv"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

// WebsiteSnapshotHandler handles website snapshot endpoints
type WebsiteSnapshotHandler struct {
	svc *service.WebsiteSnapshotFacade
}

// NewWebsiteSnapshotHandler creates a new website snapshot handler
func NewWebsiteSnapshotHandler(svc *service.WebsiteSnapshotFacade) *WebsiteSnapshotHandler {
	return &WebsiteSnapshotHandler{svc: svc}
}

// BulkUpsert creates website snapshots and syncs to asset table
// POST /api/scans/:id/websites/bulk-upsert
func (h *WebsiteSnapshotHandler) BulkUpsert(c *gin.Context) {
	scanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	var req dto.BulkUpsertWebsiteSnapshotsRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	snapshotCount, assetCount, err := h.svc.SaveAndSync(scanID, req.TargetID, toWebsiteSnapshotItemsInput(req.Websites))
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			dto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrTargetMismatch) {
			dto.BadRequest(c, "targetId does not match scan's target")
			return
		}
		dto.InternalError(c, "Failed to save snapshots")
		return
	}

	dto.Success(c, dto.BulkUpsertWebsiteSnapshotsResponse{
		SnapshotCount: int(snapshotCount),
		AssetCount:    int(assetCount),
	})
}

// List returns paginated website snapshots for a scan
// GET /api/scans/:id/websites/
func (h *WebsiteSnapshotHandler) List(c *gin.Context) {
	scanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	var query dto.WebsiteSnapshotListQuery
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

	// Convert to response
	var resp []dto.WebsiteSnapshotResponse
	for _, s := range snapshots {
		resp = append(resp, toWebsiteSnapshotOutput(&s))
	}

	dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}

// Export exports website snapshots as CSV
// GET /api/scans/:id/websites/export/
func (h *WebsiteSnapshotHandler) Export(c *gin.Context) {
	scanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	// Get count for progress estimation
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

	headers := []string{
		"url", "host", "location", "title", "status_code",
		"content_length", "content_type", "webserver", "tech",
		"response_body", "response_headers", "vhost", "created_at",
	}

	filename := fmt.Sprintf("scan-%d-websites.csv", scanID)

	mapper := func(rows *sql.Rows) ([]string, error) {
		snapshot, err := h.svc.ScanRow(rows)
		if err != nil {
			return nil, err
		}

		statusCode := ""
		if snapshot.StatusCode != nil {
			statusCode = strconv.Itoa(*snapshot.StatusCode)
		}

		contentLength := ""
		if snapshot.ContentLength != nil {
			contentLength = strconv.Itoa(*snapshot.ContentLength)
		}

		vhost := ""
		if snapshot.Vhost != nil {
			vhost = strconv.FormatBool(*snapshot.Vhost)
		}

		tech := ""
		if len(snapshot.Tech) > 0 {
			tech = strings.Join(snapshot.Tech, "|")
		}

		return []string{
			snapshot.URL,
			snapshot.Host,
			snapshot.Location,
			snapshot.Title,
			statusCode,
			contentLength,
			snapshot.ContentType,
			snapshot.Webserver,
			tech,
			snapshot.ResponseBody,
			snapshot.ResponseHeaders,
			vhost,
			timeutil.FormatRFC3339NanoUTC(snapshot.CreatedAt),
		}, nil
	}

	if err := csv.StreamCSV(c, rows, headers, filename, mapper, count); err != nil {
		return
	}
}

// toWebsiteSnapshotOutput converts model to output DTO
func toWebsiteSnapshotOutput(s *service.WebsiteSnapshot) dto.WebsiteSnapshotResponse {
	tech := s.Tech
	if tech == nil {
		tech = []string{}
	}
	return dto.WebsiteSnapshotResponse{
		ID:              s.ID,
		ScanID:          s.ScanID,
		URL:             s.URL,
		Host:            s.Host,
		Title:           s.Title,
		StatusCode:      s.StatusCode,
		ContentLength:   s.ContentLength,
		Location:        s.Location,
		Webserver:       s.Webserver,
		ContentType:     s.ContentType,
		Tech:            tech,
		ResponseBody:    s.ResponseBody,
		Vhost:           s.Vhost,
		ResponseHeaders: s.ResponseHeaders,
		CreatedAt:       timeutil.ToUTC(s.CreatedAt),
	}
}
