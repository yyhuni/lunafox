package agentdata

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/yyhuni/lunafox/contracts/executionartifact"
	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/planeauth"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ExecutionArtifactProducer writes exact artifact bytes with backpressure and
// returns the role-specific record count. Provider content ignores the count.
type ExecutionArtifactProducer func(ctx context.Context, writer io.Writer) (uint64, error)

// AuthorizedExecutionArtifact is returned only after a resolver has verified
// the persisted plan, Agent ownership, active session/lease, operation role,
// execution scope, and exact binding.
type AuthorizedExecutionArtifact struct {
	ContentType       string
	ExpectedIntegrity *agentdatav1.ExecutionArtifactIntegrity
	// Preflight validates type-specific derived input before its stream header.
	Preflight func(context.Context) error
	Produce   ExecutionArtifactProducer

	// Execution-input blacklist telemetry stays Server-private. These callbacks
	// never become an Agent request, stream frame, trailer, or Engine binding.
	executionInputBlacklistStats                func() executionInputBlacklistStats
	executionInputBlacklistTransportInterrupted func()
}

// AgentExecutionLease identifies the authenticated Agent process session that
// opened one data-plane operation. Resolver authorization must match all fields
// against the persisted task lease and current ready control session.
type AgentExecutionLease struct {
	AgentID      int
	SessionID    string
	SessionEpoch int64
}

// ExecutionArtifactResolver keeps authorization roles distinct.
// Implementations may share lower-level plan loading but cannot flatten the
// requests into a caller-selected artifact kind.
type ExecutionArtifactResolver interface {
	AuthorizeExecutionInput(ctx context.Context, lease AgentExecutionLease, request *agentdatav1.StreamExecutionInputRequest) (AuthorizedExecutionArtifact, error)
	AuthorizeConfigResource(ctx context.Context, lease AgentExecutionLease, request *agentdatav1.StreamConfigResourceRequest) (AuthorizedExecutionArtifact, error)
	AuthorizePlatformResource(ctx context.Context, lease AgentExecutionLease, request *agentdatav1.StreamPlatformResourceContentRequest) (AuthorizedExecutionArtifact, error)
}

// ExecutionArtifactService serves typed pre-start artifact streams.
type ExecutionArtifactService struct {
	agentdatav1.UnimplementedExecutionArtifactServiceServer

	agentFinder AgentFinder
	resolver    ExecutionArtifactResolver
	// runtimeExchangeAdmission is a bounded, non-blocking gate. A nil value
	// uses the package default and is safe for direct test construction.
	runtimeExchangeAdmission chan struct{}
}

func NewExecutionArtifactService(agentFinder AgentFinder, resolver ExecutionArtifactResolver) *ExecutionArtifactService {
	return &ExecutionArtifactService{agentFinder: agentFinder, resolver: resolver}
}

func (service *ExecutionArtifactService) StreamExecutionInput(request *agentdatav1.StreamExecutionInputRequest, stream grpc.ServerStreamingServer[agentdatav1.ExecutionArtifactFrame]) (resultErr error) {
	lease, err := service.authenticate(stream.Context())
	if err != nil {
		return err
	}
	role, binding, err := validateExecutionInputRequest(request)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if service.resolver == nil {
		return status.Error(codes.Unimplemented, "execution artifact resolver is not configured")
	}
	scanID, taskID, err := resourcenames.ParseTask(request.GetTask())
	if err != nil {
		// validateExecutionInputRequest already checks this; keep the terminal
		// observation scoped only to a canonical Scan/Task identity.
		return status.Error(codes.Internal, "validated execution input task scope is invalid")
	}
	observation := newExecutionInputBlacklistObservation(role, scanID, taskID)
	if observation != nil {
		defer func() {
			observation.observe(resultErr)
		}()
	}
	source, err := service.resolver.AuthorizeExecutionInput(stream.Context(), lease, request)
	if err != nil {
		return mapExecutionArtifactError(err)
	}
	if observation != nil {
		observation.setStats(source.executionInputBlacklistStats)
		source.executionInputBlacklistTransportInterrupted = observation.markTransportInterrupted
	}
	header := &agentdatav1.ExecutionArtifactHeader{
		Task:              request.GetTask(),
		Execution:         request.GetExecution(),
		Binding:           &agentdatav1.ExecutionArtifactHeader_ExecutionInput{ExecutionInput: binding},
		ContentType:       source.ContentType,
		ExpectedIntegrity: source.ExpectedIntegrity,
	}
	return streamAuthorizedExecutionArtifact(stream.Context(), stream.Send, role, header, source)
}

func (service *ExecutionArtifactService) StreamConfigResource(request *agentdatav1.StreamConfigResourceRequest, stream grpc.ServerStreamingServer[agentdatav1.ExecutionArtifactFrame]) error {
	lease, err := service.authenticate(stream.Context())
	if err != nil {
		return err
	}
	binding, err := validateConfigResourceRequest(request)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if service.resolver == nil {
		return status.Error(codes.Unimplemented, "execution artifact resolver is not configured")
	}
	source, err := service.resolver.AuthorizeConfigResource(stream.Context(), lease, request)
	if err != nil {
		return mapExecutionArtifactError(err)
	}
	header := &agentdatav1.ExecutionArtifactHeader{
		Task:              request.GetTask(),
		Execution:         request.GetExecution(),
		Binding:           &agentdatav1.ExecutionArtifactHeader_ConfigResource{ConfigResource: binding},
		ContentType:       source.ContentType,
		ExpectedIntegrity: source.ExpectedIntegrity,
	}
	return streamAuthorizedExecutionArtifact(stream.Context(), stream.Send, artifactRoleWordlist, header, source)
}

func (service *ExecutionArtifactService) StreamPlatformResourceContent(request *agentdatav1.StreamPlatformResourceContentRequest, stream grpc.ServerStreamingServer[agentdatav1.ExecutionArtifactFrame]) error {
	lease, err := service.authenticate(stream.Context())
	if err != nil {
		return err
	}
	binding, err := validatePlatformResourceRequest(request)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if service.resolver == nil {
		return status.Error(codes.Unimplemented, "execution artifact resolver is not configured")
	}
	source, err := service.resolver.AuthorizePlatformResource(stream.Context(), lease, request)
	if err != nil {
		return mapExecutionArtifactError(err)
	}
	header := &agentdatav1.ExecutionArtifactHeader{
		Task:              request.GetTask(),
		Execution:         request.GetExecution(),
		Binding:           &agentdatav1.ExecutionArtifactHeader_PlatformResource{PlatformResource: binding},
		ContentType:       source.ContentType,
		ExpectedIntegrity: source.ExpectedIntegrity,
	}
	streamRole := artifactRoleProviderConfig
	if descriptor, ok := executionartifact.LookupPlatformResource(binding.GetResourceId()); ok && executionartifact.IsFingerprintLibraryRole(descriptor.Role) {
		streamRole = artifactRoleFingerprintLibrary
	}
	return streamAuthorizedExecutionArtifact(stream.Context(), stream.Send, streamRole, header, source)
}

func (service *ExecutionArtifactService) authenticate(ctx context.Context) (AgentExecutionLease, error) {
	if service == nil || service.agentFinder == nil {
		return AgentExecutionLease{}, status.Error(codes.Unauthenticated, "execution artifact authentication is not configured")
	}
	token, ok := grpcauth.ReadAgentAuthenticationToken(ctx)
	if !ok {
		return AgentExecutionLease{}, grpcauth.MapError(grpcauth.ErrInvalidAgentAuthenticationToken)
	}
	agent, err := service.agentFinder.FindByAuthenticationToken(ctx, token)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return AgentExecutionLease{}, mapExecutionArtifactError(err)
		case errors.Is(err, agentdomain.ErrAgentNotFound):
			return AgentExecutionLease{}, grpcauth.MapError(grpcauth.ErrInvalidAgentAuthenticationToken)
		default:
			return AgentExecutionLease{}, status.Error(codes.Unavailable, "execution artifact authentication dependency is unavailable")
		}
	}
	// Only absence or an invalid persisted identity is a credential failure;
	// storage health and caller cancellation remain distinct protocol signals.
	if agent == nil || agent.ID <= 0 {
		return AgentExecutionLease{}, grpcauth.MapError(grpcauth.ErrInvalidAgentAuthenticationToken)
	}
	sessionID, sessionEpoch, ok := grpcauth.ReadAgentSession(ctx)
	if !ok {
		return AgentExecutionLease{}, status.Error(codes.FailedPrecondition, "current Agent process session metadata is required")
	}
	return AgentExecutionLease{AgentID: agent.ID, SessionID: sessionID, SessionEpoch: sessionEpoch}, nil
}

func validateExecutionInputRequest(request *agentdatav1.StreamExecutionInputRequest) (executionArtifactRole, *agentdatav1.ExecutionInputBindingKey, error) {
	if request == nil {
		return 0, nil, fmt.Errorf("execution input request is required")
	}
	if err := validateArtifactScope(request.GetTask(), request.GetExecution()); err != nil {
		return 0, nil, err
	}
	descriptor, ok := executionartifact.LookupRoleID(request.GetRole())
	if !ok || descriptor.RoleID == "" {
		return 0, nil, fmt.Errorf("execution input role is unsupported")
	}
	role, err := executionArtifactRoleForRegistryRole(descriptor.Role)
	if err != nil {
		return 0, nil, err
	}
	return role, &agentdatav1.ExecutionInputBindingKey{Role: descriptor.RoleID}, nil
}

func validateConfigResourceRequest(request *agentdatav1.StreamConfigResourceRequest) (*agentdatav1.ConfigResourceBindingKey, error) {
	if request == nil {
		return nil, fmt.Errorf("config resource request is required")
	}
	if err := validateArtifactScope(request.GetTask(), request.GetExecution()); err != nil {
		return nil, err
	}
	if err := validateCanonicalSelector("section_id", request.GetSectionId()); err != nil {
		return nil, err
	}
	if err := validateCanonicalSelector("param_key", request.GetParamKey()); err != nil {
		return nil, err
	}
	return &agentdatav1.ConfigResourceBindingKey{SectionId: request.GetSectionId(), ParamKey: request.GetParamKey()}, nil
}

func validatePlatformResourceRequest(request *agentdatav1.StreamPlatformResourceContentRequest) (*agentdatav1.PlatformResourceBindingKey, error) {
	if request == nil {
		return nil, fmt.Errorf("platform resource request is required")
	}
	if err := validateArtifactScope(request.GetTask(), request.GetExecution()); err != nil {
		return nil, err
	}
	if err := validateCanonicalSelector("resource_id", request.GetResourceId()); err != nil {
		return nil, err
	}
	return &agentdatav1.PlatformResourceBindingKey{ResourceId: request.GetResourceId()}, nil
}

func validateArtifactScope(task, execution string) error {
	if err := validateCanonicalSelector("task", task); err != nil {
		return err
	}
	scanID, taskID, err := resourcenames.ParseTask(task)
	if err != nil || resourcenames.Task(scanID, taskID) != task {
		return fmt.Errorf("task must be a canonical scan task resource name")
	}
	if err := validateCanonicalSelector("execution", execution); err != nil {
		return err
	}
	executionID, err := resourcenames.ParseExecution(execution)
	if err != nil || resourcenames.Execution(executionID) != execution {
		return fmt.Errorf("execution must be a canonical execution resource name")
	}
	return nil
}

func validateCanonicalSelector(field, value string) error {
	if value == "" || value != strings.TrimSpace(value) || strings.ContainsAny(value, "\r\n\x00") {
		return fmt.Errorf("%s must be a non-empty canonical value", field)
	}
	return nil
}
