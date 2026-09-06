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

// SubdomainSnapshotHandler handles subdomain snapshot endpoints
type SubdomainSnapshotHandler struct {
	svc *service.SubdomainSnapshotFacade
}

type subdomainSnapshotListResponse struct {
	Results       []dto.SubdomainSnapshotResponse `json:"results"`
	NextPageToken string                          `json:"nextPageToken"`
	TotalSize     int64                           `json:"totalSize"`
}

// NewSubdomainSnapshotHandler creates a new subdomain snapshot handler
func NewSubdomainSnapshotHandler(svc *service.SubdomainSnapshotFacade) *SubdomainSnapshotHandler {
	return &SubdomainSnapshotHandler{svc: svc}
}

// BatchIngest creates subdomain snapshots and syncs to asset table
// POST /v1/scans/:scan/subdomains:batchIngest
func (h *SubdomainSnapshotHandler) BatchIngest(c *gin.Context) {
	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	var req dto.BatchUpsertSubdomainSnapshotsRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	targetID, err := httpdto.ParseResourceNameID(req.Target, "targets")
	if err != nil {
		httpdto.BadRequest(c, "Invalid target name")
		return
	}

	summary, err := h.svc.SaveAndSync(scanID, targetID, toSubdomainSnapshotItemsInput(req.Subdomains))
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		if errors.Is(err, service.ErrTargetMismatch) {
			httpdto.BadRequest(c, "target does not match scan's target")
			return
		}
		if errors.Is(err, service.ErrInvalidTargetType) {
			httpdto.BadRequest(c, "Target type must be domain")
			return
		}
		httpdto.InternalError(c, "Failed to save snapshots")
		return
	}

	httpdto.Success(c, dto.BatchUpsertSubdomainSnapshotsResponse{
		SnapshotCount: int(summary.SnapshotCount),
		AssetCount:    int(summary.AssetCount),
	})
}

// List returns paginated subdomain snapshots for a scan
// GET /v1/scans/:scan/subdomains
func (h *SubdomainSnapshotHandler) List(c *gin.Context) {
	for _, legacyParam := range []string{"page", "sort", "sortBy", "sortOrder", "keyword"} {
		if _, ok := c.GetQuery(legacyParam); ok {
			httpdto.BadRequest(c, "Unsupported subdomain snapshot list query parameter: "+legacyParam)
			return
		}
	}

	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	var query dto.SubdomainSnapshotListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	result, err := h.svc.ListByScan(scanID, service.SubdomainSnapshotListQueryInput{
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
		if errors.Is(err, service.ErrUnsupportedSubdomainSnapshotFilter) || errors.Is(err, service.ErrUnsupportedSubdomainSnapshotOrderBy) || errors.Is(err, service.ErrInvalidSubdomainSnapshotPageToken) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list snapshots")
		return
	}

	resp := make([]dto.SubdomainSnapshotResponse, 0, len(result.Subdomains))
	for _, s := range result.Subdomains {
		resp = append(resp, toSubdomainSnapshotOutput(&s))
	}

	httpdto.Success(c, subdomainSnapshotListResponse{
		Results:       resp,
		NextPageToken: result.NextPageToken,
		TotalSize:     result.TotalSize,
	})
}

// Export exports subdomain snapshots as CSV
// GET /v1/scans/:scan/subdomains/exportFiles/current
func (h *SubdomainSnapshotHandler) Export(c *gin.Context) {
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
		httpdto.InternalError(c, "Failed to export snapshots")
		return
	}

	headers := []string{"id", "scan_id", "dns_name", "created_at"}
	filename := fmt.Sprintf("scan-%d-subdomains.csv", scanID)

	producer := func(write csv.RowWriter) error {
		return h.svc.ForEachByScan(scanID, func(snapshot service.SubdomainSnapshot) error {
			return write([]string{
				strconv.Itoa(snapshot.ID),
				strconv.Itoa(snapshot.ScanID),
				snapshot.DNSName,
				timeutil.FormatRFC3339NanoUTC(snapshot.CreatedAt),
			})
		})
	}

	if err := csv.StreamCSV(c, headers, filename, producer, count); err != nil {
		return
	}
}

// toSubdomainSnapshotOutput converts model to output DTO
func toSubdomainSnapshotOutput(s *service.SubdomainSnapshot) dto.SubdomainSnapshotResponse {
	return dto.SubdomainSnapshotResponse{
		ID:        s.ID,
		ScanID:    s.ScanID,
		Name:      httpdto.SubdomainSnapshotName(s.ScanID, s.ID),
		DNSName:   s.DNSName,
		CreatedAt: timeutil.ToUTC(s.CreatedAt),
	}
}
