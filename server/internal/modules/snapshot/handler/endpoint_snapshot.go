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

// EndpointSnapshotHandler handles endpoint snapshot endpoints
type EndpointSnapshotHandler struct {
	svc *service.EndpointSnapshotFacade
}

type endpointSnapshotListResponse struct {
	Results       []dto.EndpointSnapshotResponse `json:"results"`
	NextPageToken string                         `json:"nextPageToken"`
	TotalSize     int64                          `json:"totalSize"`
}

type endpointSnapshotFilterOptionsQuery struct {
	Field string `form:"field" binding:"required"`
}

// NewEndpointSnapshotHandler creates a new endpoint snapshot handler
func NewEndpointSnapshotHandler(svc *service.EndpointSnapshotFacade) *EndpointSnapshotHandler {
	return &EndpointSnapshotHandler{svc: svc}
}

// BatchIngest creates endpoint snapshots and syncs to asset table
// POST /v1/scans/:scan/endpoints:batchIngest
func (h *EndpointSnapshotHandler) BatchIngest(c *gin.Context) {
	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	var req dto.BatchUpsertEndpointSnapshotsRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	targetID, err := httpdto.ParseResourceNameID(req.Target, "targets")
	if err != nil {
		httpdto.BadRequest(c, "Invalid target name")
		return
	}

	summary, err := h.svc.SaveAndSync(scanID, targetID, toEndpointSnapshotItemsInput(req.Endpoints))
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		httpdto.InternalError(c, "Failed to save endpoint snapshots")
		return
	}

	httpdto.Success(c, dto.BatchUpsertEndpointSnapshotsResponse{
		SnapshotCount: int(summary.SnapshotCount),
		AssetCount:    int(summary.AssetCount),
	})
}

// List returns paginated endpoint snapshots for a scan
// GET /v1/scans/:scan/endpoints
func (h *EndpointSnapshotHandler) List(c *gin.Context) {
	for _, legacyParam := range []string{"page", "sort", "sortBy", "sortOrder", "keyword"} {
		if _, ok := c.GetQuery(legacyParam); ok {
			httpdto.BadRequest(c, "Unsupported endpoint snapshot list query parameter: "+legacyParam)
			return
		}
	}

	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	var query dto.EndpointSnapshotListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	result, err := h.svc.ListByScan(scanID, service.EndpointSnapshotListQueryInput{
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
		if errors.Is(err, service.ErrUnsupportedEndpointSnapshotFilter) || errors.Is(err, service.ErrUnsupportedEndpointSnapshotOrderBy) || errors.Is(err, service.ErrInvalidEndpointSnapshotPageToken) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list endpoint snapshots")
		return
	}

	var resp []dto.EndpointSnapshotResponse
	for _, s := range result.Snapshots {
		resp = append(resp, toEndpointSnapshotOutput(&s))
	}

	httpdto.Success(c, endpointSnapshotListResponse{
		Results:       resp,
		NextPageToken: result.NextPageToken,
		TotalSize:     result.TotalSize,
	})
}

// FilterOptions returns scan-scoped endpoint snapshot filter options.
// GET /v1/scans/:scan/endpoints/filterOptions?field=tech
func (h *EndpointSnapshotHandler) FilterOptions(c *gin.Context) {
	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	var query endpointSnapshotFilterOptionsQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	options, err := h.svc.ListFilterOptionsByScan(scanID, query.Field)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrUnsupportedEndpointSnapshotFilter) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list endpoint snapshot filter options")
		return
	}

	httpdto.Success(c, httpdto.FilterOptionsResponse{Results: toEndpointSnapshotFilterOptionDTOs(options)})
}

func toEndpointSnapshotFilterOptionDTOs(options []snapshotdomain.FilterOption) []httpdto.FilterOption {
	results := make([]httpdto.FilterOption, 0, len(options))
	for _, option := range options {
		results = append(results, httpdto.FilterOption{Value: option.Value, Label: option.Label, Count: option.Count})
	}
	return results
}

// Export exports endpoint snapshots as CSV
// GET /v1/scans/:scan/endpoints/exportFiles/current
func (h *EndpointSnapshotHandler) Export(c *gin.Context) {
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
		httpdto.InternalError(c, "Failed to export endpoint snapshots")
		return
	}

	headers := []string{
		"id", "scan_id", "url", "host", "title", "status_code",
		"content_length", "content_type", "webserver", "tech",
		"created_at",
	}
	filename := fmt.Sprintf("scan-%d-endpoints.csv", scanID)

	producer := func(write csv.RowWriter) error {
		return h.svc.ForEachByScan(scanID, func(snapshot service.EndpointSnapshot) error {
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

			return write([]string{
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
			})
		})
	}

	if err := csv.StreamCSV(c, headers, filename, producer, count); err != nil {
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
		Name:            httpdto.EndpointSnapshotName(s.ScanID, s.ID),
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
