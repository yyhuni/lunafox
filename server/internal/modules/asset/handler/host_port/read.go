package hostport

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
)

// List returns paginated host-ports aggregated by IP.
// GET /api/targets/:id/host-ports
func (h *HostPortHandler) List(c *gin.Context) {
	targetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid target ID")
		return
	}

	var query dto.HostPortListQuery
	if !dto.BindQuery(c, &query) {
		return
	}

	results, total, err := h.svc.ListByTarget(targetID, query.GetPage(), query.GetPageSize(), query.Filter)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			dto.NotFound(c, "Target not found")
			return
		}
		dto.InternalError(c, "Failed to list host-ports")
		return
	}

	resp := make([]dto.HostPortResponse, 0, len(results))
	for index := range results {
		item := results[index]
		if item.Hosts == nil {
			item.Hosts = []string{}
		}
		if item.Ports == nil {
			item.Ports = []int{}
		}
		resp = append(resp, dto.HostPortResponse{
			IP:        item.IP,
			Hosts:     item.Hosts,
			Ports:     item.Ports,
			CreatedAt: item.CreatedAt,
		})
	}

	dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}
