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
	service := (&ScanCreateService{}).WithAgentLookup(scanCreateAgentLookupStub{exists: true})
	if err := service.validateSelectedAgent(context.Background(), &agentID); err != nil {
		t.Fatalf("existing Agent validation error = %v", err)
	}
}
