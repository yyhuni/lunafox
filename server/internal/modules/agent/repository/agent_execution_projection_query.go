package repository

import (
	"context"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

func projectAgentExecutionEligibility(ctx context.Context, licenses *AgentLicenseRepository, agents []*agentdomain.Agent, now time.Time) error {
	if len(agents) == 0 {
		return nil
	}
	ids, err := licenses.EligibleAgentIDsAt(ctx, now)
	if err != nil {
		return err
	}
	eligible := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		eligible[id] = struct{}{}
	}
	// Eligibility comes from the complete fleet, before any display filter or
	// pagination; an offline earlier identity still occupies its entitlement.
	for _, agent := range agents {
		agent.ExecutionEligibility = agentdomain.AgentExecutionLicenseRestricted
		if _, ok := eligible[agent.ID]; ok {
			agent.ExecutionEligibility = agentdomain.AgentExecutionEligible
		}
	}
	return nil
}
