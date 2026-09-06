package hostport

import (
	"errors"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type hostPortListResponse struct {
	Results       []dto.HostPortResponse `json:"results"`
	NextPageToken string                 `json:"nextPageToken"`
	TotalSize     int64                  `json:"totalSize"`
}

type hostPortFilterOptionsQuery struct {
	Field string `form:"field" binding:"required"`
}

// List returns paginated hostPorts aggregated by IP.
// GET /v1/targets/:target/hostPorts
func (h *HostPortHandler) List(c *gin.Context) {
	for _, legacyParam := range []string{"page", "sort", "sortBy", "sortOrder", "keyword"} {
		if _, ok := c.GetQuery(legacyParam); ok {
			httpdto.BadRequest(c, "Unsupported hostPort list query parameter: "+legacyParam)
			return
		}
	}

	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	var query dto.HostPortListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	result, err := h.svc.ListByTarget(targetID, service.HostPortListQueryInput{
		PageSize:  query.GetPageSize(),
		PageToken: query.PageToken,
		Filter:    query.Filter,
		OrderBy:   query.OrderBy,
	})
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			httpdto.NotFound(c, "Target not found")
			return
		}
		if errors.Is(err, service.ErrUnsupportedHostPortFilter) || errors.Is(err, service.ErrUnsupportedHostPortOrderBy) || errors.Is(err, service.ErrInvalidHostPortPageToken) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list hostPorts")
		return
	}

	resp := make([]dto.HostPortResponse, 0, len(result.HostPorts))
	for index := range result.HostPorts {
		item := result.HostPorts[index]
		resp = append(resp, toHostPortOutput(targetID, item))
	}

	httpdto.Success(c, hostPortListResponse{
		Results:       resp,
		NextPageToken: result.NextPageToken,
		TotalSize:     result.TotalSize,
	})
}

// FilterOptions returns parent-scoped hostPort filter options.
// GET /v1/targets/:target/hostPorts/filterOptions?field=port
func (h *HostPortHandler) FilterOptions(c *gin.Context) {
	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	var query hostPortFilterOptionsQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}
	if query.Field != "port" {
		httpdto.BadRequest(c, "unsupported hostPort filter option field")
		return
	}

	options, err := h.svc.ListPortOptionsByTarget(targetID)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			httpdto.NotFound(c, "Target not found")
			return
		}
		httpdto.InternalError(c, "Failed to list hostPort filter options")
		return
	}

	httpdto.Success(c, httpdto.FilterOptionsResponse{Results: toHostPortFilterOptionDTOs(options)})
}

func toHostPortFilterOptionDTOs(options []assetdomain.FilterOption) []httpdto.FilterOption {
	results := make([]httpdto.FilterOption, 0, len(options))
	for _, option := range options {
		results = append(results, httpdto.FilterOption{Value: option.Value, Label: option.Label, Count: option.Count})
	}
	return results
}

func toHostPortOutput(targetID int, item service.HostPortResponse) dto.HostPortResponse {
	hosts := item.Hosts
	if hosts == nil {
		hosts = []string{}
	}
	ports := item.Ports
	if ports == nil {
		ports = []int{}
	}
	return dto.HostPortResponse{
		Name:      httpdto.HostPortIPName(targetID, item.IP),
		IP:        item.IP,
		Hosts:     hosts,
		Ports:     ports,
		CreatedAt: item.CreatedAt,
	}
}
