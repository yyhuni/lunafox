package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
)

// CreateOrganization creates a new organization.
// POST /api/organizations
func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	var req dto.CreateOrganizationRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	org, err := h.svc.CreateOrganization(&req)
	if err != nil {
		if errors.Is(err, service.ErrOrganizationExists) {
			dto.BadRequest(c, "Organization name already exists")
			return
		}
		dto.InternalError(c, "Failed to create organization")
		return
	}

	dto.Created(c, newOrganizationOutput(
		org.ID,
		org.Name,
		org.Description,
		org.CreatedAt,
		0,
	))
}

// UpdateOrganization updates an organization.
// PUT /api/organizations/:id
func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid organization ID")
		return
	}

	var req dto.UpdateOrganizationRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	org, err := h.svc.UpdateOrganization(id, &req)
	if err != nil {
		if errors.Is(err, service.ErrOrganizationNotFound) {
			dto.NotFound(c, "Organization not found")
			return
		}
		if errors.Is(err, service.ErrOrganizationExists) {
			dto.BadRequest(c, "Organization name already exists")
			return
		}
		dto.InternalError(c, "Failed to update organization")
		return
	}

	dto.Success(c, newOrganizationOutput(
		org.ID,
		org.Name,
		org.Description,
		org.CreatedAt,
		0,
	))
}

// DeleteOrganization soft deletes an organization.
// DELETE /api/organizations/:id
func (h *OrganizationHandler) DeleteOrganization(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid organization ID")
		return
	}

	if err := h.svc.DeleteOrganization(id); err != nil {
		if errors.Is(err, service.ErrOrganizationNotFound) {
			dto.NotFound(c, "Organization not found")
			return
		}
		dto.InternalError(c, "Failed to delete organization")
		return
	}

	dto.NoContent(c)
}

// BulkDeleteOrganizations soft deletes multiple organizations.
// POST /api/organizations/bulk-delete
func (h *OrganizationHandler) BulkDeleteOrganizations(c *gin.Context) {
	var req dto.BulkDeleteRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	deletedCount, err := h.svc.BulkDeleteOrganizations(req.IDs)
	if err != nil {
		dto.InternalError(c, "Failed to delete organizations")
		return
	}

	dto.Success(c, dto.BulkDeleteResponse{DeletedCount: deletedCount})
}
