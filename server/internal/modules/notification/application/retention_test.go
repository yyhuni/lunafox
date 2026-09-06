package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

func TestRetentionJobDeletesOnlyBoundedExpiredRows(t *testing.T) {
	now := time.Date(2026, time.August, 8, 10, 0, 0, 0, time.UTC)
	outbox := &retentionOutboxStoreFake{deleted: 3}
	inbox := &retentionInboxStoreFake{deleted: 5}
	deliveries := &retentionDeliveryStoreFake{deleted: 7}
	job := NewRetentionJob(outbox, inbox, deliveries, RetentionJobOptions{BatchSize: 17})
	job.now = func() time.Time { return now }

	result, err := job.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if result != (RetentionResult{PublishedOutboxRows: 3, ExpiredInboxRows: 5, TerminalDeliveries: 7}) {
		t.Fatalf("result = %#v", result)
	}
	wantCutoff := now.Add(-domain.NotificationRetention)
	for _, batch := range []domain.RetentionBatch{outbox.batch, inbox.batch, deliveries.batch} {
		if batch.Limit != 17 || !batch.Before.Equal(wantCutoff) {
			t.Fatalf("retention batch = %#v, want limit=17 cutoff=%s", batch, wantCutoff)
		}
	}
}

func TestRetentionJobStopsAfterFirstRepositoryFailure(t *testing.T) {
	outbox := &retentionOutboxStoreFake{err: errors.New("database unavailable")}
	inbox := &retentionInboxStoreFake{}
	deliveries := &retentionDeliveryStoreFake{}
	job := NewRetentionJob(outbox, inbox, deliveries, RetentionJobOptions{})

	if _, err := job.RunOnce(context.Background()); err == nil {
		t.Fatal("RunOnce succeeded, want repository failure")
	}
	if inbox.called || deliveries.called {
		t.Fatal("retention continued after an outbox failure")
	}
}

type retentionOutboxStoreFake struct {
	batch   domain.RetentionBatch
	deleted int64
	err     error
}

func (*retentionOutboxStoreFake) Claim(context.Context, string, time.Time, time.Duration, int) ([]domain.OutboxEvent, error) {
	return nil, nil
}

func (*retentionOutboxStoreFake) MarkPublished(context.Context, int64, string, time.Time) error {
	return nil
}

func (*retentionOutboxStoreFake) MarkUnsupported(context.Context, int64, string, string, time.Time) error {
	return nil
}

func (*retentionOutboxStoreFake) ReleaseLease(context.Context, int64, string, time.Time) error {
	return nil
}

func (store *retentionOutboxStoreFake) DeletePublishedBefore(_ context.Context, batch domain.RetentionBatch) (int64, error) {
	store.batch = batch
	return store.deleted, store.err
}

type retentionInboxStoreFake struct {
	batch   domain.RetentionBatch
	deleted int64
	err     error
	called  bool
}

func (*retentionInboxStoreFake) List(context.Context, int, int, string, time.Time) (InboxPage, error) {
	return InboxPage{}, nil
}

func (*retentionInboxStoreFake) Get(context.Context, int, int64, time.Time) (domain.InboxItem, error) {
	return domain.InboxItem{}, nil
}

func (*retentionInboxStoreFake) UnreadCount(context.Context, int, time.Time) (int64, error) {
	return 0, nil
}

func (*retentionInboxStoreFake) MarkRead(context.Context, int, int64, time.Time) (domain.InboxItem, error) {
	return domain.InboxItem{}, nil
}

func (*retentionInboxStoreFake) MarkAllRead(context.Context, int, time.Time) error { return nil }

func (store *retentionInboxStoreFake) DeleteExpiredInbox(_ context.Context, batch domain.RetentionBatch) (int64, error) {
	store.called = true
	store.batch = batch
	return store.deleted, store.err
}

type retentionDeliveryStoreFake struct {
	batch   domain.RetentionBatch
	deleted int64
	err     error
	called  bool
}

func (*retentionDeliveryStoreFake) Claim(context.Context, string, time.Time, time.Duration, int) ([]domain.Delivery, error) {
	return nil, nil
}

func (*retentionDeliveryStoreFake) RecordAttempt(context.Context, domain.DeliveryAttempt, string) error {
	return nil
}

func (*retentionDeliveryStoreFake) MarkDelivered(context.Context, int64, string, time.Time) error {
	return nil
}

func (*retentionDeliveryStoreFake) MarkRetry(context.Context, int64, string, time.Time, time.Time) error {
	return nil
}

func (*retentionDeliveryStoreFake) MarkFailedTerminal(context.Context, int64, string, time.Time) error {
	return nil
}

func (store *retentionDeliveryStoreFake) DeleteTerminalBefore(_ context.Context, batch domain.RetentionBatch) (int64, error) {
	store.called = true
	store.batch = batch
	return store.deleted, store.err
}
