package subdomain

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
)

// BulkCreate creates multiple subdomains for a target.
// POST /api/targets/:id/subdomains/bulk-create
func (h *SubdomainHandler) BulkCreate(c *gin.Context) {
	targetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid target ID")
		return
	}

	var req dto.BulkCreateSubdomainsRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	createdCount, err := h.svc.BulkCreate(targetID, req.Names)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			dto.NotFound(c, "Target not found")
			return
		}
		if errors.Is(err, service.ErrInvalidTargetType) {
			dto.BadRequest(c, "Target type must be domain for subdomains")
			return
		}
		dto.InternalError(c, "Failed to create subdomains")
		return
	}

	dto.Created(c, dto.BulkCreateSubdomainsResponse{CreatedCount: createdCount})
}

// BulkDelete deletes multiple subdomains by IDs.
// POST /api/subdomains/bulk-delete
func (h *SubdomainHandler) BulkDelete(c *gin.Context) {
	var req dto.BulkDeleteRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	deletedCount, err := h.svc.BulkDelete(req.IDs)
	if err != nil {
		dto.InternalError(c, "Failed to delete subdomains")
		return
	}

	dto.Success(c, dto.BulkDeleteResponse{DeletedCount: deletedCount})
}
