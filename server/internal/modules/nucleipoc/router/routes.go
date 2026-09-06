package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/handler"
)

// RegisterNucleiPOCRoutes exposes only the replacement source/catalog
// boundary. The retired /nuclei/repos surface intentionally has no alias.
func RegisterNucleiPOCRoutes(protected *gin.RouterGroup, nucleiHandler *handler.NucleiPOCHandler) {
	protected.POST("/nucleiPocSources:sync", nucleiHandler.Sync)
	protected.GET("/nucleiPocSources/current", nucleiHandler.CurrentSource)
	protected.GET("/nucleiPocSyncTasks/:task", nucleiHandler.GetSyncTask)
	protected.GET("/nucleiPocs", nucleiHandler.List)
	protected.GET("/nucleiPocs/filterOptions", nucleiHandler.FilterOptions)
	protected.POST("/nucleiPocs:setActivation", nucleiHandler.SetActivation)
	protected.GET("/nucleiPocs/:nucleiPoc", nucleiHandler.Get)
	protected.PATCH("/nucleiPocs/:nucleiPoc", nucleiHandler.Update)
}
