package service

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"

	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/runtime/auth"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type AgentFinder interface {
	FindByAPIKey(ctx context.Context, apiKey string) (*agentdomain.Agent, error)
}

type RuntimeLifecycle interface {
	OnConnected(ctx context.Context, agent *agentdomain.Agent, ipAddress string) error
	OnDisconnected(ctx context.Context, agentID int) error
	HandleMessage(ctx context.Context, agentID int, message agentapp.RuntimeMessageInput) error
}

type TaskRuntime interface {
	PullTask(ctx context.Context, agentID int) (*scanapp.TaskAssignment, error)
	UpdateStatus(ctx context.Context, agentID, taskID int, status, errorMessage string) error
}

type AgentRuntimeService struct {
	runtimev1.UnimplementedAgentRuntimeServiceServer

	agentFinder AgentFinder
	lifecycle   RuntimeLifecycle
	taskRuntime TaskRuntime
	streams     *AgentStreamRegistry
}

func NewAgentRuntimeService() *AgentRuntimeService {
	return &AgentRuntimeService{}
}

func NewAgentRuntimeServiceWithDeps(agentFinder AgentFinder, lifecycle RuntimeLifecycle, taskRuntime TaskRuntime, streams ...*AgentStreamRegistry) *AgentRuntimeService {
	service := &AgentRuntimeService{
		agentFinder: agentFinder,
		lifecycle:   lifecycle,
		taskRuntime: taskRuntime,
	}
	if len(streams) > 0 {
		service.streams = streams[0]
	}
	return service
}

func (s *AgentRuntimeService) Connect(stream grpc.BidiStreamingServer[runtimev1.AgentRuntimeRequest, runtimev1.AgentRuntimeEvent]) error {
	if s.agentFinder == nil || s.lifecycle == nil || s.taskRuntime == nil {
		return status.Error(codes.Unimplemented, errRuntimeConnectUnimplemented)
	}

	ctx := stream.Context()
	agentKey, ok := grpcauth.ReadIncomingToken(ctx, grpcauth.AgentKeyHeader)
	if !ok {
		return grpcauth.MapError(grpcauth.ErrInvalidAgentKey)
	}

	agent, err := s.agentFinder.FindByAPIKey(ctx, agentKey)
	if err != nil || agent == nil {
		return grpcauth.MapError(grpcauth.ErrInvalidAgentKey)
	}

	if err := s.lifecycle.OnConnected(ctx, agent, runtimePeerIP(ctx)); err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	var unregisterStream func()
	// sendMutex ensures thread-safe writes to the gRPC stream when multiple goroutines
	// (e.g., the main loop and the event forwarder) attempt to send events simultaneously.
	sendMutex := &sync.Mutex{}
	if s.streams != nil {
		outbound, unregister := s.streams.Register(agent.ID)
		unregisterStream = unregister
		go s.forwardOutboundEvents(ctx, sendMutex, stream, outbound)
	}
	if unregisterStream != nil {
		defer unregisterStream()
	}

	defer func() {
		_ = s.lifecycle.OnDisconnected(context.Background(), agent.ID)
	}()

	// Send current scheduler/config snapshot immediately after connect so the agent
	// starts with server-authoritative limits before handling follow-up events.
	if err := sendRuntimeEvent(sendMutex, stream, &runtimev1.AgentRuntimeEvent{
		Payload: &runtimev1.AgentRuntimeEvent_ConfigUpdate{
			ConfigUpdate: toConfigUpdate(agent),
		},
	}); err != nil {
		return err
	}

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if req == nil {
			continue
		}

		switch payload := req.GetPayload().(type) {
		// Handle agent heartbeat to update status and record metrics
		case *runtimev1.AgentRuntimeRequest_Heartbeat:
			if payload.Heartbeat == nil {
				continue
			}
			if err := s.lifecycle.HandleMessage(ctx, agent.ID, toRuntimeHeartbeatInput(payload.Heartbeat)); err != nil {
				return status.Error(codes.Internal, err.Error())
			}
		// Handle task requests initiated by the agent (Pull Model)
		case *runtimev1.AgentRuntimeRequest_RequestTask:
			// Pull an available scan task from the task scheduler
			assignment, err := s.taskRuntime.PullTask(ctx, agent.ID)
			if err != nil {
				if _, ok := scanapp.AsWorkflowError(err); ok {
					if sendErr := sendRuntimeEvent(sendMutex, stream, &runtimev1.AgentRuntimeEvent{
						Payload: &runtimev1.AgentRuntimeEvent_TaskAssign{
							TaskAssign: toTaskAssign(nil),
						},
					}); sendErr != nil {
						return sendErr
					}
					continue
				}
				return mapTaskRuntimeError(err)
			}
			if err := sendRuntimeEvent(sendMutex, stream, &runtimev1.AgentRuntimeEvent{
				Payload: &runtimev1.AgentRuntimeEvent_TaskAssign{
					TaskAssign: toTaskAssign(assignment),
				},
			}); err != nil {
				return err
			}
		// Handle task execution status updates reported by the agent
		case *runtimev1.AgentRuntimeRequest_TaskStatus:
			if payload.TaskStatus == nil {
				continue
			}
			if err := s.taskRuntime.UpdateStatus(
				ctx,
				agent.ID,
				int(payload.TaskStatus.TaskId),
				payload.TaskStatus.Status,
				payload.TaskStatus.Message,
			); err != nil {
				return mapTaskRuntimeError(err)
			}
		default:
			// Ignore unknown payload variants for forward compatibility.
		}
	}
}

func mapTaskRuntimeError(err error) error {
	if err == nil {
		return nil
	}
	if workflowErr, ok := scanapp.AsWorkflowError(err); ok {
		return status.Error(mapWorkflowErrorCode(workflowErr.Code), workflowErr.Error())
	}
	return status.Error(codes.Internal, err.Error())
}

func mapWorkflowErrorCode(code string) codes.Code {
	switch code {
	case scanapp.WorkflowErrorCodeSchemaInvalid:
		return codes.InvalidArgument
	case scanapp.WorkflowErrorCodeWorkflowConfigInvalid,
		scanapp.WorkflowErrorCodeWorkflowPrereqMissing:
		return codes.FailedPrecondition
	default:
		return codes.Internal
	}
}

// forwardOutboundEvents bridges async server-push events from registry channels
// to the active stream. It exits on context cancellation or first send error;
// higher layers handle reconnect/resubscribe semantics.
func (s *AgentRuntimeService) forwardOutboundEvents(
	ctx context.Context,
	sendMutex *sync.Mutex,
	stream grpc.BidiStreamingServer[runtimev1.AgentRuntimeRequest, runtimev1.AgentRuntimeEvent],
	outbound <-chan *runtimev1.AgentRuntimeEvent,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-outbound:
			if event == nil {
				continue
			}
			if err := sendRuntimeEvent(sendMutex, stream, event); err != nil {
				return
			}
		}
	}
}

func runtimePeerIP(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	peerInfo, ok := peer.FromContext(ctx)
	if !ok || peerInfo == nil || peerInfo.Addr == nil {
		return ""
	}
	return normalizeIPAddress(peerInfo.Addr.String())
}

func normalizeIPAddress(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(trimmed)
	if err == nil {
		return parseIPAddress(host)
	}
	return parseIPAddress(trimmed)
}

func parseIPAddress(candidate string) string {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" {
		return ""
	}
	if index := strings.Index(trimmed, "%"); index >= 0 {
		trimmed = trimmed[:index]
	}
	parsed := net.ParseIP(strings.Trim(trimmed, "[]"))
	if parsed == nil {
		return ""
	}
	return parsed.String()
}
