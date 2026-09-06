package handler

import (
	"errors"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
)

// BatchLinkOrganizationTargets adds targets to an organization.
// POST /v1/organizations/:organization/targets:batchLink
func (h *OrganizationHandler) BatchLinkOrganizationTargets(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("organization"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid organization ID")
		return
	}

	var req dto.LinkTargetsRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	targetIDs, err := httpdto.ParseResourceNameIDs(req.Targets, "targets")
	if err != nil {
		httpdto.BadRequest(c, "Invalid target names")
		return
	}

	err = h.svc.LinkOrganizationTargets(id, targetIDs)
	if err != nil {
		if errors.Is(err, service.ErrOrganizationNotFound) {
			httpdto.NotFound(c, "Organization not found")
			return
		}
		if errors.Is(err, service.ErrTargetNotFound) {
			httpdto.BadRequest(c, "One or more target IDs do not exist")
			return
		}
		httpdto.InternalError(c, "Failed to link targets")
		return
	}

	httpdto.NoContent(c)
}

// BatchUnlinkOrganizationTargets removes targets from an organization.
// POST /v1/organizations/:organization/targets:batchUnlink
func (h *OrganizationHandler) BatchUnlinkOrganizationTargets(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("organization"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid organization ID")
		return
	}

	var req dto.LinkTargetsRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	targetIDs, err := httpdto.ParseResourceNameIDs(req.Targets, "targets")
	if err != nil {
		httpdto.BadRequest(c, "Invalid target names")
		return
	}

	unlinkedCount, err := h.svc.UnlinkOrganizationTargets(id, targetIDs)
	if err != nil {
		if errors.Is(err, service.ErrOrganizationNotFound) {
			httpdto.NotFound(c, "Organization not found")
			return
		}
		httpdto.InternalError(c, "Failed to unlink targets")
		return
	}

	httpdto.Success(c, gin.H{"unlinkedCount": unlinkedCount})
}
