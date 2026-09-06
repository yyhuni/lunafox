package screenshot

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

// BatchDelete deletes multiple screenshots.
// POST /v1/screenshots:batchDelete
func (h *ScreenshotHandler) BatchDelete(c *gin.Context) {
	var req dto.BatchDeleteRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	ids, err := httpdto.ParseNestedResourceNameIDs(req.Names, "targets", "screenshots")
	if err != nil {
		httpdto.BadRequest(c, "Invalid screenshot names")
		return
	}

	deletedCount, err := h.svc.BatchDelete(ids)
	if err != nil {
		httpdto.InternalError(c, "Failed to delete screenshots")
		return
	}

	httpdto.Success(c, dto.BatchDeleteResponse{DeletedCount: deletedCount})
}
