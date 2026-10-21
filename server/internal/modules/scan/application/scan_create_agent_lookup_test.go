package application

import (
	"context"
	"errors"
	"testing"
)

type scanCreateAgentLookupStub struct {
	exists bool
	err    error
}

func (stub scanCreateAgentLookupStub) AgentExists(context.Context, int) (bool, error) {
	return stub.exists, stub.err
}

func TestValidateSelectedAgentFailsFastForMissingAgent(t *testing.T) {
	agentID := 42
	service := (&ScanCreateService{}).WithAgentLookup(scanCreateAgentLookupStub{})
	if err := service.validateSelectedAgent(context.Background(), &agentID); !errors.Is(err, ErrCreateAgentNotFound) {
		t.Fatalf("missing Agent validation error = %v, want %v", err, ErrCreateAgentNotFound)
	}
	if err := service.validateSelectedAgent(context.Background(), nil); err != nil {
		t.Fatalf("automatic assignment should not need an Agent lookup: %v", err)
	}
}

func TestValidateSelectedAgentAcceptsExistingAgent(t *testing.T) {
	agentID := 42
	service := (&ScanCreateService{}).WithAgentLookup(scanCreateAgentLookupStub{exists: true}).WithAgentExecutionReader(&scanAgentExecutionReaderStub{eligibility: agentExecutionEligible})
	if err := service.validateSelectedAgent(context.Background(), &agentID); err != nil {
		t.Fatalf("existing Agent validation error = %v", err)
	}
}

type scanAgentExecutionReaderStub struct {
	eligibility string
	err         error
	calls       int
}

func (stub *scanAgentExecutionReaderStub) AgentExecutionEligibility(context.Context, int) (string, error) {
	stub.calls++
	return stub.eligibility, stub.err
}

func TestValidateSelectedAgentRejectsRestrictedAndUnavailableEntitlement(t *testing.T) {
	agentID := 42
	failure := errors.New("license database unavailable")
	for _, test := range []struct {
		name   string
		reader ScanAgentExecutionReader
		want   error
	}{
		{name: "restricted", reader: &scanAgentExecutionReaderStub{eligibility: agentExecutionLicenseRestricted}, want: ErrCreateAgentLicenseRestricted},
		{name: "unavailable", reader: &scanAgentExecutionReaderStub{err: failure}, want: failure},
		{name: "missing reader"},
		{name: "unknown eligibility", reader: &scanAgentExecutionReaderStub{eligibility: "unknown"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := (&ScanCreateService{}).WithAgentLookup(scanCreateAgentLookupStub{exists: true}).WithAgentExecutionReader(test.reader)
			err := service.validateSelectedAgent(context.Background(), &agentID)
			if err == nil || (test.want != nil && !errors.Is(err, test.want)) {
				t.Fatalf("validation = %v, want fail closed with %v", err, test.want)
			}
		})
	}
}

func TestRestrictedPinnedScanRejectsBeforeAnyTargetSideEffect(t *testing.T) {
	agentID := 42
	targetCalls := 0
	service := NewScanCreateService(nil,
		func(context.Context, int) (*TargetRef, error) {
			targetCalls++
			return nil, nil
		},
		func(context.Context, []string) (*QuickTargetResolution, error) {
			targetCalls++
			return nil, nil
		}, nil, nil,
	).WithAgentLookup(scanCreateAgentLookupStub{exists: true}).
		WithAgentExecutionReader(&scanAgentExecutionReaderStub{eligibility: agentExecutionLicenseRestricted})
	if _, err := service.CreateQuick(context.Background(), &CreateQuickInput{
		Targets: []string{"example.com"}, AgentID: &agentID,
		InputSource: InputSourceScanSnapshot, TriggerType: ScanTriggerTypeManual,
	}); !errors.Is(err, ErrCreateAgentLicenseRestricted) {
		t.Fatalf("quick rejection = %v", err)
	}
	if _, err := service.CreateBatch(context.Background(), &CreateBatchInput{
		Requests: []CreateBatchItem{{TargetID: 7}}, AgentID: &agentID,
		InputSource: InputSourceScanSnapshot, TriggerType: ScanTriggerTypeScheduled,
	}); !errors.Is(err, ErrCreateAgentLicenseRestricted) {
		t.Fatalf("batch/scheduled rejection = %v", err)
	}
	if targetCalls != 0 {
		t.Fatalf("restricted create reached Target side effects %d times", targetCalls)
	}
}
