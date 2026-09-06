package agentcontrol

import (
	"context"
	"errors"
	"sync"
	"testing"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type erroringConnectStream struct {
	ctx     context.Context
	sendErr error
}

func (s *erroringConnectStream) Context() context.Context { return s.ctx }
func (s *erroringConnectStream) Send(*agentcontrolv1.ConnectResponse) error {
	return s.sendErr
}
func (s *erroringConnectStream) Recv() (*agentcontrolv1.ConnectRequest, error) { return nil, nil }
func (s *erroringConnectStream) SetHeader(metadata.MD) error                   { return nil }
func (s *erroringConnectStream) SendHeader(metadata.MD) error                  { return nil }
func (s *erroringConnectStream) SetTrailer(metadata.MD)                        {}
func (s *erroringConnectStream) SendMsg(any) error                             { return nil }
func (s *erroringConnectStream) RecvMsg(any) error                             { return nil }

func TestRegisterSessionRejectsInvalidPayload(t *testing.T) {
	svc := NewControlPlaneService(nil, &runtimeLifecycleStub{}, nil)

	_, err := svc.registerSession(
		context.Background(),
		&agentdomain.Agent{ID: 1, InstanceID: "agt-1"},
		"",
		1,
		&sync.Mutex{},
		&fakeConnectStream{ctx: context.Background()},
		&agentcontrolv1.RegisterSession{},
	)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestRegisterSessionRejectsNilRegistry(t *testing.T) {
	svc := &ControlPlaneService{
		lifecycle: &runtimeLifecycleStub{},
		streams:   NewAgentStreamRegistry(),
		sessions:  nil,
	}

	_, err := svc.registerSession(
		context.Background(),
		&agentdomain.Agent{ID: 1, InstanceID: "agt-1"},
		"",
		1,
		&sync.Mutex{},
		&fakeConnectStream{ctx: context.Background()},
		runtimeRegisterSessionRequest("agt-1", "session-1").GetRegisterSession(),
	)
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal, got %v", err)
	}
}

func TestRegisterSessionClearsActiveStreamWhenSendFails(t *testing.T) {
	streams := NewAgentStreamRegistry()
	lifecycle := &runtimeLifecycleStub{}
	svc := &ControlPlaneService{
		lifecycle:     lifecycle,
		streams:       streams,
		sessions:      NewActiveSessionRegistry(),
		sessionFences: &taskRuntimeStub{},
	}
	agent := &agentdomain.Agent{ID: 8, InstanceID: "agt-8"}
	stream := &erroringConnectStream{ctx: context.Background(), sendErr: errors.New("send failed")}

	_, err := svc.registerSession(
		context.Background(),
		agent,
		"",
		88,
		&sync.Mutex{},
		stream,
		runtimeRegisterSessionRequest("agt-8", "session-8").GetRegisterSession(),
	)
	if !errors.Is(err, stream.sendErr) {
		t.Fatalf("expected send error, got %v", err)
	}
	if lifecycle.connectedCount != 1 {
		t.Fatalf("expected lifecycle OnConnected before send failure, got %d", lifecycle.connectedCount)
	}
	if streams.activeStreamByAgent[agent.ID] != 0 {
		t.Fatalf("expected send failure to clear active stream, got active=%d", streams.activeStreamByAgent[agent.ID])
	}
	if svc.sessions.MatchesCurrentSession(agent.ID, "session-8", 1, 88) {
		t.Fatal("expected send failure to detach active control session stream")
	}
	detached := svc.sessions.sessions[agent.ID]
	if detached.Phase != ControlSessionPhaseDetachedRecoverable {
		t.Fatalf("expected send failure to leave detached recoverable session, got %q", detached.Phase)
	}
	if detached.IsDownlinkEligible() {
		t.Fatal("expected send failure not to leave downlink-eligible session")
	}
}

func TestRegisterSessionClearsActiveStreamWhenOnConnectedFails(t *testing.T) {
	streams := NewAgentStreamRegistry()
	lifecycle := &runtimeLifecycleStub{onConnectedErr: errors.New("connect failed")}
	svc := &ControlPlaneService{
		lifecycle:     lifecycle,
		streams:       streams,
		sessions:      NewActiveSessionRegistry(),
		sessionFences: &taskRuntimeStub{},
	}
	agent := &agentdomain.Agent{ID: 18, InstanceID: "agt-18"}
	stream := &fakeConnectStream{ctx: context.Background()}

	_, err := svc.registerSession(
		context.Background(),
		agent,
		"",
		188,
		&sync.Mutex{},
		stream,
		runtimeRegisterSessionRequest("agt-18", "session-18").GetRegisterSession(),
	)
	if status.Code(err) != codes.Internal || err.Error() != "rpc error: code = Internal desc = connect failed" {
		t.Fatalf("expected internal connect failure, got %v", err)
	}
	if lifecycle.connectedCount != 0 {
		t.Fatalf("expected failed OnConnected not to increment connect count, got %d", lifecycle.connectedCount)
	}
	if streams.activeStreamByAgent[agent.ID] != 0 {
		t.Fatalf("expected OnConnected failure to clear active stream, got active=%d", streams.activeStreamByAgent[agent.ID])
	}
	if svc.sessions.MatchesCurrentSession(agent.ID, "session-18", 1, 188) {
		t.Fatal("expected OnConnected failure to detach active control session stream")
	}
	if len(stream.sent) != 0 {
		t.Fatalf("expected no session_ready event on OnConnected failure, got %+v", stream.sent)
	}
	detached := svc.sessions.sessions[agent.ID]
	if detached.Phase != ControlSessionPhaseDetachedRecoverable {
		t.Fatalf("expected OnConnected failure to leave detached recoverable session, got %q", detached.Phase)
	}
	if detached.IsDownlinkEligible() {
		t.Fatal("expected OnConnected failure not to leave downlink-eligible session")
	}
}

func TestRegisterSessionMarksSessionReadyAndDownlinkEligible(t *testing.T) {
	streams := NewAgentStreamRegistry()
	streamID, _, unregister := streams.RegisterStream(28)
	defer unregister()
	lifecycle := &runtimeLifecycleStub{}
	svc := &ControlPlaneService{
		lifecycle:     lifecycle,
		streams:       streams,
		sessions:      NewActiveSessionRegistry(),
		sessionFences: &taskRuntimeStub{},
	}
	agent := &agentdomain.Agent{ID: 28, InstanceID: "agt-28"}
	stream := &fakeConnectStream{ctx: context.Background()}

	session, err := svc.registerSession(
		context.Background(),
		agent,
		"",
		streamID,
		&sync.Mutex{},
		stream,
		runtimeRegisterSessionRequest("agt-28", "session-28").GetRegisterSession(),
	)
	if err != nil {
		t.Fatalf("expected registerSession success, got %v", err)
	}
	if session.Phase != ControlSessionPhaseReadyAttached {
		t.Fatalf("expected ready attached session, got %q", session.Phase)
	}
	if !session.IsDownlinkEligible() {
		t.Fatal("expected ready attached session to be downlink eligible")
	}
	if streams.activeStreamByAgent[agent.ID] != session.StreamID {
		t.Fatalf("expected successful registerSession to activate stream %d, got %d", session.StreamID, streams.activeStreamByAgent[agent.ID])
	}
}

func TestRegisterSessionObservesLocationOnlyAfterReadyTransition(t *testing.T) {
	observer := &readyConnectionObserverStub{}
	lifecycle := &runtimeLifecycleStub{}
	svc := NewControlPlaneService(nil, lifecycle, &taskRuntimeStub{}).WithReadyConnectionObserver(observer)
	agent := &agentdomain.Agent{ID: 29, InstanceID: "agt-29", ObservedSourceIP: "8.8.8.8", ObservedIPGeneration: 4}
	stream := &fakeConnectStream{ctx: context.Background()}

	session, err := svc.registerSession(
		context.Background(),
		agent,
		"8.8.8.8",
		29,
		&sync.Mutex{},
		stream,
		runtimeRegisterSessionRequest("agt-29", "session-29").GetRegisterSession(),
	)
	if err != nil {
		t.Fatalf("registerSession: %v", err)
	}
	if session.Phase != ControlSessionPhaseReadyAttached || len(stream.sent) != 1 || stream.sent[0].GetSessionReady() == nil {
		t.Fatalf("observer ran without a ready session: session=%#v events=%#v", session, stream.sent)
	}
	if len(observer.agents) != 1 || observer.agents[0] != agent {
		t.Fatalf("ready observations = %#v, want persisted Agent", observer.agents)
	}
}

func TestRegisterSessionDoesNotObserveLocationWhenSessionReadySendFails(t *testing.T) {
	observer := &readyConnectionObserverStub{}
	sendErr := errors.New("send failed")
	svc := NewControlPlaneService(nil, &runtimeLifecycleStub{}, &taskRuntimeStub{}).WithReadyConnectionObserver(observer)
	agent := &agentdomain.Agent{ID: 30, InstanceID: "agt-30", ObservedSourceIP: "8.8.8.8", ObservedIPGeneration: 1}

	_, err := svc.registerSession(
		context.Background(),
		agent,
		"8.8.8.8",
		30,
		&sync.Mutex{},
		&erroringConnectStream{ctx: context.Background(), sendErr: sendErr},
		runtimeRegisterSessionRequest("agt-30", "session-30").GetRegisterSession(),
	)
	if !errors.Is(err, sendErr) {
		t.Fatalf("registerSession error = %v, want %v", err, sendErr)
	}
	if len(observer.agents) != 0 {
		t.Fatalf("failed session emitted location observation: %#v", observer.agents)
	}
}

type readyConnectionObserverStub struct {
	agents []*agentdomain.Agent
}

func (observer *readyConnectionObserverStub) ObserveReadyConnection(agent *agentdomain.Agent) {
	observer.agents = append(observer.agents, agent)
}

func TestAgentControlPlaneMatchesCurrentSessionWithNilRegistry(t *testing.T) {
	svc := &ControlPlaneService{}

	if svc.matchesCurrentSession(1, ActiveControlSession{SessionID: "session-1", SessionEpoch: 1, StreamID: 1}) {
		t.Fatal("expected nil session registry not to match current session")
	}
}

func TestMapScanTaskBridgeErrorNil(t *testing.T) {
	if err := mapScanTaskBridgeError(nil); err != nil {
		t.Fatalf("expected nil error to remain nil, got %v", err)
	}
}

func TestValidateHeartbeatIdentity(t *testing.T) {
	testCases := []struct {
		name    string
		agent   *agentdomain.Agent
		payload *agentcontrolv1.Heartbeat
		wantErr string
	}{
		{name: "nil agent", payload: &agentcontrolv1.Heartbeat{Agent: resourcenames.Agent("agt-1"), CompatibilityRevision: "engine-execution-diagnostics-r1"}, wantErr: "agent is required"},
		{name: "nil payload", agent: &agentdomain.Agent{InstanceID: "agt-1"}, wantErr: "heartbeat payload is required"},
		{name: "blank payload agent", agent: &agentdomain.Agent{InstanceID: "agt-1"}, payload: &agentcontrolv1.Heartbeat{CompatibilityRevision: "engine-execution-diagnostics-r1"}, wantErr: "heartbeat agent is invalid: resource name must use agents/{resource}"},
		{name: "blank agent instance", agent: &agentdomain.Agent{}, payload: &agentcontrolv1.Heartbeat{Agent: resourcenames.Agent("agt-1"), CompatibilityRevision: "engine-execution-diagnostics-r1"}, wantErr: "agent instance_id is required"},
		{name: "mismatched instance", agent: &agentdomain.Agent{InstanceID: "agt-1"}, payload: &agentcontrolv1.Heartbeat{Agent: resourcenames.Agent("agt-2"), CompatibilityRevision: "engine-execution-diagnostics-r1"}, wantErr: "heartbeat instance_id does not match authenticated agent"},
		{name: "success", agent: &agentdomain.Agent{InstanceID: "agt-1"}, payload: &agentcontrolv1.Heartbeat{Agent: resourcenames.Agent("agt-1"), CompatibilityRevision: "engine-execution-diagnostics-r1"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateHeartbeatIdentity(tc.agent, tc.payload)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected success, got %v", err)
				}
				return
			}
			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("expected error %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestValidateRegisterSession(t *testing.T) {
	validPayload := runtimeRegisterSessionRequest("agt-1", "session-1").GetRegisterSession()

	testCases := []struct {
		name    string
		agent   *agentdomain.Agent
		payload *agentcontrolv1.RegisterSession
		wantErr string
	}{
		{name: "nil agent", payload: validPayload, wantErr: "agent is required"},
		{name: "nil payload", agent: &agentdomain.Agent{InstanceID: "agt-1"}, wantErr: "register_session payload is required"},
		{name: "blank payload agent", agent: &agentdomain.Agent{InstanceID: "agt-1"}, payload: &agentcontrolv1.RegisterSession{CompatibilityRevision: "engine-execution-diagnostics-r1"}, wantErr: "register_session agent is invalid: resource name must use agents/{resource}"},
		{name: "blank agent instance", agent: &agentdomain.Agent{}, payload: validPayload, wantErr: "agent instance_id is required"},
		{name: "mismatched instance", agent: &agentdomain.Agent{InstanceID: "agt-1"}, payload: runtimeRegisterSessionRequest("agt-2", "session-1").GetRegisterSession(), wantErr: "register_session instance_id does not match authenticated agent"},
		{name: "missing session", agent: &agentdomain.Agent{InstanceID: "agt-1"}, payload: &agentcontrolv1.RegisterSession{Agent: resourcenames.Agent("agt-1"), ObservedHostname: "node-a", AgentVersion: "v1", CompatibilityRevision: "engine-execution-diagnostics-r1"}, wantErr: "register_session session is invalid: agent session name must use agents/{agent}/sessions/{session}"},
		{name: "missing observed hostname", agent: &agentdomain.Agent{InstanceID: "agt-1"}, payload: &agentcontrolv1.RegisterSession{Agent: resourcenames.Agent("agt-1"), Session: resourcenames.AgentSession("agt-1", "session-1"), AgentVersion: "v1", CompatibilityRevision: "engine-execution-diagnostics-r1"}, wantErr: "register_session observed_hostname is required"},
		{name: "missing agent version", agent: &agentdomain.Agent{InstanceID: "agt-1"}, payload: &agentcontrolv1.RegisterSession{Agent: resourcenames.Agent("agt-1"), Session: resourcenames.AgentSession("agt-1", "session-1"), ObservedHostname: "node-a", CompatibilityRevision: "engine-execution-diagnostics-r1"}, wantErr: "register_session agent_version is required"},
		{name: "success", agent: &agentdomain.Agent{InstanceID: "agt-1"}, payload: validPayload},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRegisterSession(tc.agent, tc.payload)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected success, got %v", err)
				}
				return
			}
			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("expected error %q, got %v", tc.wantErr, err)
			}
		})
	}
}
