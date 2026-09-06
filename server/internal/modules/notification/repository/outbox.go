package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/notification/repository/persistence"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OutboxRepository owns durable notification occurrence persistence.
type OutboxRepository struct {
	db *gorm.DB
}

// NewOutboxRepository creates a repository rooted at the application database.
func NewOutboxRepository(db *gorm.DB) *OutboxRepository {
	if db == nil {
		panic("notification outbox database is required")
	}
	return &OutboxRepository{db: db}
}

// CreateOccurrence records one immutable envelope in the caller-owned business
// transaction. The unique event ID makes an identical replay a no-op.
func (repository *OutboxRepository) CreateOccurrence(tx *gorm.DB, occurrence domain.Occurrence) error {
	if tx == nil {
		return fmt.Errorf("notification outbox transaction is required")
	}
	if err := domain.ValidateOccurrence(occurrence); err != nil {
		return err
	}
	now := time.Now().UTC()
	record := model.Outbox{
		EventID:        occurrence.EventID,
		Kind:           string(occurrence.Kind),
		PayloadVersion: occurrence.PayloadVersion,
		SubjectName:    occurrence.Subject,
		OccurredAt:     occurrence.OccurredAt.UTC(),
		Priority:       string(occurrence.Priority),
		Payload:        datatypes.JSON(append([]byte(nil), occurrence.Payload...)),
		Status:         string(domain.OutboxStatusPending),
		AvailableAt:    now,
		FailureCode:    "",
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "event_id"}},
		DoNothing: true,
	}).Create(&record).Error
}

// Claim atomically leases due pending work or expired in-progress work. The
// PostgreSQL row lock prevents two workers from observing the same candidate;
// the conditional update remains a second fence for non-PostgreSQL tests.
func (repository *OutboxRepository) Claim(ctx context.Context, owner string, now time.Time, lease time.Duration, limit int) ([]domain.OutboxEvent, error) {
	if strings.TrimSpace(owner) == "" {
		return nil, fmt.Errorf("notification outbox lease owner is required")
	}
	if now.IsZero() {
		return nil, fmt.Errorf("notification outbox claim time is required")
	}
	if lease <= 0 {
		return nil, fmt.Errorf("notification outbox lease duration must be positive")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("notification outbox claim limit must be positive")
	}

	now = now.UTC()
	leaseExpiresAt := now.Add(lease)
	claimed := make([]domain.OutboxEvent, 0, limit)
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.WithContext(ctx).
			Where("(status = ? AND available_at <= ?) OR (status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?)",
				domain.OutboxStatusPending, now, domain.OutboxStatusProcessing, now).
			Order("available_at ASC").
			Order("id ASC").
			Limit(limit)
		if tx.Dialector.Name() == "postgres" {
			query = query.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
		}

		var records []model.Outbox
		if err := query.Find(&records).Error; err != nil {
			return err
		}
		for index := range records {
			record := &records[index]
			result := tx.WithContext(ctx).
				Model(&model.Outbox{}).
				Where("id = ? AND ((status = ? AND available_at <= ?) OR (status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?))",
					record.ID, domain.OutboxStatusPending, now, domain.OutboxStatusProcessing, now).
				Updates(map[string]any{
					"status":           domain.OutboxStatusProcessing,
					"lease_owner":      owner,
					"lease_expires_at": leaseExpiresAt,
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				continue
			}
			record.Status = string(domain.OutboxStatusProcessing)
			record.LeaseOwner = &owner
			record.LeaseExpiresAt = &leaseExpiresAt
			claimed = append(claimed, outboxFromRecord(*record))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

// MarkPublished completes a claim only when the current worker still owns it.
func (repository *OutboxRepository) MarkPublished(ctx context.Context, id int64, owner string, publishedAt time.Time) error {
	if id <= 0 || strings.TrimSpace(owner) == "" || publishedAt.IsZero() {
		return fmt.Errorf("notification outbox published transition requires id, owner, and time")
	}
	result := repository.db.WithContext(ctx).
		Model(&model.Outbox{}).
		Where("id = ? AND status = ? AND lease_owner = ?", id, domain.OutboxStatusProcessing, owner).
		Updates(map[string]any{
			"status":           domain.OutboxStatusPublished,
			"published_at":     publishedAt.UTC(),
			"lease_owner":      nil,
			"lease_expires_at": nil,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("notification outbox %d publish lost lease", id)
	}
	return nil
}

// MarkUnsupported terminally isolates a deterministic envelope failure while
// retaining its immutable source record for diagnosis.
func (repository *OutboxRepository) MarkUnsupported(ctx context.Context, id int64, owner, failureCode string, terminalAt time.Time) error {
	if id <= 0 || strings.TrimSpace(owner) == "" || strings.TrimSpace(failureCode) == "" || terminalAt.IsZero() {
		return fmt.Errorf("notification outbox unsupported transition requires id, owner, code, and time")
	}
	result := repository.db.WithContext(ctx).
		Model(&model.Outbox{}).
		Where("id = ? AND status = ? AND lease_owner = ?", id, domain.OutboxStatusProcessing, owner).
		Updates(map[string]any{
			"status":           domain.OutboxStatusUnsupportedTerminal,
			"failure_code":     failureCode,
			"terminal_at":      terminalAt.UTC(),
			"lease_owner":      nil,
			"lease_expires_at": nil,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("notification outbox %d unsupported transition lost lease", id)
	}
	return nil
}

// ReleaseLease makes a transiently failed event eligible for a later replay.
func (repository *OutboxRepository) ReleaseLease(ctx context.Context, id int64, owner string, availableAt time.Time) error {
	if id <= 0 || strings.TrimSpace(owner) == "" || availableAt.IsZero() {
		return fmt.Errorf("notification outbox lease release requires id, owner, and time")
	}
	result := repository.db.WithContext(ctx).
		Model(&model.Outbox{}).
		Where("id = ? AND status = ? AND lease_owner = ?", id, domain.OutboxStatusProcessing, owner).
		Updates(map[string]any{
			"status":           domain.OutboxStatusPending,
			"available_at":     availableAt.UTC(),
			"lease_owner":      nil,
			"lease_expires_at": nil,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("notification outbox %d release lost lease", id)
	}
	return nil
}

// DeletePublishedBefore removes only already-published terminal handoff rows;
// unsupported and in-flight envelopes remain available for diagnosis/recovery.
func (repository *OutboxRepository) DeletePublishedBefore(ctx context.Context, batch domain.RetentionBatch) (int64, error) {
	if batch.Limit <= 0 || batch.Before.IsZero() {
		return 0, fmt.Errorf("notification outbox retention requires positive limit and cutoff")
	}
	var deleted int64
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var records []model.Outbox
		if err := tx.WithContext(ctx).
			Where("status = ? AND published_at IS NOT NULL AND published_at < ?", domain.OutboxStatusPublished, batch.Before.UTC()).
			Order("published_at ASC").
			Order("id ASC").
			Limit(batch.Limit).
			Find(&records).Error; err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}
		ids := make([]int64, 0, len(records))
		for _, record := range records {
			ids = append(ids, record.ID)
		}
		result := tx.WithContext(ctx).Where("id IN ?", ids).Delete(&model.Outbox{})
		if result.Error != nil {
			return result.Error
		}
		deleted = result.RowsAffected
		return nil
	})
	return deleted, err
}

func outboxFromRecord(record model.Outbox) domain.OutboxEvent {
	leaseOwner := ""
	if record.LeaseOwner != nil {
		leaseOwner = *record.LeaseOwner
	}
	return domain.OutboxEvent{
		ID: record.ID,
		Occurrence: domain.Occurrence{
			EventID:        record.EventID,
			Kind:           domain.Kind(record.Kind),
			PayloadVersion: record.PayloadVersion,
			Subject:        record.SubjectName,
			OccurredAt:     record.OccurredAt.UTC(),
			Priority:       domain.Priority(record.Priority),
			Payload:        append([]byte(nil), record.Payload...),
		},
		Status:         domain.OutboxStatus(record.Status),
		AvailableAt:    record.AvailableAt.UTC(),
		LeaseOwner:     leaseOwner,
		LeaseExpiresAt: record.LeaseExpiresAt,
		FailureCode:    record.FailureCode,
		TerminalAt:     record.TerminalAt,
		PublishedAt:    record.PublishedAt,
		CreatedAt:      record.CreatedAt.UTC(),
		UpdatedAt:      record.UpdatedAt.UTC(),
	}
}
