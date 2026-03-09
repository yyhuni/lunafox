package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

// WorkflowProfileHandler handles HTTP requests for workflow profiles.
type WorkflowProfileHandler struct {
	service *service.WorkflowCatalogFacade
}

// NewWorkflowProfileHandler creates a new WorkflowProfileHandler.
func NewWorkflowProfileHandler(service *service.WorkflowCatalogFacade) *WorkflowProfileHandler {
	return &WorkflowProfileHandler{service: service}
}

// List returns all workflow profiles.
// GET /api/workflows/profiles
func (h *WorkflowProfileHandler) List(c *gin.Context) {
	profiles, err := h.service.ListProfiles()
	if err != nil {
		dto.InternalError(c, "Failed to list workflow profiles")
		return
	}

	responses := dto.NewProfileListResponse(profiles)
	dto.Success(c, responses)
}

// GetByID returns a single workflow profile by ID.
// GET /api/workflows/profiles/:id
func (h *WorkflowProfileHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	profile, err := h.service.GetProfileByID(id)
	if err != nil {
		if errors.Is(err, service.ErrWorkflowProfileNotFound) {
			dto.NotFound(c, "Workflow profile not found")
			return
		}
		dto.InternalError(c, "Failed to get workflow profile")
		return
	}

	dto.Success(c, dto.NewProfileResponse(profile))
}
