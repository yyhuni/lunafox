package agentcontrol

import (
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *ControlPlaneService) matchesCurrentSession(agentID int, session ActiveControlSession) bool {
	if s.sessions == nil {
		return false
	}
	return s.sessions.MatchesCurrentSession(agentID, session.SessionID, session.SessionEpoch, session.StreamID)
}

func logSessionRegistration(agentID int, activeSession ActiveControlSession, replacedSession *ActiveControlSession) {
	fields := []zap.Field{
		zap.Int("agent.id", agentID),
		zap.String("session.id", activeSession.SessionID),
		zap.Int64("session.epoch", activeSession.SessionEpoch),
	}

	if replacedSession == nil {
		zap.L().Info("session_registered", fields...)
		return
	}
	if replacedSession.SessionID == activeSession.SessionID && replacedSession.SessionEpoch == activeSession.SessionEpoch {
		fields = append(fields,
			zap.Uint64("previous_stream.id", replacedSession.StreamID),
			zap.Bool("reattached", true),
		)
		zap.L().Info("session_registered", fields...)
		return
	}

	fields = append(fields,
		zap.String("replaced_session.id", replacedSession.SessionID),
		zap.Int64("replaced_session.epoch", replacedSession.SessionEpoch),
	)
	zap.L().Info("session_replaced", fields...)
}

func staleSessionError(agentID int, session ActiveControlSession, action string) error {
	zap.L().Info("stale_session_rejected",
		zap.Int("agent.id", agentID),
		zap.String("action", action),
		zap.String("session.id", session.SessionID),
		zap.Int64("session.epoch", session.SessionEpoch),
	)
	return status.Error(codes.FailedPrecondition, "agent control session is stale")
}

func logExecutionClaimDeferred(agentID int, session ActiveControlSession, reason executionSnapshotAdmissionReason) {
	zap.L().Info("engine_execution_claim_deferred",
		zap.Int("agent.id", agentID),
		zap.Int64("session.epoch", session.SessionEpoch),
		zap.String("execution.phase", "scheduler_admission"),
		zap.String("error.reason", string(reason)),
	)
}
