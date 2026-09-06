package agentcontrol

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/google/uuid"
	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *ControlPlaneService) handleRegisterSession(
	ctx context.Context,
	agent *agentdomain.Agent,
	connectionIP string,
	streamID uint64,
	sendMutex *sync.Mutex,
	stream grpc.BidiStreamingServer[agentcontrolv1.ConnectRequest, agentcontrolv1.ConnectResponse],
	current ActiveControlSession,
	payload *agentcontrolv1.RegisterSession,
) (ActiveControlSession, error) {
	if current.IsRegistered() {
		return ActiveControlSession{}, status.Error(codes.FailedPrecondition, "agent control session already registered")
	}
	return s.registerSession(ctx, agent, connectionIP, streamID, sendMutex, stream, payload)
}

func (s *ControlPlaneService) handleHeartbeat(ctx context.Context, agent *agentdomain.Agent, session ActiveControlSession, payload *agentcontrolv1.Heartbeat) error {
	if err := s.requireCurrentSession(agent.ID, session, "heartbeat"); err != nil {
		return err
	}
	if payload == nil {
		return nil
	}
	if payload.GetCompatibilityRevision() != engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision {
		return status.Error(codes.InvalidArgument, "heartbeat compatibility_revision is invalid")
	}
	if err := validateHeartbeatIdentity(agent, payload); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	event, err := toAgentHeartbeatEvent(payload)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if event.SessionID != session.SessionID {
		return status.Error(codes.InvalidArgument, "heartbeat session does not match authenticated control session")
	}
	event.SessionEpoch = session.SessionEpoch
	if err := s.lifecycle.RecordHeartbeat(ctx, agent.ID, event); err != nil {
		if errors.Is(err, agentdomain.ErrStaleAgentHeartbeat) {
			return staleSessionError(agent.ID, session, "heartbeat persistence")
		}
		return status.Error(codes.Internal, err.Error())
	}
	if !s.sessions.RefreshLastSeenIfCurrent(agent.ID, session.SessionID, session.SessionEpoch, session.StreamID) {
		return staleSessionError(agent.ID, session, "heartbeat")
	}
	snapshot := executionCapabilityFromHeartbeat(event)
	if !s.sessions.UpdateExecutionSnapshotIfCurrent(agent.ID, session.SessionID, session.SessionEpoch, session.StreamID, snapshot) {
		return staleSessionError(agent.ID, session, "heartbeat capability")
	}
	return nil
}

func (s *ControlPlaneService) handleExecutionPlanRequest(
	ctx context.Context,
	agent *agentdomain.Agent,
	session ActiveControlSession,
	sendMutex *sync.Mutex,
	stream grpc.BidiStreamingServer[agentcontrolv1.ConnectRequest, agentcontrolv1.ConnectResponse],
	payload *agentcontrolv1.RequestTask,
) error {
	if err := s.requireCurrentSession(agent.ID, session, "request_task"); err != nil {
		return err
	}
	requestID := ""
	if payload != nil {
		requestID = strings.TrimSpace(payload.GetRequestId())
	}
	parsedRequestID, parseErr := uuid.Parse(requestID)
	if parseErr != nil || parsedRequestID.String() != requestID {
		return status.Error(codes.InvalidArgument, "request_task request_id is required")
	}
	snapshot, rejectionReason := s.sessions.currentExecutionSnapshotIfCurrent(agent.ID, session.SessionID, session.SessionEpoch, session.StreamID)
	if rejectionReason != "" {
		logExecutionClaimDeferred(agent.ID, session, rejectionReason)
		return s.sendTaskAssignIfCurrent(agent.ID, session, sendMutex, stream, requestID, nil)
	}
	if s.executionClaims == nil {
		return status.Error(codes.Unimplemented, "Engine execution claim bridge is not initialized")
	}
	plan, err := s.executionClaims.ClaimNextExecutionPlan(ctx, agent.ID, session.SessionID, session.SessionEpoch, requestID, snapshot)
	if err != nil {
		return mapScanTaskBridgeError(err)
	}
	return s.sendTaskAssignIfCurrent(agent.ID, session, sendMutex, stream, requestID, plan)
}

func (s *ControlPlaneService) handleTerminalTaskResult(
	ctx context.Context,
	agent *agentdomain.Agent,
	session ActiveControlSession,
	sendMutex *sync.Mutex,
	stream grpc.BidiStreamingServer[agentcontrolv1.ConnectRequest, agentcontrolv1.ConnectResponse],
	payload *agentcontrolv1.TerminalTaskResult,
) error {
	if err := s.requireCurrentSession(agent.ID, session, "terminal_task_result"); err != nil {
		return err
	}
	if payload == nil {
		return nil
	}
	if payload.GetCompatibilityRevision() != engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision {
		return status.Error(codes.InvalidArgument, "terminal_task_result compatibility_revision is invalid")
	}
	result, err := fromProtoTerminalTaskResultState(payload.Result)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	_, taskID, err := resourcenames.ParseTask(payload.GetTask())
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	diagnostics, err := toEngineExecutionDiagnostics(payload.GetDiagnostics())
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if s.diagnosticTaskBridge == nil {
		return status.Error(codes.Unimplemented, "Engine diagnostic terminal bridge is not initialized")
	}
	if err := s.diagnosticTaskBridge.ReportTerminalTaskResultWithDiagnostics(
		ctx,
		agent.ID,
		session.SessionID,
		session.SessionEpoch,
		taskID,
		result,
		toFailureDetail(payload.GetFailure()),
		diagnostics,
	); err != nil {
		return mapScanTaskBridgeError(err)
	}
	if err := s.requireCurrentSession(agent.ID, session, "terminal_task_result_ack"); err != nil {
		return err
	}
	return sendControlEvent(sendMutex, stream, &agentcontrolv1.ConnectResponse{
		Payload: &agentcontrolv1.ConnectResponse_TerminalTaskResultAck{
			TerminalTaskResultAck: &agentcontrolv1.TerminalTaskResultAck{
				Task:                  payload.GetTask(),
				SessionEpoch:          session.SessionEpoch,
				CompatibilityRevision: engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
			},
		},
	})
}

func sendTaskAssign(sendMutex *sync.Mutex, stream grpc.BidiStreamingServer[agentcontrolv1.ConnectRequest, agentcontrolv1.ConnectResponse], requestID string, plan *agentexecutionv1.ResolvedEngineExecutionPlan) error {
	assignment := &agentcontrolv1.TaskAssign{RequestId: requestID}
	if plan == nil {
		assignment.Outcome = &agentcontrolv1.TaskAssign_NoTask{NoTask: &agentcontrolv1.NoTask{}}
	} else {
		assignment.Outcome = &agentcontrolv1.TaskAssign_Plan{Plan: plan}
	}
	return sendControlEvent(sendMutex, stream, &agentcontrolv1.ConnectResponse{
		Payload: &agentcontrolv1.ConnectResponse_TaskAssign{TaskAssign: assignment},
	})
}

func (s *ControlPlaneService) sendTaskAssignIfCurrent(
	agentID int,
	session ActiveControlSession,
	sendMutex *sync.Mutex,
	stream grpc.BidiStreamingServer[agentcontrolv1.ConnectRequest, agentcontrolv1.ConnectResponse],
	requestID string,
	plan *agentexecutionv1.ResolvedEngineExecutionPlan,
) error {
	if err := s.requireCurrentSession(agentID, session, "task_assign"); err != nil {
		return err
	}
	return sendTaskAssign(sendMutex, stream, requestID, plan)
}

func (s *ControlPlaneService) requireCurrentSession(agentID int, session ActiveControlSession, action string) error {
	if !session.IsRegistered() {
		return status.Error(codes.FailedPrecondition, "agent control session is not registered")
	}
	if !s.matchesCurrentSession(agentID, session) {
		return staleSessionError(agentID, session, action)
	}
	return nil
}
