package endpoint

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

// Export exports endpoints as CSV.
// GET /api/targets/:id/endpoints/export
func (h *EndpointHandler) Export(c *gin.Context) {
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
		dto.InternalError(c, "Failed to export endpoints")
		return
	}

	rows, err := h.svc.StreamByTarget(targetID)
	if err != nil {
		dto.InternalError(c, "Failed to export endpoints")
		return
	}

	headers := []string{
		"id", "target_id", "url", "host", "location", "title", "status_code",
		"content_length", "content_type", "webserver", "tech",
		"response_body", "response_headers", "vhost", "created_at",
	}
	filename := fmt.Sprintf("target-%d-endpoints.csv", targetID)

	mapper := func(rows *sql.Rows) ([]string, error) {
		endpoint, err := h.svc.ScanRow(rows)
		if err != nil {
			return nil, err
		}

		statusCode := ""
		if endpoint.StatusCode != nil {
			statusCode = strconv.Itoa(*endpoint.StatusCode)
		}
		contentLength := ""
		if endpoint.ContentLength != nil {
			contentLength = strconv.Itoa(*endpoint.ContentLength)
		}
		vhost := ""
		if endpoint.Vhost != nil {
			vhost = strconv.FormatBool(*endpoint.Vhost)
		}
		tech := ""
		if len(endpoint.Tech) > 0 {
			tech = strings.Join(endpoint.Tech, "|")
		}

		return []string{
			strconv.Itoa(endpoint.ID),
			strconv.Itoa(endpoint.TargetID),
			endpoint.URL,
			endpoint.Host,
			endpoint.Location,
			endpoint.Title,
			statusCode,
			contentLength,
			endpoint.ContentType,
			endpoint.Webserver,
			tech,
			endpoint.ResponseBody,
			endpoint.ResponseHeaders,
			vhost,
			timeutil.FormatRFC3339NanoUTC(endpoint.CreatedAt),
		}, nil
	}

	if err := csv.StreamCSV(c, rows, headers, filename, mapper, count); err != nil {
		return
	}
}
