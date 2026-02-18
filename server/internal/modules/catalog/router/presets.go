package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
)

// registerPresetRoutes registers routes for preset engines.
// These routes must be registered BEFORE /engines/:id to avoid route conflicts.
func registerPresetRoutes(protected *gin.RouterGroup, presetHandler *handler.PresetHandler) {
	protected.GET("/engines/presets", presetHandler.List)
	protected.GET("/engines/presets/:id", presetHandler.GetByID)
}
