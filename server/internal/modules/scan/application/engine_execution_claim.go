package application

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

var ErrEngineExecutionClaimUnavailable = errors.New("Engine execution claim store is unavailable")

func (service *ScanTaskBridgeService) WithEngineExecutionClaimStore(store EngineExecutionClaimStore) *ScanTaskBridgeService {
	if service != nil {
		service.engineExecutionClaims = store
	}
	return service
}

func (service *ScanTaskBridgeService) ClaimNextExecutionPlan(
	ctx context.Context,
	agentID int,
	sessionID string,
	sessionEpoch int64,
	requestID string,
	snapshot agentdomain.AgentExecutionCapabilitySnapshot,
) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	if service == nil || service.engineExecutionClaims == nil {
		return nil, ErrEngineExecutionClaimUnavailable
	}
	requestID = strings.TrimSpace(requestID)
	parsedRequestID, err := uuid.Parse(requestID)
	if ctx == nil || agentID <= 0 || strings.TrimSpace(sessionID) == "" || sessionEpoch <= 0 || err != nil || parsedRequestID.String() != requestID {
		return nil, ErrScanTaskInvalidUpdate
	}
	if !snapshot.ContainerRuntimeReady || len(snapshot.SupportedEngineAPIMajors) == 0 {
		return nil, nil
	}
	plan, err := service.engineExecutionClaims.ClaimNextCompatibleSavedExecutionPlan(
		ctx, agentID, strings.TrimSpace(sessionID), sessionEpoch, requestID,
		append([]uint32(nil), snapshot.SupportedEngineAPIMajors...),
	)
	if errors.Is(err, scandomain.ErrAgentExecutionSessionFenced) {
		return nil, ErrScanTaskNotOwned
	}
	return plan, err
}

func (service *ScanTaskFacade) ClaimNextExecutionPlan(
	ctx context.Context,
	agentID int,
	sessionID string,
	sessionEpoch int64,
	requestID string,
	snapshot agentdomain.AgentExecutionCapabilitySnapshot,
) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	return service.taskBridgeService.ClaimNextExecutionPlan(ctx, agentID, sessionID, sessionEpoch, requestID, snapshot)
}
