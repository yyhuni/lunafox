package handler

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

// DeleteAgent deletes an agent.
// DELETE /v1/admin/agents/:agent
func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("agent"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid agent ID")
		return
	}

	if err := h.facade.DeleteAgent(c.Request.Context(), id); err != nil {
		if errors.Is(err, agentapp.ErrAgentNotFound) {
			httpdto.NotFound(c, "Agent not found")
			return
		}
		httpdto.InternalError(c, "Failed to delete agent")
		return
	}
	httpdto.NoContent(c)
}

// UpdateAgentConfig updates an agent's scheduling configuration.
// PATCH /v1/admin/agents/:agent with body updateMask.
func (h *AgentHandler) UpdateAgentConfig(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("agent"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid agent ID")
		return
	}

	var req dto.UpdateAgentConfigRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.Name) != httpdto.AgentName(id) {
		httpdto.BadRequest(c, "Agent name must match the request path")
		return
	}
	if strings.TrimSpace(req.UpdateMask) == "" {
		httpdto.BadRequest(c, "updateMask is required")
		return
	}
	if req.MaxTasks == nil && req.CPUThreshold == nil && req.MemThreshold == nil && req.DiskThreshold == nil {
		httpdto.BadRequest(c, "at least one agent config field is required")
		return
	}
	if !agentConfigUpdateMaskMatchesBody(req) {
		httpdto.BadRequest(c, "updateMask must exactly match provided config fields")
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
			httpdto.NotFound(c, "Agent not found")
			return
		}
		httpdto.InternalError(c, "Failed to update agent config")
		return
	}

	if h.controlPublisher != nil {
		h.controlPublisher.SendConfigUpdate(agent)
	}

	httpdto.Success(c, toAgentOutput(agent, nil))
}

func agentConfigUpdateMaskMatchesBody(req dto.UpdateAgentConfigRequest) bool {
	provided := map[string]bool{
		"maxTasks":      req.MaxTasks != nil,
		"cpuThreshold":  req.CPUThreshold != nil,
		"memThreshold":  req.MemThreshold != nil,
		"diskThreshold": req.DiskThreshold != nil,
	}
	for _, part := range strings.Split(req.UpdateMask, ",") {
		field := strings.TrimSpace(part)
		hasField, ok := provided[field]
		if !ok || !hasField {
			return false
		}
		delete(provided, field)
	}
	for _, hasField := range provided {
		if hasField {
			return false
		}
	}
	return true
}
