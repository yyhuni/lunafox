package subdomain

import (
	"errors"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

type subdomainListResponse struct {
	Results       []dto.SubdomainResponse `json:"results"`
	NextPageToken string                  `json:"nextPageToken"`
	TotalSize     int64                   `json:"totalSize"`
}

// List returns paginated subdomains for a target.
// GET /v1/targets/:target/subdomains
func (h *SubdomainHandler) List(c *gin.Context) {
	for _, legacyParam := range []string{"page", "sort", "sortBy", "sortOrder", "keyword"} {
		if _, ok := c.GetQuery(legacyParam); ok {
			httpdto.BadRequest(c, "Unsupported subdomain list query parameter: "+legacyParam)
			return
		}
	}

	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	var query dto.SubdomainListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	result, err := h.svc.ListByTarget(targetID, service.SubdomainListQueryInput{
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
		if errors.Is(err, service.ErrUnsupportedSubdomainFilter) || errors.Is(err, service.ErrUnsupportedSubdomainOrderBy) || errors.Is(err, service.ErrInvalidSubdomainPageToken) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list subdomains")
		return
	}

	resp := make([]dto.SubdomainResponse, 0, len(result.Subdomains))
	for _, item := range result.Subdomains {
		resp = append(resp, toSubdomainOutput(&item))
	}

	httpdto.Success(c, subdomainListResponse{
		Results:       resp,
		NextPageToken: result.NextPageToken,
		TotalSize:     result.TotalSize,
	})
}

func toSubdomainOutput(subdomain *service.Subdomain) dto.SubdomainResponse {
	return dto.SubdomainResponse{
		ID:        subdomain.ID,
		TargetID:  subdomain.TargetID,
		Name:      httpdto.SubdomainName(subdomain.TargetID, subdomain.ID),
		DNSName:   subdomain.DNSName,
		CreatedAt: timeutil.ToUTC(subdomain.CreatedAt),
	}
}
