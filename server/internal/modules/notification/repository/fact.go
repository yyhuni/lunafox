package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/notification/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FactRepository persists canonical notification facts independently of their
// per-user and per-destination projections.
type FactRepository struct {
	db *gorm.DB
}

// NewFactRepository creates a fact store rooted at the application database.
func NewFactRepository(db *gorm.DB) *FactRepository {
	if db == nil {
		panic("notification fact database is required")
	}
	return &FactRepository{db: db}
}

// CreateOrGetFact converges event replays through the stable producer event ID.
func (repository *FactRepository) CreateOrGetFact(ctx context.Context, event domain.ValidatedEvent) (domain.Fact, error) {
	if err := domain.ValidateOccurrence(event.Occurrence); err != nil {
		return domain.Fact{}, domain.NewDeterministicError("invalid_occurrence", err)
	}
	db := dbtx.Resolve(ctx, repository.db).WithContext(ctx)
	record := model.Fact{
		EventID:        event.EventID,
		Kind:           string(event.Kind),
		PayloadVersion: event.PayloadVersion,
		SubjectName:    event.Subject,
		Category:       string(event.Category),
		Priority:       string(event.Priority),
		OccurredAt:     event.OccurredAt.UTC(),
		Payload:        datatypes.JSON(append([]byte(nil), event.Occurrence.Payload...)),
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "event_id"}},
		DoNothing: true,
	}).Create(&record).Error; err != nil {
		return domain.Fact{}, err
	}
	if err := db.Where("event_id = ?", event.EventID).First(&record).Error; err != nil {
		return domain.Fact{}, fmt.Errorf("load notification fact %q: %w", event.EventID, err)
	}
	return factFromRecord(record), nil
}

func factFromRecord(record model.Fact) domain.Fact {
	return domain.Fact{
		ID:                   record.ID,
		EventID:              record.EventID,
		Kind:                 domain.Kind(record.Kind),
		PayloadVersion:       record.PayloadVersion,
		Subject:              record.SubjectName,
		Category:             domain.Category(record.Category),
		Priority:             domain.Priority(record.Priority),
		OccurredAt:           record.OccurredAt.UTC(),
		Payload:              append([]byte(nil), record.Payload...),
		AudienceFrozenAt:     copyNotificationTime(record.AudienceFrozenAt),
		DestinationsFrozenAt: copyNotificationTime(record.DestinationsFrozenAt),
		CreatedAt:            record.CreatedAt.UTC(),
	}
}

func copyNotificationTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}
