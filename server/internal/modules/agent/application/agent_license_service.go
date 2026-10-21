package application

import (
	"context"
	"errors"
	"strings"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

// AgentLicenseService exposes deployment authorization without disclosing its Key.
type AgentLicenseService struct {
	store AgentLicenseStore
	clock Clock
}

// NewAgentLicenseService requires explicit persistence and time dependencies.
func NewAgentLicenseService(store AgentLicenseStore, clock Clock) (*AgentLicenseService, error) {
	if store == nil {
		return nil, errors.New("agent license store is required")
	}
	if clock == nil {
		return nil, errors.New("agent license clock is required")
	}
	return &AgentLicenseService{store: store, clock: clock}, nil
}

// Activate imports a Key as issued; retrying never restarts its fixed duration.
func (service *AgentLicenseService) Activate(ctx context.Context, key string) (AgentLicense, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return AgentLicense{}, agentdomain.ErrAgentLicenseKeyRequired
	}
	if len(key) > 4096 {
		return AgentLicense{}, agentdomain.ErrAgentLicenseKeyInvalid
	}
	return service.store.ActivateAt(ctx, key, service.clock.NowUTC().UTC())
}
