package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
	"github.com/yyhuni/lunafox/server/internal/preset"
)

// PresetHandler handles HTTP requests for preset engines.
type PresetHandler struct {
	service *preset.Service
}

// NewPresetHandler creates a new PresetHandler.
func NewPresetHandler(service *preset.Service) *PresetHandler {
	return &PresetHandler{service: service}
}

// List returns all preset engines.
// GET /api/engines/presets
func (h *PresetHandler) List(c *gin.Context) {
	presets := h.service.List()
	responses := dto.NewPresetListResponse(presets)
	dto.Success(c, responses)
}

// GetByID returns a single preset engine by ID.
// GET /api/engines/presets/:id
func (h *PresetHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	p, err := h.service.GetByID(id)
	if err != nil {
		if errors.Is(err, preset.ErrPresetNotFound) {
			dto.NotFound(c, "Preset engine not found")
			return
		}
		dto.InternalError(c, "Failed to get preset")
		return
	}

	dto.Success(c, dto.NewPresetResponse(p))
}
