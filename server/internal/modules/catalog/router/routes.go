package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
)

// RegisterCatalogRoutes registers target/engine/wordlist/preset routes.
func RegisterCatalogRoutes(
	protected *gin.RouterGroup,
	wordlistHandler *handler.WordlistHandler,
	targetHandler *handler.TargetHandler,
	engineHandler *handler.EngineHandler,
	presetHandler *handler.PresetHandler,
) {
	registerTargetRoutes(protected, targetHandler)
	registerPresetRoutes(protected, presetHandler)
	registerEngineRoutes(protected, engineHandler)
	registerWordlistRoutes(protected, wordlistHandler)
}
