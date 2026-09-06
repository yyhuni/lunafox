package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	notificationapp "github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/notification/repository/persistence"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DeliveryRepository persists provider-independent delivery state and safe
// attempt metadata. It does not log or persist credentials in attempt rows.
type DeliveryRepository struct {
	db *gorm.DB
}

// NewDeliveryRepository creates the durable delivery work store.
func NewDeliveryRepository(db *gorm.DB) *DeliveryRepository {
	if db == nil {
		panic("notification delivery database is required")
	}
	return &DeliveryRepository{db: db}
}

// Claim leases due delivery work and recovers only expired in-progress leases.
func (repository *DeliveryRepository) Claim(ctx context.Context, owner string, now time.Time, lease time.Duration, limit int) ([]domain.Delivery, error) {
	if strings.TrimSpace(owner) == "" || now.IsZero() || lease <= 0 || limit <= 0 {
		return nil, fmt.Errorf("notification delivery claim requires owner, time, positive lease, and limit")
	}
	now = now.UTC()
	leaseExpiresAt := now.Add(lease)
	claimed := make([]domain.Delivery, 0, limit)
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.WithContext(ctx).
			Where("(status IN (?, ?) AND next_attempt_at <= ?) OR (status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?)",
				domain.DeliveryStatusPending, domain.DeliveryStatusRetrying, now, domain.DeliveryStatusProcessing, now).
			Order("next_attempt_at ASC").
			Order("id ASC").
			Limit(limit)
		if tx.Dialector.Name() == "postgres" {
			query = query.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
		}
		var records []model.Delivery
		if err := query.Find(&records).Error; err != nil {
			return err
		}
		for index := range records {
			record := &records[index]
			result := tx.WithContext(ctx).Model(&model.Delivery{}).
				Where("id = ? AND ((status IN (?, ?) AND next_attempt_at <= ?) OR (status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?))",
					record.ID, domain.DeliveryStatusPending, domain.DeliveryStatusRetrying, now, domain.DeliveryStatusProcessing, now).
				Updates(map[string]any{
					"status":           domain.DeliveryStatusProcessing,
					"lease_owner":      owner,
					"lease_expires_at": leaseExpiresAt,
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				continue
			}
			record.Status = string(domain.DeliveryStatusProcessing)
			record.LeaseOwner = &owner
			record.LeaseExpiresAt = &leaseExpiresAt
			claimed = append(claimed, deliveryFromRecord(*record))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

// RecordAttempt increments the durable budget only while the worker still
// owns the lease, so a recovered delivery cannot receive stale attempt rows.
func (repository *DeliveryRepository) RecordAttempt(ctx context.Context, attempt domain.DeliveryAttempt, owner string) error {
	if attempt.DeliveryID <= 0 || attempt.AttemptNumber <= 0 || attempt.AttemptedAt.IsZero() || attempt.CompletedAt.IsZero() || strings.TrimSpace(owner) == "" {
		return fmt.Errorf("notification delivery attempt requires delivery, number, times, and owner")
	}
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var delivery model.Delivery
		query := tx.Where("id = ? AND status = ? AND lease_owner = ?", attempt.DeliveryID, domain.DeliveryStatusProcessing, owner)
		if tx.Dialector.Name() == "postgres" {
			query = query.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if err := query.First(&delivery).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("notification delivery %d attempt lost lease", attempt.DeliveryID)
			}
			return err
		}
		if attempt.AttemptNumber > delivery.AttemptCount+1 {
			return fmt.Errorf("notification delivery %d attempt number %d skips persisted count %d", attempt.DeliveryID, attempt.AttemptNumber, delivery.AttemptCount)
		}
		record := model.DeliveryAttempt{
			DeliveryID:        attempt.DeliveryID,
			AttemptNumber:     attempt.AttemptNumber,
			AttemptedAt:       attempt.AttemptedAt.UTC(),
			CompletedAt:       attempt.CompletedAt.UTC(),
			HTTPStatus:        attempt.HTTPStatus,
			ErrorClass:        strings.TrimSpace(attempt.ErrorClass),
			ProviderRequestID: strings.TrimSpace(attempt.ProviderRequestID),
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "delivery_id"}, {Name: "attempt_number"}},
			DoNothing: true,
		}).Create(&record).Error; err != nil {
			return err
		}
		if attempt.AttemptNumber <= delivery.AttemptCount {
			return nil
		}
		updates := map[string]any{"attempt_count": attempt.AttemptNumber}
		if delivery.FirstAttemptAt == nil {
			updates["first_attempt_at"] = attempt.AttemptedAt.UTC()
		}
		return tx.Model(&model.Delivery{}).Where("id = ? AND status = ? AND lease_owner = ?", attempt.DeliveryID, domain.DeliveryStatusProcessing, owner).Updates(updates).Error
	})
}

// MarkDelivered is a terminal owner-fenced transition.
func (repository *DeliveryRepository) MarkDelivered(ctx context.Context, id int64, owner string, terminalAt time.Time) error {
	return repository.markTerminal(ctx, id, owner, domain.DeliveryStatusDelivered, terminalAt)
}

// MarkFailedTerminal is a terminal owner-fenced transition.
func (repository *DeliveryRepository) MarkFailedTerminal(ctx context.Context, id int64, owner string, terminalAt time.Time) error {
	return repository.markTerminal(ctx, id, owner, domain.DeliveryStatusFailedTerminal, terminalAt)
}

func (repository *DeliveryRepository) markTerminal(ctx context.Context, id int64, owner string, status domain.DeliveryStatus, terminalAt time.Time) error {
	if id <= 0 || strings.TrimSpace(owner) == "" || terminalAt.IsZero() {
		return fmt.Errorf("notification delivery terminal transition requires id, owner, and time")
	}
	result := repository.db.WithContext(ctx).Model(&model.Delivery{}).
		Where("id = ? AND status = ? AND lease_owner = ?", id, domain.DeliveryStatusProcessing, owner).
		Updates(map[string]any{
			"status":           status,
			"terminal_at":      terminalAt.UTC(),
			"lease_owner":      nil,
			"lease_expires_at": nil,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("notification delivery %d terminal transition lost lease", id)
	}
	return nil
}

// MarkRetry clears the lease and persists the next eligible attempt time.
func (repository *DeliveryRepository) MarkRetry(ctx context.Context, id int64, owner string, nextAttemptAt, updatedAt time.Time) error {
	if id <= 0 || strings.TrimSpace(owner) == "" || nextAttemptAt.IsZero() || updatedAt.IsZero() {
		return fmt.Errorf("notification delivery retry transition requires id, owner, and times")
	}
	result := repository.db.WithContext(ctx).Model(&model.Delivery{}).
		Where("id = ? AND status = ? AND lease_owner = ?", id, domain.DeliveryStatusProcessing, owner).
		Updates(map[string]any{
			"status":           domain.DeliveryStatusRetrying,
			"next_attempt_at":  nextAttemptAt.UTC(),
			"lease_owner":      nil,
			"lease_expires_at": nil,
			"updated_at":       updatedAt.UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("notification delivery %d retry transition lost lease", id)
	}
	return nil
}

// DeleteTerminalBefore retains in-flight state and removes only completed
// delivery rows; attempts cascade from those terminal rows only.
func (repository *DeliveryRepository) DeleteTerminalBefore(ctx context.Context, batch domain.RetentionBatch) (int64, error) {
	if batch.Limit <= 0 || batch.Before.IsZero() {
		return 0, fmt.Errorf("notification delivery retention requires positive limit and cutoff")
	}
	var deleted int64
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var records []model.Delivery
		if err := tx.Where("status IN (?, ?) AND terminal_at IS NOT NULL AND terminal_at < ?", domain.DeliveryStatusDelivered, domain.DeliveryStatusFailedTerminal, batch.Before.UTC()).
			Order("terminal_at ASC").Order("id ASC").Limit(batch.Limit).Find(&records).Error; err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}
		ids := make([]int64, 0, len(records))
		for _, record := range records {
			ids = append(ids, record.ID)
		}
		result := tx.Where("id IN ?", ids).Delete(&model.Delivery{})
		if result.Error != nil {
			return result.Error
		}
		deleted = result.RowsAffected
		return nil
	})
	return deleted, err
}

func deliveryFromRecord(record model.Delivery) domain.Delivery {
	leaseOwner := ""
	if record.LeaseOwner != nil {
		leaseOwner = *record.LeaseOwner
	}
	return domain.Delivery{
		ID:             record.ID,
		EventID:        record.EventID,
		FactID:         record.FactID,
		DestinationID:  record.DestinationID,
		Provider:       domain.Provider(record.Provider),
		Status:         domain.DeliveryStatus(record.Status),
		AttemptCount:   record.AttemptCount,
		FirstAttemptAt: copyNotificationTime(record.FirstAttemptAt),
		NextAttemptAt:  record.NextAttemptAt.UTC(),
		LeaseOwner:     leaseOwner,
		LeaseExpiresAt: copyNotificationTime(record.LeaseExpiresAt),
		RenderSnapshot: domain.RenderSnapshot{
			Locale:          domain.Locale(record.Locale),
			TemplateVersion: record.TemplateVersion,
			Title:           record.Title,
			Message:         record.Message,
			ProviderPayload: append([]byte(nil), datatypes.JSON(record.ProviderSnapshot)...),
		},
		TerminalAt: copyNotificationTime(record.TerminalAt),
		CreatedAt:  record.CreatedAt.UTC(),
		UpdatedAt:  record.UpdatedAt.UTC(),
	}
}

var _ notificationapp.DeliveryStore = (*DeliveryRepository)(nil)
