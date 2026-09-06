package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/loki"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

const (
	defaultAgentLogLimit = 200
	maxAgentLogLimit     = 500
)

var logContainerNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

type AgentLogHandler struct {
	agentLookup     agentLogAgentLookup
	logQueryService agentLogQueryService
}

type agentLogAgentLookup interface {
	GetAgent(ctx context.Context, id int) (*agentdomain.Agent, error)
}

type agentLogQueryService interface {
	Query(ctx context.Context, input agentapp.LokiLogQueryInput) (agentapp.LokiLogQueryResult, error)
}

type agentLogQuery struct {
	Container string
	Limit     int
	PageToken string
	Direction string
}

func NewAgentLogHandler(
	agentLookup agentLogAgentLookup,
	logQueryService agentLogQueryService,
) *AgentLogHandler {
	return &AgentLogHandler{
		agentLookup:     agentLookup,
		logQueryService: logQueryService,
	}
}

// List returns agent container logs from Loki.
// GET /v1/admin/agents/:agent/logEntries
func (h *AgentLogHandler) List(c *gin.Context) {
	agentID, err := httpdto.ParseResourceIDSegment(c.Param("agent"))
	if err != nil {
		httpdto.Error(c, http.StatusBadRequest, "bad_request", "Invalid agent ID")
		return
	}

	query, err := parseAgentLogQuery(c)
	if err != nil {
		httpdto.Error(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	agent, err := h.agentLookup.GetAgent(c.Request.Context(), agentID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) || errors.Is(err, agentapp.ErrAgentNotFound) {
			httpdto.Error(c, http.StatusNotFound, "agent_not_found", "Agent not found")
			return
		}
		httpdto.Error(c, http.StatusInternalServerError, "internal_error", "Failed to load agent")
		return
	}
	if agent == nil {
		httpdto.Error(c, http.StatusNotFound, "agent_not_found", "Agent not found")
		return
	}

	result, err := h.logQueryService.Query(c.Request.Context(), agentapp.LokiLogQueryInput{
		AgentID:   agentID,
		Container: query.Container,
		Limit:     query.Limit,
		Cursor:    query.PageToken,
		Direction: query.Direction,
	})
	if err != nil {
		switch {
		case errors.Is(err, agentapp.ErrLogCursorInvalid), errors.Is(err, agentapp.ErrLogCursorQueryMismatch):
			httpdto.Error(c, http.StatusBadRequest, "bad_request", "Invalid pageToken")
			return
		case errors.Is(err, agentapp.ErrLokiContainerNotFound):
			httpdto.Error(c, http.StatusNotFound, "container_not_found", "Container logs not found")
			return
		case errors.Is(err, agentapp.ErrLokiQueryTimeout):
			httpdto.Error(c, http.StatusGatewayTimeout, "query_timeout", "Log query timed out")
			return
		case errors.Is(err, loki.ErrLokiUnavailable):
			httpdto.Error(c, http.StatusServiceUnavailable, "loki_unavailable", "Loki is unavailable")
			return
		default:
			httpdto.Error(c, http.StatusInternalServerError, "internal_error", "Failed to query logs")
			return
		}
	}

	items := make([]dto.AgentLogItem, 0, len(result.Logs))
	for _, item := range result.Logs {
		items = append(items, dto.AgentLogItem{
			ID:        item.ID,
			TS:        item.TS,
			TSNs:      item.TSNs,
			Stream:    item.Stream,
			Line:      item.Line,
			Truncated: item.Truncated,
		})
	}

	httpdto.Success(c, dto.AgentLogListResponse{
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

func parseAgentLogQuery(c *gin.Context) (*agentLogQuery, error) {
	container := strings.TrimSpace(c.Query("container"))
	if container == "" {
		return nil, errors.New("container is required")
	}
	if !logContainerNamePattern.MatchString(container) {
		return nil, errors.New("invalid container format")
	}
	if strings.TrimSpace(c.Query("limit")) != "" || strings.TrimSpace(c.Query("cursor")) != "" {
		return nil, errors.New("use pageSize and pageToken for pagination")
	}

	limit := defaultAgentLogLimit
	if rawLimit := strings.TrimSpace(c.Query("pageSize")); rawLimit != "" {
		value, err := strconv.Atoi(rawLimit)
		if err != nil || value <= 0 || value > maxAgentLogLimit {
			return nil, fmt.Errorf("pageSize must be between 1 and %d", maxAgentLogLimit)
		}
		limit = value
	}

	direction := strings.ToLower(strings.TrimSpace(c.Query("direction")))
	switch direction {
	case "", "newer", "older":
	default:
		return nil, errors.New("direction must be newer or older")
	}

	return &agentLogQuery{
		Container: container,
		Limit:     limit,
		PageToken: strings.TrimSpace(c.Query("pageToken")),
		Direction: direction,
	}, nil
}
