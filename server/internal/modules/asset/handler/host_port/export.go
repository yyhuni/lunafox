package hostport

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/csv"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

// Export exports host-ports as CSV.
// GET /api/targets/:id/host-ports/export
func (h *HostPortHandler) Export(c *gin.Context) {
	targetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid target ID")
		return
	}

	var ips []string
	if ipsParam := c.Query("ips"); ipsParam != "" {
		ips = strings.Split(ipsParam, ",")
	}

	var rows *sql.Rows
	var count int64

	if len(ips) > 0 {
		rows, err = h.svc.StreamByTargetAndIPs(targetID, ips)
		count = 0
	} else {
		count, err = h.svc.CountByTarget(targetID)
		if err != nil {
			if errors.Is(err, service.ErrTargetNotFound) {
				dto.NotFound(c, "Target not found")
				return
			}
			dto.InternalError(c, "Failed to export host-ports")
			return
		}
		rows, err = h.svc.StreamByTarget(targetID)
	}

	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			dto.NotFound(c, "Target not found")
			return
		}
		dto.InternalError(c, "Failed to export host-ports")
		return
	}

	headers := []string{"ip", "host", "port", "created_at"}
	filename := fmt.Sprintf("target-%d-host-ports.csv", targetID)

	mapper := func(rows *sql.Rows) ([]string, error) {
		mapping, err := h.svc.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		return []string{
			mapping.IP,
			mapping.Host,
			strconv.Itoa(mapping.Port),
			timeutil.FormatRFC3339NanoUTC(mapping.CreatedAt),
		}, nil
	}

	if err := csv.StreamCSV(c, rows, headers, filename, mapper, count); err != nil {
		return
	}
}
