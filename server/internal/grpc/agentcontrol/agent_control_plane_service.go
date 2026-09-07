// Package agentcontrol provides server-side agent control-plane gRPC handlers and support types.
package agentcontrol

import (
	"context"
	"errors"
	"io"
	"sync"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/planeauth"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AgentFinder interface {
	FindByAuthenticationToken(ctx context.Context, authenticationToken string) (*agentdomain.Agent, error)
}

type AgentControlLifecycle interface {
	OnConnected(ctx context.Context, agent *agentdomain.Agent, connectionIP string) error
	OnControlConnectionDetached(ctx context.Context, agentID int) error
	OnDisconnected(ctx context.Context, agentID int) error
	RecordHeartbeat(ctx context.Context, agentID int, event agentdomain.AgentHeartbeatEvent) error
}

type ScanTaskBridge interface {
	ReportTerminalTaskResult(ctx context.Context, agentID int, sessionID string, sessionEpoch int64, taskID int, result string, failure *scanapp.FailureDetail) error
}

type EngineDiagnosticScanTaskBridge interface {
	ReportTerminalTaskResultWithDiagnostics(ctx context.Context, agentID int, sessionID string, sessionEpoch int64, taskID int, result string, failure *scanapp.FailureDetail, diagnostics *scanapp.EngineExecutionDiagnostics) error
}

type EngineExecutionClaimBridge interface {
	ClaimNextExecutionPlan(ctx context.Context, agentID int, sessionID string, sessionEpoch int64, requestID string, snapshot agentdomain.AgentExecutionCapabilitySnapshot) (*agentexecutionv1.ResolvedEngineExecutionPlan, error)
}

type AgentSessionFenceBridge interface {
	FenceSupersededAgentSession(ctx context.Context, agentID int, currentSessionEpoch int64) error
}

// AgentReadyConnectionObserver receives only successfully persisted and ready connection observations.
type AgentReadyConnectionObserver interface {
	ObserveReadyConnection(agent *agentdomain.Agent)
}

type ControlPlaneService struct {
	agentcontrolv1.UnimplementedControlPlaneServiceServer

	agentFinder          AgentFinder
	lifecycle            AgentControlLifecycle
	taskBridge           ScanTaskBridge
	diagnosticTaskBridge EngineDiagnosticScanTaskBridge
	executionClaims      EngineExecutionClaimBridge
	sessionFences        AgentSessionFenceBridge
	streams              *AgentStreamRegistry
	sessions             *ActiveSessionRegistry
	sourceResolver       *ConnectionSourceResolver
	readyObserver        AgentReadyConnectionObserver
	// Serializes persistence with active-session replacement. Without this,
	// a stale stream's EOF can clear the address written by its replacement.
	connectionLifecycleMu sync.Mutex
}

func NewControlPlaneService(agentFinder AgentFinder, lifecycle AgentControlLifecycle, taskBridge ScanTaskBridge, streams ...*AgentStreamRegistry) *ControlPlaneService {
	service := &ControlPlaneService{
		agentFinder:    agentFinder,
		lifecycle:      lifecycle,
		taskBridge:     taskBridge,
		streams:        NewAgentStreamRegistry(),
		sessions:       NewActiveSessionRegistry(),
		sourceResolver: NewConnectionSourceResolver(),
	}
	if claims, ok := taskBridge.(EngineExecutionClaimBridge); ok {
		service.executionClaims = claims
	}
	if diagnostics, ok := taskBridge.(EngineDiagnosticScanTaskBridge); ok {
		service.diagnosticTaskBridge = diagnostics
	}
	if fences, ok := taskBridge.(AgentSessionFenceBridge); ok {
		service.sessionFences = fences
	}
	if len(streams) > 0 {
		service.streams = streams[0]
	}
	return service
}

func (s *ControlPlaneService) WithConnectionSourceResolver(resolver *ConnectionSourceResolver) *ControlPlaneService {
	if s != nil && resolver != nil {
		s.sourceResolver = resolver
	}
	return s
}

// WithReadyConnectionObserver attaches best-effort work that starts only after SessionReady.
func (s *ControlPlaneService) WithReadyConnectionObserver(observer AgentReadyConnectionObserver) *ControlPlaneService {
	if s != nil {
		s.readyObserver = observer
	}
	return s
}

// WithActiveSessionRegistry shares the authoritative ready-session registry
// with Server-side data-plane authorization. It must be applied before Serve.
func (s *ControlPlaneService) WithActiveSessionRegistry(registry *ActiveSessionRegistry) *ControlPlaneService {
	if s != nil && registry != nil {
		s.sessions = registry
	}
	return s
}

func (s *ControlPlaneService) Connect(stream grpc.BidiStreamingServer[agentcontrolv1.ConnectRequest, agentcontrolv1.ConnectResponse]) error {
	if s.agentFinder == nil || s.lifecycle == nil || s.taskBridge == nil {
		return status.Error(codes.Unimplemented, errControlPlaneUnimplemented)
	}

	ctx := stream.Context()
	authenticationToken, ok := grpcauth.ReadAgentAuthenticationToken(ctx)
	if !ok {
		return grpcauth.MapError(grpcauth.ErrInvalidAgentAuthenticationToken)
	}

	agent, err := s.agentFinder.FindByAuthenticationToken(ctx, authenticationToken)
	if err != nil || agent == nil {
		return grpcauth.MapError(grpcauth.ErrInvalidAgentAuthenticationToken)
	}

	// Resolve this only after token admission. A direct caller can forge the
	// value, so it is persisted exclusively as a display observation.
	connectionIP := s.sourceResolver.Resolve(ctx)
	var unregisterStream func()
	var streamID uint64
	// sendMutex ensures thread-safe writes to the gRPC stream when multiple goroutines
	// (e.g., the main loop and the event forwarder) attempt to send events simultaneously.
	sendMutex := &sync.Mutex{}
	if s.streams != nil {
		registeredStreamID, outbound, unregister := s.streams.RegisterStream(agent.ID)
		streamID = registeredStreamID
		unregisterStream = unregister
		go s.forwardOutboundEvents(ctx, sendMutex, stream, outbound)
	}
	if unregisterStream != nil {
		defer unregisterStream()
	}

	var session ActiveControlSession
	defer func() {
		if !session.IsRegistered() {
			return
		}
		if s.streams != nil {
			s.streams.ClearActiveStream(agent.ID, session.StreamID)
		}
		s.detachCurrentConnection(ctx, agent.ID, session)
	}()

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			zap.L().Warn("agent_control_stream_receive_failed", zap.Int("agent.id", agent.ID), zap.Error(err))
			return err
		}
		if req == nil {
			continue
		}

		switch payload := req.GetPayload().(type) {
		case *agentcontrolv1.ConnectRequest_RegisterSession:
			activeSession, err := s.handleRegisterSession(ctx, agent, connectionIP, streamID, sendMutex, stream, session, payload.RegisterSession)
			if err != nil {
				zap.L().Warn("agent_control_register_session_failed", zap.Int("agent.id", agent.ID), zap.Error(err))
				return err
			}
			session = activeSession
		case *agentcontrolv1.ConnectRequest_Heartbeat:
			if err := s.handleHeartbeat(ctx, agent, session, payload.Heartbeat); err != nil {
				zap.L().Warn("agent_control_heartbeat_failed", zap.Int("agent.id", agent.ID), zap.Error(err))
				return err
			}
		case *agentcontrolv1.ConnectRequest_RequestTask:
			if err := s.handleExecutionPlanRequest(ctx, agent, session, sendMutex, stream, payload.RequestTask); err != nil {
				zap.L().Warn("agent_control_request_task_failed", zap.Int("agent.id", agent.ID), zap.Error(err))
				return err
			}
		case *agentcontrolv1.ConnectRequest_TerminalTaskResult:
			if err := s.handleTerminalTaskResult(ctx, agent, session, sendMutex, stream, payload.TerminalTaskResult); err != nil {
				zap.L().Warn("agent_control_terminal_result_failed", zap.Int("agent.id", agent.ID), zap.Error(err))
				return err
			}
		default:
			// Ignore unknown payload variants for forward compatibility.
		}
	}
}

func (s *ControlPlaneService) detachCurrentConnection(ctx context.Context, agentID int, session ActiveControlSession) {
	s.connectionLifecycleMu.Lock()
	defer s.connectionLifecycleMu.Unlock()

	if s.sessions == nil || !s.sessions.DetachStreamIfCurrent(agentID, session.SessionID, session.SessionEpoch, session.StreamID) {
		zap.L().Info("disconnect_ignored",
			zap.Int("agent.id", agentID),
			zap.String("session.id", session.SessionID),
			zap.Int64("session.epoch", session.SessionEpoch),
		)
		return
	}
	if err := s.lifecycle.OnControlConnectionDetached(context.WithoutCancel(ctx), agentID); err != nil {
		zap.L().Warn("agent_connection_address_clear_failed", zap.Int("agent.id", agentID), zap.Error(err))
	}
}

// forwardOutboundEvents bridges async server-push events from registry channels
// to the active stream. It exits on context cancellation or first send error;
// higher layers handle reconnect/resubscribe semantics.
func (s *ControlPlaneService) forwardOutboundEvents(
	ctx context.Context,
	sendMutex *sync.Mutex,
	stream grpc.BidiStreamingServer[agentcontrolv1.ConnectRequest, agentcontrolv1.ConnectResponse],
	outbound <-chan *agentcontrolv1.ConnectResponse,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-outbound:
			if event == nil {
				continue
			}
			if err := sendControlEvent(sendMutex, stream, event); err != nil {
				return
			}
		}
	}
}
