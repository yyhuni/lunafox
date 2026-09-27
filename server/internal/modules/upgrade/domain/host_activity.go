package domain

import (
	"fmt"
	"time"
)

// HostAction identifies a fixed, host-owned operation whose in-flight wait may
// be shown to an operator. It is deliberately not an argv, service identity,
// image reference, or free-form description.
type HostAction string

const (
	HostActionPreflight               HostAction = "preflight"
	HostActionPullImages              HostAction = "pull_images"
	HostActionUpdateServices          HostAction = "update_services"
	HostActionDatabaseMigration       HostAction = "database_migration"
	HostActionUpdateResidentAgent     HostAction = "update_resident_agent"
	HostActionWaitForServiceHealth    HostAction = "wait_for_service_health"
	HostActionVerifyRuntimeImages     HostAction = "verify_runtime_images"
	HostActionVerifyFrontendContainer HostAction = "verify_frontend_container"
	HostActionVerifyFrontendEdge      HostAction = "verify_frontend_edge"
)

// HostActionFailureReason is the bounded reason vocabulary for one failed host
// action boundary event. The original Runner error stays in host-side logs.
type HostActionFailureReason string

const (
	HostActionFailureCommandFailed     HostActionFailureReason = "command_failed"
	HostActionFailureCommandCancelled  HostActionFailureReason = "command_cancelled"
	HostActionFailureValidationFailed  HostActionFailureReason = "validation_failed"
	HostActionFailureLocalActionFailed HostActionFailureReason = "local_action_failed"
)

const (
	HostActionProgressStarted   = "hostActionStarted"
	HostActionProgressCompleted = "hostActionCompleted"
	HostActionProgressFailed    = "hostActionFailed"
)

var hostActionProgressMessages = map[string]string{
	HostActionProgressStarted:   "Host upgrade action started",
	HostActionProgressCompleted: "Host upgrade action completed",
	HostActionProgressFailed:    "Host upgrade action failed",
}

const maxHostActivityClockSkew = 5 * time.Minute

// HostActivity is one bounded current snapshot. It describes that the host is
// still waiting for a known action to return; it never claims internal progress
// within that action and must not be used as lifecycle evidence.
type HostActivity struct {
	Action          HostAction `json:"action"`
	StartedAt       time.Time  `json:"startedAt"`
	LastHeartbeatAt time.Time  `json:"lastHeartbeatAt"`
}

// IsKnownHostAction reports whether action belongs to the closed operator
// vocabulary shared by the host journal, persisted Operation, and HTTP DTO.
func IsKnownHostAction(action HostAction) bool {
	switch action {
	case HostActionPreflight, HostActionPullImages, HostActionUpdateServices,
		HostActionDatabaseMigration, HostActionUpdateResidentAgent,
		HostActionWaitForServiceHealth, HostActionVerifyRuntimeImages,
		HostActionVerifyFrontendContainer, HostActionVerifyFrontendEdge:
		return true
	default:
		return false
	}
}

// IsKnownHostActionFailureReason reports whether reason is safe to project in
// a bounded host-action failure event.
func IsKnownHostActionFailureReason(reason HostActionFailureReason) bool {
	switch reason {
	case HostActionFailureCommandFailed, HostActionFailureCommandCancelled,
		HostActionFailureValidationFailed, HostActionFailureLocalActionFailed:
		return true
	default:
		return false
	}
}

// ValidateHostActionProgressEvent reserves the three host-action boundary
// messages for the closed action/reason vocabulary. Other progress events keep
// their existing catalog validation rules.
func ValidateHostActionProgressEvent(messageKey, message string, metadata map[string]string) error {
	expectedMessage, isHostActionEvent := hostActionProgressMessages[messageKey]
	if !isHostActionEvent {
		return nil
	}
	if message != expectedMessage {
		return fmt.Errorf("host action progress event message does not match its catalog key")
	}
	if !IsKnownHostAction(HostAction(metadata["action"])) {
		return fmt.Errorf("host action progress event action is invalid")
	}
	if messageKey == HostActionProgressFailed {
		if len(metadata) != 2 || !IsKnownHostActionFailureReason(HostActionFailureReason(metadata["reason"])) {
			return fmt.Errorf("host action failure progress event reason is invalid")
		}
		return nil
	}
	if len(metadata) != 1 {
		return fmt.Errorf("host action progress event metadata is invalid")
	}
	return nil
}

// ValidateHostActivity validates optional current activity against the durable
// lifecycle. A nil snapshot is the compatible representation for old journals
// and persisted Operations that predate host-side activity observation.
func ValidateHostActivity(activity *HostActivity, status Status, mode ExecutionMode, operationStartedAt, observedAt, now time.Time) error {
	if activity == nil {
		return nil
	}
	if !status.Valid() || status.IsTerminal() {
		return fmt.Errorf("host activity requires an active upgrade status")
	}
	if mode == "" {
		mode = ExecutionModeFull
	}
	if !mode.Valid() {
		return fmt.Errorf("host activity execution mode is invalid")
	}
	if !IsKnownHostAction(activity.Action) {
		return fmt.Errorf("host activity action is invalid")
	}
	if !hostActionAllowedAt(activity.Action, status, mode) {
		return fmt.Errorf("host activity action %q is not owned by status %q and execution mode %q", activity.Action, status, mode)
	}
	if activity.StartedAt.IsZero() || activity.LastHeartbeatAt.IsZero() {
		return fmt.Errorf("host activity timestamps are required")
	}
	if activity.StartedAt.After(activity.LastHeartbeatAt) {
		return fmt.Errorf("host activity startedAt is newer than lastHeartbeatAt")
	}
	if !operationStartedAt.IsZero() && activity.StartedAt.Before(operationStartedAt) {
		return fmt.Errorf("host activity startedAt precedes the upgrade operation")
	}
	if !observedAt.IsZero() && activity.LastHeartbeatAt.After(observedAt) {
		return fmt.Errorf("host activity heartbeat is newer than the observation")
	}
	if !now.IsZero() && (activity.StartedAt.After(now.Add(maxHostActivityClockSkew)) || activity.LastHeartbeatAt.After(now.Add(maxHostActivityClockSkew))) {
		return fmt.Errorf("host activity timestamp exceeds the allowed clock skew")
	}
	return nil
}

func hostActionAllowedAt(action HostAction, status Status, mode ExecutionMode) bool {
	switch action {
	case HostActionPreflight:
		return status == StatusPreflight
	case HostActionPullImages, HostActionUpdateServices:
		return status == StatusUpdating
	case HostActionDatabaseMigration:
		return mode == ExecutionModeFull && status == StatusMigrating
	case HostActionUpdateResidentAgent:
		return mode == ExecutionModeFull && status == StatusRestarting
	case HostActionWaitForServiceHealth:
		if mode == ExecutionModeFrontendOnly {
			return status == StatusRestarting || status == StatusVerifying
		}
		return status == StatusRestarting || status == StatusAgentVerifying || status == StatusVerifying
	case HostActionVerifyRuntimeImages:
		return mode == ExecutionModeFull && status == StatusVerifying
	case HostActionVerifyFrontendContainer, HostActionVerifyFrontendEdge:
		return mode == ExecutionModeFrontendOnly && status == StatusVerifying
	default:
		return false
	}
}

// CloneHostActivity returns a defensive copy for persistence and projections.
func CloneHostActivity(activity *HostActivity) *HostActivity {
	if activity == nil {
		return nil
	}
	copy := *activity
	copy.StartedAt = copy.StartedAt.UTC()
	copy.LastHeartbeatAt = copy.LastHeartbeatAt.UTC()
	return &copy
}

// EqualHostActivity compares the complete bounded snapshot without exposing
// any mutable map or slice state.
func EqualHostActivity(left, right *HostActivity) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Action == right.Action && left.StartedAt.Equal(right.StartedAt) && left.LastHeartbeatAt.Equal(right.LastHeartbeatAt)
}
