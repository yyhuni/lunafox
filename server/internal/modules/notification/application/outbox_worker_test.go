package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

func TestOutboxWorkerMarksUnsupportedWithoutRelease(t *testing.T) {
	now := time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC)
	store := &outboxStoreFake{claimed: []domain.OutboxEvent{{ID: 41, Occurrence: testScanSucceededOccurrence(t, now)}}}
	worker := NewOutboxWorker(store, outboxMaterializerFake{err: &UnsupportedEventError{Code: "unsupported_envelope", Err: errors.New("unknown kind")}}, OutboxWorkerOptions{Owner: "worker"})
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(store.unsupported) != 1 || store.unsupported[0].id != 41 || store.unsupported[0].owner != "worker" || store.unsupported[0].code != "unsupported_envelope" {
		t.Fatalf("unsupported transitions = %#v", store.unsupported)
	}
	if len(store.released) != 0 || len(store.published) != 0 {
		t.Fatalf("unsupported event was released/published: released=%#v published=%#v", store.released, store.published)
	}
}

func TestOutboxWorkerReleasesTransientMaterializationFailure(t *testing.T) {
	now := time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC)
	store := &outboxStoreFake{claimed: []domain.OutboxEvent{{ID: 42, Occurrence: testScanSucceededOccurrence(t, now)}}}
	worker := NewOutboxWorker(store, outboxMaterializerFake{err: errors.New("database unavailable")}, OutboxWorkerOptions{Owner: "worker"})
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(store.released) != 1 || store.released[0].id != 42 {
		t.Fatalf("release transitions = %#v", store.released)
	}
	if len(store.unsupported) != 0 || len(store.published) != 0 {
		t.Fatalf("transient event was terminal/published: unsupported=%#v published=%#v", store.unsupported, store.published)
	}
}

func TestOutboxWorkerPublishesMaterializedEvent(t *testing.T) {
	now := time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC)
	store := &outboxStoreFake{claimed: []domain.OutboxEvent{{ID: 43, Occurrence: testScanSucceededOccurrence(t, now)}}}
	worker := NewOutboxWorker(store, outboxMaterializerFake{}, OutboxWorkerOptions{Owner: "worker"})
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(store.published) != 1 || store.published[0].id != 43 || store.published[0].owner != "worker" {
		t.Fatalf("published transitions = %#v", store.published)
	}
}

type outboxStoreFake struct {
	claimed     []domain.OutboxEvent
	published   []outboxTransition
	unsupported []unsupportedOutboxTransition
	released    []outboxTransition
}

type outboxTransition struct {
	id    int64
	owner string
}

type unsupportedOutboxTransition struct {
	id    int64
	owner string
	code  string
}

func (store *outboxStoreFake) Claim(context.Context, string, time.Time, time.Duration, int) ([]domain.OutboxEvent, error) {
	claimed := store.claimed
	store.claimed = nil
	return claimed, nil
}

func (store *outboxStoreFake) MarkPublished(_ context.Context, id int64, owner string, _ time.Time) error {
	store.published = append(store.published, outboxTransition{id: id, owner: owner})
	return nil
}

func (store *outboxStoreFake) MarkUnsupported(_ context.Context, id int64, owner, code string, _ time.Time) error {
	store.unsupported = append(store.unsupported, unsupportedOutboxTransition{id: id, owner: owner, code: code})
	return nil
}

func (store *outboxStoreFake) ReleaseLease(_ context.Context, id int64, owner string, _ time.Time) error {
	store.released = append(store.released, outboxTransition{id: id, owner: owner})
	return nil
}

func (*outboxStoreFake) DeletePublishedBefore(context.Context, domain.RetentionBatch) (int64, error) {
	return 0, nil
}

type outboxMaterializerFake struct{ err error }

func (fake outboxMaterializerFake) Materialize(context.Context, domain.OutboxEvent) error {
	return fake.err
}

func testScanSucceededOccurrence(t *testing.T, occurredAt time.Time) domain.Occurrence {
	t.Helper()
	occurrence, err := domain.NewScanSucceededOccurrence(1, 2, "example.test", occurredAt)
	if err != nil {
		t.Fatalf("NewScanSucceededOccurrence: %v", err)
	}
	return occurrence
}
