package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

type normalScanCreatorCapture struct {
	calls   int
	request *scanapp.CreateBatchRequest
	ctx     context.Context
	result  *scanapp.BatchScanResult
	err     error
}

func (creator *normalScanCreatorCapture) CreateBatch(ctx context.Context, request *scanapp.CreateBatchRequest) (*scanapp.BatchScanResult, error) {
	creator.calls++
	creator.ctx = ctx
	creator.request = request
	return creator.result, creator.err
}

type sequencedNormalScanCreator struct {
	requests []*scanapp.CreateBatchRequest
	results  []*scanapp.BatchScanResult
	errors   []error
}

func (creator *sequencedNormalScanCreator) CreateBatch(_ context.Context, request *scanapp.CreateBatchRequest) (*scanapp.BatchScanResult, error) {
	creator.requests = append(creator.requests, request)
	index := len(creator.requests) - 1
	var result *scanapp.BatchScanResult
	if index < len(creator.results) {
		result = creator.results[index]
	}
	var err error
	if index < len(creator.errors) {
		err = creator.errors[index]
	}
	return result, err
}

type policyAtDispatchNormalScanCreator struct {
	currentPolicy []string
	snapshots     [][]string
	requests      []*scanapp.CreateBatchRequest
}

func (creator *policyAtDispatchNormalScanCreator) CreateBatch(_ context.Context, request *scanapp.CreateBatchRequest) (*scanapp.BatchScanResult, error) {
	creator.requests = append(creator.requests, request)
	creator.snapshots = append(creator.snapshots, append([]string(nil), creator.currentPolicy...))
	return &scanapp.BatchScanResult{CreatedCount: 1, Scans: []scanapp.QueryScan{{ID: len(creator.snapshots)}}}, nil
}

func TestOccurrenceDispatcherUsesOnlyOrdinaryScanCreateInputs(t *testing.T) {
	targetID, agentID := 7, 42
	creator := &normalScanCreatorCapture{result: &scanapp.BatchScanResult{CreatedCount: 1, Scans: []scanapp.QueryScan{{ID: 91}}}}
	dispatcher := NewOccurrenceDispatcher(creator)
	input := FrozenDispatchInput{
		OccurrenceID: 13, ScheduledScanID: 5, ScheduledFor: testTime(),
		ScanWorkflowID: "default", Configuration: map[string]any{"steps": map[string]any{}},
		InputSource: scanapp.InputSourceScanSnapshot,
		TargetIDs:   []int{targetID}, TargetScoped: true, AgentID: &agentID,
	}

	if _, err := dispatcher.Dispatch(context.Background(), input); err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if creator.request == nil || creator.request.ScanWorkflow != "scanWorkflows/default" || len(creator.request.Requests) != 1 || creator.request.Requests[0].TargetID != targetID || creator.request.Requests[0].OrganizationID != 0 {
		t.Fatalf("ordinary Scan request mismatch: %+v", creator.request)
	}
	if creator.request.AgentID == nil || *creator.request.AgentID != agentID {
		t.Fatalf("ordinary Scan request lost pinned Agent: %+v", creator.request)
	}
	if creator.request.TriggerType != scanapp.ScanTriggerTypeScheduled {
		t.Fatalf("scheduled handoff trigger type = %q, want scheduled", creator.request.TriggerType)
	}
}

func TestOccurrenceDispatcherDoesNotTranslateLegacySubdomainDNSConfig(t *testing.T) {
	legacyConfiguration := map[string]any{"steps": map[string]any{
		"subdomain_discovery": map[string]any{
			"enabled": true,
			"engineConfig": map[string]any{
				"dns": map[string]any{"enabled": false, "resolvers": "resolvers.txt"},
			},
		},
	}}
	creator := &normalScanCreatorCapture{err: errors.New(`unknown config section "dns"`)}
	dispatcher := NewOccurrenceDispatcher(creator)

	result, err := dispatcher.Dispatch(context.Background(), FrozenDispatchInput{
		OccurrenceID: 14, ScheduledScanID: 5, ScheduledFor: testTime(),
		ScanWorkflowID: "default", Configuration: legacyConfiguration,
		InputSource: scanapp.InputSourceScanSnapshot,
		TargetIDs:   []int{7}, TargetScoped: true,
	})
	if err == nil || creator.calls != 1 || creator.request == nil {
		t.Fatalf("Dispatch() result=%#v error=%v calls=%d request=%#v, want one failed handoff", result, err, creator.calls, creator.request)
	}
	engineConfig := creator.request.Configuration["steps"].(map[string]any)["subdomain_discovery"].(map[string]any)["engineConfig"].(map[string]any)
	if _, ok := engineConfig["dns"]; !ok {
		t.Fatalf("scheduled handoff translated or removed retired dns config: %#v", engineConfig)
	}
	if _, hasBruteforce := engineConfig["bruteforce"]; hasBruteforce {
		t.Fatalf("scheduled handoff synthesized bruteforce resolver config: %#v", engineConfig)
	}
	if _, hasResolve := engineConfig["resolve"]; hasResolve {
		t.Fatalf("scheduled handoff synthesized resolve resolver config: %#v", engineConfig)
	}
	if outcome := ClassifyHandoff(result, err, true); outcome.Kind != HandoffScanCreateFailed {
		t.Fatalf("legacy configuration handoff outcome = %+v, want scan creation failure", outcome)
	}
}

func TestOccurrenceDispatcherConsumesFrozenOrganizationTargetsWithoutReResolvingMembership(t *testing.T) {
	creator := &normalScanCreatorCapture{result: &scanapp.BatchScanResult{
		CreatedCount: 1,
		Scans:        []scanapp.QueryScan{{ID: 91}},
		Failed:       []scanapp.CreateBatchItemOutcome{{TargetID: 8, Reason: "TARGET_NOT_FOUND"}},
	}}
	dispatcher := NewOccurrenceDispatcher(creator)
	result, err := dispatcher.Dispatch(context.Background(), FrozenDispatchInput{
		OccurrenceID: 14, ScheduledScanID: 5, ScheduledFor: testTime(),
		ScanWorkflowID: "default", Configuration: map[string]any{"steps": map[string]any{}},
		InputSource: scanapp.InputSourceScanSnapshot,
		TargetIDs:   []int{7, 8},
	})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if creator.request == nil || len(creator.request.Requests) != 2 || creator.request.Requests[0].TargetID != 7 || creator.request.Requests[1].TargetID != 8 || creator.request.Requests[0].OrganizationID != 0 || creator.request.Requests[1].OrganizationID != 0 {
		t.Fatalf("dispatcher did not consume the frozen explicit Targets: %+v", creator.request)
	}
	if outcome := ClassifyHandoff(result, nil, false); outcome.Kind != HandoffPartial {
		t.Fatalf("one deleted frozen Target should preserve other results, got %+v", outcome)
	}
}

func TestOccurrenceDispatcherPropagatesAttemptContext(t *testing.T) {
	type contextKey struct{}
	creator := &normalScanCreatorCapture{}
	dispatcher := NewOccurrenceDispatcher(creator)
	targetID := 7
	ctx := context.WithValue(context.Background(), contextKey{}, "attempt")

	if _, err := dispatcher.Dispatch(ctx, FrozenDispatchInput{
		ScanWorkflowID: "default", Configuration: map[string]any{}, InputSource: scanapp.InputSourceScanSnapshot, TargetIDs: []int{targetID}, TargetScoped: true,
	}); err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if creator.ctx == nil || creator.ctx.Value(contextKey{}) != "attempt" {
		t.Fatal("Dispatch() replaced the scheduler-owned operation context")
	}
}

func TestOccurrenceDispatcherDefersPolicyFreezingToEachNormalScanCreate(t *testing.T) {
	targetID := 7
	creator := &policyAtDispatchNormalScanCreator{currentPolicy: []string{"global.before.example"}}
	dispatcher := NewOccurrenceDispatcher(creator)
	occurrence := FrozenDispatchInput{
		OccurrenceID: 1, ScheduledScanID: 9, ScanWorkflowID: "default",
		Configuration: map[string]any{"steps": map[string]any{}}, InputSource: scanapp.InputSourceScanSnapshot, TargetIDs: []int{targetID}, TargetScoped: true,
	}

	if _, err := dispatcher.Dispatch(context.Background(), occurrence); err != nil {
		t.Fatalf("first occurrence Dispatch() error = %v", err)
	}
	creator.currentPolicy = []string{"global.after.example"}
	occurrence.OccurrenceID = 2
	if _, err := dispatcher.Dispatch(context.Background(), occurrence); err != nil {
		t.Fatalf("second occurrence Dispatch() error = %v", err)
	}

	if len(creator.requests) != 2 || len(creator.snapshots) != 2 {
		t.Fatalf("scheduled occurrences did not invoke normal create twice: requests=%d snapshots=%d", len(creator.requests), len(creator.snapshots))
	}
	if got := creator.snapshots; len(got[0]) != 1 || got[0][0] != "global.before.example" || len(got[1]) != 1 || got[1][0] != "global.after.example" {
		t.Fatalf("occurrence policy snapshots = %#v", got)
	}
}

func TestClassifyHandoffUsesStructuredResultsAndSafeMessages(t *testing.T) {
	tests := []struct {
		name   string
		result *scanapp.BatchScanResult
		err    error
		kind   HandoffOutcomeKind
		org    bool
	}{
		{name: "complete", result: &scanapp.BatchScanResult{CreatedCount: 1, Scans: []scanapp.QueryScan{{ID: 1}}}, kind: HandoffCompleted},
		{name: "organization complete", result: &scanapp.BatchScanResult{CreatedCount: 2, Scans: []scanapp.QueryScan{{ID: 1}, {ID: 2}}}, kind: HandoffCompleted, org: true},
		{name: "partial", result: &scanapp.BatchScanResult{CreatedCount: 1, Scans: []scanapp.QueryScan{{ID: 1}}, Failed: []scanapp.CreateBatchItemOutcome{{Reason: "TARGET_NOT_FOUND"}}}, kind: HandoffPartial},
		{name: "structurally incomplete", result: &scanapp.BatchScanResult{CreatedCount: 2, Scans: []scanapp.QueryScan{{ID: 1}}}, kind: HandoffPartial},
		{name: "partial with creation error", result: &scanapp.BatchScanResult{CreatedCount: 1, Scans: []scanapp.QueryScan{{ID: 1}}}, err: errors.New("second child failed"), kind: HandoffPartial, org: true},
		{name: "partial before deadline", result: &scanapp.BatchScanResult{CreatedCount: 1, Scans: []scanapp.QueryScan{{ID: 1}}}, err: context.DeadlineExceeded, kind: HandoffPartial, org: true},
		{name: "deadline", err: context.DeadlineExceeded, kind: HandoffDeadlineExceeded},
		{name: "canceled", err: context.Canceled, kind: HandoffCanceled},
		{name: "ordinary failure", err: errors.New("secret=/tmp/private-token"), kind: HandoffScanCreateFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outcome := ClassifyHandoff(test.result, test.err, !test.org)
			if outcome.Kind != test.kind {
				t.Fatalf("ClassifyHandoff() = %+v, want kind %q", outcome, test.kind)
			}
			if strings.Contains(outcome.Message, "private-token") || len(outcome.Message) > maxHandoffFailureMessageBytes {
				t.Fatalf("unsafe handoff message: %q", outcome.Message)
			}
		})
	}
}

func TestSafeHandoffOutcomeProducesBoundedValidUTF8(t *testing.T) {
	message := strings.Repeat("a", maxHandoffFailureMessageBytes-1) + "\xff" + strings.Repeat("界", 10)
	outcome := safeHandoffOutcome(HandoffScanCreateFailed, message)
	if len(outcome.Message) > maxHandoffFailureMessageBytes || !utf8.ValidString(outcome.Message) {
		t.Fatalf("unsafe bounded message: bytes=%d value=%q", len(outcome.Message), outcome.Message)
	}
}

func TestOccurrenceDispatcherTriggerTimeFailuresRecoverOnlyOnLaterOccurrence(t *testing.T) {
	tests := []struct {
		name              string
		organizationScope bool
	}{
		{name: "workflow"},
		{name: "engine"},
		{name: "target"},
		{name: "organization", organizationScope: true},
		{name: "configuration resource"},
		{name: "pinned agent"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			creator := &sequencedNormalScanCreator{
				results: []*scanapp.BatchScanResult{nil, {CreatedCount: 1, Scans: []scanapp.QueryScan{{ID: 91}}}},
				errors:  []error{errors.New(test.name + " unavailable"), nil},
			}
			dispatcher := NewOccurrenceDispatcher(creator)
			targetID, agentID := 7, 42
			first := FrozenDispatchInput{
				OccurrenceID: 1, ScheduledScanID: 9, ScanWorkflowID: "default",
				Configuration: map[string]any{"state": "broken"}, InputSource: scanapp.InputSourceScanSnapshot, TargetIDs: []int{targetID}, TargetScoped: true, AgentID: &agentID,
			}
			second := first
			second.OccurrenceID = 2
			second.Configuration = map[string]any{"state": "repaired"}
			if test.organizationScope {
				first.TargetScoped, second.TargetScoped = false, false
				first.TargetIDs = []int{targetID, targetID + 1}
				second.TargetIDs = []int{targetID, targetID + 1}
			}

			result, err := dispatcher.Dispatch(context.Background(), first)
			if outcome := ClassifyHandoff(result, err, first.TargetScoped); outcome.Kind != HandoffScanCreateFailed {
				t.Fatalf("first occurrence outcome = %+v", outcome)
			}
			result, err = dispatcher.Dispatch(context.Background(), second)
			if outcome := ClassifyHandoff(result, err, second.TargetScoped); outcome.Kind != HandoffCompleted {
				t.Fatalf("later occurrence outcome = %+v", outcome)
			}
			if len(creator.requests) != 2 || creator.requests[0].Configuration["state"] != "broken" || creator.requests[1].Configuration["state"] != "repaired" {
				t.Fatalf("current-input dispatch requests = %+v", creator.requests)
			}
			if creator.requests[0].ScanWorkflow != "scanWorkflows/default" || creator.requests[0].AgentID == nil || *creator.requests[0].AgentID != agentID {
				t.Fatalf("failure path replaced a frozen input: %+v", creator.requests[0])
			}
		})
	}
}

func testTime() time.Time {
	return time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
}

func TestClassifyHandoffAttachesPublicCauseForZeroCreatedFailures(t *testing.T) {
	causeCases := []struct {
		name string
		err  error
		want HandoffFailureCause
	}{
		{name: "workflow unavailable", err: fmt.Errorf("wrap: %w", scanapp.ErrCreateInvalidScanWorkflow), want: HandoffCauseWorkflowUnavailable},
		{name: "no workflows", err: scanapp.ErrCreateNoScanWorkflows, want: HandoffCauseWorkflowUnavailable},
		{name: "agent not found", err: scanapp.ErrCreateAgentNotFound, want: HandoffCauseAgentNotFound},
		{name: "engine unavailable", err: fmt.Errorf("step: %w", scanapp.ErrCreateScanWorkflowEngineUnavailable), want: HandoffCauseEngineUnavailable},
		{name: "target not found", err: scanapp.ErrCreateTargetNotFound, want: HandoffCauseTargetUnavailable},
		{name: "no targets", err: scanapp.ErrNoTargetsForScan, want: HandoffCauseTargetUnavailable},
		{name: "invalid config", err: scanapp.ErrCreateInvalidConfig, want: HandoffCauseConfigResourceUnavailable},
		{name: "unknown typed error", err: errors.New("database is unavailable"), want: HandoffCauseInternalUnavailable},
		{name: "no error with structured all-target failures", err: nil, want: HandoffCauseTargetUnavailable},
	}
	for _, test := range causeCases {
		t.Run(test.name, func(t *testing.T) {
			var result *scanapp.BatchScanResult
			if test.err == nil {
				result = &scanapp.BatchScanResult{
					Failed: []scanapp.CreateBatchItemOutcome{{Index: 0, Reason: "TARGET_NOT_FOUND"}},
				}
			}
			outcome := ClassifyHandoff(result, test.err, true)
			if outcome.Kind != HandoffScanCreateFailed || outcome.Cause != test.want {
				t.Fatalf("outcome = %+v; want kind=%s cause=%s", outcome, HandoffScanCreateFailed, test.want)
			}
		})
	}
}

func TestClassifyHandoffLeavesCauseEmptyForOtherOutcomes(t *testing.T) {
	partial := &scanapp.BatchScanResult{
		Scans: []scanapp.QueryScan{{TargetID: 1}}, CreatedCount: 1,
		Failed: []scanapp.CreateBatchItemOutcome{{Index: 1, Reason: "TARGET_NOT_FOUND"}},
	}
	if outcome := ClassifyHandoff(partial, nil, false); outcome.Kind != HandoffPartial || outcome.Cause != "" {
		t.Fatalf("partial outcome = %+v; want no cause", outcome)
	}
	if outcome := ClassifyHandoff(nil, context.DeadlineExceeded, false); outcome.Kind != HandoffDeadlineExceeded || outcome.Cause != "" {
		t.Fatalf("deadline outcome = %+v; want no cause", outcome)
	}
	if outcome := ClassifyHandoff(nil, context.Canceled, false); outcome.Kind != HandoffCanceled || outcome.Cause != "" {
		t.Fatalf("canceled outcome = %+v; want no cause", outcome)
	}
}

func TestHandoffRetryPolicyBounds(t *testing.T) {
	if MaxHandoffRetries != 3 {
		t.Fatalf("MaxHandoffRetries = %d; want 3", MaxHandoffRetries)
	}
	delays := []time.Duration{HandoffRetryDelay(1), HandoffRetryDelay(2), HandoffRetryDelay(3)}
	want := []time.Duration{30 * time.Second, time.Minute, 3 * time.Minute}
	for index := range delays {
		if delays[index] != want[index] {
			t.Fatalf("HandoffRetryDelay(%d) = %s; want %s", index+1, delays[index], want[index])
		}
	}
	if _, ok := ParseHandoffFailureCause("NOT_A_CAUSE"); ok {
		t.Fatal("ParseHandoffFailureCause accepted an unknown value")
	}
	for _, cause := range []string{
		"WORKFLOW_UNAVAILABLE", "AGENT_NOT_FOUND", "CONFIG_RESOURCE_UNAVAILABLE",
		"ENGINE_UNAVAILABLE", "TARGET_UNAVAILABLE", "INTERNAL_UNAVAILABLE",
	} {
		if _, ok := ParseHandoffFailureCause(cause); !ok {
			t.Fatalf("ParseHandoffFailureCause rejected enum value %s", cause)
		}
	}
}
