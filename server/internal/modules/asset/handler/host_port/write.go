package hostport

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
)

// BulkDelete deletes host-port mappings by IP list.
// POST /api/host-ports/bulk-delete
func (h *HostPortHandler) BulkDelete(c *gin.Context) {
	var req dto.BulkDeleteHostPortsRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	deletedCount, err := h.svc.BulkDeleteByIPs(req.IPs)
	if err != nil {
		dto.InternalError(c, "Failed to delete host-ports")
		return
	}

	dto.Success(c, dto.BulkDeleteHostPortsResponse{DeletedCount: deletedCount})
}
