package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	service "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
)

type organizationListResponse struct {
	Results       []dto.OrganizationResponse `json:"results"`
	NextPageToken string                     `json:"nextPageToken"`
	TotalSize     int64                      `json:"totalSize"`
}

// List returns paginated organizations.
// GET /v1/organizations
func (h *OrganizationHandler) List(c *gin.Context) {
	for _, legacyParam := range []string{"page", "sort", "sortBy", "sortOrder", "keyword"} {
		if _, ok := c.GetQuery(legacyParam); ok {
			httpdto.BadRequest(c, "Unsupported organization list query parameter: "+legacyParam)
			return
		}
	}

	var query dto.OrganizationListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	result, err := h.svc.ListOrganizations(&query)
	if err != nil {
		if errors.Is(err, service.ErrUnsupportedOrganizationFilter) || errors.Is(err, service.ErrUnsupportedOrganizationOrderBy) || errors.Is(err, service.ErrInvalidOrganizationPageToken) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list organizations")
		return
	}

	resp := make([]dto.OrganizationResponse, 0, len(result.Organizations))
	for _, org := range result.Organizations {
		resp = append(resp, newOrganizationOutput(
			org.ID,
			org.Name,
			org.Description,
			org.CreatedAt,
			org.TargetCount,
		))
	}

	httpdto.Success(c, organizationListResponse{
		Results:       resp,
		NextPageToken: result.NextPageToken,
		TotalSize:     result.TotalSize,
	})
}

// GetOrganizationByID returns an organization by ID.
// GET /v1/organizations/:organization
func (h *OrganizationHandler) GetOrganizationByID(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("organization"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid organization ID")
		return
	}

	org, err := h.svc.GetOrganizationByID(id)
	if err != nil {
		if errors.Is(err, service.ErrOrganizationNotFound) {
			httpdto.NotFound(c, "Organization not found")
			return
		}
		httpdto.InternalError(c, "Failed to get organization")
		return
	}

	httpdto.Success(c, newOrganizationOutput(
		org.ID,
		org.Name,
		org.Description,
		org.CreatedAt,
		org.TargetCount,
	))
}
