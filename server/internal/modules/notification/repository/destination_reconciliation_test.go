package repository

import (
	"context"
	"strings"
	"testing"

	notificationapp "github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/notification/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDestinationReconciliationDisablesLegacyCredentialWithoutRewritingSavedState(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Destination{}, &model.DestinationSubscription{}); err != nil {
		t.Fatalf("migrate notification destination tables: %v", err)
	}
	legacy := model.Destination{
		Provider:   string(domain.ProviderDiscord),
		Credential: "https://discord.example.test/api/webhooks/legacy/token",
		Enabled:    true,
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("create legacy destination: %v", err)
	}
	if err := db.Create(&model.DestinationSubscription{DestinationID: legacy.ID, Kind: string(domain.KindScanFailed)}).Error; err != nil {
		t.Fatalf("create legacy subscription: %v", err)
	}

	destinations := NewDestinationRepository(db)
	if err := notificationapp.ReconcileDestinations(context.Background(), destinations); err != nil {
		t.Fatalf("ReconcileDestinations: %v", err)
	}
	updated, err := destinations.Get(context.Background(), domain.ProviderDiscord)
	if err != nil {
		t.Fatalf("get reconciled destination: %v", err)
	}
	if updated.Enabled || updated.Credential != legacy.Credential || len(updated.Subscriptions) != 1 || updated.Subscriptions[0] != domain.KindScanFailed {
		t.Fatalf("reconciled destination = %#v, want only enabled false", updated)
	}

	if err := notificationapp.ReconcileDestinations(context.Background(), destinations); err != nil {
		t.Fatalf("second ReconcileDestinations: %v", err)
	}
	reconciledAgain, err := destinations.Get(context.Background(), domain.ProviderDiscord)
	if err != nil {
		t.Fatalf("get second reconciled destination: %v", err)
	}
	if reconciledAgain.Enabled != updated.Enabled || reconciledAgain.Credential != updated.Credential || len(reconciledAgain.Subscriptions) != len(updated.Subscriptions) || reconciledAgain.Subscriptions[0] != updated.Subscriptions[0] {
		t.Fatalf("second reconciliation changed destination: first=%#v second=%#v", updated, reconciledAgain)
	}
}

func TestDestinationReconciliationMaterializesFeishuInertlyAndIdempotently(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Destination{}, &model.DestinationSubscription{}, &model.Delivery{}); err != nil {
		t.Fatalf("migrate notification destination tables: %v", err)
	}
	destinations := NewDestinationRepository(db)

	if err := notificationapp.ReconcileDestinations(context.Background(), destinations); err != nil {
		t.Fatalf("first ReconcileDestinations: %v", err)
	}
	var records []model.Destination
	if err := db.Order("provider ASC").Find(&records).Error; err != nil {
		t.Fatalf("list materialized destinations: %v", err)
	}
	if len(records) != len(domain.FixedProviders()) {
		t.Fatalf("materialized destination count = %d, want %d", len(records), len(domain.FixedProviders()))
	}
	feishu, err := destinations.Get(context.Background(), domain.ProviderFeishu)
	if err != nil {
		t.Fatalf("get Feishu destination: %v", err)
	}
	if feishu.Enabled || feishu.Credential != "" || len(feishu.Subscriptions) != 0 {
		t.Fatalf("Feishu destination = %#v, want disabled empty inert state", feishu)
	}
	var deliveryCount int64
	if err := db.Model(&model.Delivery{}).Count(&deliveryCount).Error; err != nil {
		t.Fatalf("count delivery side effects: %v", err)
	}
	if deliveryCount != 0 {
		t.Fatalf("reconciliation created %d deliveries, want none", deliveryCount)
	}

	if err := notificationapp.ReconcileDestinations(context.Background(), destinations); err != nil {
		t.Fatalf("second ReconcileDestinations: %v", err)
	}
	var secondCount int64
	if err := db.Model(&model.Destination{}).Count(&secondCount).Error; err != nil {
		t.Fatalf("count destinations after second reconciliation: %v", err)
	}
	if secondCount != int64(len(domain.FixedProviders())) {
		t.Fatalf("second reconciliation destination count = %d, want %d", secondCount, len(domain.FixedProviders()))
	}
}
