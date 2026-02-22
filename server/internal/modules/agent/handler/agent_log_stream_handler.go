package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
)

const (
	defaultLogTail = 200
	maxLogTail     = 2000
)

var containerNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

// AgentLogStreamHandler serves admin SSE log streaming endpoints.
type AgentLogStreamHandler struct {
	agentRepo        agentdomain.AgentRepository
	logStreamService *agentapp.LogStreamService
}

func NewAgentLogStreamHandler(agentRepo agentdomain.AgentRepository, logStreamService *agentapp.LogStreamService) *AgentLogStreamHandler {
	return &AgentLogStreamHandler{agentRepo: agentRepo, logStreamService: logStreamService}
}

// Stream streams container logs from target agent through SSE.
// GET /api/admin/agents/:id/logs/stream
func (h *AgentLogStreamHandler) Stream(c *gin.Context) {
	agentID, ok := parsePositiveID(c.Param("id"))
	if !ok {
		dto.BadRequest(c, "Invalid agent ID")
		return
	}

	query, err := parseLogStreamQuery(c)
	if err != nil {
		dto.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	agent, err := h.agentRepo.GetByID(c.Request.Context(), agentID)
	if err != nil {
		if errors.Is(err, agentapp.ErrAgentNotFound) {
			dto.NotFound(c, "Agent not found")
			return
		}
		dto.InternalError(c, "Failed to load agent")
		return
	}
	if agent == nil {
		dto.NotFound(c, "Agent not found")
		return
	}
	if !strings.EqualFold(strings.TrimSpace(agent.Status), "online") {
		dto.Error(c, http.StatusNotFound, "AGENT_OFFLINE", "Agent is offline")
		return
	}

	sub, err := h.logStreamService.Open(c.Request.Context(), agentapp.LogStreamOpenRequest{
		AgentID:    agentID,
		Container:  query.Container,
		Tail:       query.Tail,
		Follow:     query.Follow,
		Since:      query.Since,
		Timestamps: query.Timestamps,
	})
	if err != nil {
		if errors.Is(err, agentapp.ErrLogStreamSendFailed) {
			dto.Error(c, http.StatusServiceUnavailable, "WS_SEND_FAILED", "Failed to send log request to agent")
			return
		}
		dto.InternalError(c, "Failed to open log stream")
		return
	}
	defer h.logStreamService.Cancel(sub.RequestID, "client_closed")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		dto.InternalError(c, "Streaming is not supported")
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	flusher.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event, ok := <-sub.Events:
			if !ok {
				return
			}
			if err := writeSSE(c.Writer, event); err != nil {
				return
			}
			flusher.Flush()
			if event.Type == agentapp.LogStreamEventDone {
				return
			}
		}
	}
}

type logStreamQuery struct {
	Container  string
	Tail       int
	Follow     bool
	Since      *time.Time
	Timestamps bool
}

func parseLogStreamQuery(c *gin.Context) (*logStreamQuery, error) {
	containerName := strings.TrimSpace(c.Query("container"))
	if containerName == "" {
		return nil, errors.New("container is required")
	}
	if !containerNamePattern.MatchString(containerName) {
		return nil, errors.New("invalid container format")
	}

	tail := defaultLogTail
	if rawTail := strings.TrimSpace(c.Query("tail")); rawTail != "" {
		value, err := strconv.Atoi(rawTail)
		if err != nil || value < 0 || value > maxLogTail {
			return nil, fmt.Errorf("tail must be between 0 and %d", maxLogTail)
		}
		tail = value
	}

	follow := true
	if rawFollow := strings.TrimSpace(c.Query("follow")); rawFollow != "" {
		value, err := strconv.ParseBool(rawFollow)
		if err != nil {
			return nil, errors.New("follow must be true or false")
		}
		follow = value
	}

	timestamps := true
	if rawTimestamps := strings.TrimSpace(c.Query("timestamps")); rawTimestamps != "" {
		value, err := strconv.ParseBool(rawTimestamps)
		if err != nil {
			return nil, errors.New("timestamps must be true or false")
		}
		timestamps = value
	}

	var since *time.Time
	if rawSince := strings.TrimSpace(c.Query("since")); rawSince != "" {
		parsed, err := time.Parse(time.RFC3339Nano, rawSince)
		if err != nil {
			return nil, errors.New("since must be RFC3339 datetime")
		}
		utc := parsed.UTC()
		since = &utc
	}

	return &logStreamQuery{
		Container:  containerName,
		Tail:       tail,
		Follow:     follow,
		Since:      since,
		Timestamps: timestamps,
	}, nil
}

func writeSSE(w http.ResponseWriter, event agentapp.LogStreamEvent) error {
	name := event.Type
	if strings.TrimSpace(name) == "" {
		return nil
	}

	payload := buildSSEPayload(event)
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, data); err != nil {
		return err
	}
	return nil
}

func buildSSEPayload(event agentapp.LogStreamEvent) any {
	switch event.Type {
	case agentapp.LogStreamEventStatus:
		return struct {
			RequestID string `json:"requestId"`
			Status    string `json:"status"`
		}{RequestID: event.RequestID, Status: event.Status}
	case agentapp.LogStreamEventLog:
		return struct {
			RequestID string    `json:"requestId"`
			TS        time.Time `json:"ts"`
			Stream    string    `json:"stream"`
			Line      string    `json:"line"`
			Truncated bool      `json:"truncated"`
		}{
			RequestID: event.RequestID,
			TS:        event.TS.UTC(),
			Stream:    event.Stream,
			Line:      event.Line,
			Truncated: event.Truncated,
		}
	case agentapp.LogStreamEventError:
		return struct {
			RequestID string `json:"requestId"`
			Code      string `json:"code"`
			Message   string `json:"message"`
		}{RequestID: event.RequestID, Code: event.Code, Message: event.Message}
	case agentapp.LogStreamEventDone:
		return struct {
			RequestID string `json:"requestId"`
			Reason    string `json:"reason"`
		}{RequestID: event.RequestID, Reason: event.Reason}
	case agentapp.LogStreamEventPing:
		return struct {
			RequestID string    `json:"requestId"`
			TS        time.Time `json:"ts"`
		}{RequestID: event.RequestID, TS: event.TS.UTC()}
	default:
		return struct {
			RequestID string `json:"requestId"`
		}{RequestID: event.RequestID}
	}
}

func parsePositiveID(raw string) (int, bool) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}
