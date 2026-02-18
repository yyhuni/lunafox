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

// EndpointSnapshotHandler handles endpoint snapshot endpoints
type EndpointSnapshotHandler struct {
	svc *service.EndpointSnapshotFacade
}

// NewEndpointSnapshotHandler creates a new endpoint snapshot handler
func NewEndpointSnapshotHandler(svc *service.EndpointSnapshotFacade) *EndpointSnapshotHandler {
	return &EndpointSnapshotHandler{svc: svc}
}

// BulkUpsert creates endpoint snapshots and syncs to asset table
// POST /api/scans/:id/endpoints/bulk-upsert
func (h *EndpointSnapshotHandler) BulkUpsert(c *gin.Context) {
	scanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	var req dto.BulkUpsertEndpointSnapshotsRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	snapshotCount, assetCount, err := h.svc.SaveAndSync(scanID, req.TargetID, toEndpointSnapshotItemsInput(req.Endpoints))
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			dto.NotFound(c, "Scan not found")
			return
		}
		dto.InternalError(c, "Failed to save endpoint snapshots")
		return
	}

	dto.Success(c, dto.BulkUpsertEndpointSnapshotsResponse{
		SnapshotCount: int(snapshotCount),
		AssetCount:    int(assetCount),
	})
}

// List returns paginated endpoint snapshots for a scan
// GET /api/scans/:id/endpoints
func (h *EndpointSnapshotHandler) List(c *gin.Context) {
	scanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid scan ID")
		return
	}

	var query dto.EndpointSnapshotListQuery
	if !dto.BindQuery(c, &query) {
		return
	}

	snapshots, total, err := h.svc.ListByScan(scanID, toSnapshotListQueryInput(query.GetPage(), query.GetPageSize(), query.Filter))
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			dto.NotFound(c, "Scan not found")
			return
		}
		dto.InternalError(c, "Failed to list endpoint snapshots")
		return
	}

	var resp []dto.EndpointSnapshotResponse
	for _, s := range snapshots {
		resp = append(resp, toEndpointSnapshotOutput(&s))
	}

	dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}

// Export exports endpoint snapshots as CSV
// GET /api/scans/:id/endpoints/export
func (h *EndpointSnapshotHandler) Export(c *gin.Context) {
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
		dto.InternalError(c, "Failed to export endpoint snapshots")
		return
	}

	rows, err := h.svc.StreamByScan(scanID)
	if err != nil {
		dto.InternalError(c, "Failed to export endpoint snapshots")
		return
	}

	headers := []string{
		"id", "scan_id", "url", "host", "title", "status_code",
		"content_length", "content_type", "webserver", "tech",
		"created_at",
	}
	filename := fmt.Sprintf("scan-%d-endpoints.csv", scanID)

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

		tech := ""
		if len(snapshot.Tech) > 0 {
			tech = strings.Join(snapshot.Tech, "|")
		}

		return []string{
			strconv.Itoa(snapshot.ID),
			strconv.Itoa(snapshot.ScanID),
			snapshot.URL,
			snapshot.Host,
			snapshot.Title,
			statusCode,
			contentLength,
			snapshot.ContentType,
			snapshot.Webserver,
			tech,
			timeutil.FormatRFC3339NanoUTC(snapshot.CreatedAt),
		}, nil
	}

	if err := csv.StreamCSV(c, rows, headers, filename, mapper, count); err != nil {
		return
	}
}

func toEndpointSnapshotOutput(s *service.EndpointSnapshot) dto.EndpointSnapshotResponse {
	tech := []string(s.Tech)
	if tech == nil {
		tech = []string{}
	}
	return dto.EndpointSnapshotResponse{
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
