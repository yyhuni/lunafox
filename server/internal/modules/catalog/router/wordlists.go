package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
)

func registerWordlistRoutes(protected *gin.RouterGroup, wordlistHandler *handler.WordlistHandler) {
	protected.POST("/wordlists", wordlistHandler.Create)
	protected.GET("/wordlists", wordlistHandler.List)
	protected.GET("/wordlists/:id", wordlistHandler.Get)
	protected.GET("/wordlists/:id/download", wordlistHandler.DownloadByID)
	protected.DELETE("/wordlists/:id", wordlistHandler.Delete)
	protected.GET("/wordlists/:id/content", wordlistHandler.GetContent)
	protected.PUT("/wordlists/:id/content", wordlistHandler.UpdateContent)
}
