package application

import (
	"context"
	"fmt"
)

// ScanCreateAgentLookup is intentionally limited to existence. Health and
// Engine API compatibility are mutable runtime facts checked by the claim path.
type ScanCreateAgentLookup interface {
	AgentExists(ctx context.Context, id int) (bool, error)
}

func (service *ScanCreateService) WithAgentLookup(lookup ScanCreateAgentLookup) *ScanCreateService {
	if service != nil {
		service.agentLookup = lookup
	}
	return service
}

func (service *ScanCreateService) validateSelectedAgent(ctx context.Context, agentID *int) error {
	if agentID == nil {
		return nil
	}
	if *agentID <= 0 || service == nil || service.agentLookup == nil {
		return ErrCreateAgentNotFound
	}
	exists, err := service.agentLookup.AgentExists(ctx, *agentID)
	if err != nil {
		return fmt.Errorf("lookup selected agent: %w", err)
	}
	if !exists {
		return ErrCreateAgentNotFound
	}
	return nil
}

func assignmentModeForAgent(agentID *int) string {
	if agentID != nil {
		return "pinned"
	}
	return "automatic"
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
