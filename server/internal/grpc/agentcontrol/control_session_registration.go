package agentcontrol

import (
	"context"
	"errors"
	"sync"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *ControlPlaneService) registerSession(
	ctx context.Context,
	agent *agentdomain.Agent,
	connectionIP string,
	streamID uint64,
	sendMutex *sync.Mutex,
	stream grpc.BidiStreamingServer[agentcontrolv1.ConnectRequest, agentcontrolv1.ConnectResponse],
	payload *agentcontrolv1.RegisterSession,
) (ActiveControlSession, error) {
	if err := validateRegisterSession(agent, payload); err != nil {
		return ActiveControlSession{}, status.Error(codes.InvalidArgument, err.Error())
	}
	registrationEvent, err := toAgentRegistrationHeartbeatEvent(payload)
	if err != nil {
		return ActiveControlSession{}, status.Error(codes.InvalidArgument, err.Error())
	}
	if s.sessions == nil {
		return ActiveControlSession{}, status.Error(codes.Internal, "active session registry is not initialized")
	}

	// payload.session identifies the logical session of the current agent process instance; streamID and sessionEpoch fence individual transport connections separately.
	_, sessionID, err := resourcenames.ParseAgentSession(payload.GetSession())
	if err != nil {
		return ActiveControlSession{}, status.Error(codes.InvalidArgument, err.Error())
	}
	s.connectionLifecycleMu.Lock()
	s.sessions.RestorePersistedSession(agent.ID, agent.SessionID, agent.SessionEpoch, agent.LastHeartbeat)
	activeSession, replacedSession := s.sessions.Register(agent.ID, streamID, sessionID)
	if s.streams != nil {
		s.streams.ClearActiveStream(agent.ID, activeSession.StreamID)
		if replacedSession != nil {
			s.streams.ClearActiveStream(agent.ID, replacedSession.StreamID)
		}
	}
	logSessionRegistration(agent.ID, activeSession, replacedSession)

	// Roll back the provisional registration only if this stream is still current.
	// A newer reconnect may have already replaced it by the time cleanup runs.
	rollbackRegistration := func() {
		if s.streams != nil {
			s.streams.ClearActiveStream(agent.ID, activeSession.StreamID)
		}
		s.sessions.DetachStreamIfCurrent(agent.ID, activeSession.SessionID, activeSession.SessionEpoch, activeSession.StreamID)
	}

	if err := s.lifecycle.OnConnected(ctx, agent, connectionIP); err != nil {
		s.connectionLifecycleMu.Unlock()
		rollbackRegistration()
		return ActiveControlSession{}, status.Error(codes.Internal, err.Error())
	}
	s.connectionLifecycleMu.Unlock()
	registrationEvent.SessionEpoch = activeSession.SessionEpoch
	if err := s.lifecycle.RecordHeartbeat(ctx, agent.ID, registrationEvent); err != nil {
		rollbackRegistration()
		if errors.Is(err, agentdomain.ErrStaleAgentHeartbeat) {
			return ActiveControlSession{}, status.Error(codes.FailedPrecondition, "agent control session was superseded during registration")
		}
		return ActiveControlSession{}, status.Error(codes.Internal, err.Error())
	}
	if !s.sessions.UpdateExecutionSnapshotIfCurrent(agent.ID, activeSession.SessionID, activeSession.SessionEpoch, activeSession.StreamID, executionCapabilityFromHeartbeat(registrationEvent)) {
		rollbackRegistration()
		return ActiveControlSession{}, status.Error(codes.Internal, "agent execution capability snapshot could not be attached to session")
	}
	if s.sessionFences == nil {
		rollbackRegistration()
		return ActiveControlSession{}, status.Error(codes.Unimplemented, "Agent session fence bridge is not initialized")
	}
	if err := s.sessionFences.FenceSupersededAgentSession(ctx, agent.ID, activeSession.SessionEpoch); err != nil {
		rollbackRegistration()
		return ActiveControlSession{}, status.Error(codes.Internal, err.Error())
	}

	if err := sendControlEvent(sendMutex, stream, &agentcontrolv1.ConnectResponse{
		Payload: &agentcontrolv1.ConnectResponse_SessionReady{
			SessionReady: &agentcontrolv1.SessionReady{
				SessionEpoch:          activeSession.SessionEpoch,
				ConfigSnapshot:        toConfigUpdate(agent),
				CompatibilityRevision: engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
			},
		},
	}); err != nil {
		rollbackRegistration()
		return ActiveControlSession{}, err
	}
	readySession, ok := s.sessions.MarkReadyIfCurrent(agent.ID, activeSession.SessionID, activeSession.SessionEpoch, activeSession.StreamID)
	if !ok {
		rollbackRegistration()
		return ActiveControlSession{}, status.Error(codes.Internal, "agent control session could not transition to ready state")
	}
	activeSession = readySession
	if s.streams != nil {
		if activeSession.IsDownlinkEligible() {
			s.streams.SetActiveStream(agent.ID, activeSession.StreamID)
		}
	}
	if s.readyObserver != nil {
		s.readyObserver.ObserveReadyConnection(agent)
	}
	return activeSession, nil
}
