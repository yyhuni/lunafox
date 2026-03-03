package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

// ListAgents returns a paginated list of agents.
// GET /api/admin/agents
func (h *AgentHandler) ListAgents(c *gin.Context) {
	var query struct {
		dto.PaginationQuery
		Status  string `form:"status" binding:"omitempty,oneof=online offline"`
		Include string `form:"include"`
	}
	if !dto.BindQuery(c, &query) {
		return
	}

	page := query.GetPage()
	pageSize := query.GetPageSize()
	agents, total, err := h.facade.ListAgents(c.Request.Context(), page, pageSize, query.Status)
	if err != nil {
		dto.InternalError(c, "Failed to list agents")
		return
	}

	includeHeartbeat := includesHeartbeat(query.Include)
	results := make([]dto.AgentResponse, 0, len(agents))
	for _, agent := range agents {
		var heartbeat *dto.AgentHeartbeatResponse
		if includeHeartbeat && h.heartbeatCache != nil {
			data, cacheErr := h.heartbeatCache.Get(c.Request.Context(), agent.ID)
			if cacheErr == nil && data != nil {
				heartbeat = &dto.AgentHeartbeatResponse{
					CPU:           data.CPU,
					Mem:           data.Mem,
					Disk:          data.Disk,
					Tasks:         data.Tasks,
					Uptime:        data.Uptime,
					AgentVersion:  data.AgentVersion,
					WorkerVersion: data.WorkerVersion,
					UpdatedAt:     timeutil.ToUTC(data.UpdatedAt),
					Health:        data.Health,
				}
			}
		}
		results = append(results, toAgentOutput(agent, heartbeat))
	}

	dto.Paginated(c, results, total, page, pageSize)
}

// GetAgent returns an agent by ID.
// GET /api/admin/agents/:id
func (h *AgentHandler) GetAgent(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid agent ID")
		return
	}

	agent, err := h.facade.GetAgent(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, agentapp.ErrAgentNotFound) {
			dto.NotFound(c, "Agent not found")
			return
		}
		dto.InternalError(c, "Failed to get agent")
		return
	}

	var heartbeat *dto.AgentHeartbeatResponse
	if includesHeartbeat(c.Query("include")) && h.heartbeatCache != nil {
		data, cacheErr := h.heartbeatCache.Get(c.Request.Context(), agent.ID)
		if cacheErr == nil && data != nil {
			heartbeat = &dto.AgentHeartbeatResponse{
				CPU:           data.CPU,
				Mem:           data.Mem,
				Disk:          data.Disk,
				Tasks:         data.Tasks,
				Uptime:        data.Uptime,
				AgentVersion:  data.AgentVersion,
				WorkerVersion: data.WorkerVersion,
				UpdatedAt:     timeutil.ToUTC(data.UpdatedAt),
				Health:        data.Health,
			}
		}
	}

	dto.Success(c, toAgentOutput(agent, heartbeat))
}

func includesHeartbeat(include string) bool {
	if include == "" {
		return false
	}
	for _, part := range strings.Split(include, ",") {
		if strings.EqualFold(strings.TrimSpace(part), "heartbeat") {
			return true
		}
	}
	return false
}
