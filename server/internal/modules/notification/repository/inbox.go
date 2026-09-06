package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	notificationapp "github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/notification/repository/persistence"
	"gorm.io/gorm"
)

const inboxPageTokenVersion = 1

// InboxRepository owns user-scoped durable inbox state and the persisted locale
// used by asynchronous notification workers.
type InboxRepository struct {
	db *gorm.DB
}

// NewInboxRepository creates the current-user notification state store.
func NewInboxRepository(db *gorm.DB) *InboxRepository {
	if db == nil {
		panic("notification inbox database is required")
	}
	return &InboxRepository{db: db}
}

type inboxPageToken struct {
	Version   int    `json:"v"`
	CreatedAt string `json:"createdAt"`
	ID        int64  `json:"id"`
}

// List returns only non-expired projections owned by the current user.
func (repository *InboxRepository) List(ctx context.Context, userID, pageSize int, pageToken string, now time.Time) (notificationapp.InboxPage, error) {
	if userID <= 0 || pageSize <= 0 || now.IsZero() {
		return notificationapp.InboxPage{}, fmt.Errorf("notification inbox list requires user, positive page size, and time")
	}
	cursor, err := decodeInboxPageToken(pageToken)
	if err != nil {
		return notificationapp.InboxPage{}, err
	}
	cutoff := now.UTC().Add(-domain.NotificationRetention)
	db := repository.db.WithContext(ctx)
	base := db.Model(&model.Inbox{}).Where("user_id = ? AND created_at >= ?", userID, cutoff)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return notificationapp.InboxPage{}, err
	}
	query := base.Order("created_at DESC").Order("id DESC").Limit(pageSize + 1)
	if cursor != nil {
		query = query.Where("created_at < ? OR (created_at = ? AND id < ?)", cursor.CreatedAt, cursor.CreatedAt, cursor.ID)
	}
	var records []model.Inbox
	if err := query.Find(&records).Error; err != nil {
		return notificationapp.InboxPage{}, err
	}
	page := notificationapp.InboxPage{TotalSize: total}
	hasNext := len(records) > pageSize
	if hasNext {
		records = records[:pageSize]
	}
	for _, record := range records {
		page.Results = append(page.Results, inboxFromRecord(record))
	}
	if hasNext {
		last := records[len(records)-1]
		page.NextPageToken, err = encodeInboxPageToken(inboxPageToken{Version: inboxPageTokenVersion, CreatedAt: last.CreatedAt.UTC().Format(time.RFC3339Nano), ID: last.ID})
		if err != nil {
			return notificationapp.InboxPage{}, err
		}
	}
	return page, nil
}

// Get returns one owned, non-expired notification by the stable fact resource
// identifier embedded in its canonical users/{user}/notifications/{fact} name.
func (repository *InboxRepository) Get(ctx context.Context, userID int, notificationFactID int64, now time.Time) (domain.InboxItem, error) {
	if userID <= 0 || notificationFactID <= 0 || now.IsZero() {
		return domain.InboxItem{}, fmt.Errorf("notification inbox get requires user, notification, and time")
	}
	var record model.Inbox
	err := repository.db.WithContext(ctx).
		Where("user_id = ? AND fact_id = ? AND created_at >= ?", userID, notificationFactID, now.UTC().Add(-domain.NotificationRetention)).
		First(&record).Error
	if err != nil {
		return domain.InboxItem{}, err
	}
	return inboxFromRecord(record), nil
}

// UnreadCount is authoritative and excludes rows immediately once they age out
// of the 90-day visibility window, even before physical retention runs.
func (repository *InboxRepository) UnreadCount(ctx context.Context, userID int, now time.Time) (int64, error) {
	if userID <= 0 || now.IsZero() {
		return 0, fmt.Errorf("notification unread count requires user and time")
	}
	var count int64
	err := repository.db.WithContext(ctx).Model(&model.Inbox{}).
		Where("user_id = ? AND read_at IS NULL AND created_at >= ?", userID, now.UTC().Add(-domain.NotificationRetention)).
		Count(&count).Error
	return count, err
}

// MarkRead is idempotent and preserves the first read timestamp with COALESCE.
func (repository *InboxRepository) MarkRead(ctx context.Context, userID int, notificationFactID int64, readAt time.Time) (domain.InboxItem, error) {
	if userID <= 0 || notificationFactID <= 0 || readAt.IsZero() {
		return domain.InboxItem{}, fmt.Errorf("notification mark read requires user, notification, and time")
	}
	item, err := repository.Get(ctx, userID, notificationFactID, readAt)
	if err != nil {
		return domain.InboxItem{}, err
	}
	result := repository.db.WithContext(ctx).Model(&model.Inbox{}).
		Where("id = ? AND user_id = ?", item.ID, userID).
		UpdateColumn("read_at", gorm.Expr("COALESCE(read_at, ?)", readAt.UTC()))
	if result.Error != nil {
		return domain.InboxItem{}, result.Error
	}
	return repository.Get(ctx, userID, notificationFactID, readAt)
}

// MarkAllRead captures the visible maximum before modifying rows. It uses a
// stable (created_at, id) high-water tuple so later projections are outside the
// update set and remain unread.
func (repository *InboxRepository) MarkAllRead(ctx context.Context, userID int, readAt time.Time) error {
	if userID <= 0 || readAt.IsZero() {
		return fmt.Errorf("notification mark all read requires user and time")
	}
	cutoff := readAt.UTC().Add(-domain.NotificationRetention)
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var highWater model.Inbox
		err := tx.Where("user_id = ? AND created_at >= ?", userID, cutoff).
			Order("created_at DESC").
			Order("id DESC").
			First(&highWater).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return tx.Model(&model.Inbox{}).
			Where("user_id = ? AND read_at IS NULL AND created_at >= ? AND (created_at < ? OR (created_at = ? AND id <= ?))",
				userID, cutoff, highWater.CreatedAt, highWater.CreatedAt, highWater.ID).
			UpdateColumn("read_at", readAt.UTC()).Error
	})
}

// DeleteExpiredInbox physically removes only expired per-user projections.
func (repository *InboxRepository) DeleteExpiredInbox(ctx context.Context, batch domain.RetentionBatch) (int64, error) {
	if batch.Limit <= 0 || batch.Before.IsZero() {
		return 0, fmt.Errorf("notification inbox retention requires positive limit and cutoff")
	}
	var deleted int64
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var records []model.Inbox
		if err := tx.Where("created_at < ?", batch.Before.UTC()).Order("created_at ASC").Order("id ASC").Limit(batch.Limit).Find(&records).Error; err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}
		ids := make([]int64, 0, len(records))
		for _, record := range records {
			ids = append(ids, record.ID)
		}
		result := tx.Where("id IN ?", ids).Delete(&model.Inbox{})
		if result.Error != nil {
			return result.Error
		}
		deleted = result.RowsAffected
		return nil
	})
	return deleted, err
}

// GetLocale reads only the persisted auth_user value; workers never receive a
// request header or browser fallback as an asynchronous locale authority.
func (repository *InboxRepository) GetLocale(ctx context.Context, userID int) (domain.Locale, error) {
	if userID <= 0 {
		return "", fmt.Errorf("notification locale user id is required")
	}
	var row struct {
		Locale string `gorm:"column:locale"`
	}
	if err := repository.db.WithContext(ctx).Table("auth_user").Select("locale").Where("id = ?", userID).Take(&row).Error; err != nil {
		return "", err
	}
	locale := domain.Locale(strings.TrimSpace(row.Locale))
	if err := domain.ValidateLocale(locale); err != nil {
		return "", domain.NewDeterministicError("invalid_persisted_locale", err)
	}
	return locale, nil
}

// UpdateLocale accepts only the two persisted product locales and is safe to
// repeat when the page shell synchronizes the same locale more than once.
func (repository *InboxRepository) UpdateLocale(ctx context.Context, userID int, locale domain.Locale) error {
	if userID <= 0 {
		return fmt.Errorf("notification locale user id is required")
	}
	if err := domain.ValidateLocale(locale); err != nil {
		return err
	}
	now := time.Now().UTC()
	result := repository.db.WithContext(ctx).Table("auth_user").Where("id = ?", userID).Updates(map[string]any{
		"locale":                string(locale),
		"locale_initialized_at": gorm.Expr("COALESCE(locale_initialized_at, ?)", now),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func encodeInboxPageToken(token inboxPageToken) (string, error) {
	encoded, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeInboxPageToken(raw string) (*struct {
	CreatedAt time.Time
	ID        int64
}, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, notificationapp.ErrInvalidInboxPageToken
	}
	var token inboxPageToken
	if err := json.Unmarshal(decoded, &token); err != nil || token.Version != inboxPageTokenVersion || token.ID <= 0 || strings.TrimSpace(token.CreatedAt) == "" {
		return nil, notificationapp.ErrInvalidInboxPageToken
	}
	createdAt, err := time.Parse(time.RFC3339Nano, token.CreatedAt)
	if err != nil {
		return nil, notificationapp.ErrInvalidInboxPageToken
	}
	return &struct {
		CreatedAt time.Time
		ID        int64
	}{CreatedAt: createdAt.UTC(), ID: token.ID}, nil
}

var _ notificationapp.InboxStore = (*InboxRepository)(nil)
var _ notificationapp.LocaleStore = (*InboxRepository)(nil)
