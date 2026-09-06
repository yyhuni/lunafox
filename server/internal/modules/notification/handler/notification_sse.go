package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type notificationRefreshBroker interface {
	Subscribe(int) (<-chan struct{}, func())
}

const defaultNotificationSSEHeartbeatInterval = 15 * time.Second

// NotificationSSEHandler exposes a one-way, authenticated invalidation stream.
type NotificationSSEHandler struct {
	broker            notificationRefreshBroker
	heartbeatInterval time.Duration
}

// NewNotificationSSEHandler creates the fetch-SSE boundary.
func NewNotificationSSEHandler(broker notificationRefreshBroker) *NotificationSSEHandler {
	if broker == nil {
		panic("notification SSE broker is required")
	}
	return &NotificationSSEHandler{
		broker:            broker,
		heartbeatInterval: defaultNotificationSSEHeartbeatInterval,
	}
}

// Stream handles GET /v1/users/current/notifications:stream.
func (handler *NotificationSSEHandler) Stream(c *gin.Context) {
	if len(c.Request.URL.Query()) != 0 {
		httpdto.BadRequest(c, "Notification stream does not accept query parameters")
		return
	}
	claims, ok := middleware.GetUserClaims(c)
	if !ok || claims.ExpiresAt == nil || claims.ExpiresAt.Time.IsZero() {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	deadline := claims.ExpiresAt.Time.UTC()
	if !deadline.After(time.Now().UTC()) {
		httpdto.Error(c, http.StatusUnauthorized, "TOKEN_EXPIRED", "Token has expired")
		return
	}
	updates, unsubscribe := handler.broker.Subscribe(claims.UserID)
	defer unsubscribe()
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		httpdto.InternalError(c, "Streaming response is unavailable")
		return
	}
	// The Server's ordinary WriteTimeout is shorter than an access token. Move
	// this response deadline to JWT expiry so the durable-stream contract, not
	// the global request default, determines when a valid SSE stream ends.
	if err := http.NewResponseController(c.Writer).SetWriteDeadline(deadline); err != nil && !errors.Is(err, http.ErrNotSupported) {
		httpdto.InternalError(c, "Streaming response is unavailable")
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	c.Writer.WriteHeaderNow()
	flusher.Flush()

	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	heartbeatTimer := time.NewTimer(handler.heartbeatInterval)
	defer heartbeatTimer.Stop()
	resetHeartbeatTimer := func() {
		if !heartbeatTimer.Stop() {
			select {
			case <-heartbeatTimer.C:
			default:
			}
		}
		heartbeatTimer.Reset(handler.heartbeatInterval)
	}
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-timer.C:
			return
		case <-heartbeatTimer.C:
			// Comment heartbeats keep idle proxy paths active without creating a
			// notification invalidation or changing the durable inbox authority.
			if _, err := fmt.Fprint(c.Writer, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
			heartbeatTimer.Reset(handler.heartbeatInterval)
		case <-updates:
			if _, err := fmt.Fprint(c.Writer, "event: refresh\ndata: {}\n\n"); err != nil {
				return
			}
			flusher.Flush()
			resetHeartbeatTimer()
		}
	}
}
