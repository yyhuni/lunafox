package website

import (
	"errors"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

type websiteListResponse struct {
	Results       []dto.WebsiteResponse `json:"results"`
	NextPageToken string                `json:"nextPageToken"`
	TotalSize     int64                 `json:"totalSize"`
}

type websiteFilterOptionsQuery struct {
	Field string `form:"field" binding:"required"`
}

// List returns paginated websites for a target.
// GET /v1/targets/:target/websites
func (h *WebsiteHandler) List(c *gin.Context) {
	for _, legacyParam := range []string{"page", "sort", "sortBy", "sortOrder", "keyword"} {
		if _, ok := c.GetQuery(legacyParam); ok {
			httpdto.BadRequest(c, "Unsupported website list query parameter: "+legacyParam)
			return
		}
	}

	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	var query dto.WebsiteListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	result, err := h.svc.ListByTarget(targetID, service.WebsiteListQueryInput{
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
		if errors.Is(err, service.ErrUnsupportedWebsiteFilter) || errors.Is(err, service.ErrUnsupportedWebsiteOrderBy) || errors.Is(err, service.ErrInvalidWebsitePageToken) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list websites")
		return
	}

	resp := make([]dto.WebsiteResponse, 0, len(result.Websites))
	for _, website := range result.Websites {
		resp = append(resp, toWebsiteOutput(&website))
	}

	httpdto.Success(c, websiteListResponse{
		Results:       resp,
		NextPageToken: result.NextPageToken,
		TotalSize:     result.TotalSize,
	})
}

// Get returns one Website resource and its optional Screenshot summary.
// GET /v1/websites/:website
func (h *WebsiteHandler) Get(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("website"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid website ID")
		return
	}

	website, err := h.svc.Get(id)
	if err != nil {
		if errors.Is(err, service.ErrWebsiteNotFound) {
			httpdto.NotFound(c, "Website not found")
			return
		}
		httpdto.InternalError(c, "Failed to get website")
		return
	}

	httpdto.Success(c, toWebsiteOutput(website))
}

// FilterOptions returns parent-scoped website filter options.
// GET /v1/targets/:target/websites/filterOptions?field=tech
func (h *WebsiteHandler) FilterOptions(c *gin.Context) {
	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	var query websiteFilterOptionsQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	options, err := h.svc.ListFilterOptionsByTarget(targetID, query.Field)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			httpdto.NotFound(c, "Target not found")
			return
		}
		if errors.Is(err, service.ErrUnsupportedWebsiteFilter) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list website filter options")
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

func toWebsiteOutput(readModel *service.WebsiteReadModel) dto.WebsiteResponse {
	website := &readModel.Website
	tech := website.Tech
	if tech == nil {
		tech = []string{}
	}
	output := dto.WebsiteResponse{
		ID:              website.ID,
		Name:            httpdto.WebsiteName(website.TargetID, website.ID),
		URL:             website.URL,
		Host:            website.Host,
		Location:        website.Location,
		Title:           website.Title,
		Webserver:       website.Webserver,
		ContentType:     website.ContentType,
		StatusCode:      website.StatusCode,
		ContentLength:   website.ContentLength,
		ResponseBody:    website.ResponseBody,
		Tech:            tech,
		Vhost:           website.Vhost,
		ResponseHeaders: website.ResponseHeaders,
		CreatedAt:       timeutil.ToUTC(website.CreatedAt),
	}
	if readModel.Screenshot != nil {
		output.Screenshot = &dto.WebsiteScreenshotSummary{
			ID:         readModel.Screenshot.ID,
			Name:       httpdto.ScreenshotName(website.TargetID, readModel.Screenshot.ID),
			URL:        readModel.Screenshot.URL,
			StatusCode: readModel.Screenshot.StatusCode,
			CreatedAt:  timeutil.ToUTC(readModel.Screenshot.CreatedAt),
			UpdatedAt:  timeutil.ToUTC(readModel.Screenshot.UpdatedAt),
		}
	}
	return output
}
