package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Status is the durable user-visible lifecycle of one system upgrade.
// Migration and Agent verification are deliberately terminally distinct from
// a generic failed preflight so recovery tooling can apply the right policy.
type Status string

const (
	StatusQueued         Status = "queued"
	StatusStopping       Status = "stopping"
	StatusPreflight      Status = "preflight"
	StatusUpdating       Status = "updating"
	StatusMigrating      Status = "migrating"
	StatusRestarting     Status = "restarting"
	StatusAgentVerifying Status = "agent_verifying"
	StatusVerifying      Status = "verifying"
	StatusSucceeded      Status = "succeeded"
	StatusFailed         Status = "failed"
	StatusNeedsRecovery  Status = "needs_recovery"
	StatusNeedsAttention Status = "needs_attention"
)

type MigrationStatus string

const (
	MigrationStatusNotStarted MigrationStatus = "not_started"
	MigrationStatusRunning    MigrationStatus = "running"
	MigrationStatusSucceeded  MigrationStatus = "succeeded"
	MigrationStatusFailed     MigrationStatus = "failed"
	MigrationStatusUnknown    MigrationStatus = "unknown"
)

type AgentSummary struct {
	Expected  int `json:"expected"`
	Ready     int `json:"ready"`
	Missing   int `json:"missing"`
	Unhealthy int `json:"unhealthy"`
}

// AgentExpectation is the immutable-at-start snapshot and subsequent
// verification projection for one enabled Agent. It intentionally contains
// only operational evidence; authentication material is never copied into an
// Upgrade Operation.
type AgentExpectation struct {
	AgentID         int        `json:"agentId"`
	DesiredVersion  string     `json:"desiredVersion"`
	TargetDigest    string     `json:"targetDigest"`
	ObservedVersion string     `json:"observedVersion,omitempty"`
	ObservedDigest  string     `json:"observedDigest,omitempty"`
	Connected       bool       `json:"connected"`
	Healthy         bool       `json:"healthy"`
	Paused          bool       `json:"paused"`
	ClaimReady      bool       `json:"claimReady"`
	LastObservedAt  *time.Time `json:"lastObservedAt,omitempty"`
	Diagnostic      string     `json:"diagnostic,omitempty"`
}

// Ready reports whether the Agent has supplied enough authenticated runtime
// evidence to participate in a completed upgrade. The current control-plane
// heartbeat exposes the Agent version but not its container image digest; an
// empty ObservedDigest is therefore kept unknown rather than fabricated. When
// a future heartbeat supplies a digest, it is compared strictly to the target.
func (expectation AgentExpectation) Ready() bool {
	if !expectation.Connected || !expectation.Healthy || expectation.Paused || !expectation.ClaimReady {
		return false
	}
	if strings.TrimSpace(expectation.DesiredVersion) == "" || expectation.ObservedVersion != expectation.DesiredVersion {
		return false
	}
	if expectation.ObservedDigest != "" && expectation.TargetDigest != "" && expectation.ObservedDigest != expectation.TargetDigest {
		return false
	}
	return true
}

// Operation contains only bounded, non-secret operational evidence. Free-form
// command output and credentials must never be copied into Diagnostic fields.
type Operation struct {
	OperationID               string
	RequestID                 string
	OperatorID                int
	ManifestID                string
	ManifestDigest            string
	ReleaseVersion            string
	CompatibilityRange        string
	MaintenanceWindowMinutes  int
	Status                    Status
	MigrationStatus           MigrationStatus
	MigrationType             string
	MigrationID               string
	MigrationChecksum         string
	CancelledScanCount        int
	CancelledTaskCount        int
	AgentDesiredVersion       string
	AgentTargetDigest         string
	AgentSummary              AgentSummary
	AgentExpectations         []AgentExpectation
	AgentVerificationDeadline *time.Time
	ObservedDigests           map[string]string
	ProgressEvents            []ProgressEvent
	Diagnostic                string
	StageTimes                map[Status]time.Time
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
	CompletedAt               *time.Time
}

func (status Status) IsTerminal() bool {
	switch status {
	case StatusSucceeded, StatusFailed, StatusNeedsRecovery, StatusNeedsAttention:
		return true
	default:
		return false
	}
}

func (status Status) Valid() bool {
	switch status {
	case StatusQueued, StatusStopping, StatusPreflight, StatusUpdating,
		StatusMigrating, StatusRestarting, StatusAgentVerifying, StatusVerifying,
		StatusSucceeded, StatusFailed, StatusNeedsRecovery, StatusNeedsAttention:
		return true
	default:
		return false
	}
}

// ValidateTransition is the explicit lifecycle graph for an Upgrade Operation.
// Host events are untrusted observations, so accepting arbitrary rank jumps
// would allow a forged/late completion to skip stopping, migration, or health
// verification. Recovery outcomes are the only cross-cutting exits.
func ValidateTransition(from, to Status) error {
	if !from.Valid() || !to.Valid() {
		return fmt.Errorf("unsupported upgrade status transition %q -> %q", from, to)
	}
	if from.IsTerminal() {
		// A terminal non-success outcome may be escalated when a later checkpoint
		// proves migration failure/uncertainty. This preserves the safety priority
		// of needs_recovery without reopening normal execution. A succeeded
		// operation remains authoritative and cannot be rewritten by a late event.
		if to == StatusNeedsRecovery && from != StatusSucceeded {
			return nil
		}
		return fmt.Errorf("terminal upgrade operation cannot transition from %q to %q", from, to)
	}
	if from == to {
		return nil
	}
	// Recovery outcomes are explicit safety fences and may be entered from any
	// active phase. They are never a route back into normal execution.
	if to == StatusNeedsRecovery || to == StatusNeedsAttention {
		return nil
	}
	allowed := map[Status]map[Status]struct{}{
		StatusQueued: {
			StatusStopping: {},
		},
		StatusStopping: {
			StatusPreflight: {},
		},
		StatusPreflight: {
			StatusUpdating: {},
			StatusFailed:   {},
		},
		StatusUpdating: {
			StatusMigrating:  {},
			StatusRestarting: {},
			StatusFailed:     {},
		},
		StatusMigrating: {
			StatusRestarting: {},
		},
		StatusRestarting: {
			StatusAgentVerifying: {},
			StatusVerifying:      {},
		},
		StatusAgentVerifying: {
			StatusVerifying: {},
		},
		StatusVerifying: {
			StatusSucceeded: {},
			StatusFailed:    {},
		},
	}
	if _, ok := allowed[from][to]; !ok {
		return fmt.Errorf("upgrade operation transition from %q to %q is not allowed", from, to)
	}
	return nil
}

func (operation *Operation) Validate() error {
	if operation == nil {
		return fmt.Errorf("upgrade operation is required")
	}
	if strings.TrimSpace(operation.OperationID) == "" || strings.TrimSpace(operation.RequestID) == "" {
		return fmt.Errorf("operationId and requestId are required")
	}
	if operation.OperatorID <= 0 {
		return fmt.Errorf("operatorId is required")
	}
	if strings.TrimSpace(operation.ManifestID) == "" || strings.TrimSpace(operation.ManifestDigest) == "" {
		return fmt.Errorf("manifest identity and digest are required")
	}
	if !operation.Status.Valid() {
		return fmt.Errorf("unsupported upgrade status %q", operation.Status)
	}
	if operation.MigrationStatus == "" {
		return fmt.Errorf("migration status is required")
	}
	if operation.CreatedAt.IsZero() || operation.UpdatedAt.IsZero() {
		return fmt.Errorf("operation timestamps are required")
	}
	if operation.AgentVerificationDeadline != nil && operation.AgentVerificationDeadline.Before(operation.CreatedAt) {
		return fmt.Errorf("agent verification deadline cannot precede operation creation")
	}
	if err := ValidateProgressEvents(operation.ProgressEvents); err != nil {
		return fmt.Errorf("invalid upgrade progress events: %w", err)
	}
	return nil
}

// Repository is the durable Server-side source of truth for user-visible
// Upgrade Operations. Implementations must make requestId unique.
type Repository interface {
	CreateOrGet(context.Context, *Operation) (*Operation, bool, error)
	Get(context.Context, string) (*Operation, error)
	Update(context.Context, *Operation) error
	FindActive(context.Context) (*Operation, error)
}

// TransitionRepository adds an atomic compare-and-set boundary without
// breaking older in-memory repositories used by callers and tests. Production
// event reconciliation MUST use this interface when available.
type TransitionRepository interface {
	Repository
	UpdateTransition(context.Context, *Operation, Status) error
}

// RetryRepository adds the one operation that is intentionally allowed to
// reopen a terminal, retryable operation.  It is separate from Update so a
// normal event writer can never accidentally turn a terminal result back into
// active work. Implementations must compare the persisted status and target
// identity in the same transaction before applying the reset.
type RetryRepository interface {
	Repository
	ResetForRetry(context.Context, *Operation, Status) error
}
