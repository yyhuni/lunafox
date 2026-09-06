package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	notificationapp "github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/dto"
	"gorm.io/gorm"
)

type inboxNotificationService interface {
	List(context.Context, int, notificationapp.InboxListInput) (notificationapp.InboxPage, error)
	Get(context.Context, int, int64) (domain.InboxItem, error)
	UnreadCount(context.Context, int) (int64, error)
	MarkRead(context.Context, int, int64) (domain.InboxItem, error)
	MarkAllRead(context.Context, int) error
	Locale(context.Context, int) (domain.Locale, error)
	UpdateLocale(context.Context, int, domain.Locale) error
}

// NotificationInboxHandler serves only current-user notification resources.
type NotificationInboxHandler struct {
	service inboxNotificationService
}

// NewNotificationInboxHandler creates the current-user inbox HTTP boundary.
func NewNotificationInboxHandler(service inboxNotificationService) *NotificationInboxHandler {
	if service == nil {
		panic("notification inbox handler service is required")
	}
	return &NotificationInboxHandler{service: service}
}

// List handles GET /v1/users/current/notifications.
func (handler *NotificationInboxHandler) List(c *gin.Context) {
	if hasUnsupportedNotificationListQuery(c) {
		httpdto.BadRequest(c, "Unsupported query parameter")
		return
	}
	userID, ok := notificationCurrentUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	var query dto.InboxListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}
	page, err := handler.service.List(c.Request.Context(), userID, notificationapp.InboxListInput{PageSize: query.PageSize, PageToken: query.PageToken})
	if err != nil {
		handler.writeInboxError(c, err, "list notifications")
		return
	}
	results := make([]dto.NotificationResponse, 0, len(page.Results))
	for _, item := range page.Results {
		results = append(results, toNotificationOutput(item))
	}
	httpdto.Success(c, dto.NotificationListResponse{Results: results, NextPageToken: page.NextPageToken, TotalSize: page.TotalSize})
}

// Get handles GET /v1/users/current/notifications/:notification.
func (handler *NotificationInboxHandler) Get(c *gin.Context) {
	userID, ok := notificationCurrentUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	notificationID, err := parseNotificationResourceSegment(c.Param("notification"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid notification resource")
		return
	}
	item, err := handler.service.Get(c.Request.Context(), userID, notificationID)
	if err != nil {
		handler.writeInboxError(c, err, "get notification")
		return
	}
	httpdto.Success(c, toNotificationOutput(item))
}

// MarkRead handles POST /v1/users/current/notifications/:notification:markRead.
func (handler *NotificationInboxHandler) MarkRead(c *gin.Context) {
	if !notificationCommandHasEmptyBody(c) {
		httpdto.BadRequest(c, "markRead does not accept request fields")
		return
	}
	userID, ok := notificationCurrentUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	notificationID, err := parseNotificationMarkReadSegment(c.Param("notification"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid notification markRead resource")
		return
	}
	item, err := handler.service.MarkRead(c.Request.Context(), userID, notificationID)
	if err != nil {
		handler.writeInboxError(c, err, "mark notification read")
		return
	}
	httpdto.Success(c, toNotificationOutput(item))
}

// MarkAllRead handles POST /v1/users/current/notifications:markAllRead.
func (handler *NotificationInboxHandler) MarkAllRead(c *gin.Context) {
	if !notificationCommandHasEmptyBody(c) {
		httpdto.BadRequest(c, "markAllRead does not accept request fields")
		return
	}
	userID, ok := notificationCurrentUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	if err := handler.service.MarkAllRead(c.Request.Context(), userID); err != nil {
		handler.writeInboxError(c, err, "mark notifications read")
		return
	}
	httpdto.NoContent(c)
}

// UnreadCount handles GET /v1/users/current/notifications:unreadCount.
func (handler *NotificationInboxHandler) UnreadCount(c *gin.Context) {
	userID, ok := notificationCurrentUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	count, err := handler.service.UnreadCount(c.Request.Context(), userID)
	if err != nil {
		handler.writeInboxError(c, err, "get unread notification count")
		return
	}
	httpdto.Success(c, dto.UnreadCountResponse{UnreadCount: count})
}

// GetLocale handles GET /v1/users/current/notificationLocale.
func (handler *NotificationInboxHandler) GetLocale(c *gin.Context) {
	userID, ok := notificationCurrentUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	locale, err := handler.service.Locale(c.Request.Context(), userID)
	if err != nil {
		handler.writeInboxError(c, err, "get notification locale")
		return
	}
	httpdto.Success(c, dto.LocaleResponse{Locale: string(locale)})
}

// UpdateLocale handles PATCH /v1/users/current/notificationLocale.
func (handler *NotificationInboxHandler) UpdateLocale(c *gin.Context) {
	userID, ok := notificationCurrentUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	var request dto.LocaleUpdateRequest
	if !httpdto.BindJSON(c, &request) {
		return
	}
	if err := handler.service.UpdateLocale(c.Request.Context(), userID, domain.Locale(strings.TrimSpace(request.Locale))); err != nil {
		if errors.Is(err, notificationapp.ErrNotificationPermissionDenied) {
			httpdto.Forbidden(c, "Permission denied")
			return
		}
		httpdto.BadRequest(c, "Unsupported notification locale")
		return
	}
	httpdto.Success(c, dto.LocaleResponse{Locale: strings.TrimSpace(request.Locale)})
}

func (handler *NotificationInboxHandler) writeInboxError(c *gin.Context, err error, operation string) {
	if errors.Is(err, notificationapp.ErrInvalidInboxPageToken) || errors.Is(err, notificationapp.ErrInvalidInboxPageSize) {
		httpdto.BadRequest(c, err.Error())
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		httpdto.NotFound(c, "Notification not found")
		return
	}
	httpdto.InternalError(c, "Failed to "+operation)
}

func notificationCurrentUserID(c *gin.Context) (int, bool) {
	return middleware.GetUserID(c)
}

func hasUnsupportedNotificationListQuery(c *gin.Context) bool {
	for key := range c.Request.URL.Query() {
		if key != "pageSize" && key != "pageToken" {
			return true
		}
	}
	return false
}

func notificationCommandHasEmptyBody(c *gin.Context) bool {
	if c.Request.ContentLength > 0 {
		return false
	}
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		return true
	}
	data, err := io.ReadAll(c.Request.Body)
	return err == nil && len(strings.TrimSpace(string(data))) == 0
}

func parseNotificationResourceSegment(raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid notification resource")
	}
	return value, nil
}

func parseNotificationMarkReadSegment(raw string) (int64, error) {
	idSegment, verb, ok := strings.Cut(strings.TrimSpace(raw), ":")
	if !ok || verb != "markRead" {
		return 0, fmt.Errorf("invalid notification markRead resource")
	}
	return parseNotificationResourceSegment(idSegment)
}

func toNotificationOutput(item domain.InboxItem) dto.NotificationResponse {
	return dto.NotificationResponse{
		Name:       item.Name,
		Kind:       string(item.Kind),
		Category:   string(item.Category),
		Priority:   string(item.Priority),
		Subject:    item.Subject,
		Title:      item.Title,
		Message:    item.Message,
		OccurredAt: item.OccurredAt,
		CreatedAt:  item.CreatedAt,
		ReadAt:     item.ReadAt,
	}
}
