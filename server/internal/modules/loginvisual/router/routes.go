package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/handler"
)

func RegisterLoginVisualRoutes(api, protected *gin.RouterGroup, visual *handler.LoginVisualHandler) {
	if api == nil || protected == nil || visual == nil {
		panic("login visual route dependencies are required")
	}
	api.GET("/loginVisual/current", visual.PublicCurrent)
	api.GET("/loginVisual/current/media", visual.PublicMedia)
	api.GET("/loginVisual/current/poster", visual.PublicPoster)
	protected.GET("/settings/loginVisual", visual.GetSettings)
	protected.GET("/settings/loginVisual:customMethod", func(c *gin.Context) {
		httpdto.DispatchCustomMethod(c, "customMethod", map[string]gin.HandlerFunc{
			"checkDiscoverability": visual.CheckDiscoverability,
			"preview":              visual.Preview,
		})
	})
	protected.POST("/settings/loginVisual:customMethod", func(c *gin.Context) {
		httpdto.DispatchCustomMethod(c, "customMethod", map[string]gin.HandlerFunc{
			"unlockDiscoverability": visual.UnlockDiscoverability,
			"upload":                visual.Upload,
			"publish":               visual.Publish,
			"restoreDefault":        visual.RestoreDefault,
		})
	})
}
