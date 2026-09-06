package handler

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	service "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/snapshot/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/csv"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

// HostPortSnapshotHandler handles host-port snapshot endpoints
type HostPortSnapshotHandler struct {
	svc *service.HostPortSnapshotFacade
}

type hostPortSnapshotListResponse struct {
	Results       []dto.HostPortSnapshotAggregateResponse `json:"results"`
	NextPageToken string                                  `json:"nextPageToken"`
	TotalSize     int64                                   `json:"totalSize"`
}

type hostPortSnapshotFilterOptionsQuery struct {
	Field string `form:"field" binding:"required"`
}

// NewHostPortSnapshotHandler creates a new host-port snapshot handler
func NewHostPortSnapshotHandler(svc *service.HostPortSnapshotFacade) *HostPortSnapshotHandler {
	return &HostPortSnapshotHandler{svc: svc}
}

// BatchIngest creates host-port snapshots and syncs to asset table
// POST /v1/scans/:scan/hostPorts:batchIngest
func (h *HostPortSnapshotHandler) BatchIngest(c *gin.Context) {
	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	var req dto.BatchUpsertHostPortSnapshotsRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	targetID, err := httpdto.ParseResourceNameID(req.Target, "targets")
	if err != nil {
		httpdto.BadRequest(c, "Invalid target name")
		return
	}

	summary, err := h.svc.SaveAndSync(scanID, targetID, toHostPortSnapshotItemsInput(req.HostPorts))
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

	httpdto.Success(c, dto.BatchUpsertHostPortSnapshotsResponse{
		SnapshotCount: int(summary.SnapshotCount),
		AssetCount:    int(summary.AssetCount),
	})
}

// List returns paginated host-port snapshots for a scan
// GET /v1/scans/:scan/hostPorts
func (h *HostPortSnapshotHandler) List(c *gin.Context) {
	for _, legacyParam := range []string{"page", "sort", "sortBy", "sortOrder", "keyword"} {
		if _, ok := c.GetQuery(legacyParam); ok {
			httpdto.BadRequest(c, "Unsupported hostPort snapshot list query parameter: "+legacyParam)
			return
		}
	}

	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	var query dto.HostPortSnapshotListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	result, err := h.svc.ListByScan(scanID, service.HostPortSnapshotListQueryInput{
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
		if errors.Is(err, service.ErrUnsupportedHostPortSnapshotFilter) || errors.Is(err, service.ErrUnsupportedHostPortSnapshotOrderBy) || errors.Is(err, service.ErrInvalidHostPortSnapshotPageToken) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list snapshots")
		return
	}

	resp := make([]dto.HostPortSnapshotAggregateResponse, 0, len(result.HostPorts))
	for _, s := range result.HostPorts {
		resp = append(resp, toHostPortSnapshotAggregateOutput(scanID, s))
	}

	httpdto.Success(c, hostPortSnapshotListResponse{
		Results:       resp,
		NextPageToken: result.NextPageToken,
		TotalSize:     result.TotalSize,
	})
}

// FilterOptions returns scan-scoped hostPort filter options.
// GET /v1/scans/:scan/hostPorts/filterOptions?field=port
func (h *HostPortSnapshotHandler) FilterOptions(c *gin.Context) {
	scanID, err := httpdto.ParseResourceIDSegment(c.Param("scan"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid scan ID")
		return
	}

	var query hostPortSnapshotFilterOptionsQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}
	if query.Field != "port" {
		httpdto.BadRequest(c, "unsupported hostPort snapshot filter option field")
		return
	}

	options, err := h.svc.ListPortOptionsByScan(scanID)
	if err != nil {
		if errors.Is(err, service.ErrScanNotFoundForSnapshot) {
			httpdto.NotFound(c, "Scan not found")
			return
		}
		httpdto.InternalError(c, "Failed to list hostPort snapshot filter options")
		return
	}

	httpdto.Success(c, httpdto.FilterOptionsResponse{Results: toHostPortSnapshotFilterOptionDTOs(options)})
}

func toHostPortSnapshotFilterOptionDTOs(options []snapshotdomain.FilterOption) []httpdto.FilterOption {
	results := make([]httpdto.FilterOption, 0, len(options))
	for _, option := range options {
		results = append(results, httpdto.FilterOption{Value: option.Value, Label: option.Label, Count: option.Count})
	}
	return results
}

// Export exports host-port snapshots as CSV
// GET /v1/scans/:scan/hostPorts/exportFiles/current
func (h *HostPortSnapshotHandler) Export(c *gin.Context) {
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

	headers := []string{"id", "scan_id", "host", "ip", "port", "created_at"}
	filename := fmt.Sprintf("scan-%d-hostPorts.csv", scanID)

	producer := func(write csv.RowWriter) error {
		return h.svc.ForEachByScan(scanID, func(snapshot service.HostPortSnapshot) error {
			return write([]string{
				strconv.Itoa(snapshot.ID),
				strconv.Itoa(snapshot.ScanID),
				snapshot.Host,
				snapshot.IP,
				strconv.Itoa(snapshot.Port),
				timeutil.FormatRFC3339NanoUTC(snapshot.CreatedAt),
			})
		})
	}

	if err := csv.StreamCSV(c, headers, filename, producer, count); err != nil {
		return
	}
}

// toHostPortSnapshotOutput converts model to output DTO
func toHostPortSnapshotOutput(s *service.HostPortSnapshot) dto.HostPortSnapshotResponse {
	return dto.HostPortSnapshotResponse{
		ID:        s.ID,
		ScanID:    s.ScanID,
		Name:      httpdto.HostPortSnapshotName(s.ScanID, s.ID),
		Host:      s.Host,
		IP:        s.IP,
		Port:      s.Port,
		CreatedAt: timeutil.ToUTC(s.CreatedAt),
	}
}

func toHostPortSnapshotAggregateOutput(scanID int, s service.HostPortSnapshotAggregate) dto.HostPortSnapshotAggregateResponse {
	hosts := s.Hosts
	if hosts == nil {
		hosts = []string{}
	}
	ports := s.Ports
	if ports == nil {
		ports = []int{}
	}
	return dto.HostPortSnapshotAggregateResponse{
		Name:      httpdto.HostPortSnapshotIPName(scanID, s.IP),
		IP:        s.IP,
		Hosts:     hosts,
		Ports:     ports,
		CreatedAt: timeutil.ToUTC(s.CreatedAt),
	}
}
