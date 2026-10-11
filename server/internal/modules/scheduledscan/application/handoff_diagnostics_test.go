package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func diagnosticFieldMap(t *testing.T, fields []zap.Field) (map[string]any, string) {
	t.Helper()
	core, logs := observer.New(zap.DebugLevel)
	zap.New(core).Error("diagnostic", fields...)
	result := logs.All()[0].ContextMap()
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	return result, string(encoded)
}

func TestHandoffDiagnosticErrorChainPreservesTypedPrivateCause(t *testing.T) {
	databaseErr := &pgconn.PgError{Code: "42P01", Message: `relation "wordlist" does not exist`, Detail: "private SQL detail"}
	resourceErr := scanapp.NewConfigResourceInternalError("configuration.steps.dns.engineConfig.resolvers", "wordlist", "wordlists/2", databaseErr)
	err := fmt.Errorf("create scan: %w", &scanapp.PlanTaskError{Kind: scanapp.PlanTaskResourceError, Err: resourceErr})
	fields, encoded := diagnosticFieldMap(t, diagnosticErrorFields("error", err))
	if fields["error.plan_task.kind"] != string(scanapp.PlanTaskResourceError) || fields["error.resource.cause"] != "internal" || fields["error.resource.kind"] != "wordlist" {
		t.Fatalf("typed context = %s", encoded)
	}
	if !strings.Contains(encoded, "42P01") || !strings.Contains(encoded, "does not exist") || !strings.Contains(encoded, "pgconn.PgError") {
		t.Fatalf("private cause lost behind safe Error(): %s", encoded)
	}
	if strings.Contains(encoded, databaseErr.Detail) {
		t.Fatalf("SQL detail leaked: %s", encoded)
	}
	outcome := ClassifyHandoff(nil, err, true)
	if outcome.Cause != HandoffCauseInternalUnavailable || strings.Contains(outcome.Message, "wordlist") || strings.Contains(outcome.Message, "42P01") {
		t.Fatalf("public outcome changed: %+v", outcome)
	}
}

func TestHandoffDiagnosticRedactsSensitiveAndConfigErrors(t *testing.T) {
	for _, message := range []string{
		"connect failed password=example-value", "TOKEN: example-value", "Authorization: Bearer example-value",
		"api_key=example-value", "private-key example-value", "Cookie: session=example-value",
		"connection postgres://user:example-value@localhost/db", "GET https://host.invalid/path?value=example-value",
		"credentials rejected: example-value", "secret=example-value",
	} {
		t.Run(message, func(t *testing.T) {
			_, encoded := diagnosticFieldMap(t, diagnosticErrorFields("error", fmt.Errorf("handoff: %w", errors.New(message))))
			if strings.Contains(encoded, "example-value") || !strings.Contains(encoded, "REDACTED") {
				t.Fatalf("sensitive error leaked: %s", encoded)
			}
		})
	}
	// The value has no credential marker; a parent wrapper must still omit it
	// when the underlying typed error is a configuration-input rejection.
	err := fmt.Errorf("create: %w", &scanapp.PlanTaskError{Kind: scanapp.PlanTaskConfigError, Err: errors.New("bad value opaque-user-value")})
	fields, encoded := diagnosticFieldMap(t, diagnosticErrorFields("error", err))
	if strings.Contains(encoded, "opaque-user-value") || fields["error.plan_task.kind"] != string(scanapp.PlanTaskConfigError) {
		t.Fatalf("config value leaked or type lost: %s", encoded)
	}
	inputErr := fmt.Errorf("selection: %w", &scanapp.ConfigResourceInputError{Field: "wordlist", Err: errors.New("opaque-user-value")})
	_, encoded = diagnosticFieldMap(t, diagnosticErrorFields("error", inputErr))
	if strings.Contains(encoded, "opaque-user-value") {
		t.Fatalf("resource input leaked: %s", encoded)
	}
}

func TestHandoffDiagnosticBoundsChainsAndSupportsJoinedErrors(t *testing.T) {
	err := errors.Join(errors.New("dependency failure"), fmt.Errorf("request: %w", context.DeadlineExceeded))
	fields, encoded := diagnosticFieldMap(t, diagnosticErrorFields("error", err))
	if fields["error.kind"] != "deadline_exceeded" || !strings.Contains(encoded, "dependency failure") || !strings.Contains(encoded, "context deadline exceeded") {
		t.Fatalf("joined causes lost: %s", encoded)
	}
	for range maxDiagnosticErrorNodes {
		err = fmt.Errorf("wrapper: %w", err)
	}
	fields, encoded = diagnosticFieldMap(t, diagnosticErrorFields("error", err))
	nodes := fields["error.chain"].([]diagnosticErrorNode)
	if len(nodes) != maxDiagnosticErrorNodes || fields["error.chain_truncated"] != true || strings.Contains(encoded, "dependency failure") {
		t.Fatalf("chain limit or incomplete-chain redaction failed: %s", encoded)
	}
	if fields, _ := diagnosticFieldMap(t, diagnosticErrorFields("error", nil)); len(fields) != 0 {
		t.Fatalf("nil error diagnostics = %+v", fields)
	}
}

func TestHandoffDiagnosticTextIsBoundedValidAndSingleLine(t *testing.T) {
	text := diagnosticText(strings.Repeat("诊断", 1000) + "\n\x00\xff")
	if len(text) > maxDiagnosticTextBytes || !utf8.ValidString(text) || !strings.HasSuffix(text, " [truncated]") {
		t.Fatalf("invalid diagnostic text length=%d valid=%t", len(text), utf8.ValidString(text))
	}
	if got := diagnosticText("operation\nfailed\x00\xff"); got != "operation failed " {
		t.Fatalf("control/UTF-8 normalization = %q", got)
	}
	if got := diagnosticText(strings.Repeat("a", 2048) + " password=hidden"); strings.Contains(got, "aaaa") {
		t.Fatalf("redaction must precede truncation: %q", got)
	}
}

func TestHandoffDiagnosticsBoundsBatchAndExcludesConfiguration(t *testing.T) {
	agentID := 12
	input := FrozenDispatchInput{ScanWorkflowID: "default", AgentID: &agentID, TargetIDs: []int{1, 2, 3, 4, 5, 6, 7}, Configuration: map[string]any{"value": "private-config-value"}}
	result := &scanapp.BatchScanResult{CreatedCount: 2, Scans: []scanapp.QueryScan{{ID: 20}, {ID: 21}}}
	for index := range 7 {
		result.Failed = append(result.Failed, scanapp.CreateBatchItemOutcome{Index: index, TargetID: index + 1, Reason: "TARGET_NOT_FOUND", Message: "lookup failed"})
		result.Skipped = append(result.Skipped, scanapp.CreateBatchItemOutcome{Index: index, TargetID: index + 1, Reason: "DUPLICATE_TARGET", Message: "token=hidden-value"})
	}
	fields, encoded := diagnosticFieldMap(t, handoffDiagnosticFields(input, result, nil))
	if fmt.Sprint(fields["scan.batch.failed_count"]) != "7" || fmt.Sprint(fields["scan.batch.created_count"]) != "2" || fields["scan.batch.failed_sample_truncated"] != true || fields["scheduled_scan.target_sample_truncated"] != true {
		t.Fatalf("batch context = %s", encoded)
	}
	if len(fields["scan.batch.failed_sample"].([]diagnosticBatchItem)) != maxDiagnosticSamples || len(fields["scan.batch.skipped_sample"].([]diagnosticBatchItem)) != maxDiagnosticSamples {
		t.Fatalf("unbounded samples: %s", encoded)
	}
	if strings.Contains(encoded, "private-config-value") || strings.Contains(encoded, "hidden-value") || strings.Contains(encoded, "error.chain") {
		t.Fatalf("payload/error leakage: %s", encoded)
	}
}

func TestSchedulerDiagnosticsSurviveOutcomeWriteback(t *testing.T) {
	for _, test := range []struct {
		name     string
		batch    *scanapp.BatchScanResult
		dispatch error
		record   error
		recorded bool
		message  string
	}{
		{name: "failed creation and writeback", dispatch: errors.New("package cache unavailable"), record: errors.New("database unavailable"), message: "Scheduled scan handoff did not complete"},
		{name: "completed creation and failed writeback", batch: &scanapp.BatchScanResult{CreatedCount: 1, Scans: []scanapp.QueryScan{{ID: 7}}}, record: errors.New("database unavailable"), message: "Scheduled scan handoff outcome could not be recorded"},
		{name: "deleted occurrence with creation failure", dispatch: errors.New("package cache unavailable"), message: "Scheduled scan handoff did not complete"},
		{name: "result-only failure", batch: &scanapp.BatchScanResult{Failed: []scanapp.CreateBatchItemOutcome{{TargetID: 7, Reason: "TARGET_NOT_FOUND", Message: "catalog lookup failed"}}}, recorded: true, message: "Scheduled scan handoff did not complete"},
		{name: "nil result", recorded: true, message: "Scheduled scan handoff did not complete"},
		{name: "partial success", batch: &scanapp.BatchScanResult{CreatedCount: 1, Scans: []scanapp.QueryScan{{ID: 7}}, Failed: []scanapp.CreateBatchItemOutcome{{Reason: "TARGET_NOT_FOUND"}}}, recorded: true, message: "Scheduled scan handoff did not complete"},
		{name: "deleted occurrence with success", batch: &scanapp.BatchScanResult{CreatedCount: 1, Scans: []scanapp.QueryScan{{ID: 7}}}, message: "Scheduled scan handoff occurrence no longer exists"},
		{name: "completed", batch: &scanapp.BatchScanResult{CreatedCount: 1, Scans: []scanapp.QueryScan{{ID: 7}}}, recorded: true, message: "Scheduled scan handoff completed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			now := testTime()
			repository := &schedulerRepositoryStub{
				candidate:  &OccurrenceCandidate{ID: 9, ScheduledScanID: 1},
				startInput: &FrozenDispatchInput{OccurrenceID: 9, ScheduledScanID: 1, ScheduledFor: now, ScanWorkflowID: "default", TargetIDs: []int{7}, TargetScoped: true, RetryCount: 2},
				recordErr:  test.record, recorded: test.recorded,
			}
			core, logs := observer.New(zap.DebugLevel)
			controller := NewSchedulerController(repository, &dispatcherStub{result: test.batch, err: test.dispatch}).WithRuntime(fixedClock{now}, &cancelingWaiter{}, zapLoggerAdapter{logger: zap.New(core)})
			pass := controller.RunPass(context.Background())
			if (pass.Err != nil) != (test.record != nil) || !pass.Attempted || logs.Len() != 1 || logs.FilterMessage(test.message).Len() != 1 {
				t.Fatalf("pass=%+v logs=%+v", pass, logs.All())
			}
			fields := logs.All()[0].ContextMap()
			if fields["outcome.recorded"] != test.recorded || fmt.Sprint(fields["scheduled_scan.attempt_number"]) != "3" {
				t.Fatalf("settlement/attempt context = %+v", fields)
			}
			if test.dispatch != nil && !strings.Contains(fmt.Sprint(fields["error.chain"]), test.dispatch.Error()) {
				t.Fatalf("dispatch diagnostics lost: %+v", fields)
			}
			if test.record != nil && (!strings.Contains(fmt.Sprint(fields["outcome.error.chain"]), test.record.Error()) || pass.FailureStage != "record_outcome") {
				t.Fatalf("writeback diagnostics lost: %+v", fields)
			}
			if test.message == "Scheduled scan handoff completed" {
				if _, found := fields["error.chain"]; found || logs.FilterLevelExact(zap.InfoLevel).Len() != 1 {
					t.Fatalf("success emitted failure diagnostics: %+v", fields)
				}
			}
		})
	}
}
