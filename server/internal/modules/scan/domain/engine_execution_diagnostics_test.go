package domain

import (
	"testing"

	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
)

func TestValidateTerminalEngineExecutionDiagnosticsRequiresOutcomeConsistentEvidence(t *testing.T) {
	failedStage := "result_submit"
	errorType := "result_submit_failed"
	tests := []struct {
		name        string
		status      TaskStatus
		diagnostics *EngineExecutionDiagnostics
		wantErr     bool
	}{
		{
			name:   "successful zero result delivery is complete",
			status: TaskStatusSucceeded,
			diagnostics: &EngineExecutionDiagnostics{
				CompatibilityRevision: engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
				Availability:          EngineDiagnosticAvailabilityAvailable,
				ResultState:           EngineDiagnosticResultStateComplete,
			},
		},
		{
			name:   "failed acknowledged result delivery is partial",
			status: TaskStatusFailed,
			diagnostics: &EngineExecutionDiagnostics{
				CompatibilityRevision: engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
				Availability:          EngineDiagnosticAvailabilityAvailable,
				ResultState:           EngineDiagnosticResultStatePartial,
				FailedStage:           failedStage,
				ErrorType:             errorType,
				ResultTypeWatermarks: []ResultTypeWatermark{{
					ResultType:    "network.port",
					ReceivedItems: 1, EncodedItems: 1, SubmittedItems: 1, AcknowledgedItems: 1,
					SubmittedBatches: 1, AcknowledgedBatches: 1,
				}},
			},
		},
		{
			name:   "failed unresolved delivery is unknown",
			status: TaskStatusFailed,
			diagnostics: &EngineExecutionDiagnostics{
				CompatibilityRevision: engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
				Availability:          EngineDiagnosticAvailabilityAvailable,
				ResultState:           EngineDiagnosticResultStateUnknown,
				FailedStage:           failedStage,
				ErrorType:             errorType,
				ResultTypeWatermarks: []ResultTypeWatermark{{
					ResultType:    "network.port",
					ReceivedItems: 1, EncodedItems: 1, SubmittedItems: 1, SubmittedBatches: 1,
				}},
			},
		},
		{
			name:   "unavailable remains explicit evidence loss",
			status: TaskStatusSucceeded,
			diagnostics: &EngineExecutionDiagnostics{
				CompatibilityRevision: engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
				Availability:          EngineDiagnosticAvailabilityUnavailable,
				ResultState:           EngineDiagnosticResultStateUnknown,
			},
		},
		{
			name:   "successful task cannot carry failure classification",
			status: TaskStatusSucceeded,
			diagnostics: &EngineExecutionDiagnostics{
				CompatibilityRevision: engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
				Availability:          EngineDiagnosticAvailabilityAvailable,
				ResultState:           EngineDiagnosticResultStateComplete,
				FailedStage:           failedStage,
				ErrorType:             errorType,
			},
			wantErr: true,
		},
		{
			name:   "failed task cannot claim complete delivery",
			status: TaskStatusFailed,
			diagnostics: &EngineExecutionDiagnostics{
				CompatibilityRevision: engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
				Availability:          EngineDiagnosticAvailabilityAvailable,
				ResultState:           EngineDiagnosticResultStateComplete,
				FailedStage:           failedStage,
				ErrorType:             errorType,
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateTerminalEngineExecutionDiagnostics(test.status, test.diagnostics)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateTerminalEngineExecutionDiagnostics() error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}
}

func TestValidateTerminalEngineExecutionDiagnosticsAcceptsResultProtocolFailure(t *testing.T) {
	diagnostics := &EngineExecutionDiagnostics{
		CompatibilityRevision: engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
		Availability:          EngineDiagnosticAvailabilityAvailable,
		ResultState:           EngineDiagnosticResultStateNone,
		FailedStage:           "handler",
		ErrorType:             "result_protocol_failed",
		ResultTypeWatermarks:  []ResultTypeWatermark{{ResultType: "nuclei.vulnerability"}},
	}
	if err := ValidateTerminalEngineExecutionDiagnostics(TaskStatusFailed, diagnostics); err != nil {
		t.Fatalf("ValidateTerminalEngineExecutionDiagnostics() rejected result protocol failure: %v", err)
	}
}
