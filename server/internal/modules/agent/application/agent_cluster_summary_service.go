package application

import (
	"context"
	"errors"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

// AgentClusterSummaryService is the read-only complete-fleet summary boundary.
type AgentClusterSummaryService struct {
	store AgentClusterSummaryStore
	clock Clock
}

func NewAgentClusterSummaryService(store AgentClusterSummaryStore, clock Clock) (*AgentClusterSummaryService, error) {
	if store == nil {
		return nil, errors.New("Agent cluster summary store is required")
	}
	if clock == nil {
		return nil, errors.New("Agent cluster summary clock is required")
	}
	return &AgentClusterSummaryService{store: store, clock: clock}, nil
}

func (service *AgentClusterSummaryService) Current(ctx context.Context) (AgentClusterSummary, error) {
	generatedAt := service.clock.NowUTC().UTC()
	aggregate, err := service.store.GetClusterAggregate(ctx, generatedAt)
	if err != nil {
		return AgentClusterSummary{}, err
	}
	conclusion := mapAgentClusterConclusion(aggregate)
	return AgentClusterSummary{
		GeneratedAt:               generatedAt,
		ExecutionFreshnessSeconds: int(agentdomain.AgentExecutionFreshness.Seconds()),
		Nodes: AgentClusterNodeCounts{
			Total:   aggregate.TotalNodes,
			Healthy: aggregate.HealthyCount,
			Warning: aggregate.WarningCount,
			Offline: aggregate.OfflineCount,
			Unknown: aggregate.UnknownCount,
			Stale:   aggregate.StaleAgentCount,
		},
		ExecutionCapacity: AgentClusterExecutionCapacity{
			Configured:    aggregate.ConfiguredSlots,
			Occupied:      aggregate.OccupiedSlots,
			Available:     aggregate.AvailableSlots,
			Unavailable:   aggregate.UnavailableSlots,
			Overcommitted: aggregate.OvercommittedSlots,
		},
		State:       conclusion.State,
		ReasonCodes: append([]AgentClusterReasonCode(nil), conclusion.ReasonCodes...),
		LocationCoverage: AgentClusterLocationCoverage{
			Positioned:   aggregate.PositionedCount,
			Unpositioned: aggregate.UnpositionedCount,
		},
	}, nil
}
