package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
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
	if err == nil && plan != nil && service.runtimeLogs != nil {
		scanID, taskID, parseErr := resourcenames.ParseTask(plan.GetTask())
		if parseErr != nil {
			pkg.Error("claimed task runtime log scope is invalid", zap.Error(parseErr))
		} else {
			service.appendTaskRuntimeLog(ctx, &ScanTaskRecord{ID: taskID, ScanID: scanID}, "assignment:"+requestID, "info", fmt.Sprintf("[server] task assigned to Agent %d; state=running", agentID))
		}
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
