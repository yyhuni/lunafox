package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type residentAgentReadinessStoreStub struct {
	instanceID string
	agent      *agentdomain.Agent
	bindingErr error
	agentErr   error
}

func (stub residentAgentReadinessStoreStub) boundInstanceID(context.Context) (string, error) {
	return stub.instanceID, stub.bindingErr
}

func (stub residentAgentReadinessStoreStub) findAgent(context.Context, string) (*agentdomain.Agent, error) {
	return stub.agent, stub.agentErr
}

func readyAgent(now time.Time, version string) *agentdomain.Agent {
	heartbeat := now.Add(-5 * time.Second)
	return &agentdomain.Agent{
		ID:                       3,
		InstanceID:               "instance-3",
		DisplayName:              "lunafox-agent",
		Status:                   "online",
		HealthState:              "healthy",
		AgentVersion:             version,
		OperatingSystem:          "linux",
		Architecture:             "arm64",
		ContainerRuntimeReady:    true,
		SupportedEngineAPIMajors: []uint32{5},
		SessionID:                "session-3",
		SessionEpoch:             7,
		LastHeartbeat:            &heartbeat,
		AuthenticationToken:      "must-not-leak",
	}
}

// The lifecycle probe must accept an Agent that can claim work today, even when
// its version is not the version a pending Upgrade Operation targets.
func TestResidentAgentReadinessIgnoresUpgradeTargetVersion(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	store := residentAgentReadinessStoreStub{instanceID: "instance-3", agent: readyAgent(now, "1.4.0")}
	readiness, err := residentAgentReadiness(context.Background(), store, now)
	if err != nil {
		t.Fatal(err)
	}
	if !readiness.Ready() || readiness.InstanceID != "instance-3" || readiness.Diagnostic != "" {
		t.Fatalf("readiness=%#v", readiness)
	}
	if !strings.Contains(readiness.Describe(), "claim-ready") {
		t.Fatalf("description=%q", readiness.Describe())
	}
}

func TestResidentAgentReadinessClassifiesUnavailableAgents(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	stale := now.Add(-4 * time.Minute)
	cases := []struct {
		name           string
		store          residentAgentReadinessStoreStub
		wantReady      bool
		wantDiagnostic string
	}{
		{
			name:           "missing binding",
			store:          residentAgentReadinessStoreStub{},
			wantDiagnostic: "no Agent binding",
		},
		{
			name:           "bound instance is not registered",
			store:          residentAgentReadinessStoreStub{instanceID: "instance-9"},
			wantDiagnostic: "not registered",
		},
		{
			name: "stale heartbeat",
			store: residentAgentReadinessStoreStub{instanceID: "instance-3", agent: func() *agentdomain.Agent {
				agent := readyAgent(now, "1.4.0")
				agent.LastHeartbeat = &stale
				return agent
			}()},
			wantDiagnostic: "heartbeat is missing or stale",
		},
		{
			name: "paused agent",
			store: residentAgentReadinessStoreStub{instanceID: "instance-3", agent: func() *agentdomain.Agent {
				agent := readyAgent(now, "1.4.0")
				agent.HealthState = "paused"
				return agent
			}()},
			wantDiagnostic: "paused",
		},
		{
			name: "runtime cannot accept work",
			store: residentAgentReadinessStoreStub{instanceID: "instance-3", agent: func() *agentdomain.Agent {
				agent := readyAgent(now, "1.4.0")
				agent.ContainerRuntimeReady = false
				return agent
			}()},
			wantDiagnostic: "not ready to claim work",
		},
		{
			name: "no runtime session",
			store: residentAgentReadinessStoreStub{instanceID: "instance-3", agent: func() *agentdomain.Agent {
				agent := readyAgent(now, "1.4.0")
				agent.SessionID = ""
				agent.SessionEpoch = 0
				return agent
			}()},
			wantDiagnostic: "not ready to claim work",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			readiness, err := residentAgentReadiness(context.Background(), testCase.store, now)
			if err != nil {
				t.Fatal(err)
			}
			if readiness.Ready() {
				t.Fatalf("unexpected ready result: %#v", readiness)
			}
			if !strings.Contains(readiness.Diagnostic, testCase.wantDiagnostic) {
				t.Fatalf("diagnostic=%q want substring %q", readiness.Diagnostic, testCase.wantDiagnostic)
			}
			for _, secret := range []string{"must-not-leak"} {
				if strings.Contains(readiness.Describe(), secret) {
					t.Fatalf("description leaked a credential: %q", readiness.Describe())
				}
			}
		})
	}
}

func TestResidentAgentReadinessReportsStoreFailures(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	if _, err := residentAgentReadiness(context.Background(), residentAgentReadinessStoreStub{bindingErr: errors.New("database offline")}, now); err == nil {
		t.Fatal("binding failure was accepted")
	}
	if _, err := residentAgentReadiness(context.Background(), residentAgentReadinessStoreStub{instanceID: "instance-3", agentErr: errors.New("query failed")}, now); err == nil {
		t.Fatal("agent lookup failure was accepted")
	}
}
