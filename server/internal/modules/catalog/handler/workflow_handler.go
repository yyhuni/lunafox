package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

// WorkflowHandler handles read-only workflow catalog endpoints.
type WorkflowHandler struct {
	service *service.WorkflowCatalogFacade
}

// NewWorkflowHandler creates a new WorkflowHandler.
func NewWorkflowHandler(service *service.WorkflowCatalogFacade) *WorkflowHandler {
	return &WorkflowHandler{service: service}
}

// List returns all available workflows.
// GET /api/workflows
func (h *WorkflowHandler) List(c *gin.Context) {
	workflows, err := h.service.ListWorkflows()
	if err != nil {
		dto.InternalError(c, "Failed to list workflows")
		return
	}

	dto.Success(c, dto.NewWorkflowListResponse(workflows))
}

// GetByWorkflowID returns a workflow by id.
// GET /api/workflows/:workflowId
func (h *WorkflowHandler) GetByWorkflowID(c *gin.Context) {
	workflowID := c.Param("workflowId")

	workflow, err := h.service.GetWorkflowByID(workflowID)
	if err != nil {
		if errors.Is(err, service.ErrWorkflowNotFound) {
			dto.NotFound(c, "Workflow not found")
			return
		}
		dto.InternalError(c, "Failed to get workflow")
		return
	}

	dto.Success(c, dto.NewWorkflowResponse(workflow))
}
