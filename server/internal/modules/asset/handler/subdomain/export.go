package subdomain

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

// Export exports subdomains as CSV.
// GET /v1/targets/:target/subdomains/exportFiles/current
func (h *SubdomainHandler) Export(c *gin.Context) {
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
		httpdto.InternalError(c, "Failed to export subdomains")
		return
	}

	headers := []string{"id", "target_id", "dns_name", "created_at"}
	filename := fmt.Sprintf("target-%d-subdomains.csv", targetID)

	producer := func(write csv.RowWriter) error {
		return h.svc.ForEachByTarget(targetID, func(subdomain service.Subdomain) error {
			return write([]string{
				strconv.Itoa(subdomain.ID),
				strconv.Itoa(subdomain.TargetID),
				subdomain.DNSName,
				timeutil.FormatRFC3339NanoUTC(subdomain.CreatedAt),
			})
		})
	}

	if err := csv.StreamCSV(c, headers, filename, producer, count); err != nil {
		return
	}
}
