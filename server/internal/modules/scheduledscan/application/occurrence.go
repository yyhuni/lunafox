package application

import (
	"context"
	"time"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

const (
	DueScheduleBatchSize      = 100
	OccurrenceDeleteBatchSize = 1000
)

type DueSchedule struct {
	ID          int
	NextRunTime time.Time
}

type OccurrenceCandidate struct {
	ID              int64
	ScheduledScanID int
	ScheduledFor    time.Time
}

// FrozenDispatchInput is copied while the Schedule row is locked. TargetIDs is
// the complete immutable execution scope for one attempt; it intentionally
// carries neither an Organization reference nor a durable provenance edge.
type FrozenDispatchInput struct {
	OccurrenceID    int64
	ScheduledScanID int
	ScheduledFor    time.Time
	ScanWorkflowID  string
	Configuration   map[string]any
	InputSource     scandomain.InputSource
	TargetIDs       []int
	TargetScoped    bool
	AgentID         *int
}

type HandoffOutcomeKind string

const (
	HandoffCompleted        HandoffOutcomeKind = "completed"
	HandoffPartial          HandoffOutcomeKind = "partial"
	HandoffDeadlineExceeded HandoffOutcomeKind = "deadline_exceeded"
	HandoffCanceled         HandoffOutcomeKind = "canceled"
	HandoffScanCreateFailed HandoffOutcomeKind = "scan_create_failed"
)

type HandoffOutcome struct {
	Kind    HandoffOutcomeKind
	Message string
}

func (outcome HandoffOutcome) Completed() bool {
	return outcome.Kind == HandoffCompleted
}

type ExecutionRepository interface {
	ListDueSchedules(ctx context.Context, evaluationAt time.Time) ([]DueSchedule, error)
	MaterializeDue(ctx context.Context, scheduledScanID int, evaluationAt time.Time) (bool, error)
	SelectAttemptCandidate(ctx context.Context) (*OccurrenceCandidate, error)
	StartAttempt(ctx context.Context, candidate OccurrenceCandidate, attemptedAt time.Time) (*FrozenDispatchInput, error)
	RecordOutcome(ctx context.Context, occurrenceID int64, outcome HandoffOutcome, recordedAt time.Time) (bool, error)
}

type OccurrenceRetentionRepository interface {
	DeleteOccurrenceBatch(ctx context.Context, attemptedAtCutoff time.Time) (int64, error)
}
