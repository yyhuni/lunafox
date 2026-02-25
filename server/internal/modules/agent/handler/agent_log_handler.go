package handler

import (
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
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

const (
	defaultAgentLogLimit = 200
	maxAgentLogLimit     = 500
)

var logContainerNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

type AgentLogHandler struct {
	agentRepo       agentdomain.AgentRepository
	logQueryService *agentapp.LokiLogQueryService
}

type agentLogQuery struct {
	Container string
	Limit     int
	Cursor    string
}

func NewAgentLogHandler(
	agentRepo agentdomain.AgentRepository,
	logQueryService *agentapp.LokiLogQueryService,
) *AgentLogHandler {
	return &AgentLogHandler{
		agentRepo:       agentRepo,
		logQueryService: logQueryService,
	}
}

// List returns agent container logs from Loki.
// GET /api/admin/agents/:id/logs
func (h *AgentLogHandler) List(c *gin.Context) {
	agentID, ok := parseAgentLogPositiveID(c.Param("id"))
	if !ok {
		dto.Error(c, http.StatusBadRequest, "bad_request", "Invalid agent ID")
		return
	}

	query, err := parseAgentLogQuery(c)
	if err != nil {
		dto.Error(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	agent, err := h.agentRepo.GetByID(c.Request.Context(), agentID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			dto.Error(c, http.StatusNotFound, "agent_not_found", "Agent not found")
			return
		}
		dto.Error(c, http.StatusInternalServerError, "internal_error", "Failed to load agent")
		return
	}
	if agent == nil {
		dto.Error(c, http.StatusNotFound, "agent_not_found", "Agent not found")
		return
	}

	result, err := h.logQueryService.Query(c.Request.Context(), agentapp.LokiLogQueryInput{
		AgentID:   agentID,
		Container: query.Container,
		Limit:     query.Limit,
		Cursor:    query.Cursor,
	})
	if err != nil {
		switch {
		case errors.Is(err, agentapp.ErrLogCursorInvalid), errors.Is(err, agentapp.ErrLogCursorQueryMismatch):
			dto.Error(c, http.StatusBadRequest, "bad_request", "Invalid cursor")
			return
		case errors.Is(err, agentapp.ErrLokiContainerNotFound):
			dto.Error(c, http.StatusNotFound, "container_not_found", "Container logs not found")
			return
		case errors.Is(err, agentapp.ErrLokiQueryTimeout):
			dto.Error(c, http.StatusGatewayTimeout, "query_timeout", "Log query timed out")
			return
		case errors.Is(err, loki.ErrLokiUnavailable):
			dto.Error(c, http.StatusServiceUnavailable, "loki_unavailable", "Loki is unavailable")
			return
		default:
			dto.Error(c, http.StatusInternalServerError, "internal_error", "Failed to query logs")
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

	dto.Success(c, dto.AgentLogListResponse{
		Logs:       items,
		NextCursor: result.NextCursor,
		HasMore:    result.HasMore,
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

	limit := defaultAgentLogLimit
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		value, err := strconv.Atoi(rawLimit)
		if err != nil || value <= 0 || value > maxAgentLogLimit {
			return nil, fmt.Errorf("limit must be between 1 and %d", maxAgentLogLimit)
		}
		limit = value
	}

	if _, exists := c.GetQuery("direction"); exists {
		return nil, errors.New("direction is deprecated, please remove it")
	}

	return &agentLogQuery{
		Container: container,
		Limit:     limit,
		Cursor:    strings.TrimSpace(c.Query("cursor")),
	}, nil
}

func parseAgentLogPositiveID(raw string) (int, bool) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}
