package screenshot

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

// ListByTargetID returns screenshots for a target.
// GET /api/targets/:id/screenshots
func (h *ScreenshotHandler) ListByTargetID(c *gin.Context) {
	targetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid target ID")
		return
	}

	var query dto.ScreenshotListQuery
	if !dto.BindQuery(c, &query) {
		return
	}

	screenshots, total, err := h.svc.ListByTargetID(targetID, query.GetPage(), query.GetPageSize(), query.Filter)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			dto.NotFound(c, "Target not found")
			return
		}
		dto.InternalError(c, "Failed to list screenshots")
		return
	}

	resp := make([]dto.ScreenshotResponse, 0, len(screenshots))
	for _, item := range screenshots {
		resp = append(resp, dto.ScreenshotResponse{
			ID:         item.ID,
			URL:        item.URL,
			StatusCode: item.StatusCode,
			CreatedAt:  timeutil.ToUTC(item.CreatedAt),
			UpdatedAt:  timeutil.ToUTC(item.UpdatedAt),
		})
	}

	dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}

// GetImage returns screenshot image binary data.
// GET /api/screenshots/:id/image
func (h *ScreenshotHandler) GetImage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid screenshot ID")
		return
	}

	screenshot, err := h.svc.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrScreenshotNotFound) {
			dto.NotFound(c, "Screenshot not found")
			return
		}
		dto.InternalError(c, "Failed to get screenshot")
		return
	}
	if len(screenshot.Image) == 0 {
		dto.NotFound(c, "Screenshot image not found")
		return
	}

	c.Header("Content-Type", "image/webp")
	c.Header("Content-Disposition", "inline; filename=\"screenshot_"+strconv.Itoa(id)+".webp\"")
	c.Data(200, "image/webp", screenshot.Image)
}
