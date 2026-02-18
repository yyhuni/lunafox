package directory

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

// List returns paginated directories for a target.
// GET /api/targets/:id/directories
func (h *DirectoryHandler) List(c *gin.Context) {
	targetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid target ID")
		return
	}

	var query dto.DirectoryListQuery
	if !dto.BindQuery(c, &query) {
		return
	}

	directories, total, err := h.svc.ListByTarget(targetID, query.GetPage(), query.GetPageSize(), query.Filter)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			dto.NotFound(c, "Target not found")
			return
		}
		dto.InternalError(c, "Failed to list directories")
		return
	}

	resp := make([]dto.DirectoryResponse, 0, len(directories))
	for _, item := range directories {
		resp = append(resp, toDirectoryOutput(&item))
	}

	dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}

func toDirectoryOutput(directory *service.Directory) dto.DirectoryResponse {
	return dto.DirectoryResponse{
		ID:            directory.ID,
		TargetID:      directory.TargetID,
		URL:           directory.URL,
		Status:        directory.Status,
		ContentLength: directory.ContentLength,
		ContentType:   directory.ContentType,
		Duration:      directory.Duration,
		CreatedAt:     timeutil.ToUTC(directory.CreatedAt),
	}
}
