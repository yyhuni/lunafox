package website

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
)

// BulkCreate creates multiple websites for a target.
// POST /api/targets/:id/websites/bulk-create
func (h *WebsiteHandler) BulkCreate(c *gin.Context) {
	targetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid target ID")
		return
	}

	var req dto.BulkCreateWebsitesRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	createdCount, err := h.svc.BulkCreate(targetID, req.URLs)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			dto.NotFound(c, "Target not found")
			return
		}
		dto.InternalError(c, "Failed to create websites")
		return
	}

	dto.Created(c, dto.BulkCreateWebsitesResponse{CreatedCount: createdCount})
}

// Delete deletes a website by ID.
// DELETE /api/websites/:id
func (h *WebsiteHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid website ID")
		return
	}

	if err := h.svc.Delete(id); err != nil {
		if errors.Is(err, service.ErrWebsiteNotFound) {
			dto.NotFound(c, "Website not found")
			return
		}
		dto.InternalError(c, "Failed to delete website")
		return
	}

	dto.NoContent(c)
}

// BulkDelete deletes multiple websites by IDs.
// POST /api/websites/bulk-delete
func (h *WebsiteHandler) BulkDelete(c *gin.Context) {
	var req dto.BulkDeleteRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	deletedCount, err := h.svc.BulkDelete(req.IDs)
	if err != nil {
		dto.InternalError(c, "Failed to delete websites")
		return
	}

	dto.Success(c, dto.BulkDeleteResponse{DeletedCount: deletedCount})
}
