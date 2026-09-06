package directory

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/pkg/csv"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

// Export exports directories as CSV.
// GET /v1/targets/:target/directories/exportFiles/current
func (h *DirectoryHandler) Export(c *gin.Context) {
	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	count, err := h.svc.CountByTarget(targetID)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			httpdto.NotFound(c, "Target not found")
			return
		}
		httpdto.InternalError(c, "Failed to export directories")
		return
	}

	headers := []string{"id", "target_id", "url", "status", "content_length", "content_type", "duration", "created_at"}
	filename := fmt.Sprintf("target-%d-directories.csv", targetID)

	producer := func(write csv.RowWriter) error {
		return h.svc.ForEachByTarget(targetID, func(directory service.Directory) error {
			status := ""
			if directory.Status != nil {
				status = strconv.Itoa(*directory.Status)
			}
			contentLength := ""
			if directory.ContentLength != nil {
				contentLength = strconv.FormatInt(*directory.ContentLength, 10)
			}
			duration := ""
			if directory.Duration != nil {
				duration = strconv.FormatInt(*directory.Duration, 10)
			}

			return write([]string{
				strconv.Itoa(directory.ID),
				strconv.Itoa(directory.TargetID),
				directory.URL,
				status,
				contentLength,
				directory.ContentType,
				duration,
				timeutil.FormatRFC3339NanoUTC(directory.CreatedAt),
			})
		})
	}

	if err := csv.StreamCSV(c, headers, filename, producer, count); err != nil {
		return
	}
}
