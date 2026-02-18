package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
)

// ListOrganizationTargets returns paginated targets for an organization.
// GET /api/organizations/:id/targets
func (handler *OrganizationHandler) ListOrganizationTargets(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid organization ID")
		return
	}

	var query dto.TargetListQuery
	if !dto.BindQuery(c, &query) {
		return
	}

	targets, total, err := handler.svc.ListOrganizationTargets(id, &query)
	if err != nil {
		if errors.Is(err, service.ErrOrganizationNotFound) {
			dto.NotFound(c, "Organization not found")
			return
		}
		dto.InternalError(c, "Failed to list targets")
		return
	}

	resp := make([]dto.TargetResponse, 0, len(targets))
	for _, target := range targets {
		resp = append(resp, toTargetOutput(target))
	}

	dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}
