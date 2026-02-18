package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
)

// ListOrganizations returns paginated organizations.
// GET /api/organizations
func (h *OrganizationHandler) ListOrganizations(c *gin.Context) {
	var query dto.OrganizationListQuery
	if !dto.BindQuery(c, &query) {
		return
	}

	orgs, total, err := h.svc.ListOrganizations(&query)
	if err != nil {
		dto.InternalError(c, "Failed to list organizations")
		return
	}

	resp := make([]dto.OrganizationResponse, 0, len(orgs))
	for _, org := range orgs {
		resp = append(resp, newOrganizationOutput(
			org.ID,
			org.Name,
			org.Description,
			org.CreatedAt,
			org.TargetCount,
		))
	}

	dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}

// GetOrganizationByID returns an organization by ID.
// GET /api/organizations/:id
func (h *OrganizationHandler) GetOrganizationByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid organization ID")
		return
	}

	org, err := h.svc.GetOrganizationByID(id)
	if err != nil {
		if errors.Is(err, service.ErrOrganizationNotFound) {
			dto.NotFound(c, "Organization not found")
			return
		}
		dto.InternalError(c, "Failed to get organization")
		return
	}

	dto.Success(c, newOrganizationOutput(
		org.ID,
		org.Name,
		org.Description,
		org.CreatedAt,
		org.TargetCount,
	))
}
