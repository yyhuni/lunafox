package application

import (
	"context"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

type InboxPage struct {
	Results       []domain.InboxItem
	NextPageToken string
	TotalSize     int64
}

// InboxStore owns current-user resource reads and transactional read commands.
type InboxStore interface {
	List(context.Context, int, int, string, time.Time) (InboxPage, error)
	Get(context.Context, int, int64, time.Time) (domain.InboxItem, error)
	UnreadCount(context.Context, int, time.Time) (int64, error)
	MarkRead(context.Context, int, int64, time.Time) (domain.InboxItem, error)
	MarkAllRead(context.Context, int, time.Time) error
	DeleteExpiredInbox(context.Context, domain.RetentionBatch) (int64, error)
}

type LocaleStore interface {
	GetLocale(context.Context, int) (domain.Locale, error)
	UpdateLocale(context.Context, int, domain.Locale) error
}
