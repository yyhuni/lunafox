package domain

import (
	"fmt"
	"strings"
	"unicode/utf8"

	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
)

// EngineDiagnosticResultTypeLimit mirrors the Engine API v2 fixed diagnostic
// cardinality. Keeping the persisted model bounded prevents scan history from
// turning a result stream into a per-item diagnostic store.
const EngineDiagnosticResultTypeLimit = engineexecutionpb.DiagnosticResultTypeLimit

const (
	EngineDiagnosticAvailabilityAvailable   = "available"
	EngineDiagnosticAvailabilityUnavailable = "unavailable"

	EngineDiagnosticResultStateComplete = "complete"
	EngineDiagnosticResultStatePartial  = "partial"
	EngineDiagnosticResultStateNone     = "none"
	EngineDiagnosticResultStateUnknown  = "unknown"
)

// EngineExecutionDiagnostics is the Server-owned persisted projection of the
// Agent terminal diagnostic snapshot. It intentionally contains only closed
// classifications and watermarks, never raw errors, logs, payloads, paths, or
// credentials.
type EngineExecutionDiagnostics struct {
	CompatibilityRevision string                `json:"compatibilityRevision"`
	Availability          string                `json:"availability"`
	ResultState           string                `json:"resultState"`
	FailedStage           string                `json:"failedStage,omitempty"`
	ErrorType             string                `json:"errorType,omitempty"`
	ResultTypeWatermarks  []ResultTypeWatermark `json:"resultTypeWatermarks,omitempty"`
}

// ResultTypeWatermark records only monotonic common transport boundaries for
// one canonical result type. Unresolved batches remain derivable as submitted
// minus acknowledged batches; no failed-batch or per-item record is stored.
type ResultTypeWatermark struct {
	ResultType          string `json:"resultType"`
	ReceivedItems       uint64 `json:"receivedItems"`
	EncodedItems        uint64 `json:"encodedItems"`
	SubmittedItems      uint64 `json:"submittedItems"`
	AcknowledgedItems   uint64 `json:"acknowledgedItems"`
	SubmittedBatches    uint64 `json:"submittedBatches"`
	AcknowledgedBatches uint64 `json:"acknowledgedBatches"`
}

// UnavailableEngineExecutionDiagnostics is the only valid representation when
// the Engine side channel never supplied a complete trusted observation.
func UnavailableEngineExecutionDiagnostics() *EngineExecutionDiagnostics {
	return &EngineExecutionDiagnostics{
		CompatibilityRevision: engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
		Availability:          EngineDiagnosticAvailabilityUnavailable,
		ResultState:           EngineDiagnosticResultStateUnknown,
	}
}

// CloneEngineExecutionDiagnostics returns a detached bounded snapshot for
// retries and read projections.
func CloneEngineExecutionDiagnostics(source *EngineExecutionDiagnostics) *EngineExecutionDiagnostics {
	if source == nil {
		return nil
	}
	cloned := *source
	if len(source.ResultTypeWatermarks) > 0 {
		cloned.ResultTypeWatermarks = append([]ResultTypeWatermark(nil), source.ResultTypeWatermarks...)
	}
	return &cloned
}

// EqualEngineExecutionDiagnostics compares persisted terminal evidence for
// replay idempotency without broadening the public terminal result contract.
func EqualEngineExecutionDiagnostics(left, right *EngineExecutionDiagnostics) bool {
	if left == nil || right == nil {
		return left == right
	}
	if left.CompatibilityRevision != right.CompatibilityRevision || left.Availability != right.Availability || left.ResultState != right.ResultState ||
		left.FailedStage != right.FailedStage || left.ErrorType != right.ErrorType ||
		len(left.ResultTypeWatermarks) != len(right.ResultTypeWatermarks) {
		return false
	}
	for index := range left.ResultTypeWatermarks {
		if left.ResultTypeWatermarks[index] != right.ResultTypeWatermarks[index] {
			return false
		}
	}
	return true
}

// ValidateEngineExecutionDiagnostics validates the fixed Server projection at
// both the Agent-control boundary and persistence boundary. The Server never
// repairs or infers a malformed snapshot from logs or workspace state.
func ValidateEngineExecutionDiagnostics(diagnostics *EngineExecutionDiagnostics) error {
	if diagnostics == nil {
		return fmt.Errorf("Engine execution diagnostics are required")
	}
	if diagnostics.CompatibilityRevision != engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision {
		return fmt.Errorf("Engine diagnostics compatibility revision is invalid")
	}
	if !validDiagnosticAvailability(diagnostics.Availability) {
		return fmt.Errorf("Engine diagnostic availability is invalid")
	}
	if !validDiagnosticResultState(diagnostics.ResultState) {
		return fmt.Errorf("Engine diagnostic result state is invalid")
	}
	if diagnostics.Availability == EngineDiagnosticAvailabilityUnavailable {
		if diagnostics.ResultState != EngineDiagnosticResultStateUnknown || diagnostics.FailedStage != "" || diagnostics.ErrorType != "" || len(diagnostics.ResultTypeWatermarks) != 0 {
			return fmt.Errorf("unavailable Engine diagnostics must not contain Engine evidence")
		}
		return nil
	}
	if (diagnostics.FailedStage == "") != (diagnostics.ErrorType == "") {
		return fmt.Errorf("Engine diagnostic failure fields must be set together")
	}
	if diagnostics.FailedStage != "" && !validDiagnosticFailedStage(diagnostics.FailedStage) {
		return fmt.Errorf("Engine diagnostic failed stage is invalid")
	}
	if diagnostics.ErrorType != "" && !validDiagnosticErrorType(diagnostics.ErrorType) {
		return fmt.Errorf("Engine diagnostic error type is invalid")
	}
	if len(diagnostics.ResultTypeWatermarks) > EngineDiagnosticResultTypeLimit {
		return fmt.Errorf("Engine diagnostic result type limit exceeded")
	}
	seen := make(map[string]struct{}, len(diagnostics.ResultTypeWatermarks))
	for _, watermark := range diagnostics.ResultTypeWatermarks {
		if err := validateResultTypeWatermark(watermark); err != nil {
			return err
		}
		if _, duplicate := seen[watermark.ResultType]; duplicate {
			return fmt.Errorf("Engine diagnostic result types must be unique")
		}
		seen[watermark.ResultType] = struct{}{}
	}
	return nil
}

// ValidateTerminalEngineExecutionDiagnostics verifies that the Agent-owned
// snapshot agrees with the terminal task outcome before Server persists both
// facts in one fenced write. Server never repairs a disagreement from logs or
// recalculates Engine-side counters.
func ValidateTerminalEngineExecutionDiagnostics(status TaskStatus, diagnostics *EngineExecutionDiagnostics) error {
	if err := ValidateEngineExecutionDiagnostics(diagnostics); err != nil {
		return err
	}
	if diagnostics.Availability == EngineDiagnosticAvailabilityUnavailable {
		return nil
	}

	expectedResultState, err := diagnosticResultStateForTerminal(status, diagnostics.ResultTypeWatermarks)
	if err != nil {
		return err
	}
	if diagnostics.ResultState != expectedResultState {
		return fmt.Errorf("Engine diagnostic result state does not match terminal status")
	}

	switch status {
	case TaskStatusSucceeded:
		if diagnostics.FailedStage != "" || diagnostics.ErrorType != "" {
			return fmt.Errorf("successful Engine diagnostics must not contain a failure classification")
		}
	case TaskStatusFailed, TaskStatusCancelled:
		if diagnostics.FailedStage == "" || diagnostics.ErrorType == "" {
			return fmt.Errorf("non-successful Engine diagnostics require a failure classification")
		}
	default:
		return fmt.Errorf("Engine diagnostics require an Agent terminal task status")
	}
	return nil
}

func diagnosticResultStateForTerminal(status TaskStatus, watermarks []ResultTypeWatermark) (string, error) {
	acknowledged := false
	unresolved := false
	for _, watermark := range watermarks {
		if watermark.AcknowledgedItems > 0 || watermark.AcknowledgedBatches > 0 {
			acknowledged = true
		}
		if watermark.SubmittedBatches > watermark.AcknowledgedBatches {
			unresolved = true
		}
	}

	switch status {
	case TaskStatusSucceeded:
		if unresolved {
			return EngineDiagnosticResultStateUnknown, nil
		}
		return EngineDiagnosticResultStateComplete, nil
	case TaskStatusFailed, TaskStatusCancelled:
		if acknowledged {
			return EngineDiagnosticResultStatePartial, nil
		}
		if unresolved {
			return EngineDiagnosticResultStateUnknown, nil
		}
		return EngineDiagnosticResultStateNone, nil
	default:
		return "", fmt.Errorf("Engine diagnostics require an Agent terminal task status")
	}
}

func validDiagnosticAvailability(value string) bool {
	return value == EngineDiagnosticAvailabilityAvailable || value == EngineDiagnosticAvailabilityUnavailable
}

func validDiagnosticResultState(value string) bool {
	switch value {
	case EngineDiagnosticResultStateComplete, EngineDiagnosticResultStatePartial, EngineDiagnosticResultStateNone, EngineDiagnosticResultStateUnknown:
		return true
	default:
		return false
	}
}

func validDiagnosticFailedStage(value string) bool {
	switch value {
	case "unknown", "plan_validation", "input_materialization", "resource_materialization", "context_build", "image_prepare", "container_create", "container_start", "container_wait", "container_cleanup", "protocol_bootstrap", "execution_projection", "handler", "progress_report", "result_encode", "result_submit":
		return true
	default:
		return false
	}
}

func validDiagnosticErrorType(value string) bool {
	switch value {
	case "execution_failed", "plan_validation_failed", "input_materialization_failed", "resource_materialization_failed", "context_build_failed", "image_prepare_failed", "container_create_failed", "container_start_failed", "container_wait_failed", "container_cleanup_failed", "protocol_bootstrap_failed", "handler_failed", "progress_report_failed", "result_encode_failed", "result_submit_failed", "timeout", "cancelled", "oom_killed", "agent_session_failed", "invalid_outcome", "result_protocol_failed":
		return true
	default:
		return false
	}
}

func validateResultTypeWatermark(watermark ResultTypeWatermark) error {
	if err := validateDiagnosticResultType(watermark.ResultType); err != nil {
		return err
	}
	if watermark.ReceivedItems < watermark.EncodedItems ||
		watermark.EncodedItems < watermark.SubmittedItems ||
		watermark.SubmittedItems < watermark.AcknowledgedItems ||
		watermark.SubmittedBatches < watermark.AcknowledgedBatches {
		return fmt.Errorf("Engine diagnostic result watermarks are out of order")
	}
	return nil
}

func validateDiagnosticResultType(resultType string) error {
	if resultType == "" || resultType != strings.TrimSpace(resultType) || !utf8.ValidString(resultType) || len([]byte(resultType)) > 128 {
		return fmt.Errorf("Engine diagnostic result type is invalid")
	}
	segments := strings.Split(resultType, ".")
	if len(segments) < 2 {
		return fmt.Errorf("Engine diagnostic result type is invalid")
	}
	for _, segment := range segments {
		if segment == "" || segment[0] == '_' || segment[0] == '-' || segment[len(segment)-1] == '_' || segment[len(segment)-1] == '-' {
			return fmt.Errorf("Engine diagnostic result type is invalid")
		}
		for _, character := range segment {
			if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || character == '_' || character == '-' {
				continue
			}
			return fmt.Errorf("Engine diagnostic result type is invalid")
		}
	}
	return nil
}
