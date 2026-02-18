package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
)

// DeleteAgent deletes an agent.
// DELETE /api/admin/agents/:id
func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid agent ID")
		return
	}

	if err := h.facade.DeleteAgent(c.Request.Context(), id); err != nil {
		if errors.Is(err, agentapp.ErrAgentNotFound) {
			dto.NotFound(c, "Agent not found")
			return
		}
		dto.InternalError(c, "Failed to delete agent")
		return
	}
	dto.NoContent(c)
}

// UpdateAgentConfig updates an agent's scheduling configuration.
// PATCH /api/admin/agents/:id/config
func (h *AgentHandler) UpdateAgentConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid agent ID")
		return
	}

	var req dto.UpdateAgentConfigRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	update := agentdomain.AgentConfigUpdate{
		MaxTasks:      req.MaxTasks,
		CPUThreshold:  req.CPUThreshold,
		MemThreshold:  req.MemThreshold,
		DiskThreshold: req.DiskThreshold,
	}

	agent, err := h.facade.UpdateAgentConfig(c.Request.Context(), id, update)
	if err != nil {
		if errors.Is(err, agentapp.ErrAgentNotFound) {
			dto.NotFound(c, "Agent not found")
			return
		}
		dto.InternalError(c, "Failed to update agent config")
		return
	}

	if h.runtimeService != nil {
		h.runtimeService.SendConfigUpdate(agent)
	}

	dto.Success(c, toAgentOutput(agent, nil))
}
