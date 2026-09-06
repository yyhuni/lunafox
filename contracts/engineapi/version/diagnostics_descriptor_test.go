package version

import (
	"testing"

	agentcontrolpb "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestEngineDiagnosticsServiceIsIsolatedFromStatusOnlyReporting(t *testing.T) {
	diagnosticsFile := engineexecutionpb.File_lunafox_engine_execution_v2_engine_execution_diagnostics_proto
	diagnostics := diagnosticsFile.Services().ByName("EngineExecutionDiagnosticsService")
	if diagnostics == nil || diagnostics.Methods().Len() != 2 || diagnostics.Methods().Get(0).Name() != "EstablishExecutionDiagnostics" || diagnostics.Methods().Get(1).Name() != "ReportTerminalDiagnostics" {
		t.Fatalf("diagnostics service descriptor = %v, want exactly the fixed establishment and terminal methods", diagnostics)
	}
	establishment := diagnostics.Methods().Get(0).Input()
	if establishment.Fields().Len() != 1 || !hasDescriptorField(establishment, "compatibility_revision", "compatibilityRevision", 1, protoreflect.StringKind) {
		t.Fatalf("diagnostic establishment request fields = %v, want exactly compatibility_revision", establishment.Fields())
	}
	for _, forbidden := range []string{"task", "scan", "execution", "agent", "session", "credential"} {
		if establishment.Fields().ByName(protoreflect.Name(forbidden)) != nil || !establishment.ReservedNames().Has(protoreflect.Name(forbidden)) {
			t.Fatalf("diagnostic establishment request must reserve and omit control-plane identity %q", forbidden)
		}
	}
	if diagnostics.Methods().Get(0).Output().Fields().Len() != 0 {
		t.Fatal("diagnostic establishment response must retain its empty fixed shape")
	}
	request := diagnostics.Methods().Get(1).Input()
	if request.Fields().Len() != 1 || !hasDescriptorField(request, "snapshot", "snapshot", 1, protoreflect.MessageKind) {
		t.Fatalf("diagnostic request fields = %v, want exactly snapshot", request.Fields())
	}
	for _, forbidden := range []string{"task", "scan", "execution", "agent", "session", "credential"} {
		if request.Fields().ByName(protoreflect.Name(forbidden)) != nil || !request.ReservedNames().Has(protoreflect.Name(forbidden)) {
			t.Fatalf("diagnostic request must reserve and omit control-plane identity %q", forbidden)
		}
	}
	if diagnostics.Methods().Get(1).Output().Fields().Len() != 0 {
		t.Fatal("diagnostics response must retain its status-only empty shape")
	}

	reporting := engineexecutionpb.File_lunafox_engine_execution_v2_engine_execution_reporting_proto.Services().ByName("EngineExecutionReportingService")
	if reporting == nil || reporting.Methods().Len() != 2 {
		t.Fatal("reporting service must retain exactly its two status-only methods")
	}
	for index := 0; index < reporting.Methods().Len(); index++ {
		method := reporting.Methods().Get(index)
		if method.Name() != "ReportProgress" && method.Name() != "SubmitResultBatch" {
			t.Fatalf("reporting service gained non-reporting method %q", method.Name())
		}
	}
}

func TestEngineDiagnosticsDescriptorClosesFieldsAndCanonicalNames(t *testing.T) {
	file := engineexecutionpb.File_lunafox_engine_execution_v2_engine_execution_diagnostics_proto
	snapshot := file.Messages().ByName("EngineTerminalDiagnosticSnapshot")
	if snapshot == nil || snapshot.Fields().Len() != 4 {
		t.Fatal("EngineTerminalDiagnosticSnapshot must have only bounded diagnostic fields")
	}
	if !hasDescriptorField(snapshot, "failed_stage", "failedStage", 1, protoreflect.EnumKind) ||
		!hasDescriptorField(snapshot, "error_type", engineexecutionpb.DiagnosticErrorTypeJSONField, 2, protoreflect.EnumKind) ||
		!hasDescriptorField(snapshot, "result_type_watermarks", "resultTypeWatermarks", 3, protoreflect.MessageKind) ||
		!hasDescriptorField(snapshot, "compatibility_revision", "compatibilityRevision", 4, protoreflect.StringKind) {
		t.Fatalf("Engine terminal diagnostic field naming drifted: %v", snapshot.Fields())
	}
	if got := engineexecutionpb.DiagnosticErrorTypeProtoField; got != "error_type" {
		t.Fatalf("error type proto field = %q", got)
	}
	if got := engineexecutionpb.DiagnosticErrorTypeLogField; got != "error.type" {
		t.Fatalf("error type log field = %q", got)
	}
	for _, forbidden := range []string{"error", "message", "display_message", "tool_output", "stdout", "stderr", "path", "credential"} {
		if snapshot.Fields().ByName(protoreflect.Name(forbidden)) != nil || !snapshot.ReservedNames().Has(protoreflect.Name(forbidden)) {
			t.Fatalf("terminal snapshot must reserve and omit unsafe field %q", forbidden)
		}
	}

	watermark := file.Messages().ByName("ResultTypeWatermark")
	if watermark == nil || watermark.Fields().Len() != 7 {
		t.Fatal("ResultTypeWatermark must retain exactly seven fixed counters")
	}
	if limit := file.Enums().ByName("EngineExecutionDiagnosticLimit").Values().ByName("ENGINE_EXECUTION_DIAGNOSTIC_LIMIT_RESULT_TYPE_COUNT"); limit == nil || limit.Number() != protoreflect.EnumNumber(engineexecutionpb.DiagnosticResultTypeLimit) {
		t.Fatalf("diagnostic result type limit drifted: %v", limit)
	}

	terminal := agentcontrolpb.File_lunafox_agent_control_v1_agent_control_proto.Messages().ByName("TerminalTaskResult")
	if terminal == nil || !hasDescriptorField(terminal, "diagnostics", "diagnostics", 5, protoreflect.MessageKind) ||
		!hasDescriptorField(terminal, "compatibility_revision", "compatibilityRevision", 6, protoreflect.StringKind) {
		t.Fatal("Agent terminal result must carry final diagnostics and matching revision")
	}
}

func TestEngineDiagnosticsEnumClosure(t *testing.T) {
	if !engineexecutionpb.ValidEngineExecutionFailedStage(engineexecutionpb.EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_UNKNOWN) ||
		engineexecutionpb.ValidEngineExecutionFailedStage(engineexecutionpb.EngineExecutionFailedStage(99)) {
		t.Fatal("failed_stage closure drifted")
	}
	if !engineexecutionpb.ValidEngineExecutionErrorType(engineexecutionpb.EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_EXECUTION_FAILED) ||
		engineexecutionpb.ValidEngineExecutionErrorType(engineexecutionpb.EngineExecutionErrorType(99)) {
		t.Fatal("error_type closure drifted")
	}
}

func hasDescriptorField(message protoreflect.MessageDescriptor, name, jsonName string, number protoreflect.FieldNumber, kind protoreflect.Kind) bool {
	field := message.Fields().ByName(protoreflect.Name(name))
	return field != nil && field.JSONName() == jsonName && field.Number() == number && field.Kind() == kind
}
