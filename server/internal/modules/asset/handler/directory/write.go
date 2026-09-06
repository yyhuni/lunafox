package directory

import (
	"errors"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
)

// BatchCreate creates multiple directories for a target.
// POST /v1/targets/:target/directories:batchCreate
func (h *DirectoryHandler) BatchCreate(c *gin.Context) {
	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	var req dto.BatchCreateDirectoriesRequest
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
		httpdto.InternalError(c, "Failed to create directories")
		return
	}

	httpdto.Created(c, dto.BatchCreateDirectoriesResponse{CreatedCount: createdCount})
}

// BatchDelete deletes multiple directories by IDs.
// POST /v1/directories:batchDelete
func (h *DirectoryHandler) BatchDelete(c *gin.Context) {
	var req dto.BatchDeleteRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	ids, err := httpdto.ParseNestedResourceNameIDs(req.Names, "targets", "directories")
	if err != nil {
		httpdto.BadRequest(c, "Invalid directory names")
		return
	}

	deletedCount, err := h.svc.BatchDelete(ids)
	if err != nil {
		httpdto.InternalError(c, "Failed to delete directories")
		return
	}

	httpdto.Success(c, dto.BatchDeleteResponse{DeletedCount: deletedCount})
}
