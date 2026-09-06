package hostport

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/pkg/csv"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

// Export exports hostPorts as CSV.
// GET /v1/targets/:target/hostPorts/exportFiles/current
func (h *HostPortHandler) Export(c *gin.Context) {
	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	var ips []string
	if ipsParam := c.Query("ips"); ipsParam != "" {
		ips = strings.Split(ipsParam, ",")
	}

	count, err := h.svc.CountByTarget(targetID)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			httpdto.NotFound(c, "Target not found")
			return
		}
		httpdto.InternalError(c, "Failed to export hostPorts")
		return
	}

	headers := []string{"ip", "host", "port", "created_at"}
	filename := fmt.Sprintf("target-%d-hostPorts.csv", targetID)

	producer := func(write csv.RowWriter) error {
		visit := func(mapping service.HostPort) error {
			return write([]string{
				mapping.IP,
				mapping.Host,
				strconv.Itoa(mapping.Port),
				timeutil.FormatRFC3339NanoUTC(mapping.CreatedAt),
			})
		}

		if len(ips) > 0 {
			return h.svc.ForEachByTargetAndIPs(targetID, ips, visit)
		}
		return h.svc.ForEachByTarget(targetID, visit)
	}

	if len(ips) > 0 {
		count = 0
	}
	if err := csv.StreamCSV(c, headers, filename, producer, count); err != nil {
		return
	}
}
