package handler

import (
	"errors"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
)

// CreateOrganization creates a new organization.
// POST /v1/organizations
func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	var req dto.CreateOrganizationRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	org, err := h.svc.CreateOrganization(&req)
	if err != nil {
		if errors.Is(err, service.ErrOrganizationExists) {
			httpdto.BadRequest(c, "Organization name already exists")
			return
		}
		httpdto.InternalError(c, "Failed to create organization")
		return
	}

	httpdto.Created(c, newOrganizationOutput(
		org.ID,
		org.Name,
		org.Description,
		org.CreatedAt,
		0,
	))
}

// UpdateOrganization updates an organization.
// PATCH /v1/organizations/:organization with body updateMask.
func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("organization"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid organization ID")
		return
	}

	var req dto.UpdateOrganizationRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}
	if req.Name != httpdto.OrganizationName(id) {
		httpdto.BadRequest(c, "Organization name must match the request path")
		return
	}
	if req.UpdateMask != "displayName,description" && req.UpdateMask != "description,displayName" {
		httpdto.BadRequest(c, "updateMask must include displayName and description")
		return
	}

	org, err := h.svc.UpdateOrganization(id, &req)
	if err != nil {
		if errors.Is(err, service.ErrOrganizationNotFound) {
			httpdto.NotFound(c, "Organization not found")
			return
		}
		if errors.Is(err, service.ErrOrganizationExists) {
			httpdto.BadRequest(c, "Organization name already exists")
			return
		}
		httpdto.InternalError(c, "Failed to update organization")
		return
	}

	httpdto.Success(c, newOrganizationOutput(
		org.ID,
		org.Name,
		org.Description,
		org.CreatedAt,
		0,
	))
}

// DeleteOrganization soft deletes an organization.
// DELETE /v1/organizations/:organization
func (h *OrganizationHandler) DeleteOrganization(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("organization"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid organization ID")
		return
	}

	if err := h.svc.DeleteOrganization(id); err != nil {
		if errors.Is(err, service.ErrOrganizationNotFound) {
			httpdto.NotFound(c, "Organization not found")
			return
		}
		httpdto.InternalError(c, "Failed to delete organization")
		return
	}

	httpdto.NoContent(c)
}

// BatchDeleteOrganizations soft deletes multiple organizations.
// POST /v1/organizations:batchDelete
func (h *OrganizationHandler) BatchDeleteOrganizations(c *gin.Context) {
	var req dto.BatchDeleteRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	ids, err := httpdto.ParseResourceNameIDs(req.Names, "organizations")
	if err != nil {
		httpdto.BadRequest(c, "Invalid organization names")
		return
	}

	deletedCount, err := h.svc.BatchDeleteOrganizations(ids)
	if err != nil {
		httpdto.InternalError(c, "Failed to delete organizations")
		return
	}

	httpdto.Success(c, dto.BatchDeleteResponse{DeletedCount: deletedCount})
}
