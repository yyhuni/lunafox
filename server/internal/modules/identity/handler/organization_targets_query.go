package handler

import (
	"errors"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
)

// ListOrganizationTargets returns paginated targets for an organization.
// GET /v1/organizations/:organization/targets
func (h *OrganizationHandler) ListOrganizationTargets(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("organization"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid organization ID")
		return
	}

	var query dto.TargetListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	targets, total, err := h.svc.ListOrganizationTargets(id, &query)
	if err != nil {
		if errors.Is(err, service.ErrOrganizationNotFound) {
			httpdto.NotFound(c, "Organization not found")
			return
		}
		httpdto.InternalError(c, "Failed to list targets")
		return
	}

	resp := make([]dto.TargetResponse, 0, len(targets))
	for _, target := range targets {
		resp = append(resp, toTargetOutput(target))
	}

	httpdto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}
