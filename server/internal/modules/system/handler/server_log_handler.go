package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/loki"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	systemapp "github.com/yyhuni/lunafox/server/internal/modules/system/application"
	"github.com/yyhuni/lunafox/server/internal/modules/system/dto"
)

const (
	defaultServerLogPageSize = 200
	maxServerLogPageSize     = 500
)

type ServerLogHandler struct {
	logQueryService serverLogQueryService
}

type serverLogQueryService interface {
	Query(ctx context.Context, input systemapp.ServerLogQueryInput) (systemapp.ServerLogQueryResult, error)
}

type serverLogQuery struct {
	Limit     int
	PageToken string
	Direction string
}

func NewServerLogHandler(logQueryService serverLogQueryService) *ServerLogHandler {
	return &ServerLogHandler{logQueryService: logQueryService}
}

// List returns Server/control-plane container logs from Loki.
// GET /v1/admin/system/logEntries
func (h *ServerLogHandler) List(c *gin.Context) {
	if h.logQueryService == nil {
		httpdto.Error(c, http.StatusInternalServerError, "internal_error", "Server log service is not configured")
		return
	}

	query, err := parseServerLogQuery(c)
	if err != nil {
		httpdto.Error(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	result, err := h.logQueryService.Query(c.Request.Context(), systemapp.ServerLogQueryInput{
		Limit:     query.Limit,
		Cursor:    query.PageToken,
		Direction: query.Direction,
	})
	if err != nil {
		switch {
		case errors.Is(err, systemapp.ErrServerLogCursorInvalid), errors.Is(err, systemapp.ErrServerLogCursorQueryMismatch):
			httpdto.Error(c, http.StatusBadRequest, "bad_request", "Invalid pageToken")
			return
		case errors.Is(err, systemapp.ErrServerLogQueryTimeout):
			httpdto.Error(c, http.StatusGatewayTimeout, "query_timeout", "Log query timed out")
			return
		case errors.Is(err, loki.ErrLokiUnavailable):
			httpdto.Error(c, http.StatusServiceUnavailable, "loki_unavailable", "Loki is unavailable")
			return
		default:
			httpdto.Error(c, http.StatusInternalServerError, "internal_error", "Failed to query server logs")
			return
		}
	}

	items := make([]dto.ServerLogItem, 0, len(result.Logs))
	for _, item := range result.Logs {
		items = append(items, dto.ServerLogItem{
			ID:        item.ID,
			TS:        item.TS,
			TSNs:      item.TSNs,
			Stream:    item.Stream,
			Line:      item.Line,
			Truncated: item.Truncated,
		})
	}

	httpdto.Success(c, dto.ServerLogListResponse{
		Results:           items,
		NextPageToken:     result.NextCursor,
		PreviousPageToken: result.PreviousCursor,
		HasOlder:          result.HasOlder,
		HasNewer:          result.HasNewer,
		CaughtUp:          result.CaughtUp,
		Gap:               result.Gap,
		GapReason:         result.GapReason,
	})
}

func parseServerLogQuery(c *gin.Context) (*serverLogQuery, error) {
	if strings.TrimSpace(c.Query("container")) != "" ||
		strings.TrimSpace(c.Query("file")) != "" ||
		strings.TrimSpace(c.Query("query")) != "" ||
		strings.TrimSpace(c.Query("selector")) != "" {
		return nil, errors.New("server log source is fixed and cannot be overridden")
	}
	if strings.TrimSpace(c.Query("limit")) != "" || strings.TrimSpace(c.Query("cursor")) != "" {
		return nil, errors.New("use pageSize and pageToken for pagination")
	}

	limit := defaultServerLogPageSize
	if rawLimit := strings.TrimSpace(c.Query("pageSize")); rawLimit != "" {
		value, err := strconv.Atoi(rawLimit)
		if err != nil || value <= 0 || value > maxServerLogPageSize {
			return nil, fmt.Errorf("pageSize must be between 1 and %d", maxServerLogPageSize)
		}
		limit = value
	}

	direction := strings.ToLower(strings.TrimSpace(c.Query("direction")))
	switch direction {
	case "", "newer", "older":
	default:
		return nil, errors.New("direction must be newer or older")
	}

	return &serverLogQuery{
		Limit:     limit,
		PageToken: strings.TrimSpace(c.Query("pageToken")),
		Direction: direction,
	}, nil
}
