package search

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

type globalAssetSearchResponse struct {
	Results       any    `json:"results"`
	NextPageToken string `json:"nextPageToken,omitempty"`
}

// Search returns one selected current-state asset type from all active Targets.
// GET /v1/assets:search
func (handler *GlobalAssetSearchHandler) Search(c *gin.Context) {
	if handler == nil || handler.service == nil {
		httpdto.InternalError(c, "Global asset search is unavailable")
		return
	}
	var query dto.GlobalAssetSearchQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	result, err := handler.service.Search(c.Request.Context(), service.GlobalAssetSearchInput{
		Query:     query.Q,
		AssetType: service.GlobalAssetSearchAssetType(query.AssetType),
		PageSize:  query.PageSize,
		PageToken: query.PageToken,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidGlobalAssetSearchQuery) || errors.Is(err, service.ErrInvalidGlobalAssetSearchPageToken) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrGlobalAssetSearchTimeout) {
			httpdto.ErrorWithStatus(c, http.StatusGatewayTimeout, "GLOBAL_ASSET_SEARCH_TIMEOUT", "DEADLINE_EXCEEDED", "Global asset search timed out")
			return
		}
		httpdto.InternalError(c, "Failed to search assets")
		return
	}

	if result.AssetType == service.GlobalAssetSearchAssetTypeWebsite {
		items := make([]dto.WebsiteResponse, 0, len(result.Websites))
		for index := range result.Websites {
			items = append(items, globalSearchWebsiteResponse(result.Websites[index]))
		}
		httpdto.Success(c, globalAssetSearchResponse{Results: items, NextPageToken: result.NextPageToken})
		return
	}

	items := make([]dto.EndpointResponse, 0, len(result.Endpoints))
	for index := range result.Endpoints {
		items = append(items, globalSearchEndpointResponse(result.Endpoints[index]))
	}
	httpdto.Success(c, globalAssetSearchResponse{Results: items, NextPageToken: result.NextPageToken})
}

func globalSearchWebsiteResponse(website assetdomain.Website) dto.WebsiteResponse {
	tech := website.Tech
	if tech == nil {
		tech = []string{}
	}
	return dto.WebsiteResponse{
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
}

func globalSearchEndpointResponse(endpoint assetdomain.Endpoint) dto.EndpointResponse {
	tech := endpoint.Tech
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
