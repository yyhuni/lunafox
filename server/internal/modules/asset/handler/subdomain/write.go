package subdomain

import (
	"errors"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
)

// BatchCreate creates multiple subdomains for a target.
// POST /v1/targets/:target/subdomains:batchCreate
func (h *SubdomainHandler) BatchCreate(c *gin.Context) {
	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	var req dto.BatchCreateSubdomainsRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	createdCount, err := h.svc.BatchCreate(targetID, req.DNSNames)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			httpdto.NotFound(c, "Target not found")
			return
		}
		if errors.Is(err, service.ErrInvalidTargetType) {
			httpdto.BadRequest(c, "Target type must be domain for subdomains")
			return
		}
		httpdto.InternalError(c, "Failed to create subdomains")
		return
	}

	httpdto.Created(c, dto.BatchCreateSubdomainsResponse{CreatedCount: createdCount})
}

// BatchDelete deletes multiple subdomains by IDs.
// POST /v1/subdomains:batchDelete
func (h *SubdomainHandler) BatchDelete(c *gin.Context) {
	var req dto.BatchDeleteRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	ids, err := httpdto.ParseNestedResourceNameIDs(req.Names, "targets", "subdomains")
	if err != nil {
		httpdto.BadRequest(c, "Invalid subdomain names")
		return
	}

	deletedCount, err := h.svc.BatchDelete(ids)
	if err != nil {
		httpdto.InternalError(c, "Failed to delete subdomains")
		return
	}

	httpdto.Success(c, dto.BatchDeleteResponse{DeletedCount: deletedCount})
}
