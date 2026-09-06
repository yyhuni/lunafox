package version

import (
	"testing"

	agentexecutionpb "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestEngineInputServiceIsIndependentFromReporting(t *testing.T) {
	inputFile := engineexecutionpb.File_lunafox_engine_execution_v2_engine_execution_input_proto
	service := inputFile.Services().ByName("EngineExecutionInputService")
	if service == nil {
		t.Fatal("EngineExecutionInputService descriptor is missing")
	}
	if service.Methods().Len() != 1 || service.Methods().Get(0).Name() != "MaterializeExecutionInput" {
		t.Fatalf("input service methods = %v, want one MaterializeExecutionInput method", service.Methods())
	}
	request := service.Methods().Get(0).Input()
	if request.Fields().Len() != 1 || request.Fields().Get(0).Name() != "role" || request.Fields().Get(0).Number() != 1 || request.Fields().Get(0).Kind() != protoreflect.StringKind {
		t.Fatalf("input request fields do not have the closed role shape")
	}
	response := service.Methods().Get(0).Output()
	if response.Fields().Len() != 1 || response.Fields().Get(0).Name() != "path" || response.Fields().Get(0).Number() != 1 || response.Fields().Get(0).Kind() != protoreflect.StringKind {
		t.Fatalf("input response fields do not have the private path shape")
	}

	reporting := engineexecutionpb.File_lunafox_engine_execution_v2_engine_execution_reporting_proto.Services().ByName("EngineExecutionReportingService")
	if reporting == nil || reporting.Methods().Len() != 2 {
		t.Fatalf("reporting service must retain exactly its two reporting methods")
	}
	for i := 0; i < reporting.Methods().Len(); i++ {
		name := reporting.Methods().Get(i).Name()
		if name != "ReportProgress" && name != "SubmitResultBatch" {
			t.Fatalf("reporting service gained non-reporting method %q", name)
		}
	}
}

func TestRemovedInputBindingsAreReservedAndAbsent(t *testing.T) {
	context := engineexecutionpb.File_lunafox_engine_execution_v2_engine_execution_context_proto
	message := context.Messages().ByName("EngineExecutionContext")
	if message == nil {
		t.Fatal("EngineExecutionContext descriptor is missing")
	}
	if message.Fields().ByName("inputs") != nil || !message.ReservedNames().Has("inputs") {
		t.Fatal("Context.inputs must remain absent and reserved")
	}
	revision := message.Fields().ByName("compatibility_revision")
	if revision == nil || revision.Number() != 2 || revision.Kind() != protoreflect.StringKind {
		t.Fatal("Context.compatibility_revision must occupy hard-cut field 2")
	}
	if context.Messages().ByName("EngineExecutionInputs") != nil || context.Messages().ByName("EngineInputBinding") != nil {
		t.Fatal("retired Context input messages must not remain in the descriptor")
	}

	plan := agentexecutionpb.File_lunafox_agent_execution_v1_resolved_engine_execution_plan_proto
	planMessage := plan.Messages().ByName("ResolvedEngineExecutionPlan")
	if planMessage == nil {
		t.Fatal("ResolvedEngineExecutionPlan descriptor is missing")
	}
	if planMessage.Fields().ByName("inputs") != nil || !planMessage.ReservedRanges().Has(8) || !planMessage.ReservedNames().Has("inputs") {
		t.Fatal("saved-plan inputs must be absent and reserve field 8/name inputs")
	}
	if plan.Messages().ByName("ResolvedExecutionInputs") != nil || plan.Messages().ByName("ExecutionInputBinding") != nil {
		t.Fatal("retired saved-plan input messages must not remain in the descriptor")
	}
}
