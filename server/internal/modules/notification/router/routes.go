// Package router registers notification HTTP and realtime resources.
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/handler"
)

// RegisterNotificationRoutes registers only the current-user Inbox, settings,
// and fetch-SSE contracts. Delivery history/redrive remains intentionally absent.
func RegisterNotificationRoutes(protected *gin.RouterGroup, inbox *handler.NotificationInboxHandler, destinations *handler.NotificationDestinationHandler, stream *handler.NotificationSSEHandler) {
	if protected == nil || inbox == nil || destinations == nil || stream == nil {
		panic("notification route dependencies are required")
	}
	protected.GET("/users/current/notifications", inbox.List)
	protected.GET("/users/current/notifications:customMethod", func(c *gin.Context) {
		httpdto.DispatchCustomMethod(c, "customMethod", map[string]gin.HandlerFunc{
			"unreadCount": inbox.UnreadCount,
			"stream":      stream.Stream,
		})
	})
	protected.GET("/users/current/notifications/:notification", inbox.Get)
	protected.POST("/users/current/notifications", func(c *gin.Context) {
		httpdto.NotFound(c, "Notification command not found")
	})
	protected.POST("/users/current/notifications:customMethod", func(c *gin.Context) {
		httpdto.DispatchCustomMethod(c, "customMethod", map[string]gin.HandlerFunc{
			"markAllRead": inbox.MarkAllRead,
		})
	})
	protected.POST("/users/current/notifications/:notification", inbox.MarkRead)

	protected.GET("/users/current/notificationLocale", inbox.GetLocale)
	protected.PATCH("/users/current/notificationLocale", inbox.UpdateLocale)

	protected.GET("/settings/notificationDestinations", destinations.List)
	// Gin permits only one wildcard in this segment. The handler deliberately
	// parses `discord:testDelivery` so the public AIP custom-method URL remains
	// canonical without inventing a second resource path.
	protected.POST("/settings/notificationDestinations/:destinationTestMethod", destinations.TestDelivery)
	protected.GET("/settings/notificationDestinations/:destination", destinations.Get)
	protected.PATCH("/settings/notificationDestinations/:destination", destinations.Update)
}
