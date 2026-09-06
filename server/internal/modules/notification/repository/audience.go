package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	notificationapp "github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/notification/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AudienceRepository freezes active-user recipients and their fixed inbox
// projections at the same transaction point as canonical fact materialization.
type AudienceRepository struct {
	db *gorm.DB
}

// NewAudienceRepository creates an audience projection store.
func NewAudienceRepository(db *gorm.DB) *AudienceRepository {
	if db == nil {
		panic("notification audience database is required")
	}
	return &AudienceRepository{db: db}
}

type activeNotificationUser struct {
	ID     int    `gorm:"column:id"`
	Locale string `gorm:"column:locale"`
}

// CreateRecipientsAndInbox performs the one-time audience snapshot. Once the
// fact is frozen, a replay loads the original recipients instead of adding
// accounts activated after the event was materialized.
func (repository *AudienceRepository) CreateRecipientsAndInbox(ctx context.Context, fact domain.Fact, render func(domain.Locale) (domain.RenderSnapshot, error)) (recipientsAndInbox notificationapp.AudienceProjection, err error) {
	if fact.ID <= 0 {
		return recipientsAndInbox, fmt.Errorf("notification audience requires a persisted fact")
	}
	if render == nil {
		return recipientsAndInbox, fmt.Errorf("notification audience renderer is required")
	}

	db := dbtx.Resolve(ctx, repository.db).WithContext(ctx)
	var factRecord model.Fact
	query := db.Where("id = ?", fact.ID)
	if db.Dialector.Name() == "postgres" {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := query.First(&factRecord).Error; err != nil {
		return recipientsAndInbox, fmt.Errorf("load notification fact %d for audience: %w", fact.ID, err)
	}
	if factRecord.AudienceFrozenAt != nil {
		return repository.loadFrozenAudience(db, fact.ID)
	}

	var users []activeNotificationUser
	if err := db.Table("auth_user").
		Select("id, locale").
		Where("is_active = ?", true).
		Order("id ASC").
		Find(&users).Error; err != nil {
		return recipientsAndInbox, fmt.Errorf("list active notification users: %w", err)
	}

	for _, user := range users {
		locale := domain.Locale(strings.TrimSpace(user.Locale))
		if err := domain.ValidateLocale(locale); err != nil {
			return recipientsAndInbox, domain.NewDeterministicError("invalid_recipient_locale", err)
		}
		recipientRecord := model.Recipient{
			FactID:   fact.ID,
			UserID:   user.ID,
			Locale:   string(locale),
			Category: string(fact.Category),
		}
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "fact_id"}, {Name: "user_id"}},
			DoNothing: true,
		}).Create(&recipientRecord).Error; err != nil {
			return recipientsAndInbox, err
		}
		if err := db.Where("fact_id = ? AND user_id = ?", fact.ID, user.ID).First(&recipientRecord).Error; err != nil {
			return recipientsAndInbox, fmt.Errorf("load notification recipient: %w", err)
		}
		recipient := recipientFromRecord(recipientRecord)
		recipientsAndInbox.Recipients = append(recipientsAndInbox.Recipients, recipient)
		snapshot, renderErr := render(locale)
		if renderErr != nil {
			return recipientsAndInbox, renderErr
		}
		if err := validateRenderSnapshot(snapshot); err != nil {
			return recipientsAndInbox, domain.NewDeterministicError("invalid_render_snapshot", err)
		}
		inboxRecord := model.Inbox{
			FactID:      fact.ID,
			RecipientID: recipient.ID,
			UserID:      user.ID,
			Name:        fmt.Sprintf("users/%d/notifications/%d", user.ID, fact.ID),
			Kind:        string(fact.Kind),
			Category:    string(fact.Category),
			Priority:    string(fact.Priority),
			SubjectName: fact.Subject,
			Locale:      string(snapshot.Locale),
			Title:       snapshot.Title,
			Message:     snapshot.Message,
			OccurredAt:  fact.OccurredAt.UTC(),
		}
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "fact_id"}, {Name: "user_id"}},
			DoNothing: true,
		}).Create(&inboxRecord).Error; err != nil {
			return recipientsAndInbox, err
		}
		if err := db.Where("fact_id = ? AND user_id = ?", fact.ID, user.ID).First(&inboxRecord).Error; err != nil {
			return recipientsAndInbox, fmt.Errorf("load notification inbox projection: %w", err)
		}
		recipientsAndInbox.InboxItems = append(recipientsAndInbox.InboxItems, inboxFromRecord(inboxRecord))
	}

	frozenAt := time.Now().UTC()
	if err := db.Model(&model.Fact{}).
		Where("id = ? AND audience_frozen_at IS NULL", fact.ID).
		Update("audience_frozen_at", frozenAt).Error; err != nil {
		return recipientsAndInbox, err
	}
	return recipientsAndInbox, nil
}

func (repository *AudienceRepository) loadFrozenAudience(db *gorm.DB, factID int64) (notificationapp.AudienceProjection, error) {
	projection := notificationapp.AudienceProjection{}
	var recipientRecords []model.Recipient
	if err := db.Where("fact_id = ?", factID).Order("id ASC").Find(&recipientRecords).Error; err != nil {
		return projection, err
	}
	for _, record := range recipientRecords {
		projection.Recipients = append(projection.Recipients, recipientFromRecord(record))
	}
	var inboxRecords []model.Inbox
	if err := db.Where("fact_id = ?", factID).Order("id ASC").Find(&inboxRecords).Error; err != nil {
		return projection, err
	}
	for _, record := range inboxRecords {
		projection.InboxItems = append(projection.InboxItems, inboxFromRecord(record))
	}
	return projection, nil
}

func recipientFromRecord(record model.Recipient) domain.Recipient {
	return domain.Recipient{
		ID:        record.ID,
		FactID:    record.FactID,
		UserID:    record.UserID,
		Locale:    domain.Locale(record.Locale),
		Category:  domain.Category(record.Category),
		CreatedAt: record.CreatedAt.UTC(),
	}
}

func inboxFromRecord(record model.Inbox) domain.InboxItem {
	return domain.InboxItem{
		ID:          record.ID,
		FactID:      record.FactID,
		RecipientID: record.RecipientID,
		UserID:      record.UserID,
		Name:        record.Name,
		Kind:        domain.Kind(record.Kind),
		Category:    domain.Category(record.Category),
		Priority:    domain.Priority(record.Priority),
		Subject:     record.SubjectName,
		Locale:      domain.Locale(record.Locale),
		Title:       record.Title,
		Message:     record.Message,
		OccurredAt:  record.OccurredAt.UTC(),
		CreatedAt:   record.CreatedAt.UTC(),
		ReadAt:      copyNotificationTime(record.ReadAt),
	}
}

func validateRenderSnapshot(snapshot domain.RenderSnapshot) error {
	if err := domain.ValidateLocale(snapshot.Locale); err != nil {
		return err
	}
	if snapshot.TemplateVersion <= 0 {
		return fmt.Errorf("notification render snapshot template version is required")
	}
	if strings.TrimSpace(snapshot.Title) == "" || strings.TrimSpace(snapshot.Message) == "" {
		return fmt.Errorf("notification render snapshot title and message are required")
	}
	var providerPayload map[string]json.RawMessage
	if len(snapshot.ProviderPayload) == 0 || json.Unmarshal(snapshot.ProviderPayload, &providerPayload) != nil || providerPayload == nil {
		return fmt.Errorf("notification render snapshot provider payload must be a JSON object")
	}
	return nil
}
