// Package notificationwiring assembles the notification module at the
// composition root without leaking its persistence or provider details.
package notificationwiring

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/handler"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/provider"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/repository"
	"gorm.io/gorm"
)

// Module contains the notification boundaries and managed workers that must
// share one process-local realtime broker and one database composition root.
type Module struct {
	InboxHandler       *handler.NotificationInboxHandler
	DestinationHandler *handler.NotificationDestinationHandler
	SSEHandler         *handler.NotificationSSEHandler
	OutboxWorker       *application.OutboxWorker
	DeliveryWorker     *application.DeliveryWorker
	RetentionJob       *application.RetentionJob
}

// NewNotificationWorkerOwner creates a process-unique lease owner. The owner
// is persisted only while a worker holds a lease and is never user input.
func NewNotificationWorkerOwner() (string, error) {
	entropy := make([]byte, 16)
	if _, err := rand.Read(entropy); err != nil {
		return "", fmt.Errorf("generate notification worker owner: %w", err)
	}
	return "notification-" + hex.EncodeToString(entropy), nil
}

// NewNotificationModule wires handlers, workers, repositories, templates,
// and provider adapters around one notification database and lease owner.
func NewNotificationModule(db *gorm.DB, workerOwner string) (*Module, error) {
	if db == nil {
		panic("notification wiring database is required")
	}
	workerOwner = strings.TrimSpace(workerOwner)
	if workerOwner == "" {
		panic("notification wiring worker owner is required")
	}

	outboxStore := repository.NewOutboxRepository(db)
	inboxStore := repository.NewInboxRepository(db)
	destinationStore := repository.NewDestinationRepository(db)
	deliveryStore := repository.NewDeliveryRepository(db)
	authorizer := repository.NewActiveSuperuserRepository(db)
	adapters := []application.ProviderAdapter{
		provider.NewDiscordAdapter(nil),
		provider.NewWeComAdapter(nil),
		provider.NewFeishuAdapter(nil),
	}
	if err := application.ValidateFixedProviderAdapters(adapters); err != nil {
		return nil, fmt.Errorf("validate notification provider adapters: %w", err)
	}
	if err := application.ReconcileDestinations(context.Background(), destinationStore); err != nil {
		return nil, fmt.Errorf("reconcile notification destinations: %w", err)
	}
	realtimeHub := application.NewRealtimeHub()
	materializer := application.NewMaterializer(
		repository.NewTransactionCoordinator(db),
		repository.NewFactRepository(db),
		repository.NewAudienceRepository(db),
		destinationStore,
		application.NewTemplates(),
		realtimeHub,
	)
	inboxService := application.NewInboxService(inboxStore, inboxStore)
	destinationSettings := application.NewDestinationSettingsService(destinationStore, authorizer)
	destinationHandler := handler.NewNotificationDestinationHandler(destinationSettings)
	destinationHandler.SetTestDeliveryService(application.NewTestDeliveryService(authorizer, adapters))

	return &Module{
		InboxHandler:       handler.NewNotificationInboxHandler(inboxService),
		DestinationHandler: destinationHandler,
		SSEHandler:         handler.NewNotificationSSEHandler(realtimeHub),
		OutboxWorker: application.NewOutboxWorker(outboxStore, materializer, application.OutboxWorkerOptions{
			Owner: workerOwner + "-outbox",
		}),
		DeliveryWorker: application.NewDeliveryWorker(
			deliveryStore,
			destinationStore,
			adapters,
			application.DeliveryWorkerOptions{Owner: workerOwner + "-delivery"},
		),
		RetentionJob: application.NewRetentionJob(outboxStore, inboxStore, deliveryStore, application.RetentionJobOptions{}),
	}, nil
}
