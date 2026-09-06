package protocol

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// DiagnosticResultTypeLimit is the maximum number of canonical result types
// retained by one execution. The bound prevents diagnostic memory from growing
// with an Engine's result volume or descriptor breadth.
const DiagnosticResultTypeLimit = int(EngineExecutionDiagnosticLimit_ENGINE_EXECUTION_DIAGNOSTIC_LIMIT_RESULT_TYPE_COUNT)

const (
	// EngineExecutionDiagnosticsCompatibilityRevision is the required hard-cut
	// value shared by the Engine Context, UDS diagnostics, Agent control plane,
	// Server projection, and frontend contract. It intentionally has no
	// negotiation or fallback path during disposable development.
	EngineExecutionDiagnosticsCompatibilityRevision = "engine-execution-diagnostics-r1"

	// DiagnosticErrorTypeProtoField is the canonical protobuf field name.
	DiagnosticErrorTypeProtoField = "error_type"
	// DiagnosticErrorTypeJSONField is the canonical HTTP JSON field name.
	DiagnosticErrorTypeJSONField = "errorType"
	// DiagnosticErrorTypeLogField is the canonical structured-log attribute.
	DiagnosticErrorTypeLogField = "error.type"
)

// ValidateEngineExecutionDiagnosticsEstablishment validates the mandatory
// matching-revision session confirmation before an Engine can use the task
// UDS. This is intentionally a fixed equality check, not capability probing.
func ValidateEngineExecutionDiagnosticsEstablishment(request *EstablishExecutionDiagnosticsRequest) error {
	if request == nil {
		return fmt.Errorf("Engine execution diagnostics establishment is required")
	}
	if hasDiagnosticUnknownFields(request.ProtoReflect()) {
		return fmt.Errorf("Engine execution diagnostics establishment contains unsupported fields")
	}
	if request.GetCompatibilityRevision() != EngineExecutionDiagnosticsCompatibilityRevision {
		return fmt.Errorf("Engine execution diagnostics establishment compatibility_revision is invalid")
	}
	return nil
}

// ValidateEngineTerminalDiagnosticSnapshot validates the bounded Engine-owned
// observation accepted by the side-channel RPC.
func ValidateEngineTerminalDiagnosticSnapshot(snapshot *EngineTerminalDiagnosticSnapshot) error {
	if snapshot == nil {
		return fmt.Errorf("Engine terminal diagnostic snapshot is required")
	}
	if hasDiagnosticUnknownFields(snapshot.ProtoReflect()) {
		return fmt.Errorf("Engine terminal diagnostic snapshot contains unsupported fields")
	}
	if snapshot.GetCompatibilityRevision() != EngineExecutionDiagnosticsCompatibilityRevision {
		return fmt.Errorf("Engine terminal diagnostic snapshot compatibility_revision is invalid")
	}
	if (snapshot.FailedStage == nil) != (snapshot.ErrorType == nil) {
		return fmt.Errorf("Engine terminal diagnostic failure fields must be set together")
	}
	if snapshot.FailedStage != nil {
		if !ValidEngineExecutionFailedStage(snapshot.GetFailedStage()) {
			return fmt.Errorf("Engine terminal diagnostic failed_stage is invalid")
		}
		if !ValidEngineExecutionErrorType(snapshot.GetErrorType()) {
			return fmt.Errorf("Engine terminal diagnostic error_type is invalid")
		}
	}
	return validateResultTypeWatermarks(snapshot.GetResultTypeWatermarks())
}

// ValidateEngineExecutionDiagnostics validates the Agent-owned terminal
// snapshot before it crosses the Agent-control or HTTP boundary.
func ValidateEngineExecutionDiagnostics(diagnostics *EngineExecutionDiagnostics) error {
	if diagnostics == nil {
		return fmt.Errorf("Engine execution diagnostics are required")
	}
	if hasDiagnosticUnknownFields(diagnostics.ProtoReflect()) {
		return fmt.Errorf("Engine execution diagnostics contain unsupported fields")
	}
	if diagnostics.GetCompatibilityRevision() != EngineExecutionDiagnosticsCompatibilityRevision {
		return fmt.Errorf("Engine execution diagnostics compatibility_revision is invalid")
	}
	if !ValidEngineExecutionDiagnosticAvailability(diagnostics.GetAvailability()) {
		return fmt.Errorf("Engine execution diagnostic availability is invalid")
	}
	if !ValidEngineExecutionResultState(diagnostics.GetResultState()) {
		return fmt.Errorf("Engine execution diagnostic result_state is invalid")
	}
	if diagnostics.GetAvailability() == EngineExecutionDiagnosticAvailability_ENGINE_EXECUTION_DIAGNOSTIC_AVAILABILITY_UNAVAILABLE {
		if diagnostics.FailedStage != nil || diagnostics.ErrorType != nil || len(diagnostics.GetResultTypeWatermarks()) != 0 || diagnostics.GetResultState() != EngineExecutionResultState_ENGINE_EXECUTION_RESULT_STATE_UNKNOWN {
			return fmt.Errorf("unavailable Engine diagnostics must not contain Engine evidence")
		}
		return nil
	}
	if (diagnostics.FailedStage == nil) != (diagnostics.ErrorType == nil) {
		return fmt.Errorf("Engine execution diagnostic failure fields must be set together")
	}
	if diagnostics.FailedStage != nil {
		if !ValidEngineExecutionFailedStage(diagnostics.GetFailedStage()) {
			return fmt.Errorf("Engine execution diagnostic failed_stage is invalid")
		}
		if !ValidEngineExecutionErrorType(diagnostics.GetErrorType()) {
			return fmt.Errorf("Engine execution diagnostic error_type is invalid")
		}
	}
	return validateResultTypeWatermarks(diagnostics.GetResultTypeWatermarks())
}

// ValidEngineExecutionFailedStage reports whether value belongs to the fixed
// execution-boundary stage vocabulary.
func ValidEngineExecutionFailedStage(value EngineExecutionFailedStage) bool {
	switch value {
	case EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_UNKNOWN,
		EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_PLAN_VALIDATION,
		EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_INPUT_MATERIALIZATION,
		EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_RESOURCE_MATERIALIZATION,
		EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_CONTEXT_BUILD,
		EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_IMAGE_PREPARE,
		EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_CONTAINER_CREATE,
		EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_CONTAINER_START,
		EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_CONTAINER_WAIT,
		EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_CONTAINER_CLEANUP,
		EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_PROTOCOL_BOOTSTRAP,
		EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_EXECUTION_PROJECTION,
		EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_HANDLER,
		EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_PROGRESS_REPORT,
		EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_RESULT_ENCODE,
		EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_RESULT_SUBMIT:
		return true
	default:
		return false
	}
}

// ValidEngineExecutionErrorType reports whether value belongs to the stable,
// low-cardinality diagnostic reason catalog.
func ValidEngineExecutionErrorType(value EngineExecutionErrorType) bool {
	switch value {
	case EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_EXECUTION_FAILED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_PLAN_VALIDATION_FAILED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_INPUT_MATERIALIZATION_FAILED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_RESOURCE_MATERIALIZATION_FAILED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_CONTEXT_BUILD_FAILED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_IMAGE_PREPARE_FAILED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_CONTAINER_CREATE_FAILED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_CONTAINER_START_FAILED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_CONTAINER_WAIT_FAILED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_CONTAINER_CLEANUP_FAILED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_PROTOCOL_BOOTSTRAP_FAILED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_HANDLER_FAILED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_PROGRESS_REPORT_FAILED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_RESULT_ENCODE_FAILED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_RESULT_SUBMIT_FAILED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_TIMEOUT,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_CANCELLED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_OOM_KILLED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_AGENT_SESSION_FAILED,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_INVALID_OUTCOME,
		EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_RESULT_PROTOCOL_FAILED:
		return true
	default:
		return false
	}
}

// ValidEngineExecutionDiagnosticAvailability reports whether value is a
// concrete availability state rather than its protobuf zero value.
func ValidEngineExecutionDiagnosticAvailability(value EngineExecutionDiagnosticAvailability) bool {
	return value == EngineExecutionDiagnosticAvailability_ENGINE_EXECUTION_DIAGNOSTIC_AVAILABILITY_AVAILABLE ||
		value == EngineExecutionDiagnosticAvailability_ENGINE_EXECUTION_DIAGNOSTIC_AVAILABILITY_UNAVAILABLE
}

// ValidEngineExecutionResultState reports whether value is a concrete
// result-delivery evidence state rather than its protobuf zero value.
func ValidEngineExecutionResultState(value EngineExecutionResultState) bool {
	switch value {
	case EngineExecutionResultState_ENGINE_EXECUTION_RESULT_STATE_COMPLETE,
		EngineExecutionResultState_ENGINE_EXECUTION_RESULT_STATE_PARTIAL,
		EngineExecutionResultState_ENGINE_EXECUTION_RESULT_STATE_NONE,
		EngineExecutionResultState_ENGINE_EXECUTION_RESULT_STATE_UNKNOWN:
		return true
	default:
		return false
	}
}

func validateResultTypeWatermarks(watermarks []*ResultTypeWatermark) error {
	if len(watermarks) > DiagnosticResultTypeLimit {
		return fmt.Errorf("Engine diagnostic result type limit exceeded")
	}
	seen := make(map[string]struct{}, len(watermarks))
	for _, watermark := range watermarks {
		if watermark == nil || hasDiagnosticUnknownFields(watermark.ProtoReflect()) {
			return fmt.Errorf("Engine diagnostic result watermark is invalid")
		}
		if err := ValidateResultTypeSyntax(watermark.GetResultType()); err != nil {
			return fmt.Errorf("Engine diagnostic result type is invalid")
		}
		if _, duplicate := seen[watermark.GetResultType()]; duplicate {
			return fmt.Errorf("Engine diagnostic result types must be unique")
		}
		if watermark.GetReceivedItems() < watermark.GetEncodedItems() ||
			watermark.GetEncodedItems() < watermark.GetSubmittedItems() ||
			watermark.GetSubmittedItems() < watermark.GetAcknowledgedItems() ||
			watermark.GetSubmittedBatches() < watermark.GetAcknowledgedBatches() {
			return fmt.Errorf("Engine diagnostic result watermarks are out of order")
		}
		seen[watermark.GetResultType()] = struct{}{}
	}
	return nil
}

func hasDiagnosticUnknownFields(message protoreflect.Message) bool {
	if !message.IsValid() || len(message.GetUnknown()) != 0 {
		return true
	}
	var unknown bool
	message.Range(func(descriptor protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if descriptor.IsList() {
			list := value.List()
			for index := 0; index < list.Len(); index++ {
				if descriptor.Kind() == protoreflect.MessageKind && hasDiagnosticUnknownFields(list.Get(index).Message()) {
					unknown = true
					return false
				}
			}
			return true
		}
		if descriptor.Kind() == protoreflect.MessageKind && hasDiagnosticUnknownFields(value.Message()) {
			unknown = true
			return false
		}
		return true
	})
	return unknown
}
