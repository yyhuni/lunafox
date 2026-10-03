package application

import (
	"context"
	"time"
)

// OccurrenceStatus is the closed public status enum for the occurrence
// history projection. Values only grow; they are never renamed, repurposed,
// or removed. The status is derived server-side from the ledger columns, so
// clients never re-derive it from raw timestamps.
type OccurrenceStatus string

const (
	// OccurrenceStatusPending marks a materialized occurrence no attempt has
	// started yet.
	OccurrenceStatusPending OccurrenceStatus = "PENDING"
	// OccurrenceStatusDispatching marks an attempted, unsettled occurrence
	// with no pending retry. It covers both an in-flight handoff and one stuck
	// past the handoff deadline; the raw attemptedAt stays visible so clients
	// can flag suspected-stuck rows.
	OccurrenceStatusDispatching OccurrenceStatus = "DISPATCHING"
	// OccurrenceStatusRetrying marks an unsettled occurrence waiting for its
	// bounded retry deadline.
	OccurrenceStatusRetrying OccurrenceStatus = "RETRYING"
	// OccurrenceStatusSucceeded marks an occurrence whose handoff completed.
	OccurrenceStatusSucceeded OccurrenceStatus = "SUCCEEDED"
	// OccurrenceStatusFailed marks an occurrence with a terminal failure.
	OccurrenceStatusFailed OccurrenceStatus = "FAILED"
)

func ParseOccurrenceStatus(value string) (OccurrenceStatus, bool) {
	switch OccurrenceStatus(value) {
	case OccurrenceStatusPending,
		OccurrenceStatusDispatching,
		OccurrenceStatusRetrying,
		OccurrenceStatusSucceeded,
		OccurrenceStatusFailed:
		return OccurrenceStatus(value), true
	default:
		return "", false
	}
}

// OccurrenceSnapshot is the raw persisted ledger row handed to the
// application layer; it carries no derived state.
type OccurrenceSnapshot struct {
	ID              int64
	ScheduledScanID int
	ScheduledFor    time.Time
	AttemptedAt     *time.Time
	DispatchedAt    *time.Time
	FailureKind     *string
	FailureMessage  *string
	RetryCount      int
	NextRetryAt     *time.Time
	LastFailureCause *string
}

// OccurrenceRecord is the read-only occurrence history projection. DurationMs
// is populated only for SUCCEEDED rows (dispatchedAt minus attemptedAt); a
// terminal failure has no settlement timestamp, so it stays nil instead of a
// synthesized value.
type OccurrenceRecord struct {
	ID             int64
	ScheduledFor   time.Time
	AttemptedAt    *time.Time
	DispatchedAt   *time.Time
	Status         OccurrenceStatus
	FailureKind    *string
	FailureCause   *HandoffFailureCause
	FailureMessage *string
	RetryCount     int
	NextRetryAt    *time.Time
	DurationMs     *int64
}

// OccurrenceStatusCounts aggregates every retained occurrence of one
// Scheduled Scan by projected status. Counts reflect only rows that survive
// the occurrence retention policy; they are lifetime counters and MUST NOT be
// coerced against runCount or the handoff aggregates.
type OccurrenceStatusCounts struct {
	Pending     int64
	Dispatching int64
	Retrying    int64
	Succeeded   int64
	Failed      int64
}

func (counts OccurrenceStatusCounts) Total() int64 {
	return counts.Pending + counts.Dispatching + counts.Retrying + counts.Succeeded + counts.Failed
}

// OccurrenceHistoryQuery requests one page of newest-first occurrence
// history for one Scheduled Scan.
type OccurrenceHistoryQuery struct {
	ScheduledScanID int
	Page            int
	PageSize        int
}

type OccurrenceHistory struct {
	Records []OccurrenceRecord
	Total   int64
	Counts  OccurrenceStatusCounts
}

// ScheduledScanOccurrenceStore is the read-only occurrence ledger port. It is
// deliberately separate from ScheduledScanStore so the command lifecycle
// interfaces stay unchanged.
type ScheduledScanOccurrenceStore interface {
	ListOccurrences(ctx context.Context, query OccurrenceHistoryQuery) ([]OccurrenceSnapshot, int64, error)
	CountOccurrencesByStatus(ctx context.Context, scheduledScanID int) (OccurrenceStatusCounts, error)
}

// DeriveOccurrenceStatus maps one raw ledger row onto the public status enum.
// The precedence mirrors the RecordOutcome settlement gates: a dispatched
// timestamp or terminal failure kind settles first, then the pending retry,
// then the unsettled attempt, and an unattempted row stays pending.
func DeriveOccurrenceStatus(snapshot OccurrenceSnapshot) OccurrenceStatus {
	switch {
	case snapshot.DispatchedAt != nil:
		return OccurrenceStatusSucceeded
	case snapshot.FailureKind != nil:
		return OccurrenceStatusFailed
	case snapshot.AttemptedAt != nil && snapshot.NextRetryAt != nil:
		return OccurrenceStatusRetrying
	case snapshot.AttemptedAt != nil:
		return OccurrenceStatusDispatching
	default:
		return OccurrenceStatusPending
	}
}

func projectOccurrenceRecord(snapshot OccurrenceSnapshot) OccurrenceRecord {
	record := OccurrenceRecord{
		ID:             snapshot.ID,
		ScheduledFor:   snapshot.ScheduledFor.UTC(),
		AttemptedAt:    cloneTimePtr(snapshot.AttemptedAt),
		DispatchedAt:   cloneTimePtr(snapshot.DispatchedAt),
		Status:         DeriveOccurrenceStatus(snapshot),
		FailureKind:    cloneStringPtr(snapshot.FailureKind),
		FailureMessage: cloneStringPtr(snapshot.FailureMessage),
		RetryCount:     snapshot.RetryCount,
		NextRetryAt:    cloneTimePtr(snapshot.NextRetryAt),
	}
	if snapshot.LastFailureCause != nil {
		cause := HandoffFailureCause(*snapshot.LastFailureCause)
		record.FailureCause = &cause
	}
	if record.Status == OccurrenceStatusSucceeded && snapshot.AttemptedAt != nil && snapshot.DispatchedAt != nil {
		duration := snapshot.DispatchedAt.Sub(*snapshot.AttemptedAt).Milliseconds()
		if duration < 0 {
			duration = 0
		}
		record.DurationMs = &duration
	}
	return record
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
