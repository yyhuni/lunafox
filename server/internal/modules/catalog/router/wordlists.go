package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
)

func registerWordlistRoutes(protected *gin.RouterGroup, wordlistHandler *handler.WordlistHandler) {
	protected.POST("/wordlists", wordlistHandler.Create)
	protected.GET("/wordlists", wordlistHandler.List)
	protected.GET("/wordlistTags", wordlistHandler.ListTags)
	protected.GET("/wordlists/:wordlist", wordlistHandler.GetByID)
	protected.PATCH("/wordlists/:wordlist", wordlistHandler.Update)
	protected.GET("/wordlists/:wordlist/blob", wordlistHandler.DownloadByID)
	protected.DELETE("/wordlists/:wordlist", wordlistHandler.Delete)
	protected.GET("/wordlists/:wordlist/text", wordlistHandler.GetContent)
	protected.PATCH("/wordlists/:wordlist/text", wordlistHandler.UpdateContent)
}
