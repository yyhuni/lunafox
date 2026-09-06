package hostport

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

// BatchDelete deletes host-port mappings by IP list.
// POST /v1/hostPorts:batchDelete
func (h *HostPortHandler) BatchDelete(c *gin.Context) {
	var req dto.BatchDeleteHostPortsRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	deletedCount, err := h.svc.BatchDeleteByIPs(req.IPs)
	if err != nil {
		httpdto.InternalError(c, "Failed to delete hostPorts")
		return
	}

	httpdto.Success(c, dto.BatchDeleteHostPortsResponse{DeletedCount: deletedCount})
}
