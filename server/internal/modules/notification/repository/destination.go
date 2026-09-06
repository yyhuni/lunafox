package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	notificationapp "github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/notification/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DestinationRepository owns the installation-scoped Webhook settings and
// creates durable delivery work without consulting personal inbox preferences.
type DestinationRepository struct {
	db *gorm.DB
}

// NewDestinationRepository creates a destination and fanout store.
func NewDestinationRepository(db *gorm.DB) *DestinationRepository {
	if db == nil {
		panic("notification destination database is required")
	}
	return &DestinationRepository{db: db}
}

// Get returns one provider's complete credential only to the caller that has
// already passed the application/HTTP authorization boundary.
func (repository *DestinationRepository) Get(ctx context.Context, provider domain.Provider) (domain.Destination, error) {
	if err := domain.ValidateProvider(provider); err != nil {
		return domain.Destination{}, err
	}
	db := repository.db.WithContext(ctx)
	record, err := ensureDestination(db, provider)
	if err != nil {
		return domain.Destination{}, err
	}
	return destinationFromRecord(db, record)
}

// Update atomically replaces one provider's explicit exact-kind allowlist.
func (repository *DestinationRepository) Update(ctx context.Context, destination domain.Destination) (domain.Destination, error) {
	if err := domain.ValidateDestination(destination); err != nil {
		return domain.Destination{}, err
	}
	var updated domain.Destination
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record, err := ensureDestination(tx, destination.Provider)
		if err != nil {
			return err
		}
		result := tx.Model(&model.Destination{}).Where("id = ?", record.ID).Updates(map[string]any{
			"credential": destinationCredentialForStorage(destination),
			"enabled":    destination.Enabled,
		})
		if result.Error != nil {
			return result.Error
		}
		if err := tx.Where("destination_id = ?", record.ID).Delete(&model.DestinationSubscription{}).Error; err != nil {
			return err
		}
		for _, kind := range destination.Subscriptions {
			if err := tx.Create(&model.DestinationSubscription{DestinationID: record.ID, Kind: string(kind)}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("id = ?", record.ID).First(&record).Error; err != nil {
			return err
		}
		updated, err = destinationFromRecord(tx, record)
		return err
	})
	return updated, err
}

// List returns every fixed installation destination for startup reconciliation.
func (repository *DestinationRepository) List(ctx context.Context) ([]domain.Destination, error) {
	providers := domain.FixedProviders()
	values := make([]domain.Destination, 0, len(providers))
	for _, provider := range providers {
		destination, err := repository.Get(ctx, provider)
		if err != nil {
			return nil, err
		}
		values = append(values, destination)
	}
	return values, nil
}

func destinationCredentialForStorage(destination domain.Destination) string {
	if destination.Provider == domain.ProviderFeishu {
		return destination.Credential
	}
	return strings.TrimSpace(destination.Credential)
}

// DisableForWebhookRemediation changes only enablement for a legacy credential
// that cannot pass current validation. It intentionally bypasses Update's
// strict parser so reconciliation retains the original credential for repair.
func (repository *DestinationRepository) DisableForWebhookRemediation(ctx context.Context, provider domain.Provider) (domain.Destination, error) {
	if err := domain.ValidateProvider(provider); err != nil {
		return domain.Destination{}, err
	}
	var updated domain.Destination
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record, err := ensureDestination(tx, provider)
		if err != nil {
			return err
		}
		if record.Enabled {
			if err := tx.Model(&model.Destination{}).Where("id = ?", record.ID).Update("enabled", false).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("id = ?", record.ID).First(&record).Error; err != nil {
			return err
		}
		updated, err = destinationFromRecord(tx, record)
		return err
	})
	return updated, err
}

// ListEnabledForKind returns only destinations that independently opted in to
// the exact canonical kind; payload version and inbox switches are irrelevant.
func (repository *DestinationRepository) ListEnabledForKind(ctx context.Context, kind domain.Kind) ([]domain.Destination, error) {
	if !domain.IsExternallyDeliverableKind(kind) {
		return nil, fmt.Errorf("unsupported notification destination kind %q", kind)
	}
	return listEnabledDestinationsForKind(repository.db.WithContext(ctx), kind)
}

// CreateDeliveries freezes destination fanout in the materializer transaction.
// A retry reads the existing frozen set rather than new settings.
func (repository *DestinationRepository) CreateDeliveries(ctx context.Context, fact domain.Fact, snapshot domain.RenderSnapshot) error {
	if fact.ID <= 0 {
		return fmt.Errorf("notification delivery fanout requires a persisted fact")
	}
	if err := validateRenderSnapshot(snapshot); err != nil {
		return domain.NewDeterministicError("invalid_render_snapshot", err)
	}
	db := dbtx.Resolve(ctx, repository.db).WithContext(ctx)
	var factRecord model.Fact
	query := db.Where("id = ?", fact.ID)
	if db.Dialector.Name() == "postgres" {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := query.First(&factRecord).Error; err != nil {
		return fmt.Errorf("load notification fact %d for destination fanout: %w", fact.ID, err)
	}
	if factRecord.DestinationsFrozenAt != nil {
		return nil
	}
	if !domain.IsExternallyDeliverableKind(fact.Kind) {
		// Inbox-only facts deliberately freeze an empty destination set. This
		// check stays at the fanout boundary so stale subscriptions cannot leak
		// a new inbox-only kind to an external provider.
		return db.Model(&model.Fact{}).
			Where("id = ? AND destinations_frozen_at IS NULL", fact.ID).
			Update("destinations_frozen_at", time.Now().UTC()).Error
	}

	destinations, err := listEnabledDestinationsForKind(db, fact.Kind)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, destination := range destinations {
		record := model.Delivery{
			EventID:          fact.EventID,
			FactID:           fact.ID,
			DestinationID:    destination.ID,
			Provider:         string(destination.Provider),
			Status:           string(domain.DeliveryStatusPending),
			AttemptCount:     0,
			NextAttemptAt:    now,
			Locale:           string(snapshot.Locale),
			TemplateVersion:  snapshot.TemplateVersion,
			Title:            snapshot.Title,
			Message:          snapshot.Message,
			ProviderSnapshot: datatypes.JSON(append([]byte(nil), snapshot.ProviderPayload...)),
		}
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "event_id"}, {Name: "destination_id"}},
			DoNothing: true,
		}).Create(&record).Error; err != nil {
			return err
		}
	}
	return db.Model(&model.Fact{}).
		Where("id = ? AND destinations_frozen_at IS NULL", fact.ID).
		Update("destinations_frozen_at", now).Error
}

func ensureDestination(db *gorm.DB, provider domain.Provider) (model.Destination, error) {
	var record model.Destination
	err := db.Where("provider = ?", string(provider)).First(&record).Error
	if err == nil {
		return record, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Destination{}, err
	}
	seed := model.Destination{Provider: string(provider), Credential: "", Enabled: false}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "provider"}},
		DoNothing: true,
	}).Create(&seed).Error; err != nil {
		return model.Destination{}, err
	}
	if err := db.Where("provider = ?", string(provider)).First(&record).Error; err != nil {
		return model.Destination{}, err
	}
	return record, nil
}

func destinationFromRecord(db *gorm.DB, record model.Destination) (domain.Destination, error) {
	var subscriptions []model.DestinationSubscription
	if err := db.Where("destination_id = ?", record.ID).Order("kind ASC").Find(&subscriptions).Error; err != nil {
		return domain.Destination{}, err
	}
	kinds := make([]domain.Kind, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		kind := domain.Kind(subscription.Kind)
		if !domain.IsExternallyDeliverableKind(kind) {
			return domain.Destination{}, domain.NewDeterministicError("invalid_destination_subscription", fmt.Errorf("destination %d has unsupported kind %q", record.ID, subscription.Kind))
		}
		kinds = append(kinds, kind)
	}
	return domain.Destination{
		ID:            record.ID,
		Provider:      domain.Provider(record.Provider),
		Credential:    record.Credential,
		Enabled:       record.Enabled,
		Subscriptions: kinds,
		CreatedAt:     record.CreatedAt.UTC(),
		UpdatedAt:     record.UpdatedAt.UTC(),
	}, nil
}

func listEnabledDestinationsForKind(db *gorm.DB, kind domain.Kind) ([]domain.Destination, error) {
	var records []model.Destination
	if err := db.Table((model.Destination{}).TableName()+" AS destination").
		Select("destination.*").
		Joins("JOIN notification_destination_subscription subscription ON subscription.destination_id = destination.id").
		Where("destination.enabled = ? AND subscription.kind = ?", true, string(kind)).
		Order("destination.id ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	destinations := make([]domain.Destination, 0, len(records))
	for _, record := range records {
		destination, err := destinationFromRecord(db, record)
		if err != nil {
			return nil, err
		}
		destinations = append(destinations, destination)
	}
	sort.Slice(destinations, func(left, right int) bool { return destinations[left].ID < destinations[right].ID })
	return destinations, nil
}

var _ notificationapp.DestinationStore = (*DestinationRepository)(nil)
var _ notificationapp.DestinationFanoutStore = (*DestinationRepository)(nil)
