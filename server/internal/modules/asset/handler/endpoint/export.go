package endpoint

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

// Export exports endpoints as CSV.
// GET /v1/targets/:target/endpoints/exportFiles/current
func (h *EndpointHandler) Export(c *gin.Context) {
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
		httpdto.InternalError(c, "Failed to export endpoints")
		return
	}

	headers := []string{
		"id", "target_id", "url", "host", "location", "title", "status_code",
		"content_length", "content_type", "webserver", "tech",
		"response_body", "response_headers", "vhost", "created_at",
	}
	filename := fmt.Sprintf("target-%d-endpoints.csv", targetID)

	producer := func(write csv.RowWriter) error {
		return h.svc.ForEachByTarget(targetID, func(endpoint service.Endpoint) error {
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

			return write([]string{
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
			})
		})
	}

	if err := csv.StreamCSV(c, headers, filename, producer, count); err != nil {
		return
	}
}
