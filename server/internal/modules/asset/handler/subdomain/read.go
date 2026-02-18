package subdomain

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

// List returns paginated subdomains for a target.
// GET /api/targets/:id/subdomains
func (h *SubdomainHandler) List(c *gin.Context) {
	targetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid target ID")
		return
	}

	var query dto.SubdomainListQuery
	if !dto.BindQuery(c, &query) {
		return
	}

	subdomains, total, err := h.svc.ListByTarget(targetID, query.GetPage(), query.GetPageSize(), query.Filter)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			dto.NotFound(c, "Target not found")
			return
		}
		dto.InternalError(c, "Failed to list subdomains")
		return
	}

	resp := make([]dto.SubdomainResponse, 0, len(subdomains))
	for _, item := range subdomains {
		resp = append(resp, toSubdomainOutput(&item))
	}

	dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}

func toSubdomainOutput(subdomain *service.Subdomain) dto.SubdomainResponse {
	return dto.SubdomainResponse{
		ID:        subdomain.ID,
		TargetID:  subdomain.TargetID,
		Name:      subdomain.Name,
		CreatedAt: timeutil.ToUTC(subdomain.CreatedAt),
	}
}
