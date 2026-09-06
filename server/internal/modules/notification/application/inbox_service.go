package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

const (
	defaultInboxPageSize = 50
	maxInboxPageSize     = 100
)

// InboxListInput controls the current user's durable inbox resource list.
type InboxListInput struct {
	PageSize  int
	PageToken string
}

// InboxService owns current-user read commands and locale updates.
type InboxService struct {
	inbox   InboxStore
	locales LocaleStore
	now     func() time.Time
}

// NewInboxService creates the notification inbox application service.
func NewInboxService(inbox InboxStore, locales LocaleStore) *InboxService {
	if inbox == nil || locales == nil {
		panic("notification inbox service dependencies are required")
	}
	return &InboxService{inbox: inbox, locales: locales, now: func() time.Time { return time.Now().UTC() }}
}

// List returns current-user items with bounded token pagination.
func (service *InboxService) List(ctx context.Context, userID int, input InboxListInput) (InboxPage, error) {
	if userID <= 0 {
		return InboxPage{}, fmt.Errorf("notification inbox user id is required")
	}
	pageSize := input.PageSize
	if pageSize == 0 {
		pageSize = defaultInboxPageSize
	}
	if pageSize < 0 || pageSize > maxInboxPageSize {
		return InboxPage{}, fmt.Errorf("%w: must be between 1 and %d", ErrInvalidInboxPageSize, maxInboxPageSize)
	}
	return service.inbox.List(ctx, userID, pageSize, strings.TrimSpace(input.PageToken), service.now())
}

// Get returns a user-owned notification fact projection.
func (service *InboxService) Get(ctx context.Context, userID int, notificationFactID int64) (domain.InboxItem, error) {
	if userID <= 0 || notificationFactID <= 0 {
		return domain.InboxItem{}, fmt.Errorf("notification inbox resource name is required")
	}
	return service.inbox.Get(ctx, userID, notificationFactID, service.now())
}

// UnreadCount reads the authoritative persisted unread state.
func (service *InboxService) UnreadCount(ctx context.Context, userID int) (int64, error) {
	if userID <= 0 {
		return 0, fmt.Errorf("notification inbox user id is required")
	}
	return service.inbox.UnreadCount(ctx, userID, service.now())
}

// MarkRead repeats safely and retains the first read time.
func (service *InboxService) MarkRead(ctx context.Context, userID int, notificationFactID int64) (domain.InboxItem, error) {
	if userID <= 0 || notificationFactID <= 0 {
		return domain.InboxItem{}, fmt.Errorf("notification inbox resource name is required")
	}
	return service.inbox.MarkRead(ctx, userID, notificationFactID, service.now())
}

// MarkAllRead captures its own high-water mark in the store transaction.
func (service *InboxService) MarkAllRead(ctx context.Context, userID int) error {
	if userID <= 0 {
		return fmt.Errorf("notification inbox user id is required")
	}
	return service.inbox.MarkAllRead(ctx, userID, service.now())
}

// Locale returns the persisted asynchronous rendering authority.
func (service *InboxService) Locale(ctx context.Context, userID int) (domain.Locale, error) {
	if userID <= 0 {
		return "", fmt.Errorf("notification locale user id is required")
	}
	return service.locales.GetLocale(ctx, userID)
}

// UpdateLocale rejects unsupported values before changing the worker source.
func (service *InboxService) UpdateLocale(ctx context.Context, userID int, locale domain.Locale) error {
	if userID <= 0 {
		return fmt.Errorf("notification locale user id is required")
	}
	if err := domain.ValidateLocale(locale); err != nil {
		return err
	}
	return service.locales.UpdateLocale(ctx, userID, locale)
}
