package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/notification/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateOccurrenceRejectsInvalidPriorityBeforeWrite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:notification_outbox_invalid_priority?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Outbox{}); err != nil {
		t.Fatalf("migrate outbox: %v", err)
	}
	repository := NewOutboxRepository(db)
	err = db.Transaction(func(tx *gorm.DB) error {
		return repository.CreateOccurrence(tx, domain.Occurrence{
			EventID: "scan:1:succeeded", Kind: domain.KindScanSucceeded, PayloadVersion: domain.PayloadVersionOne,
			Subject: "scans/1", OccurredAt: time.Now().UTC(), Priority: domain.Priority("low"),
			Payload: []byte(`{"scanId":1,"targetId":2,"targetName":"example.test"}`),
		})
	})
	if err == nil {
		t.Fatal("CreateOccurrence accepted low priority")
	}
}

func TestOutboxClaimRecoversExpiredLeaseAndFencesOwnerTransitions(t *testing.T) {
	repository := newOutboxRepositoryForTest(t)
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	createOutboxOccurrence(t, repository, "scan:1:succeeded", now)

	claimed, err := repository.Claim(context.Background(), "worker-a", now, time.Minute, 10)
	if err != nil {
		t.Fatalf("Claim first lease: %v", err)
	}
	if len(claimed) != 1 || claimed[0].LeaseOwner != "worker-a" {
		t.Fatalf("Claim first lease = %#v, want worker-a event", claimed)
	}

	claimed, err = repository.Claim(context.Background(), "worker-b", now.Add(30*time.Second), time.Minute, 10)
	if err != nil {
		t.Fatalf("Claim unexpired lease: %v", err)
	}
	if len(claimed) != 0 {
		t.Fatalf("Claim unexpired lease = %#v, want no work", claimed)
	}

	claimed, err = repository.Claim(context.Background(), "worker-b", now.Add(time.Minute), time.Minute, 10)
	if err != nil {
		t.Fatalf("Claim expired lease: %v", err)
	}
	if len(claimed) != 1 || claimed[0].LeaseOwner != "worker-b" {
		t.Fatalf("Claim expired lease = %#v, want worker-b event", claimed)
	}
	if err := repository.MarkPublished(context.Background(), claimed[0].ID, "worker-a", now.Add(2*time.Minute)); err == nil {
		t.Fatal("MarkPublished accepted stale lease owner")
	}
	if err := repository.MarkPublished(context.Background(), claimed[0].ID, "worker-b", now.Add(2*time.Minute)); err != nil {
		t.Fatalf("MarkPublished current owner: %v", err)
	}
}

func TestOutboxRetentionKeepsUnsupportedAndInFlightRows(t *testing.T) {
	repository := newOutboxRepositoryForTest(t)
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	createOutboxOccurrence(t, repository, "scan:1:succeeded", now)
	createOutboxOccurrence(t, repository, "scan:2:succeeded", now)
	createOutboxOccurrence(t, repository, "scan:3:succeeded", now)

	claimed, err := repository.Claim(context.Background(), "worker", now, time.Minute, 3)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if len(claimed) != 3 {
		t.Fatalf("Claim count = %d, want 3", len(claimed))
	}
	if err := repository.MarkPublished(context.Background(), claimed[0].ID, "worker", now.Add(time.Minute)); err != nil {
		t.Fatalf("MarkPublished: %v", err)
	}
	if err := repository.MarkUnsupported(context.Background(), claimed[1].ID, "worker", "unsupported_envelope", now.Add(time.Minute)); err != nil {
		t.Fatalf("MarkUnsupported: %v", err)
	}

	deleted, err := repository.DeletePublishedBefore(context.Background(), domain.RetentionBatch{Limit: 10, Before: now.Add(2 * time.Minute)})
	if err != nil {
		t.Fatalf("DeletePublishedBefore: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("DeletePublishedBefore deleted = %d, want 1", deleted)
	}

	var records []model.Outbox
	if err := repository.db.Order("id ASC").Find(&records).Error; err != nil {
		t.Fatalf("list remaining outbox rows: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("remaining outbox rows = %d, want 2", len(records))
	}
	if records[0].Status != string(domain.OutboxStatusUnsupportedTerminal) || records[1].Status != string(domain.OutboxStatusProcessing) {
		t.Fatalf("remaining statuses = %q, %q, want unsupported then processing", records[0].Status, records[1].Status)
	}
}

func newOutboxRepositoryForTest(t *testing.T) *OutboxRepository {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Outbox{}); err != nil {
		t.Fatalf("migrate outbox: %v", err)
	}
	return NewOutboxRepository(db)
}

func createOutboxOccurrence(t *testing.T, repository *OutboxRepository, eventID string, occurredAt time.Time) {
	t.Helper()
	err := repository.db.Transaction(func(tx *gorm.DB) error {
		return repository.CreateOccurrence(tx, domain.Occurrence{
			EventID:        eventID,
			Kind:           domain.KindScanSucceeded,
			PayloadVersion: domain.PayloadVersionOne,
			Subject:        "scans/1",
			OccurredAt:     occurredAt,
			Priority:       domain.PriorityNormal,
			Payload:        []byte(`{"scanId":1,"targetId":2,"targetName":"example.test"}`),
		})
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			t.Fatalf("unexpected duplicate test occurrence %q: %v", eventID, err)
		}
		t.Fatalf("CreateOccurrence %q: %v", eventID, err)
	}
	if err := repository.db.Model(&model.Outbox{}).Where("event_id = ?", eventID).Update("available_at", occurredAt).Error; err != nil {
		t.Fatalf("set test occurrence availability %q: %v", eventID, err)
	}
}
