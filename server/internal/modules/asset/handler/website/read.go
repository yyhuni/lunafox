package website

import (
	"errors"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
)

// List returns paginated websites for a target.
// GET /api/targets/:id/websites
func (h *WebsiteHandler) List(c *gin.Context) {
	targetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid target ID")
		return
	}

	var query dto.WebsiteListQuery
	if !dto.BindQuery(c, &query) {
		return
	}

	websites, total, err := h.svc.ListByTarget(targetID, query.GetPage(), query.GetPageSize(), query.Filter)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			dto.NotFound(c, "Target not found")
			return
		}
		dto.InternalError(c, "Failed to list websites")
		return
	}

	resp := make([]dto.WebsiteResponse, 0, len(websites))
	for _, website := range websites {
		resp = append(resp, toWebsiteOutput(&website))
	}

	dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}

func toWebsiteOutput(website *service.Website) dto.WebsiteResponse {
	tech := website.Tech
	if tech == nil {
		tech = []string{}
	}
	return dto.WebsiteResponse{
		ID:              website.ID,
		URL:             website.URL,
		Host:            website.Host,
		Location:        website.Location,
		Title:           website.Title,
		Webserver:       website.Webserver,
		ContentType:     website.ContentType,
		StatusCode:      website.StatusCode,
		ContentLength:   website.ContentLength,
		ResponseBody:    website.ResponseBody,
		Tech:            tech,
		Vhost:           website.Vhost,
		ResponseHeaders: website.ResponseHeaders,
		CreatedAt:       timeutil.ToUTC(website.CreatedAt),
	}
}
