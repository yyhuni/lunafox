package endpoint

import (
	"errors"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
)

// BatchCreate creates multiple endpoints for a target.
// POST /v1/targets/:target/endpoints:batchCreate
func (h *EndpointHandler) BatchCreate(c *gin.Context) {
	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	var req dto.BatchCreateEndpointsRequest
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
		httpdto.InternalError(c, "Failed to create endpoints")
		return
	}

	httpdto.Created(c, dto.BatchCreateEndpointsResponse{CreatedCount: createdCount})
}

// Delete deletes an endpoint by ID.
// DELETE /v1/endpoints/:endpoint
func (h *EndpointHandler) Delete(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("endpoint"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid endpoint ID")
		return
	}

	if err := h.svc.Delete(id); err != nil {
		if errors.Is(err, service.ErrEndpointNotFound) {
			httpdto.NotFound(c, "Endpoint not found")
			return
		}
		httpdto.InternalError(c, "Failed to delete endpoint")
		return
	}

	httpdto.NoContent(c)
}

// BatchDelete deletes multiple endpoints by IDs.
// POST /v1/endpoints:batchDelete
func (h *EndpointHandler) BatchDelete(c *gin.Context) {
	var req dto.BatchDeleteRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	ids, err := httpdto.ParseNestedResourceNameIDs(req.Names, "targets", "endpoints")
	if err != nil {
		httpdto.BadRequest(c, "Invalid endpoint names")
		return
	}

	deletedCount, err := h.svc.BatchDelete(ids)
	if err != nil {
		httpdto.InternalError(c, "Failed to delete endpoints")
		return
	}

	httpdto.Success(c, dto.BatchDeleteResponse{DeletedCount: deletedCount})
}
