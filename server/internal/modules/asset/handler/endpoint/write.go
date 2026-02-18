package endpoint

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
)

// BulkCreate creates multiple endpoints for a target.
// POST /api/targets/:id/endpoints/bulk-create
func (h *EndpointHandler) BulkCreate(c *gin.Context) {
	targetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid target ID")
		return
	}

	var req dto.BulkCreateEndpointsRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	createdCount, err := h.svc.BulkCreate(targetID, req.URLs)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			dto.NotFound(c, "Target not found")
			return
		}
		dto.InternalError(c, "Failed to create endpoints")
		return
	}

	dto.Created(c, dto.BulkCreateEndpointsResponse{CreatedCount: createdCount})
}

// Delete deletes an endpoint by ID.
// DELETE /api/endpoints/:id
func (h *EndpointHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid endpoint ID")
		return
	}

	if err := h.svc.Delete(id); err != nil {
		if errors.Is(err, service.ErrEndpointNotFound) {
			dto.NotFound(c, "Endpoint not found")
			return
		}
		dto.InternalError(c, "Failed to delete endpoint")
		return
	}

	dto.NoContent(c)
}

// BulkDelete deletes multiple endpoints by IDs.
// POST /api/endpoints/bulk-delete
func (h *EndpointHandler) BulkDelete(c *gin.Context) {
	var req dto.BulkDeleteRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	deletedCount, err := h.svc.BulkDelete(req.IDs)
	if err != nil {
		dto.InternalError(c, "Failed to delete endpoints")
		return
	}

	dto.Success(c, dto.BulkDeleteResponse{DeletedCount: deletedCount})
}
