package directory

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/csv"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

// Export exports directories as CSV.
// GET /api/targets/:id/directories/export
func (h *DirectoryHandler) Export(c *gin.Context) {
	targetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid target ID")
		return
	}

	count, err := h.svc.CountByTarget(targetID)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			dto.NotFound(c, "Target not found")
			return
		}
		dto.InternalError(c, "Failed to export directories")
		return
	}

	rows, err := h.svc.StreamByTarget(targetID)
	if err != nil {
		dto.InternalError(c, "Failed to export directories")
		return
	}

	headers := []string{"id", "target_id", "url", "status", "content_length", "content_type", "duration", "created_at"}
	filename := fmt.Sprintf("target-%d-directories.csv", targetID)

	mapper := func(rows *sql.Rows) ([]string, error) {
		directory, err := h.svc.ScanRow(rows)
		if err != nil {
			return nil, err
		}

		status := ""
		if directory.Status != nil {
			status = strconv.Itoa(*directory.Status)
		}
		contentLength := ""
		if directory.ContentLength != nil {
			contentLength = strconv.Itoa(*directory.ContentLength)
		}
		duration := ""
		if directory.Duration != nil {
			duration = strconv.Itoa(*directory.Duration)
		}

		return []string{
			strconv.Itoa(directory.ID),
			strconv.Itoa(directory.TargetID),
			directory.URL,
			status,
			contentLength,
			directory.ContentType,
			duration,
			timeutil.FormatRFC3339NanoUTC(directory.CreatedAt),
		}, nil
	}

	if err := csv.StreamCSV(c, rows, headers, filename, mapper, count); err != nil {
		return
	}
}
