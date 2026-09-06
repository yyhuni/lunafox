package website

import (
	"errors"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
)

// BatchCreate creates multiple websites for a target.
// POST /v1/targets/:target/websites:batchCreate
func (h *WebsiteHandler) BatchCreate(c *gin.Context) {
	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	var req dto.BatchCreateWebsitesRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	createdCount, err := h.svc.BatchCreate(targetID, req.URLs)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			httpdto.NotFound(c, "Target not found")
			return
		}
		if errors.Is(err, service.ErrInvalidObservedAssetURL) {
			httpdto.BadRequest(c, "Invalid URL")
			return
		}
		httpdto.InternalError(c, "Failed to create websites")
		return
	}

	httpdto.Created(c, dto.BatchCreateWebsitesResponse{CreatedCount: createdCount})
}

// Delete deletes a website by ID.
// DELETE /v1/websites/:website
func (h *WebsiteHandler) Delete(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("website"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid website ID")
		return
	}

	if err := h.svc.Delete(id); err != nil {
		if errors.Is(err, service.ErrWebsiteNotFound) {
			httpdto.NotFound(c, "Website not found")
			return
		}
		httpdto.InternalError(c, "Failed to delete website")
		return
	}

	httpdto.NoContent(c)
}

// BatchDelete deletes multiple websites by IDs.
// POST /v1/websites:batchDelete
func (h *WebsiteHandler) BatchDelete(c *gin.Context) {
	var req dto.BatchDeleteRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	ids, err := httpdto.ParseNestedResourceNameIDs(req.Names, "targets", "websites")
	if err != nil {
		httpdto.BadRequest(c, "Invalid website names")
		return
	}

	deletedCount, err := h.svc.BatchDelete(ids)
	if err != nil {
		httpdto.InternalError(c, "Failed to delete websites")
		return
	}

	httpdto.Success(c, dto.BatchDeleteResponse{DeletedCount: deletedCount})
}
