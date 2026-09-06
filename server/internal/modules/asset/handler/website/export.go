package website

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

// Export exports websites as CSV.
// GET /v1/targets/:target/websites/exportFiles/current
func (h *WebsiteHandler) Export(c *gin.Context) {
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
		httpdto.InternalError(c, "Failed to export websites")
		return
	}

	headers := []string{
		"id", "target_id", "url", "host", "location", "title", "status_code",
		"content_length", "content_type", "webserver", "tech",
		"response_body", "response_headers", "vhost", "created_at",
	}
	filename := fmt.Sprintf("target-%d-websites.csv", targetID)

	producer := func(write csv.RowWriter) error {
		return h.svc.ForEachByTarget(targetID, func(website service.Website) error {
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

			return write([]string{
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
			})
		})
	}

	if err := csv.StreamCSV(c, headers, filename, producer, count); err != nil {
		return
	}
}
