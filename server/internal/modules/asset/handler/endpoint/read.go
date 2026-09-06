package endpoint

import (
	"errors"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

type endpointListResponse struct {
	Results       []dto.EndpointResponse `json:"results"`
	NextPageToken string                 `json:"nextPageToken"`
	TotalSize     int64                  `json:"totalSize"`
}

type endpointFilterOptionsQuery struct {
	Field string `form:"field" binding:"required"`
}

// List returns paginated endpoints for a target.
// GET /v1/targets/:target/endpoints
func (h *EndpointHandler) List(c *gin.Context) {
	for _, legacyParam := range []string{"page", "sort", "sortBy", "sortOrder", "keyword"} {
		if _, ok := c.GetQuery(legacyParam); ok {
			httpdto.BadRequest(c, "Unsupported endpoint list query parameter: "+legacyParam)
			return
		}
	}

	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	var query dto.EndpointListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	result, err := h.svc.ListByTarget(targetID, service.EndpointListQueryInput{
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
		if errors.Is(err, service.ErrUnsupportedEndpointFilter) || errors.Is(err, service.ErrUnsupportedEndpointOrderBy) || errors.Is(err, service.ErrInvalidEndpointPageToken) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list endpoints")
		return
	}

	resp := make([]dto.EndpointResponse, 0, len(result.Endpoints))
	for _, endpoint := range result.Endpoints {
		resp = append(resp, toEndpointOutput(&endpoint))
	}

	httpdto.Success(c, endpointListResponse{
		Results:       resp,
		NextPageToken: result.NextPageToken,
		TotalSize:     result.TotalSize,
	})
}

// FilterOptions returns parent-scoped endpoint filter options.
// GET /v1/targets/:target/endpoints/filterOptions?field=tech
func (h *EndpointHandler) FilterOptions(c *gin.Context) {
	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	var query endpointFilterOptionsQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	options, err := h.svc.ListFilterOptionsByTarget(targetID, query.Field)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			httpdto.NotFound(c, "Target not found")
			return
		}
		if errors.Is(err, service.ErrUnsupportedEndpointFilter) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list endpoint filter options")
		return
	}

	httpdto.Success(c, httpdto.FilterOptionsResponse{Results: toFilterOptionDTOs(options)})
}

func toFilterOptionDTOs(options []assetdomain.FilterOption) []httpdto.FilterOption {
	results := make([]httpdto.FilterOption, 0, len(options))
	for _, option := range options {
		results = append(results, httpdto.FilterOption{Value: option.Value, Label: option.Label, Count: option.Count})
	}
	return results
}

// GetByID returns an endpoint by ID.
// GET /v1/endpoints/:endpoint
func (h *EndpointHandler) GetByID(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("endpoint"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid endpoint ID")
		return
	}

	endpoint, err := h.svc.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrEndpointNotFound) {
			httpdto.NotFound(c, "Endpoint not found")
			return
		}
		httpdto.InternalError(c, "Failed to get endpoint")
		return
	}

	httpdto.Success(c, toEndpointOutput(endpoint))
}

func toEndpointOutput(endpoint *service.Endpoint) dto.EndpointResponse {
	tech := []string(endpoint.Tech)
	if tech == nil {
		tech = []string{}
	}

	return dto.EndpointResponse{
		ID:                       endpoint.ID,
		TargetID:                 endpoint.TargetID,
		Name:                     httpdto.EndpointName(endpoint.TargetID, endpoint.ID),
		URL:                      endpoint.URL,
		Host:                     endpoint.Host,
		Location:                 endpoint.Location,
		Title:                    endpoint.Title,
		Webserver:                endpoint.Webserver,
		ContentType:              endpoint.ContentType,
		StatusCode:               endpoint.StatusCode,
		ContentLength:            endpoint.ContentLength,
		ResponseBody:             endpoint.ResponseBody,
		ResponseBodyTruncated:    endpoint.ResponseBodyTruncated,
		Tech:                     tech,
		Vhost:                    endpoint.Vhost,
		ResponseHeaders:          endpoint.ResponseHeaders,
		ResponseHeadersTruncated: endpoint.ResponseHeadersTruncated,
		CreatedAt:                timeutil.ToUTC(endpoint.CreatedAt),
	}
}
