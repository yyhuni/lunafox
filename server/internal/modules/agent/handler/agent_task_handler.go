package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scandto "github.com/yyhuni/lunafox/server/internal/modules/scan/dto"
)

// AgentTaskHandler handles task APIs for agents.
type AgentTaskHandler struct {
	service *agentapp.AgentTaskService
}

// NewAgentTaskHandler creates a new AgentTaskHandler.
func NewAgentTaskHandler(service *agentapp.AgentTaskService) *AgentTaskHandler {
	return &AgentTaskHandler{service: service}
}

// PullTask assigns a task to the agent.
// POST /api/agent/tasks/pull
func (h *AgentTaskHandler) PullTask(c *gin.Context) {
	agentID, ok := c.Get("agentID")
	if !ok {
		dto.Unauthorized(c, "Missing agent context")
		return
	}
	id, ok := agentID.(int)
	if !ok {
		dto.Unauthorized(c, "Invalid agent context")
		return
	}

	task, err := h.service.PullTask(c.Request.Context(), id)
	if err != nil {
		dto.InternalError(c, "Failed to pull task")
		return
	}
	if task == nil {
		dto.NoContent(c)
		return
	}

	dto.Success(c, toScanTaskAssignmentOutput(task))
}

// UpdateTaskStatus updates task status for the agent.
// PATCH /api/agent/tasks/:taskId/status
func (h *AgentTaskHandler) UpdateTaskStatus(c *gin.Context) {
	agentID, ok := c.Get("agentID")
	if !ok {
		dto.Unauthorized(c, "Missing agent context")
		return
	}
	id, ok := agentID.(int)
	if !ok {
		dto.Unauthorized(c, "Invalid agent context")
		return
	}

	taskID, err := strconv.Atoi(c.Param("taskId"))
	if err != nil || taskID <= 0 {
		dto.BadRequest(c, "Invalid taskId")
		return
	}

	var req scandto.TaskStatusUpdateRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	if !isValidAgentTaskStatus(req.Status) {
		dto.BadRequest(c, "Invalid status value")
		return
	}
	if len(req.ErrorMessage) > 4096 {
		dto.BadRequest(c, "Error message exceeds 4KB limit")
		return
	}

	err = h.service.UpdateStatus(c.Request.Context(), id, taskID, req.Status, req.ErrorMessage)
	if err != nil {
		switch {
		case errors.Is(err, agentapp.ErrAgentTaskNotFound):
			dto.NotFound(c, "Task not found")
		case errors.Is(err, agentapp.ErrAgentTaskNotOwned):
			dto.Forbidden(c, "Task not owned by this agent")
		case errors.Is(err, agentapp.ErrAgentTaskInvalidTransition):
			dto.BadRequest(c, "Invalid status transition")
		case errors.Is(err, agentapp.ErrAgentTaskInvalidUpdate):
			dto.BadRequest(c, "Invalid task status update")
		default:
			dto.InternalError(c, "Failed to update task status")
		}
		return
	}

	c.Status(http.StatusOK)
}

func isValidAgentTaskStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "completed", "failed", "cancelled":
		return true
	default:
		return false
	}
}

func toScanTaskAssignmentOutput(task *scanapp.TaskAssignment) scandto.TaskAssignment {
	if task == nil {
		return scandto.TaskAssignment{}
	}
	return scandto.TaskAssignment{
		TaskID:       task.TaskID,
		ScanID:       task.ScanID,
		Stage:        task.Stage,
		WorkflowName: task.WorkflowName,
		TargetID:     task.TargetID,
		TargetName:   task.TargetName,
		TargetType:   task.TargetType,
		WorkspaceDir: task.WorkspaceDir,
		Config:       task.Config,
	}
}
