package subdomain

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

// Export exports subdomains as CSV.
// GET /api/targets/:id/subdomains/export
func (h *SubdomainHandler) Export(c *gin.Context) {
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
		dto.InternalError(c, "Failed to export subdomains")
		return
	}

	rows, err := h.svc.StreamByTarget(targetID)
	if err != nil {
		dto.InternalError(c, "Failed to export subdomains")
		return
	}

	headers := []string{"id", "target_id", "name", "created_at"}
	filename := fmt.Sprintf("target-%d-subdomains.csv", targetID)

	mapper := func(rows *sql.Rows) ([]string, error) {
		subdomain, err := h.svc.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		return []string{
			strconv.Itoa(subdomain.ID),
			strconv.Itoa(subdomain.TargetID),
			subdomain.Name,
			timeutil.FormatRFC3339NanoUTC(subdomain.CreatedAt),
		}, nil
	}

	if err := csv.StreamCSV(c, rows, headers, filename, mapper, count); err != nil {
		return
	}
}
