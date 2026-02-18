package website

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

// Export exports websites as CSV.
// GET /api/targets/:id/websites/export
func (h *WebsiteHandler) Export(c *gin.Context) {
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
		dto.InternalError(c, "Failed to export websites")
		return
	}

	rows, err := h.svc.StreamByTarget(targetID)
	if err != nil {
		dto.InternalError(c, "Failed to export websites")
		return
	}

	headers := []string{
		"id", "target_id", "url", "host", "location", "title", "status_code",
		"content_length", "content_type", "webserver", "tech",
		"response_body", "response_headers", "vhost", "created_at",
	}
	filename := fmt.Sprintf("target-%d-websites.csv", targetID)

	mapper := func(rows *sql.Rows) ([]string, error) {
		website, err := h.svc.ScanRow(rows)
		if err != nil {
			return nil, err
		}

		statusCode := ""
		if website.StatusCode != nil {
			statusCode = strconv.Itoa(*website.StatusCode)
		}
		contentLength := ""
		if website.ContentLength != nil {
			contentLength = strconv.Itoa(*website.ContentLength)
		}
		vhost := ""
		if website.Vhost != nil {
			vhost = strconv.FormatBool(*website.Vhost)
		}
		tech := ""
		if len(website.Tech) > 0 {
			tech = strings.Join(website.Tech, "|")
		}

		return []string{
			strconv.Itoa(website.ID),
			strconv.Itoa(website.TargetID),
			website.URL,
			website.Host,
			website.Location,
			website.Title,
			statusCode,
			contentLength,
			website.ContentType,
			website.Webserver,
			tech,
			website.ResponseBody,
			website.ResponseHeaders,
			vhost,
			timeutil.FormatRFC3339NanoUTC(website.CreatedAt),
		}, nil
	}

	if err := csv.StreamCSV(c, rows, headers, filename, mapper, count); err != nil {
		return
	}
}
