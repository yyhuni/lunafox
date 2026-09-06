package handler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	service "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/snapshot/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/csv"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

// WebsiteSnapshotHandler handles website snapshot endpoints
type WebsiteSnapshotHandler struct {
	svc *service.WebsiteSnapshotFacade
}

type websiteSnapshotListResponse struct {
	Results       []dto.WebsiteSnapshotResponse `json:"results"`
	NextPageToken string                        `json:"nextPageToken"`
	TotalSize     int64                         `json:"totalSize"`
}

type websiteSnapshotFilterOptionsQuery struct {
	Field string `form:"field" binding:"required"`
}

// NewWebsiteSnapshotHandler creates a new website snapshot handler
func NewWebsiteSnapshotHandler(svc *service.WebsiteSnapshotFacade) *WebsiteSnapshotHandler {
	return &WebsiteSnapshotHandler{svc: svc}
}

// BatchIngest creates website snapshots and syncs to asset table
// POST /v1/scans/:scan/websites:batchIngest
func (h *WebsiteSnapshotHandler) BatchIngest(c *gin.Context) {
	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	var req dto.BatchUpsertWebsiteSnapshotsRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	targetID, err := httpdto.ParseResourceNameID(req.Target, "targets")
	if err != nil {
		httpdto.BadRequest(c, "Invalid target name")
		return
	}

	summary, err := h.svc.SaveAndSync(scanID, targetID, toWebsiteSnapshotItemsInput(req.Websites))
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrTargetMismatch) {
			httpdto.BadRequest(c, "target does not match scan's target")
			return
		}
		httpdto.InternalError(c, "Failed to save snapshots")
		return
	}

	httpdto.Success(c, dto.BatchUpsertWebsiteSnapshotsResponse{
		SnapshotCount: int(summary.SnapshotCount),
		AssetCount:    int(summary.AssetCount),
	})
}

// List returns paginated website snapshots for a scan
// GET /v1/scans/:scan/websites
func (h *WebsiteSnapshotHandler) List(c *gin.Context) {
	for _, legacyParam := range []string{"page", "sort", "sortBy", "sortOrder", "keyword"} {
		if _, ok := c.GetQuery(legacyParam); ok {
			httpdto.BadRequest(c, "Unsupported website snapshot list query parameter: "+legacyParam)
			return
		}
	}

	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	var query dto.WebsiteSnapshotListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	result, err := h.svc.ListByScan(scanID, service.WebsiteSnapshotListQueryInput{
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
		if errors.Is(err, service.ErrUnsupportedWebsiteSnapshotFilter) || errors.Is(err, service.ErrUnsupportedWebsiteSnapshotOrderBy) || errors.Is(err, service.ErrInvalidWebsiteSnapshotPageToken) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list snapshots")
		return
	}

	// Convert to response
	var resp []dto.WebsiteSnapshotResponse
	for _, s := range result.Snapshots {
		resp = append(resp, toWebsiteSnapshotOutput(&s))
	}

	httpdto.Success(c, websiteSnapshotListResponse{
		Results:       resp,
		NextPageToken: result.NextPageToken,
		TotalSize:     result.TotalSize,
	})
}

// FilterOptions returns scan-scoped website snapshot filter options.
// GET /v1/scans/:scan/websites/filterOptions?field=tech
func (h *WebsiteSnapshotHandler) FilterOptions(c *gin.Context) {
	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	var query websiteSnapshotFilterOptionsQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	options, err := h.svc.ListFilterOptionsByScan(scanID, query.Field)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrUnsupportedWebsiteSnapshotFilter) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list website snapshot filter options")
		return
	}

	httpdto.Success(c, httpdto.FilterOptionsResponse{Results: toSnapshotFilterOptionDTOs(options)})
}

func toSnapshotFilterOptionDTOs(options []snapshotdomain.FilterOption) []httpdto.FilterOption {
	results := make([]httpdto.FilterOption, 0, len(options))
	for _, option := range options {
		results = append(results, httpdto.FilterOption{Value: option.Value, Label: option.Label, Count: option.Count})
	}
	return results
}

// Export exports website snapshots as CSV
// GET /v1/scans/:scan/websites/exportFiles/current
func (h *WebsiteSnapshotHandler) Export(c *gin.Context) {
	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	// Get count for progress estimation
	count, err := h.svc.CountByScan(scanID)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		httpdto.InternalError(c, "Failed to export snapshots")
		return
	}

	headers := []string{
		"url", "host", "location", "title", "status_code",
		"content_length", "content_type", "webserver", "tech",
		"response_body", "response_headers", "vhost", "created_at",
	}

	filename := fmt.Sprintf("scan-%d-websites.csv", scanID)

	producer := func(write csv.RowWriter) error {
		return h.svc.ForEachByScan(scanID, func(snapshot service.WebsiteSnapshot) error {
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

			return write([]string{
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
			})
		})
	}

	if err := csv.StreamCSV(c, headers, filename, producer, count); err != nil {
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
		Name:            httpdto.WebsiteSnapshotName(s.ScanID, s.ID),
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
