package screenshot

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

type screenshotListResponse struct {
	Results       []dto.ScreenshotResponse `json:"results"`
	NextPageToken string                   `json:"nextPageToken"`
	TotalSize     int64                    `json:"totalSize"`
}

type screenshotFilterOptionsQuery struct {
	Field string `form:"field" binding:"required"`
}

// List returns screenshots for a target.
// GET /v1/targets/:target/screenshots
func (h *ScreenshotHandler) List(c *gin.Context) {
	for _, legacyParam := range []string{"page", "sort", "sortBy", "sortOrder", "keyword"} {
		if _, ok := c.GetQuery(legacyParam); ok {
			httpdto.BadRequest(c, "Unsupported screenshot list query parameter: "+legacyParam)
			return
		}
	}

	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	var query dto.ScreenshotListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	result, err := h.svc.ListByTarget(targetID, service.ScreenshotListQueryInput{
		PageSize:  query.GetPageSize(),
		PageToken: query.PageToken,
		Filter:    query.Filter,
		OrderBy:   query.OrderBy,
	})
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			httpdto.NotFound(c, "Target not found")
			return
		}
		if errors.Is(err, service.ErrUnsupportedScreenshotFilter) || errors.Is(err, service.ErrUnsupportedScreenshotOrderBy) || errors.Is(err, service.ErrInvalidScreenshotPageToken) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list screenshots")
		return
	}

	resp := make([]dto.ScreenshotResponse, 0, len(result.Screenshots))
	for _, item := range result.Screenshots {
		resp = append(resp, toScreenshotOutput(&item))
	}

	httpdto.Success(c, screenshotListResponse{
		Results:       resp,
		NextPageToken: result.NextPageToken,
		TotalSize:     result.TotalSize,
	})
}

// FilterOptions returns parent-scoped screenshot filter options.
// GET /v1/targets/:target/screenshots/filterOptions?field=statusCode
func (h *ScreenshotHandler) FilterOptions(c *gin.Context) {
	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	var query screenshotFilterOptionsQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	options, err := h.svc.ListFilterOptionsByTarget(targetID, query.Field)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			httpdto.NotFound(c, "Target not found")
			return
		}
		if errors.Is(err, service.ErrUnsupportedScreenshotFilter) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list screenshot filter options")
		return
	}

	httpdto.Success(c, httpdto.FilterOptionsResponse{Results: toScreenshotFilterOptionDTOs(options)})
}

func toScreenshotFilterOptionDTOs(options []assetdomain.FilterOption) []httpdto.FilterOption {
	results := make([]httpdto.FilterOption, 0, len(options))
	for _, option := range options {
		results = append(results, httpdto.FilterOption{Value: option.Value, Label: option.Label, Count: option.Count})
	}
	return results
}

// GetImage returns screenshot image binary data.
// GET /v1/screenshots/:screenshot/blob
func (h *ScreenshotHandler) GetImage(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("screenshot"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid screenshot ID")
		return
	}

	screenshot, err := h.svc.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrScreenshotNotFound) {
			httpdto.NotFound(c, "Screenshot not found")
			return
		}
		httpdto.InternalError(c, "Failed to get screenshot")
		return
	}
	if len(screenshot.Image) == 0 {
		httpdto.NotFound(c, "Screenshot image not found")
		return
	}

	c.Header("Content-Type", "image/webp")
	c.Header("Content-Disposition", "inline; filename=\"screenshot_"+strconv.Itoa(id)+".webp\"")
	c.Data(200, "image/webp", screenshot.Image)
}

func toScreenshotOutput(screenshot *service.Screenshot) dto.ScreenshotResponse {
	return dto.ScreenshotResponse{
		ID:         screenshot.ID,
		Name:       httpdto.ScreenshotName(screenshot.TargetID, screenshot.ID),
		URL:        screenshot.URL,
		StatusCode: screenshot.StatusCode,
		CreatedAt:  timeutil.ToUTC(screenshot.CreatedAt),
		UpdatedAt:  timeutil.ToUTC(screenshot.UpdatedAt),
	}
}
