package protocol

import (
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func TestValidateEngineExecutionDiagnosticsEstablishment(t *testing.T) {
	if err := ValidateEngineExecutionDiagnosticsEstablishment(&EstablishExecutionDiagnosticsRequest{CompatibilityRevision: EngineExecutionDiagnosticsCompatibilityRevision}); err != nil {
		t.Fatalf("ValidateEngineExecutionDiagnosticsEstablishment() rejected matching revision: %v", err)
	}
	for _, request := range []*EstablishExecutionDiagnosticsRequest{
		nil,
		{},
		{CompatibilityRevision: "pre-cut"},
		func() *EstablishExecutionDiagnosticsRequest {
			request := &EstablishExecutionDiagnosticsRequest{CompatibilityRevision: EngineExecutionDiagnosticsCompatibilityRevision}
			request.ProtoReflect().SetUnknown(protowire.AppendTag(nil, 99, protowire.BytesType))
			return request
		}(),
	} {
		if err := ValidateEngineExecutionDiagnosticsEstablishment(request); err == nil {
			t.Fatalf("ValidateEngineExecutionDiagnosticsEstablishment() accepted malformed request %#v", request)
		}
	}
}

func TestValidateEngineTerminalDiagnosticSnapshot(t *testing.T) {
	if err := ValidateEngineTerminalDiagnosticSnapshot(validEngineTerminalDiagnosticSnapshot()); err != nil {
		t.Fatalf("ValidateEngineTerminalDiagnosticSnapshot() rejected valid snapshot: %v", err)
	}
	if err := ValidateEngineTerminalDiagnosticSnapshot(&EngineTerminalDiagnosticSnapshot{CompatibilityRevision: EngineExecutionDiagnosticsCompatibilityRevision}); err != nil {
		t.Fatalf("ValidateEngineTerminalDiagnosticSnapshot() rejected legal zero-result success: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*EngineTerminalDiagnosticSnapshot)
	}{
		{
			name: "custom stage",
			mutate: func(snapshot *EngineTerminalDiagnosticSnapshot) {
				stage := EngineExecutionFailedStage(99)
				snapshot.FailedStage = &stage
			},
		},
		{
			name: "custom error type",
			mutate: func(snapshot *EngineTerminalDiagnosticSnapshot) {
				errorType := EngineExecutionErrorType(99)
				snapshot.ErrorType = &errorType
			},
		},
		{
			name: "incomplete failure fields",
			mutate: func(snapshot *EngineTerminalDiagnosticSnapshot) {
				snapshot.ErrorType = nil
			},
		},
		{
			name: "out of order item watermarks",
			mutate: func(snapshot *EngineTerminalDiagnosticSnapshot) {
				snapshot.ResultTypeWatermarks[0].EncodedItems = snapshot.ResultTypeWatermarks[0].ReceivedItems + 1
			},
		},
		{
			name: "out of order batch watermarks",
			mutate: func(snapshot *EngineTerminalDiagnosticSnapshot) {
				snapshot.ResultTypeWatermarks[0].AcknowledgedBatches = snapshot.ResultTypeWatermarks[0].SubmittedBatches + 1
			},
		},
		{
			name: "duplicate result type",
			mutate: func(snapshot *EngineTerminalDiagnosticSnapshot) {
				snapshot.ResultTypeWatermarks = append(snapshot.ResultTypeWatermarks, cloneWatermark(snapshot.ResultTypeWatermarks[0]))
			},
		},
		{
			name: "over result type limit",
			mutate: func(snapshot *EngineTerminalDiagnosticSnapshot) {
				snapshot.ResultTypeWatermarks = nil
				for index := 0; index <= DiagnosticResultTypeLimit; index++ {
					snapshot.ResultTypeWatermarks = append(snapshot.ResultTypeWatermarks, &ResultTypeWatermark{ResultType: "asset.type" + string(rune('a'+index)) + ".v1"})
				}
			},
		},
		{
			name: "unknown raw diagnostic field",
			mutate: func(snapshot *EngineTerminalDiagnosticSnapshot) {
				snapshot.ProtoReflect().SetUnknown(protowire.AppendTag(nil, 99, protowire.BytesType))
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validEngineTerminalDiagnosticSnapshot()
			test.mutate(snapshot)
			if err := ValidateEngineTerminalDiagnosticSnapshot(snapshot); err == nil {
				t.Fatal("ValidateEngineTerminalDiagnosticSnapshot() accepted malformed snapshot")
			}
		})
	}
}

func TestValidateEngineExecutionDiagnostics(t *testing.T) {
	available := &EngineExecutionDiagnostics{
		CompatibilityRevision: EngineExecutionDiagnosticsCompatibilityRevision,
		Availability:          EngineExecutionDiagnosticAvailability_ENGINE_EXECUTION_DIAGNOSTIC_AVAILABILITY_AVAILABLE,
		ResultState:           EngineExecutionResultState_ENGINE_EXECUTION_RESULT_STATE_COMPLETE,
		ResultTypeWatermarks:  []*ResultTypeWatermark{{ResultType: "asset.subdomain.v1"}},
	}
	if err := ValidateEngineExecutionDiagnostics(available); err != nil {
		t.Fatalf("ValidateEngineExecutionDiagnostics() rejected available snapshot: %v", err)
	}
	unavailable := &EngineExecutionDiagnostics{
		CompatibilityRevision: EngineExecutionDiagnosticsCompatibilityRevision,
		Availability:          EngineExecutionDiagnosticAvailability_ENGINE_EXECUTION_DIAGNOSTIC_AVAILABILITY_UNAVAILABLE,
		ResultState:           EngineExecutionResultState_ENGINE_EXECUTION_RESULT_STATE_UNKNOWN,
	}
	if err := ValidateEngineExecutionDiagnostics(unavailable); err != nil {
		t.Fatalf("ValidateEngineExecutionDiagnostics() rejected unavailable snapshot: %v", err)
	}

	for _, diagnostics := range []*EngineExecutionDiagnostics{
		{},
		{Availability: EngineExecutionDiagnosticAvailability_ENGINE_EXECUTION_DIAGNOSTIC_AVAILABILITY_UNAVAILABLE, ResultState: EngineExecutionResultState_ENGINE_EXECUTION_RESULT_STATE_COMPLETE},
		{Availability: EngineExecutionDiagnosticAvailability_ENGINE_EXECUTION_DIAGNOSTIC_AVAILABILITY_UNAVAILABLE, ResultState: EngineExecutionResultState_ENGINE_EXECUTION_RESULT_STATE_UNKNOWN, ResultTypeWatermarks: []*ResultTypeWatermark{{ResultType: "asset.subdomain.v1"}}},
	} {
		if err := ValidateEngineExecutionDiagnostics(diagnostics); err == nil {
			t.Fatalf("ValidateEngineExecutionDiagnostics() accepted malformed snapshot %#v", diagnostics)
		}
	}
}

func validEngineTerminalDiagnosticSnapshot() *EngineTerminalDiagnosticSnapshot {
	stage := EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_RESULT_SUBMIT
	errorType := EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_RESULT_SUBMIT_FAILED
	return &EngineTerminalDiagnosticSnapshot{
		CompatibilityRevision: EngineExecutionDiagnosticsCompatibilityRevision,
		FailedStage:           &stage,
		ErrorType:             &errorType,
		ResultTypeWatermarks: []*ResultTypeWatermark{{
			ResultType:          "asset.subdomain.v1",
			ReceivedItems:       4,
			EncodedItems:        3,
			SubmittedItems:      2,
			AcknowledgedItems:   1,
			SubmittedBatches:    2,
			AcknowledgedBatches: 1,
		}},
	}
}

func cloneWatermark(source *ResultTypeWatermark) *ResultTypeWatermark {
	if source == nil {
		return nil
	}
	copy := *source
	return &copy
}
