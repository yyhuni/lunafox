package agentdata

import (
	"context"
	"errors"
	"strings"

	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/planeauth"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AgentFinder interface {
	FindByAuthenticationToken(ctx context.Context, authenticationToken string) (*agentdomain.Agent, error)
}

// requireAuthenticatedAgentSession is used by task-scoped writes. The
// long-lived Agent token proves node identity; the canonical metadata tuple
// proves which fenced process session issued this operation. A successful
// return is this RPC's admission point: takeover rejects later admissions but
// does not retroactively revoke an already admitted in-flight operation.
func (s *DataPlaneService) requireAuthenticatedAgentSession(ctx context.Context) (AgentExecutionLease, error) {
	if s == nil || s.agentFinder == nil {
		return AgentExecutionLease{}, grpcauth.MapError(grpcauth.ErrInvalidAgentAuthenticationToken)
	}
	authenticationToken, ok := grpcauth.ReadAgentAuthenticationToken(ctx)
	if !ok {
		return AgentExecutionLease{}, grpcauth.MapError(grpcauth.ErrInvalidAgentAuthenticationToken)
	}
	agent, err := s.agentFinder.FindByAuthenticationToken(ctx, authenticationToken)
	if err != nil {
		return AgentExecutionLease{}, mapAgentAuthenticationLookupError(err)
	}
	if agent == nil || agent.ID <= 0 {
		return AgentExecutionLease{}, grpcauth.MapError(grpcauth.ErrInvalidAgentAuthenticationToken)
	}
	sessionID, sessionEpoch, ok := grpcauth.ReadAgentSession(ctx)
	if !ok || strings.TrimSpace(agent.SessionID) == "" || agent.SessionEpoch <= 0 || sessionID != agent.SessionID || sessionEpoch != agent.SessionEpoch {
		return AgentExecutionLease{}, status.Error(codes.FailedPrecondition, "Agent process session is not current")
	}
	if s.sessions == nil {
		return AgentExecutionLease{}, status.Error(codes.FailedPrecondition, "Agent process session authority is unavailable")
	}
	current, ok := s.sessions.CurrentLeaseSession(agent.ID)
	if !ok || current.AgentID != agent.ID || current.SessionID != sessionID || current.SessionEpoch != sessionEpoch {
		return AgentExecutionLease{}, status.Error(codes.FailedPrecondition, "Agent process session is not current")
	}
	return AgentExecutionLease{AgentID: agent.ID, SessionID: sessionID, SessionEpoch: sessionEpoch}, nil
}

func mapAgentAuthenticationLookupError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "Agent authentication lookup cancelled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "Agent authentication lookup deadline exceeded")
	case errors.Is(err, agentdomain.ErrAgentNotFound), dberrors.IsRecordNotFound(err):
		return grpcauth.MapError(grpcauth.ErrInvalidAgentAuthenticationToken)
	default:
		return status.Error(codes.Unavailable, "Agent authentication authority is unavailable")
	}
}
