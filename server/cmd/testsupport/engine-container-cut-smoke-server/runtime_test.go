package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/contracts/agentexecution"
	enginecontract "github.com/yyhuni/lunafox/contracts/enginemanifest"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/repositoryname"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"github.com/yyhuni/lunafox/contracts/results"
	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
	"github.com/yyhuni/lunafox/server/internal/grpc/agentcontrol"
	"github.com/yyhuni/lunafox/server/internal/grpc/agentdata"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	resultingestapp "github.com/yyhuni/lunafox/server/internal/modules/resultingest/application"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/yaml.v3"
)

const smokeTestSessionID = "session-smoke-test"

func TestSmokeNucleiTemplateContentUsesCanonicalBaseURLExpression(t *testing.T) {
	for _, scenario := range []smokeScenario{smokeScenarioNucleiHit, smokeScenarioNucleiAckFailure} {
		t.Run(string(scenario), func(t *testing.T) {
			content := smokeNucleiTemplateContent(smokePlanRecordSnapshot{scenario: scenario})
			if strings.Contains(content, "{{{{") || strings.Contains(content, "}}}}") {
				t.Fatalf("template contains over-escaped Nuclei expression: %q", content)
			}
			if !strings.Contains(content, "{{BaseURL}}") {
				t.Fatalf("template does not contain canonical BaseURL expression: %q", content)
			}
			if scenario == smokeScenarioNucleiAckFailure {
				if got := strings.Count(content, "{{BaseURL}}\""); got != 1 {
					t.Fatalf("ack template must keep exactly one root finding path, got %d: %q", got, content)
				}
				delayedPaths := 0
				for index := 1; index < 8; index++ {
					delayedPaths += strings.Count(content, fmt.Sprintf("{{BaseURL}}/smoke-%d\"", index))
				}
				if delayedPaths != 7 {
					t.Fatalf("ack template must keep seven bounded delayed paths, got %d: %q", delayedPaths, content)
				}
				if strings.Contains(content, "smoke-ack") || strings.Contains(content, "port == 18080") || !strings.Contains(content, "type: status") || !strings.Contains(content, "- 200") {
					t.Fatalf("ack template must use the in-scope root URL status matcher: %q", content)
				}
			}
			var document yaml.Node
			if err := yaml.Unmarshal([]byte(content), &document); err != nil {
				t.Fatalf("generated smoke template is not valid YAML: %v", err)
			}
		})
	}
}

func TestScenarioOrderAddsOnlyFixtureNucleiCases(t *testing.T) {
	base := scenarioOrder(smokeConfig{})
	if len(base) != 15 {
		t.Fatalf("base scenario count = %d, want 15", len(base))
	}
	fixture := scenarioOrder(smokeConfig{nucleiFixtureImageRef: "localhost:5000/fixture@sha256:" + strings.Repeat("a", 64)})
	if len(fixture) != 18 || fixture[15] != smokeScenarioNucleiMalformed || fixture[16] != smokeScenarioNucleiMixed || fixture[17] != smokeScenarioNucleiNonzero {
		t.Fatalf("fixture scenario order = %#v, want three appended Nuclei cases", fixture)
	}
}

func TestFixtureNucleiScenarioPlansUseWebsiteIPTarget(t *testing.T) {
	cfg := smokeConfig{fixtureIP: "192.0.2.10", fixturePort: 18080}
	for _, scenario := range []smokeScenario{smokeScenarioNucleiMalformed, smokeScenarioNucleiMixed, smokeScenarioNucleiNonzero} {
		spec, err := scenarioPlan(cfg, scenario)
		if err != nil {
			t.Fatalf("scenarioPlan(%q): %v", scenario, err)
		}
		if spec.engineID != engineNucleiID || spec.targetType != scanapp.ExecutionTargetTypeIP || spec.targetValue != cfg.fixtureIP {
			t.Fatalf("scenarioPlan(%q) = engine=%q target=%s/%s", scenario, spec.engineID, spec.targetType, spec.targetValue)
		}
	}
}

func TestForceNucleiAcknowledgedBatchPlanNarrowsOnlyBatchItemLimit(t *testing.T) {
	store, err := newSmokeScanStore()
	if err != nil {
		t.Fatalf("new smoke store: %v", err)
	}
	t.Cleanup(store.cleanup)
	plan := validSmokePlanForTest()
	plan.EngineRelease.Engine = engineNucleiID
	encoded, err := agentexecution.MarshalResolvedEngineExecutionPlan(plan)
	if err != nil {
		t.Fatalf("encode Nuclei plan: %v", err)
	}
	task := smokePersistedTask{ID: 1, EngineID: engineNucleiID, ResolvedExecutionPlan: encoded}
	if err := store.db.Exec(`INSERT INTO target (id, name, type) VALUES (1, '192.0.2.10', 'ip')`).Error; err != nil {
		t.Fatalf("persist test target: %v", err)
	}
	if err := store.db.Exec(`INSERT INTO scan (id, target_id, scan_workflow_id, input_source, trigger_type) VALUES (1, 1, 'smoke', 'scan_snapshot', 'manual')`).Error; err != nil {
		t.Fatalf("persist test scan: %v", err)
	}
	if err := store.db.Exec(`INSERT INTO scan_task (id, scan_id, stage_id, step_id, engine_id, resolved_execution_plan) VALUES (?, 1, 'nuclei', 'nuclei_vulnerability', ?, ?)`, task.ID, engineNucleiID, encoded).Error; err != nil {
		t.Fatalf("persist test plan: %v", err)
	}
	if err := store.forceNucleiAcknowledgedBatchPlan(&task); err != nil {
		t.Fatalf("force acknowledgement batch plan: %v", err)
	}
	updated, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(task.ResolvedExecutionPlan)
	if err != nil {
		t.Fatalf("decode updated plan: %v", err)
	}
	if updated.GetLimits().GetResultBatchMaxItems() != 1 || updated.GetLimits().GetResultBatchMaxBytes() != plan.GetLimits().GetResultBatchMaxBytes() || updated.GetLimits().GetMaxExecutionDuration().AsDuration() != plan.GetLimits().GetMaxExecutionDuration().AsDuration() {
		t.Fatalf("updated Nuclei limits = %#v", updated.GetLimits())
	}
}

func newSmokeBridgeForTest(t *testing.T, scenario smokeScenario) (*smokeTaskBridge, *smokePlanRecord) {
	t.Helper()
	plan := validSmokePlanForTest()
	encoded, err := agentexecution.MarshalResolvedEngineExecutionPlan(plan)
	if err != nil {
		t.Fatalf("encode test plan: %v", err)
	}
	authority := newSmokeAuthority("cut-smoke-agent-token-test", nil)
	authority.mu.Lock()
	authority.agent.SessionID = smokeTestSessionID
	authority.agent.SessionEpoch = 7
	authority.mu.Unlock()
	record := &smokePlanRecord{
		scenario: scenario, scanID: 1, taskID: 1, targetID: 1,
		engineID: enginePortID, task: plan.GetTask(), planBytes: encoded,
	}
	return newSmokeTaskBridge(authority, []*smokePlanRecord{record}), record
}

func validSmokePlanForTest() *agentexecutionv1.ResolvedEngineExecutionPlan {
	return &agentexecutionv1.ResolvedEngineExecutionPlan{
		Execution: "executions/1",
		Task:      "scans/1/tasks/1",
		Target: &agentexecutionv1.CanonicalTarget{
			Resource: "targets/1",
			Type:     agentexecutionv1.TargetType_TARGET_TYPE_IP,
			Value:    "192.0.2.10",
		},
		WorkflowStep: &agentexecutionv1.WorkflowStepScope{
			Scan: "scans/1", Workflow: "scanWorkflows/default", StageId: "ports", StepId: "port_scan",
		},
		EngineRelease: &agentexecutionv1.EngineRelease{
			Engine: "engine.lunafox.port_scan", PackageDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", EngineApiMajor: 2, CompatibilityRevision: "engine-execution-diagnostics-r1",
		},
		RuntimeImage: &agentexecutionv1.RuntimeImage{Refs: []string{
			"docker.io/yyhuni/lunafox-engine-runtime-port-scan@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		}},
		Config: &agentexecutionv1.FinalEngineConfig{Sections: []*agentexecutionv1.EngineConfigSection{{
			SectionId: "naabu_active", Enabled: true, Params: []*agentexecutionv1.EngineConfigParam{{
				Key: "threads", Value: &agentexecutionv1.EngineConfigScalar{Value: &agentexecutionv1.EngineConfigScalar_IntegerValue{IntegerValue: 1}},
			}},
		}}},
		Limits: &agentexecutionv1.ExecutionLimits{
			MaxExecutionDuration: durationpb.New(time.Minute), ProgressMessageMaxBytes: 1024,
			ResultBatchMaxItems: 10, ResultBatchMaxBytes: 1024,
		},
	}
}

func smokeTestCapability() agentdomain.AgentExecutionCapabilitySnapshot {
	return agentdomain.AgentExecutionCapabilitySnapshot{
		OperatingSystem: "linux", Architecture: "amd64", ContainerRuntimeReady: true,
		SupportedEngineAPIMajors: []uint32{2},
	}
}

func smokeAvailableCompleteDiagnostics() *scanapp.EngineExecutionDiagnostics {
	return &scanapp.EngineExecutionDiagnostics{
		CompatibilityRevision: engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
		Availability:          "available",
		ResultState:           "complete",
		ResultTypeWatermarks: []scanapp.ResultTypeWatermark{{
			ResultType:          "asset.host_port.v1",
			ReceivedItems:       1,
			EncodedItems:        1,
			SubmittedItems:      1,
			AcknowledgedItems:   1,
			SubmittedBatches:    1,
			AcknowledgedBatches: 1,
		}},
	}
}

func claimSmokePlanAfterInjectedFailure(t *testing.T, bridge *smokeTaskBridge, requestID string) *agentexecutionv1.ResolvedEngineExecutionPlan {
	t.Helper()
	plan, err := bridge.ClaimNextExecutionPlan(context.Background(), smokeAgentID, smokeTestSessionID, 7, requestID, smokeTestCapability())
	if err == nil || plan != nil {
		t.Fatalf("first claim = %#v, %v; want injected post-commit failure", plan, err)
	}
	plan, err = bridge.ClaimNextExecutionPlan(context.Background(), smokeAgentID, smokeTestSessionID, 7, requestID, smokeTestCapability())
	if err != nil || plan == nil {
		t.Fatalf("replayed claim = %#v, %v", plan, err)
	}
	return plan
}

func TestSmokeClaimReplayIsImmutableAndCountedOnce(t *testing.T) {
	bridge, _ := newSmokeBridgeForTest(t, smokeScenarioPortSuccess)
	requestID := uuid.NewString()
	want := claimSmokePlanAfterInjectedFailure(t, bridge, requestID)
	replayed, err := bridge.ClaimNextExecutionPlan(context.Background(), smokeAgentID, smokeTestSessionID, 7, requestID, smokeTestCapability())
	if err != nil || replayed.GetTask() != want.GetTask() || replayed.GetExecution() != want.GetExecution() {
		t.Fatalf("duplicate replay = %#v, %v; want same plan", replayed, err)
	}
	counters := bridge.countersSnapshot()
	if counters.planAssignments != 1 || counters.claimRecoveryFailures != 1 || counters.claimRecoveryReplays != 1 {
		t.Fatalf("claim counters = %#v; want one distinct assignment and one recovery", counters)
	}
	if err := bridge.ReportTerminalTaskResult(context.Background(), smokeAgentID, smokeTestSessionID, 7, 1, "succeeded", nil); err != nil {
		t.Fatalf("terminal result: %v", err)
	}
	if err := bridge.ReportTerminalTaskResult(context.Background(), smokeAgentID, smokeTestSessionID, 7, 1, "succeeded", nil); err != nil {
		t.Fatalf("duplicate terminal result: %v", err)
	}
	if got := bridge.countersSnapshot().terminalAcks; got != 1 {
		t.Fatalf("terminal ack count = %d; want 1", got)
	}
	if _, err := bridge.ClaimNextExecutionPlan(context.Background(), smokeAgentID, smokeTestSessionID, 7, requestID, smokeTestCapability()); err == nil {
		t.Fatal("claim replay after terminal observation unexpectedly succeeded")
	}
}

func TestSmokeNoTaskRequestIsNotPersisted(t *testing.T) {
	bridge, record := newSmokeBridgeForTest(t, smokeScenarioPortSuccess)
	record.terminalAcked = false
	bridge.nextIndex = 1
	requestID := uuid.NewString()
	plan, err := bridge.ClaimNextExecutionPlan(context.Background(), smokeAgentID, smokeTestSessionID, 7, requestID, smokeTestCapability())
	if err != nil || plan != nil {
		t.Fatalf("blocked claim = %#v, %v; want no task", plan, err)
	}
	if len(bridge.claims) != 0 {
		t.Fatalf("no-task claim persisted %d request(s)", len(bridge.claims))
	}
	record.terminalAcked = true
	bridge.nextIndex = 0
	plan, err = bridge.ClaimNextExecutionPlan(context.Background(), smokeAgentID, smokeTestSessionID, 7, requestID, smokeTestCapability())
	if err == nil || plan != nil {
		t.Fatalf("same request after ready task = %#v, %v; want injected post-commit failure", plan, err)
	}
	if len(bridge.claims) != 1 {
		t.Fatalf("ready task claim persistence count = %d; want 1", len(bridge.claims))
	}
}

func TestSmokeClaimRejectsSessionEpochAndCapabilityMismatch(t *testing.T) {
	bridge, _ := newSmokeBridgeForTest(t, smokeScenarioPortSuccess)
	requestID := uuid.NewString()
	tests := []struct {
		name       string
		sessionID  string
		epoch      int64
		capability agentdomain.AgentExecutionCapabilitySnapshot
	}{
		{name: "session", sessionID: "session-other", epoch: 7, capability: smokeTestCapability()},
		{name: "epoch", sessionID: smokeTestSessionID, epoch: 8, capability: smokeTestCapability()},
		{name: "runtime", sessionID: smokeTestSessionID, epoch: 7, capability: agentdomain.AgentExecutionCapabilitySnapshot{OperatingSystem: "linux", Architecture: "amd64", SupportedEngineAPIMajors: []uint32{2}}},
		{name: "api", sessionID: smokeTestSessionID, epoch: 7, capability: agentdomain.AgentExecutionCapabilitySnapshot{OperatingSystem: "linux", Architecture: "amd64", ContainerRuntimeReady: true, SupportedEngineAPIMajors: []uint32{1}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := bridge.ClaimNextExecutionPlan(context.Background(), smokeAgentID, test.sessionID, test.epoch, requestID, test.capability); err == nil {
				t.Fatal("mismatched claim unexpectedly succeeded")
			}
		})
	}
	if len(bridge.claims) != 0 || bridge.nextIndex != 0 {
		t.Fatal("rejected claim mutated bridge state")
	}
}

func TestSmokeTerminalAckRequiresExactReplay(t *testing.T) {
	bridge, _ := newSmokeBridgeForTest(t, smokeScenarioWebsiteEmpty)
	claimSmokePlanAfterInjectedFailure(t, bridge, uuid.NewString())
	if err := bridge.ReportTerminalTaskResult(context.Background(), smokeAgentID, smokeTestSessionID, 7, 1, "succeeded", nil); err == nil {
		t.Fatal("first terminal report did not inject the acknowledgement failure")
	}
	if counters := bridge.countersSnapshot(); counters.terminalAcks != 0 || counters.terminalRecoveryFails != 1 || counters.recoveryReplays != 0 {
		t.Fatalf("pre-replay terminal counters = %#v", counters)
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if bridge.waitTerminal(waitCtx, 1) {
		t.Fatal("terminal waiter was notified before acknowledgement")
	}
	if err := bridge.ReportTerminalTaskResult(context.Background(), smokeAgentID, smokeTestSessionID, 7, 1, "failed", &scanapp.FailureDetail{Kind: smokeEngineExitFailureKind, Message: smokeEngineExitFailureMessage}); err == nil {
		t.Fatal("changed terminal replay unexpectedly succeeded")
	}
	if err := bridge.ReportTerminalTaskResult(context.Background(), smokeAgentID, smokeTestSessionID, 7, 1, "succeeded", nil); err != nil {
		t.Fatalf("exact terminal replay: %v", err)
	}
	if counters := bridge.countersSnapshot(); counters.terminalAcks != 1 || counters.terminalRecoveryFails != 1 || counters.recoveryReplays != 1 {
		t.Fatalf("post-replay terminal counters = %#v", counters)
	}
}

func TestSmokeTerminalDiagnosticsAreImmutableReplayEvidence(t *testing.T) {
	bridge, _ := newSmokeBridgeForTest(t, smokeScenarioPortSuccess)
	claimSmokePlanAfterInjectedFailure(t, bridge, uuid.NewString())

	diagnostics := smokeAvailableCompleteDiagnostics()
	if err := bridge.ReportTerminalTaskResultWithDiagnostics(context.Background(), smokeAgentID, smokeTestSessionID, 7, 1, "succeeded", nil, diagnostics); err != nil {
		t.Fatalf("terminal result with diagnostics: %v", err)
	}
	diagnostics.ResultTypeWatermarks[0].ReceivedItems = 2
	record, ok := bridge.recordForTaskID(1)
	if !ok || record.terminal == nil || record.terminal.diagnostics == nil {
		t.Fatalf("stored terminal diagnostics = %#v", record)
	}
	if got := record.terminal.diagnostics.ResultTypeWatermarks[0].ReceivedItems; got != 1 {
		t.Fatalf("stored terminal diagnostic watermark = %d, want detached value 1", got)
	}

	changed := smokeAvailableCompleteDiagnostics()
	changed.ResultTypeWatermarks[0].ReceivedItems = 2
	if err := bridge.ReportTerminalTaskResultWithDiagnostics(context.Background(), smokeAgentID, smokeTestSessionID, 7, 1, "succeeded", nil, changed); err == nil {
		t.Fatal("changed terminal diagnostics replay unexpectedly succeeded")
	}
	if err := bridge.ReportTerminalTaskResultWithDiagnostics(context.Background(), smokeAgentID, smokeTestSessionID, 7, 1, "succeeded", nil, smokeAvailableCompleteDiagnostics()); err != nil {
		t.Fatalf("exact terminal diagnostics replay: %v", err)
	}
}

func TestSmokeDiagnosticEvidenceClassifiesBoundedTerminalSnapshots(t *testing.T) {
	succeeded := smokeDiagnosticEvidenceFor(smokePlanRecordSnapshot{terminal: &smokeTerminalSnapshot{
		result:      "succeeded",
		diagnostics: smokeAvailableCompleteDiagnostics(),
	}})
	if !smokeConfirmedResultDiagnostics(succeeded) || succeeded.AcknowledgedItems != 1 || succeeded.UnresolvedBatches != 0 {
		t.Fatalf("successful diagnostic evidence = %#v", succeeded)
	}

	zero := smokeAvailableCompleteDiagnostics()
	zero.ResultTypeWatermarks[0] = scanapp.ResultTypeWatermark{ResultType: "asset.host_port.v1"}
	zeroEvidence := smokeDiagnosticEvidenceFor(smokePlanRecordSnapshot{terminal: &smokeTerminalSnapshot{
		result: "succeeded", diagnostics: zero,
	}})
	if !smokeZeroResultDiagnostics(zeroEvidence) {
		t.Fatalf("zero-result diagnostic evidence = %#v", zeroEvidence)
	}

	failed := &scanapp.EngineExecutionDiagnostics{
		CompatibilityRevision: engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
		Availability:          "available",
		ResultState:           "none",
		FailedStage:           "handler",
		ErrorType:             "handler_failed",
		ResultTypeWatermarks:  []scanapp.ResultTypeWatermark{{ResultType: "asset.host_port.v1"}},
	}
	failedEvidence := smokeDiagnosticEvidenceFor(smokePlanRecordSnapshot{terminal: &smokeTerminalSnapshot{
		result: "failed", diagnostics: failed,
	}})
	if !smokeNoConfirmedFailureDiagnostics(failedEvidence) {
		t.Fatalf("failed diagnostic evidence = %#v", failedEvidence)
	}

	unavailable := smokeDiagnosticEvidenceFor(smokePlanRecordSnapshot{terminal: &smokeTerminalSnapshot{
		result: "failed", diagnostics: &scanapp.EngineExecutionDiagnostics{
			CompatibilityRevision: engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
			Availability:          "unavailable",
			ResultState:           "unknown",
		},
	}})
	if !smokeUnavailableDiagnostics(unavailable) {
		t.Fatalf("unavailable diagnostic evidence = %#v", unavailable)
	}
}

func TestCanonicalSmokeFailureRequiresStableKindMessagePair(t *testing.T) {
	tests := []struct {
		name    string
		failure *scanapp.FailureDetail
		want    bool
	}{
		{name: "engine exit", failure: &scanapp.FailureDetail{Kind: smokeEngineExitFailureKind, Message: smokeEngineExitFailureMessage}, want: true},
		{name: "result protocol", failure: &scanapp.FailureDetail{Kind: smokeResultProtocolFailureKind, Message: smokeResultProtocolFailureMessage}, want: true},
		{name: "task timeout", failure: &scanapp.FailureDetail{Kind: smokeTaskTimeoutFailureKind, Message: smokeTaskTimeoutFailureMessage}, want: true},
		{name: "nil"},
		{name: "no enabled Nuclei templates", failure: &scanapp.FailureDetail{Kind: smokeNoTemplatesFailureKind, Message: smokeNoTemplatesFailureMessage}, want: true},
		{name: "blank", failure: &scanapp.FailureDetail{}},
		{name: "kind copied into message", failure: &scanapp.FailureDetail{Kind: smokeEngineExitFailureKind, Message: smokeEngineExitFailureKind}},
		{name: "message swapped", failure: &scanapp.FailureDetail{Kind: smokeEngineExitFailureKind, Message: smokeTaskTimeoutFailureMessage}},
		{name: "unknown kind", failure: &scanapp.FailureDetail{Kind: "container_failed", Message: smokeEngineExitFailureMessage}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isCanonicalSmokeFailure(test.failure); got != test.want {
				t.Fatalf("isCanonicalSmokeFailure(%#v) = %t; want %t", test.failure, got, test.want)
			}
		})
	}
}

func TestSmokeProgressRequiresExactLeaseShapeAndRequestReplay(t *testing.T) {
	bridge, _ := newSmokeBridgeForTest(t, smokeScenarioPortSuccess)
	claimSmokePlanAfterInjectedFailure(t, bridge, uuid.NewString())
	sink := &smokeProgressSink{bridge: bridge}
	batch := agentdata.TaskProgressLogBatch{
		ScanID: 1, TaskID: 1, AgentID: smokeAgentID, SessionID: smokeTestSessionID, SessionEpoch: 7,
		RequestID: uuid.NewString(),
		Entries:   []agentdata.TaskProgressLogEntry{{Sequence: 1, Level: "info", Content: "naabu started", EmittedAt: time.Now().UTC()}},
	}
	accepted, duplicates, err := sink.WriteTaskProgressLogs(context.Background(), batch)
	if err != nil || accepted != 1 || duplicates != 0 {
		t.Fatalf("first progress = %d, %d, %v", accepted, duplicates, err)
	}
	accepted, duplicates, err = sink.WriteTaskProgressLogs(context.Background(), batch)
	if err != nil || accepted != 0 || duplicates != 1 {
		t.Fatalf("duplicate progress = %d, %d, %v", accepted, duplicates, err)
	}
	if got := bridge.countersSnapshot().progressMessages; got != 1 {
		t.Fatalf("progress evidence count = %d; want 1", got)
	}
	changed := batch
	changed.Entries = append([]agentdata.TaskProgressLogEntry(nil), batch.Entries...)
	changed.Entries[0].Content = "changed"
	if _, _, err := sink.WriteTaskProgressLogs(context.Background(), changed); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("changed progress replay error = %v", err)
	}

	invalid := []struct {
		name   string
		mutate func(*agentdata.TaskProgressLogBatch)
	}{
		{name: "uuid version", mutate: func(value *agentdata.TaskProgressLogBatch) { value.RequestID = "6ba7b810-9dad-11d1-80b4-00c04fd430c8" }},
		{name: "multiple entries", mutate: func(value *agentdata.TaskProgressLogBatch) {
			value.RequestID = uuid.NewString()
			value.Entries = append(value.Entries, value.Entries[0])
		}},
		{name: "sequence", mutate: func(value *agentdata.TaskProgressLogBatch) {
			value.RequestID = uuid.NewString()
			value.Entries[0].Sequence = 2
		}},
		{name: "level", mutate: func(value *agentdata.TaskProgressLogBatch) {
			value.RequestID = uuid.NewString()
			value.Entries[0].Level = "warning"
		}},
		{name: "content", mutate: func(value *agentdata.TaskProgressLogBatch) {
			value.RequestID = uuid.NewString()
			value.Entries[0].Content = "  "
		}},
		{name: "lease", mutate: func(value *agentdata.TaskProgressLogBatch) {
			value.RequestID = uuid.NewString()
			value.SessionEpoch = 8
		}},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			candidate := batch
			candidate.Entries = append([]agentdata.TaskProgressLogEntry(nil), batch.Entries...)
			test.mutate(&candidate)
			if _, _, err := sink.WriteTaskProgressLogs(context.Background(), candidate); err == nil {
				t.Fatal("invalid progress unexpectedly succeeded")
			}
		})
	}
}

func TestSmokeProgressCancelsNucleiCancelScenario(t *testing.T) {
	bridge, record := newSmokeBridgeForTest(t, smokeScenarioNucleiCancel)
	bridge.cancelPublisher = agentcontrol.NewAgentControlEventPublisher(agentcontrol.NewAgentStreamRegistry())
	claimSmokePlanAfterInjectedFailure(t, bridge, uuid.NewString())
	sink := &smokeProgressSink{bridge: bridge}
	batch := agentdata.TaskProgressLogBatch{
		ScanID: 1, TaskID: 1, AgentID: smokeAgentID, SessionID: smokeTestSessionID, SessionEpoch: 7,
		RequestID: uuid.NewString(),
		Entries:   []agentdata.TaskProgressLogEntry{{Sequence: 1, Level: "info", Content: "scan_started candidates=2", EmittedAt: time.Now().UTC()}},
	}
	if accepted, duplicates, err := sink.WriteTaskProgressLogs(context.Background(), batch); err != nil || accepted != 1 || duplicates != 0 {
		t.Fatalf("Nuclei cancel progress = %d, %d, %v", accepted, duplicates, err)
	}
	if !record.cancelIssued {
		t.Fatal("Nuclei cancel progress did not persist cancellation intent")
	}
}

func TestSmokeProviderArtifactUsesCompleteCurrentRegistryMapping(t *testing.T) {
	source := catalogExecutionProviderSource(&smokeProviderSettingsStore{})
	content, err := source.GetExecutionSubfinderProviderConfig(context.Background())
	if err != nil {
		t.Fatalf("produce provider artifact: %v", err)
	}
	var decoded map[string][]string
	if err := yaml.Unmarshal([]byte(content), &decoded); err != nil {
		t.Fatalf("decode provider artifact: %v", err)
	}
	definitions := catalogdomain.SubfinderProviderDefinitions()
	if len(decoded) != len(definitions) {
		t.Fatalf("provider key count = %d, want %d", len(decoded), len(definitions))
	}
	for _, definition := range definitions {
		values, ok := decoded[definition.SourceName]
		if !ok || values == nil || len(values) != 0 {
			t.Fatalf("provider %q = %#v (present=%v), want explicit empty list", definition.SourceName, values, ok)
		}
	}
}

func TestSmokeResultEvidenceRequiresRealHostPortMaterialization(t *testing.T) {
	bridge, _ := newSmokeBridgeForTest(t, smokeScenarioPortSuccess)
	claimSmokePlanAfterInjectedFailure(t, bridge, uuid.NewString())
	cfg := smokeConfig{fixtureIP: "192.0.2.10", fixturePort: 8080}
	resultJSON, err := results.EncodeHostPort(results.HostPort{Host: cfg.fixtureIP, IP: cfg.fixtureIP, Port: cfg.fixturePort})
	if err != nil {
		t.Fatalf("encode HostPort result: %v", err)
	}
	ingest := &smokeResultIngest{bridge: bridge, inner: newSmokeResultFacade(newSmokeResultEvidenceStore(cfg))}
	outcome, err := ingest.Ingest(context.Background(), resultingestCommandForSmokeTest(resultJSON))
	if err != nil {
		t.Fatalf("ingest HostPort result: %v", err)
	}
	if outcome.ReceivedItems != 1 || outcome.SnapshotCount != 1 || outcome.AssetCount != 1 {
		t.Fatalf("HostPort result outcome = %#v", outcome)
	}
	snapshot, ok := bridge.recordForTaskID(1)
	if !ok || snapshot.resultBatches != 1 || snapshot.resultItems != 1 || !snapshot.resultMatched {
		t.Fatalf("HostPort result evidence = %#v; want one matched batch", snapshot)
	}
}

func TestSmokeWebsiteMaterializerRequiresExactFixtureResponse(t *testing.T) {
	cfg := smokeConfig{fixtureIP: "192.0.2.10", fixturePort: 8080}
	expectedURL := "http://192.0.2.10:8080"
	okStatus := 200
	wrongStatus := 201
	tests := []struct {
		name string
		item snapshotapp.WebsiteSnapshotItem
	}{
		{name: "wrong URL", item: snapshotapp.WebsiteSnapshotItem{URL: "https://192.0.2.10:8080", Host: cfg.fixtureIP, StatusCode: &okStatus}},
		{name: "wrong host", item: snapshotapp.WebsiteSnapshotItem{URL: expectedURL, Host: "192.0.2.11", StatusCode: &okStatus}},
		{name: "missing status", item: snapshotapp.WebsiteSnapshotItem{URL: expectedURL, Host: cfg.fixtureIP}},
		{name: "wrong status", item: snapshotapp.WebsiteSnapshotItem{URL: expectedURL, Host: cfg.fixtureIP, StatusCode: &wrongStatus}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			materializer := &smokeWebsiteMaterializer{store: newSmokeResultEvidenceStore(cfg)}
			if _, err := materializer.SaveAndSyncContext(context.Background(), 1, 1, []snapshotapp.WebsiteSnapshotItem{test.item}); err == nil {
				t.Fatal("non-exact Website fixture unexpectedly materialized")
			}
		})
	}

	bridge, record := newSmokeBridgeForTest(t, smokeScenarioWebsiteSuccess)
	record.engineID = engineWebsiteID
	claimSmokePlanAfterInjectedFailure(t, bridge, uuid.NewString())
	resultJSON, err := results.EncodeWebsite(results.Website{
		URL: expectedURL, Host: cfg.fixtureIP, StatusCode: &okStatus,
	})
	if err != nil {
		t.Fatalf("encode Website result: %v", err)
	}
	command := resultingestCommandForSmokeTest(resultJSON)
	command.ResultType = results.ResultKindAssetWebsite
	store := newSmokeResultEvidenceStore(cfg)
	ingest := &smokeResultIngest{bridge: bridge, inner: newSmokeResultFacade(store)}
	outcome, err := ingest.Ingest(context.Background(), command)
	if err != nil {
		t.Fatalf("ingest exact Website result: %v", err)
	}
	if outcome.ReceivedItems != 1 || outcome.SnapshotCount != 1 || outcome.AssetCount != 1 {
		t.Fatalf("Website result outcome = %#v", outcome)
	}
	store.mu.RLock()
	materialized := append([]snapshotapp.WebsiteSnapshotItem(nil), store.websites[1]...)
	store.mu.RUnlock()
	if len(materialized) != 1 || materialized[0].URL != expectedURL || materialized[0].Host != cfg.fixtureIP || materialized[0].StatusCode == nil || *materialized[0].StatusCode != okStatus {
		t.Fatalf("materialized Website = %#v; want exact fixture response", materialized)
	}
	snapshot, ok := bridge.recordForTaskID(1)
	if !ok || snapshot.resultBatches != 1 || snapshot.resultItems != 1 || !snapshot.resultMatched {
		t.Fatalf("Website result evidence = %#v; want one matched batch", snapshot)
	}
}

func resultingestCommandForSmokeTest(item string) resultingestapp.ResultIngestCommand {
	return resultingestapp.ResultIngestCommand{
		TaskID: 1, ScanID: 1, TargetID: 1,
		ResultType: results.ResultKindAssetHostPort,
		Items:      [][]byte{[]byte(item)},
	}
}

func smokePackageReaderForRepositoryTest(t *testing.T) *packageMapReader {
	t.Helper()
	repositoryRoot := filepath.Clean(filepath.Join("..", "..", "..", ".."))
	engineDirs := map[string]string{
		engineSubdomainID: "subdomain_discovery",
		enginePortID:      "port_scan",
		engineWebsiteID:   "website_discovery",
		engineNucleiID:    "nuclei_vulnerability",
	}
	packages := make(map[string]scanapp.PlanTaskPackage, len(engineDirs))
	for engineID, directory := range engineDirs {
		path := filepath.Join(repositoryRoot, "extensions", "engines", directory, "engine.json")
		payload, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		definition, err := enginecontract.DecodeEngineDefinition(payload, path)
		if err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		runtimeRepository, err := repositoryname.FirstPartyRuntimeImageRepositoryName(engineID)
		if err != nil {
			t.Fatalf("derive Runtime Image repository for %s: %v", engineID, err)
		}
		packages[engineID] = scanapp.PlanTaskPackage{
			Identity: scanapp.PlanTaskPackageIdentity{
				EngineID:      engineID,
				PackageDigest: "sha256:" + strings.Repeat("a", 64),
			},
			PackageVersion: "0.0.0-test",
			Definition:     definition,
			RuntimeImageRefs: []string{
				fmt.Sprintf("docker.io/yyhuni/%s@sha256:%s", runtimeRepository, strings.Repeat("b", 64)),
			},
		}
	}
	return &packageMapReader{packages: packages, cacheEmptyObserved: true}
}

func TestSmokeScanCreateRepositoryInputAndWorkflowEvidence(t *testing.T) {
	ctx := context.Background()
	packages := smokePackageReaderForRepositoryTest(t)
	wordlists, err := newSmokeWordlists()
	if err != nil {
		t.Fatalf("newSmokeWordlists: %v", err)
	}
	defer wordlists.cleanup()
	compiler, err := scanapp.NewPlanTaskCompiler(packages, wordlists)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler: %v", err)
	}
	store, err := newSmokeScanStore()
	if err != nil {
		t.Fatalf("newSmokeScanStore: %v", err)
	}
	defer store.cleanup()

	workflow, err := runSmokeWorkflowEvidence(ctx, store, compiler, packages)
	if err != nil {
		t.Fatalf("runSmokeWorkflowEvidence: %v", err)
	}
	inputs, err := runSmokeInputSemanticsEvidence(ctx, store, compiler, packages, "192.0.2.10")
	if err != nil {
		t.Fatalf("runSmokeInputSemanticsEvidence: %v", err)
	}
	cfg := smokeConfig{fixtureIP: "192.0.2.10", fixturePort: 18080}
	records, err := createSmokeRuntimeRecords(ctx, store, compiler, packages, wordlists, cfg)
	if err != nil {
		t.Fatalf("createSmokeRuntimeRecords: %v", err)
	}
	if len(records) != len(scenarioOrder()) {
		t.Fatalf("runtime record count = %d, want %d", len(records), len(scenarioOrder()))
	}
	for _, record := range records {
		if record == nil || len(record.planBytes) == 0 {
			t.Fatalf("runtime record has no persisted plan: %#v", record)
		}
	}
	recordByScenario := make(map[smokeScenario]*smokePlanRecord, len(records))
	for _, record := range records {
		recordByScenario[record.scenario] = record
	}
	portSuccess := recordByScenario[smokeScenarioPortSuccess]
	websiteSuccess := recordByScenario[smokeScenarioWebsiteSuccess]
	portEmpty := recordByScenario[smokeScenarioPortEmpty]
	websiteEmpty := recordByScenario[smokeScenarioWebsiteEmpty]
	subdomainFailure := recordByScenario[smokeScenarioSubdomainFail]
	wordlistCacheHit := recordByScenario[smokeScenarioWordlistCacheHit]
	if portSuccess == nil || websiteSuccess == nil || portSuccess.scanID <= 0 || portSuccess.scanID != websiteSuccess.scanID {
		t.Fatalf("IP runtime chain is not linked to one scan: port=%#v website=%#v", portSuccess, websiteSuccess)
	}
	if portEmpty == nil || websiteEmpty == nil || portEmpty.scanID <= 0 || portEmpty.scanID != websiteEmpty.scanID {
		t.Fatalf("CIDR runtime chain is not linked to one scan: port=%#v website=%#v", portEmpty, websiteEmpty)
	}
	if portSuccess.scanID == portEmpty.scanID {
		t.Fatalf("IP and CIDR runtime chains unexpectedly share scan %d", portSuccess.scanID)
	}
	if subdomainFailure == nil || wordlistCacheHit == nil {
		t.Fatalf("subdomain resource smoke records are missing: failure=%#v cacheHit=%#v", subdomainFailure, wordlistCacheHit)
	}
	wordlistResources := make(map[smokeScenario]string, 2)
	for _, record := range []*smokePlanRecord{subdomainFailure, wordlistCacheHit} {
		subdomainPlan, decodeErr := decodeSmokePlanBytes(record.planBytes)
		if decodeErr != nil {
			t.Fatalf("decode %s plan: %v", record.scenario, decodeErr)
		}
		resolveEnabled := false
		resolveSeen := false
		for _, section := range subdomainPlan.GetConfig().GetSections() {
			if section.GetSectionId() == "resolve" {
				resolveSeen = true
				resolveEnabled = section.GetEnabled()
				break
			}
		}
		if !resolveSeen || resolveEnabled {
			t.Fatalf("%s smoke plan must disable resolve to exercise deterministic Engine failure: %#v", record.scenario, subdomainPlan.GetConfig().GetSections())
		}
		for _, binding := range subdomainPlan.GetConfigResourceBindings() {
			if binding.GetSectionId() == "resolve" {
				t.Fatalf("%s smoke plan retained a resolve config resource: %#v", record.scenario, binding)
			}
			if binding.GetSectionId() == "bruteforce" && binding.GetParamKey() == "wordlist" && binding.GetWordlist() != nil {
				wordlistResources[record.scenario] = binding.GetWordlist().GetResource()
			}
		}
		if wordlistResources[record.scenario] == "" {
			t.Fatalf("%s smoke plan has no bruteforce.wordlist config resource: %#v", record.scenario, subdomainPlan.GetConfigResourceBindings())
		}
	}
	if wordlistResources[smokeScenarioSubdomainFail] != wordlistResources[smokeScenarioWordlistCacheHit] {
		t.Fatalf("cache-hit smoke plan wordlist = %q, want warmed wordlist %q", wordlistResources[smokeScenarioWordlistCacheHit], wordlistResources[smokeScenarioSubdomainFail])
	}
	websitePlan, err := decodeSmokePlanBytes(websiteSuccess.planBytes)
	if err != nil {
		t.Fatalf("decode Website success plan: %v", err)
	}
	wantHTTPXParams := map[string]int64{
		"timeout": 60, "threads": 1, "rate-limit": 100, "request-timeout": 1, "retries": 0,
	}
	gotHTTPXParams := make(map[string]int64, len(wantHTTPXParams))
	httpxEnabled := false
	for _, section := range websitePlan.GetConfig().GetSections() {
		if section.GetSectionId() != "httpx" {
			continue
		}
		httpxEnabled = section.GetEnabled()
		for _, param := range section.GetParams() {
			if _, expected := wantHTTPXParams[param.GetKey()]; expected {
				gotHTTPXParams[param.GetKey()] = param.GetValue().GetIntegerValue()
			}
		}
	}
	if !httpxEnabled || len(gotHTTPXParams) != len(wantHTTPXParams) {
		t.Fatalf("Website httpx config is incomplete: %#v", websitePlan.GetConfig().GetSections())
	}
	for key, want := range wantHTTPXParams {
		if got := gotHTTPXParams[key]; got != want {
			t.Fatalf("Website httpx config %q = %d, want %d", key, got, want)
		}
	}
	emptyPlan, err := decodeSmokePlanBytes(portEmpty.planBytes)
	if err != nil {
		t.Fatalf("decode CIDR zero-result plan: %v", err)
	}
	if emptyPlan.GetTarget().GetValue() != smokeEmptyCIDR {
		t.Fatalf("CIDR zero-result Target = %q, want %q", emptyPlan.GetTarget().GetValue(), smokeEmptyCIDR)
	}
	sectionStates := map[string]bool{"naabu_active": false, "naabu_passive": true}
	for _, section := range emptyPlan.GetConfig().GetSections() {
		if expected, ok := sectionStates[section.GetSectionId()]; ok {
			sectionStates[section.GetSectionId()] = section.GetEnabled() == expected
		}
	}
	if !sectionStates["naabu_active"] || !sectionStates["naabu_passive"] {
		t.Fatalf("CIDR zero-result naabu section states are unexpected: %#v", emptyPlan.GetConfig().GetSections())
	}
	blacklist, err := newSmokeEmptyExecutionInputBlacklistFilter()
	if err != nil {
		t.Fatalf("create empty blacklist filter: %v", err)
	}
	producer, err := scanapp.NewSubdomainsProducer(ctx, portEmpty.scanID, smokeDNSCursor{}, blacklist)
	if err != nil {
		t.Fatalf("create CIDR Subdomains producer: %v", err)
	}
	var cidrSubdomains bytes.Buffer
	recordCount, err := scanapp.WriteCanonicalLineStream(ctx, &cidrSubdomains, producer)
	if err != nil {
		t.Fatalf("stream CIDR Subdomains: %v", err)
	}
	if recordCount != 0 || cidrSubdomains.Len() != 0 {
		t.Fatalf("CIDR Subdomains = %d records/%d bytes, want finalized snapshot facts to be empty", recordCount, cidrSubdomains.Len())
	}
	scanCreate, err := store.scanCreateEvidence(ctx)
	if err != nil {
		t.Fatalf("scanCreateEvidence: %v", err)
	}
	if !workflow.Passed || !inputs.Passed || !scanCreate.Passed {
		t.Fatalf("smoke evidence did not pass: workflow=%#v inputs=%#v scanCreate=%#v", workflow, inputs, scanCreate)
	}
	if scanCreate.CreatedScans != 18 || scanCreate.PersistedTaskRows != 30 ||
		scanCreate.ExecutableTasks != 27 || scanCreate.SavedPlanRepositoryReads != 27 ||
		scanCreate.PlanningSkippedTasks != 3 || scanCreate.SkippedWithoutPlanReads != 3 {
		t.Fatalf("unexpected scan-create repository counts: %#v", scanCreate)
	}
}

func TestSmokeProductionBridgePersistsClaimReplayAndTerminalRecovery(t *testing.T) {
	ctx := context.Background()
	packages := smokePackageReaderForRepositoryTest(t)
	wordlists, err := newSmokeWordlists()
	if err != nil {
		t.Fatalf("newSmokeWordlists: %v", err)
	}
	defer wordlists.cleanup()
	compiler, err := scanapp.NewPlanTaskCompiler(packages, wordlists)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler: %v", err)
	}
	store, err := newSmokeScanStore()
	if err != nil {
		t.Fatalf("newSmokeScanStore: %v", err)
	}
	defer store.cleanup()
	record, err := createSmokeRecord(ctx, store, compiler, packages, wordlists, smokeConfig{
		fixtureIP: "192.0.2.10", fixturePort: 18080,
	}, smokeScenarioWebsiteEmpty)
	if err != nil {
		t.Fatalf("createSmokeRecord: %v", err)
	}
	authority := newSmokeAuthority("cut-smoke-agent-token-production", nil)
	authority.UseSessionStore(store)
	if err := authority.RecordHeartbeat(ctx, smokeAgentID, agentdomain.AgentHeartbeatEvent{
		SessionID: smokeTestSessionID, SessionEpoch: 7,
		OperatingSystem: "linux", Architecture: "amd64", ContainerRuntimeReady: true,
		SupportedEngineAPIMajors: []uint32{2},
	}); err != nil {
		t.Fatalf("RecordHeartbeat: %v", err)
	}
	bridge := newSmokeTaskBridge(authority, []*smokePlanRecord{record})
	bridge.UseProductionBridge(
		scanapp.NewScanTaskBridgeService(store.tasks, store.scans).
			WithEngineExecutionClaimStore(store.tasks),
	)

	requestID := uuid.NewString()
	if plan, err := bridge.ClaimNextExecutionPlan(ctx, smokeAgentID, smokeTestSessionID, 7, requestID, smokeTestCapability()); err == nil || plan != nil {
		t.Fatalf("first production claim = %#v, %v; want injected response loss", plan, err)
	}
	var claimed struct {
		Status               string  `gorm:"column:status"`
		AssignedAgentID      *int    `gorm:"column:assigned_agent_id"`
		AssignedSessionID    *string `gorm:"column:assigned_session_id"`
		AssignedSessionEpoch *int64  `gorm:"column:assigned_session_epoch"`
		AssignedRequestID    *string `gorm:"column:assigned_request_id"`
	}
	if err := store.db.Table("scan_task").Where("id = ?", record.taskID).Take(&claimed).Error; err != nil {
		t.Fatalf("read claimed task: %v", err)
	}
	if claimed.Status != "running" || claimed.AssignedAgentID == nil || *claimed.AssignedAgentID != smokeAgentID ||
		claimed.AssignedSessionID == nil || *claimed.AssignedSessionID != smokeTestSessionID ||
		claimed.AssignedSessionEpoch == nil || *claimed.AssignedSessionEpoch != 7 ||
		claimed.AssignedRequestID == nil || *claimed.AssignedRequestID != requestID {
		t.Fatalf("production claim tuple = %#v", claimed)
	}
	replayed, err := bridge.ClaimNextExecutionPlan(ctx, smokeAgentID, smokeTestSessionID, 7, requestID, smokeTestCapability())
	if err != nil || replayed == nil || replayed.GetTask() != record.task {
		t.Fatalf("production claim replay = %#v, %v", replayed, err)
	}
	noTaskRequestID := uuid.NewString()
	noTask, err := bridge.ClaimNextExecutionPlan(ctx, smokeAgentID, smokeTestSessionID, 7, noTaskRequestID, smokeTestCapability())
	if err != nil || noTask != nil {
		t.Fatalf("production no-task = %#v, %v", noTask, err)
	}
	var noTaskRows int64
	if err := store.db.Table("scan_task").Where("assigned_request_id = ?", noTaskRequestID).Count(&noTaskRows).Error; err != nil || noTaskRows != 0 {
		t.Fatalf("no-task request persistence: rows=%d err=%v", noTaskRows, err)
	}

	diagnostics := smokeAvailableCompleteDiagnostics()
	if err := bridge.ReportTerminalTaskResultWithDiagnostics(ctx, smokeAgentID, smokeTestSessionID, 7, record.taskID, "succeeded", nil, diagnostics); err == nil {
		t.Fatal("first production terminal did not inject acknowledgement loss")
	}
	var terminal struct {
		TaskStatus                    string `gorm:"column:task_status"`
		TerminalReconciliationPending bool   `gorm:"column:terminal_reconciliation_pending"`
		ScanStatus                    string `gorm:"column:scan_status"`
	}
	if err := store.db.Table("scan_task AS st").
		Joins("JOIN scan AS s ON s.id = st.scan_id").
		Select("st.status AS task_status, st.terminal_reconciliation_pending, s.status AS scan_status").
		Where("st.id = ?", record.taskID).
		Take(&terminal).Error; err != nil {
		t.Fatalf("read terminal task: %v", err)
	}
	if terminal.TaskStatus != "succeeded" || terminal.ScanStatus != "succeeded" || terminal.TerminalReconciliationPending {
		t.Fatalf("production terminal state = %#v", terminal)
	}
	persisted, err := store.tasks.GetByID(ctx, record.taskID)
	if err != nil {
		t.Fatalf("read persisted terminal diagnostics: %v", err)
	}
	if persisted.Diagnostics == nil || persisted.Diagnostics.Availability != "available" || persisted.Diagnostics.ResultState != "complete" ||
		len(persisted.Diagnostics.ResultTypeWatermarks) != 1 || persisted.Diagnostics.ResultTypeWatermarks[0].AcknowledgedItems != 1 {
		t.Fatalf("persisted terminal diagnostics = %#v", persisted.Diagnostics)
	}
	if err := bridge.ReportTerminalTaskResult(ctx, smokeAgentID, smokeTestSessionID, 7, record.taskID, "failed", &scanapp.FailureDetail{
		Kind: smokeEngineExitFailureKind, Message: smokeEngineExitFailureMessage,
	}); err == nil {
		t.Fatal("conflicting production terminal replay unexpectedly succeeded")
	}
	if err := bridge.ReportTerminalTaskResultWithDiagnostics(ctx, smokeAgentID, smokeTestSessionID, 7, record.taskID, "succeeded", nil, diagnostics); err != nil {
		t.Fatalf("exact production terminal replay: %v", err)
	}
	counters := bridge.countersSnapshot()
	if counters.planAssignments != 1 || counters.claimRecoveryFailures != 1 || counters.claimRecoveryReplays != 1 ||
		counters.productionClaimCalls < 3 || counters.terminalAcks != 1 || counters.terminalRecoveryFails != 1 ||
		counters.recoveryReplays != 1 || counters.productionTerminalCalls != 1 {
		t.Fatalf("production bridge counters = %#v", counters)
	}
}

func TestSmokeProductionBridgeCompletesEveryRuntimeScenarioInOrder(t *testing.T) {
	ctx := context.Background()
	packages := smokePackageReaderForRepositoryTest(t)
	wordlists, err := newSmokeWordlists()
	if err != nil {
		t.Fatalf("newSmokeWordlists: %v", err)
	}
	defer wordlists.cleanup()
	compiler, err := scanapp.NewPlanTaskCompiler(packages, wordlists)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler: %v", err)
	}
	store, err := newSmokeScanStore()
	if err != nil {
		t.Fatalf("newSmokeScanStore: %v", err)
	}
	defer store.cleanup()
	records, err := createSmokeRuntimeRecords(ctx, store, compiler, packages, wordlists, smokeConfig{
		fixtureIP: "192.0.2.10", fixturePort: 18080,
	})
	if err != nil {
		t.Fatalf("createSmokeRuntimeRecords: %v", err)
	}
	authority := newSmokeAuthority("cut-smoke-agent-token-production-all", nil)
	authority.UseSessionStore(store)
	if err := authority.RecordHeartbeat(ctx, smokeAgentID, agentdomain.AgentHeartbeatEvent{
		SessionID: smokeTestSessionID, SessionEpoch: 7,
		OperatingSystem: "linux", Architecture: "amd64", ContainerRuntimeReady: true,
		SupportedEngineAPIMajors: []uint32{2},
	}); err != nil {
		t.Fatalf("RecordHeartbeat: %v", err)
	}
	bridge := newSmokeTaskBridge(authority, records)
	bridge.UseProductionBridge(
		scanapp.NewScanTaskBridgeService(store.tasks, store.scans).
			WithEngineExecutionClaimStore(store.tasks),
	)

	for index, record := range records {
		requestID := uuid.NewString()
		plan, claimErr := bridge.ClaimNextExecutionPlan(ctx, smokeAgentID, smokeTestSessionID, 7, requestID, smokeTestCapability())
		if index == 0 {
			if claimErr == nil || plan != nil {
				t.Fatalf("first production claim = %#v, %v; want injected response loss", plan, claimErr)
			}
			plan, claimErr = bridge.ClaimNextExecutionPlan(ctx, smokeAgentID, smokeTestSessionID, 7, requestID, smokeTestCapability())
		}
		if claimErr != nil || plan == nil || plan.GetTask() != record.task {
			t.Fatalf("claim %s = %#v, %v", record.scenario, plan, claimErr)
		}

		result, failure := smokeScenarioTerminalResult(record.scenario)
		terminalErr := bridge.ReportTerminalTaskResult(ctx, smokeAgentID, smokeTestSessionID, 7, record.taskID, result, failure)
		if record.scenario == smokeScenarioWebsiteEmpty {
			if terminalErr == nil {
				t.Fatal("website empty terminal did not inject acknowledgement loss")
			}
			terminalErr = bridge.ReportTerminalTaskResult(ctx, smokeAgentID, smokeTestSessionID, 7, record.taskID, result, failure)
		}
		if terminalErr != nil {
			t.Fatalf("terminal %s: %v", record.scenario, terminalErr)
		}
	}
	if !bridge.allTerminalsAcked() {
		t.Fatal("all runtime smoke terminals were not acknowledged")
	}
}

func smokeScenarioTerminalResult(scenario smokeScenario) (string, *scanapp.FailureDetail) {
	switch scenario {
	case smokeScenarioArtifactIntegrity:
		return "failed", &scanapp.FailureDetail{Kind: smokeConfigHashFailureKind, Message: smokeConfigHashFailureMessage}
	case smokeScenarioSubdomainFail, smokeScenarioWordlistCacheHit:
		return "failed", &scanapp.FailureDetail{Kind: smokeEngineExitFailureKind, Message: smokeEngineExitFailureMessage}
	case smokeScenarioPortCancel:
		return "cancelled", nil
	case smokeScenarioPortTimeout:
		return "failed", &scanapp.FailureDetail{Kind: smokeTaskTimeoutFailureKind, Message: smokeTaskTimeoutFailureMessage}
	default:
		return "succeeded", nil
	}
}

func TestSmokeSessionStoreRejectsStaleEpochOverwrite(t *testing.T) {
	store, err := newSmokeScanStore()
	if err != nil {
		t.Fatalf("newSmokeScanStore: %v", err)
	}
	defer store.cleanup()
	if err := store.setAgentExecutionSession(smokeAgentID, "session-new", 9); err != nil {
		t.Fatalf("persist current session: %v", err)
	}
	if err := store.setAgentExecutionSession(smokeAgentID, "session-old", 8); !errors.Is(err, errSmokeStaleAgentExecutionSession) {
		t.Fatalf("stale session overwrite error = %v", err)
	}
	var row struct {
		SessionID    string `gorm:"column:session_id"`
		SessionEpoch int64  `gorm:"column:session_epoch"`
	}
	if err := store.db.Table("agent_runtime_status").Where("agent_id = ?", smokeAgentID).Take(&row).Error; err != nil {
		t.Fatalf("read current session: %v", err)
	}
	if row.SessionID != "session-new" || row.SessionEpoch != 9 {
		t.Fatalf("stale session replaced current row: %#v", row)
	}
}
