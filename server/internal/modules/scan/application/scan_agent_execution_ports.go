package application

import (
	"context"
	"fmt"
)

// ScanAgentExecutionReader exposes the current deployment-owned eligibility of
// one Agent without coupling Scan use-cases to Agent persistence.
type ScanAgentExecutionReader interface {
	AgentExecutionEligibility(ctx context.Context, agentID int) (string, error)
}

const (
	agentExecutionEligible          = "eligible"
	agentExecutionLicenseRestricted = "license_restricted"
	// ExecutionBlockedReasonAgentLicenseRestricted identifies a resumable wait
	// for the bound Agent's current license entitlement.
	ExecutionBlockedReasonAgentLicenseRestricted = "agent_license_restricted"
)

func readAgentExecutionEligibility(ctx context.Context, reader ScanAgentExecutionReader, agentID int) (string, error) {
	if reader == nil {
		return "", fmt.Errorf("scan Agent execution reader is required")
	}
	eligibility, err := reader.AgentExecutionEligibility(ctx, agentID)
	if err != nil {
		return "", fmt.Errorf("read scan Agent execution eligibility: %w", err)
	}
	switch eligibility {
	case agentExecutionEligible, agentExecutionLicenseRestricted:
		return eligibility, nil
	default:
		return "", fmt.Errorf("invalid scan Agent execution eligibility %q", eligibility)
	}
}
