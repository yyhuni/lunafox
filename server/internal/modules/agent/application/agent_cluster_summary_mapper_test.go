package application

import (
	"reflect"
	"testing"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

func TestMapAgentClusterConclusionPolicyAndReasonOrder(t *testing.T) {
	tests := []struct {
		name      string
		aggregate agentdomain.AgentClusterAggregate
		wantState AgentClusterState
		wantCodes []AgentClusterReasonCode
	}{
		{
			name:      "successful empty cluster",
			wantState: AgentClusterStateEmpty,
			wantCodes: []AgentClusterReasonCode{AgentClusterReasonNoAgents},
		},
		{
			name: "non-empty zero capacity",
			aggregate: agentdomain.AgentClusterAggregate{
				TotalNodes:   1,
				HealthyCount: 1,
			},
			wantState: AgentClusterStateCritical,
			wantCodes: []AgentClusterReasonCode{AgentClusterReasonNoAvailableSlots},
		},
		{
			name: "critical reasons remain deterministically ordered",
			aggregate: agentdomain.AgentClusterAggregate{
				TotalNodes:         5,
				OfflineCount:       1,
				WarningCount:       1,
				UnknownCount:       1,
				StaleAgentCount:    1,
				OvercommittedSlots: 2,
			},
			wantState: AgentClusterStateCritical,
			wantCodes: []AgentClusterReasonCode{
				AgentClusterReasonNoAvailableSlots,
				AgentClusterReasonOfflineAgents,
				AgentClusterReasonWarningAgents,
				AgentClusterReasonUnknownAgents,
				AgentClusterReasonStaleRuntimeObservations,
				AgentClusterReasonOvercommittedSlots,
			},
		},
		{
			name: "positive capacity with signals needs attention",
			aggregate: agentdomain.AgentClusterAggregate{
				TotalNodes:         5,
				AvailableSlots:     1,
				OfflineCount:       1,
				WarningCount:       1,
				UnknownCount:       1,
				StaleAgentCount:    1,
				OvercommittedSlots: 2,
			},
			wantState: AgentClusterStateNeedsAttention,
			wantCodes: []AgentClusterReasonCode{
				AgentClusterReasonOfflineAgents,
				AgentClusterReasonWarningAgents,
				AgentClusterReasonUnknownAgents,
				AgentClusterReasonStaleRuntimeObservations,
				AgentClusterReasonOvercommittedSlots,
			},
		},
		{
			name: "low but positive clean capacity remains healthy",
			aggregate: agentdomain.AgentClusterAggregate{
				TotalNodes:      1,
				HealthyCount:    1,
				ConfiguredSlots: 100,
				OccupiedSlots:   99,
				AvailableSlots:  1,
			},
			wantState: AgentClusterStateHealthy,
			wantCodes: []AgentClusterReasonCode{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conclusion := mapAgentClusterConclusion(test.aggregate)
			if conclusion.State != test.wantState || !reflect.DeepEqual(conclusion.ReasonCodes, test.wantCodes) {
				t.Fatalf("conclusion = %#v, want state=%s codes=%v", conclusion, test.wantState, test.wantCodes)
			}
		})
	}
}
