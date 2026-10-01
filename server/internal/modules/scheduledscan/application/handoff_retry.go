package application

import (
	"errors"
	"time"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

// HandoffFailureCause is the closed public cause enum for zero-created Scan
// Create failures. It enters the API contract on the Scheduled Scan resource
// and may only grow: values are never renamed, repurposed, or removed.
type HandoffFailureCause string

const (
	HandoffCauseWorkflowUnavailable       HandoffFailureCause = "WORKFLOW_UNAVAILABLE"
	HandoffCauseAgentNotFound             HandoffFailureCause = "AGENT_NOT_FOUND"
	HandoffCauseConfigResourceUnavailable HandoffFailureCause = "CONFIG_RESOURCE_UNAVAILABLE"
	HandoffCauseEngineUnavailable         HandoffFailureCause = "ENGINE_UNAVAILABLE"
	HandoffCauseTargetUnavailable         HandoffFailureCause = "TARGET_UNAVAILABLE"
	HandoffCauseInternalUnavailable       HandoffFailureCause = "INTERNAL_UNAVAILABLE"
)

func ParseHandoffFailureCause(value string) (HandoffFailureCause, bool) {
	switch cause := HandoffFailureCause(value); cause {
	case HandoffCauseWorkflowUnavailable,
		HandoffCauseAgentNotFound,
		HandoffCauseConfigResourceUnavailable,
		HandoffCauseEngineUnavailable,
		HandoffCauseTargetUnavailable,
		HandoffCauseInternalUnavailable:
		return cause, true
	default:
		return "", false
	}
}

// MaxHandoffRetries bounds the automatic retries of one occurrence after a
// zero-created failure. Only zero-created failures are retryable: any created
// Scan makes a retry duplicate work.
const MaxHandoffRetries = 3

// HandoffRetryDelay returns the backoff before the retry that follows the
// given 1-based committed retryable-failure count.
func HandoffRetryDelay(retryCount int) time.Duration {
	switch retryCount {
	case 1:
		return 30 * time.Second
	case 2:
		return time.Minute
	case MaxHandoffRetries:
		return 3 * time.Minute
	default:
		return 0
	}
}

// RetryableAfterFailures reports whether this outcome should schedule another
// bounded retry given the occurrence's committed retryable-failure count.
// Only zero-created creation failures qualify: any created Scan makes a retry
// duplicate work, and the scan_create_failed kind already implies zero created
// Scans by classification.
func (outcome HandoffOutcome) RetryableAfterFailures(retryCount int) bool {
	return !outcome.Completed() &&
		outcome.Kind == HandoffScanCreateFailed &&
		retryCount < MaxHandoffRetries
}

// ClassifyHandoffFailureCause maps typed Scan Create errors onto the public
// cause enum. Unknown error shapes fail closed to INTERNAL_UNAVAILABLE rather
// than surfacing raw error text.
func ClassifyHandoffFailureCause(err error) HandoffFailureCause {
	switch {
	case errors.Is(err, scanapp.ErrCreateInvalidScanWorkflow),
		errors.Is(err, scanapp.ErrCreateNoScanWorkflows):
		return HandoffCauseWorkflowUnavailable
	case errors.Is(err, scanapp.ErrCreateAgentNotFound):
		return HandoffCauseAgentNotFound
	case errors.Is(err, scanapp.ErrCreateScanWorkflowEngineUnavailable):
		return HandoffCauseEngineUnavailable
	case errors.Is(err, scanapp.ErrCreateTargetNotFound),
		errors.Is(err, scanapp.ErrNoTargetsForScan):
		return HandoffCauseTargetUnavailable
	case errors.Is(err, scanapp.ErrCreateInvalidConfig),
		errors.Is(err, scanapp.ErrCreateInvalidInputSource):
		return HandoffCauseConfigResourceUnavailable
	default:
		return HandoffCauseInternalUnavailable
	}
}
