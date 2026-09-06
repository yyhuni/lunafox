package application

import (
	"context"

	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
)

// EngineExecutionClaimStore owns the atomic saved-plan claim/replay boundary.
// It receives only generic compatibility facts, never package/config inputs.
type EngineExecutionClaimStore interface {
	ClaimNextCompatibleSavedExecutionPlan(ctx context.Context, agentID int, sessionID string, sessionEpoch int64, requestID string, supportedEngineAPIMajors []uint32) (*agentexecutionv1.ResolvedEngineExecutionPlan, error)
}
