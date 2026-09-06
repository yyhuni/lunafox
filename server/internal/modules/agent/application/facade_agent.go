package application

import (
	"context"
	"errors"
	"fmt"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

// AgentFacade handles agent registration and management operations.
type AgentFacade struct {
	queryService        *AgentQueryService
	commandService      *AgentCommandService
	registrationService *AgentRegistrationService
}

// NewAgentFacade creates a new agent facade.
func NewAgentFacade(
	queryService *AgentQueryService,
	commandService *AgentCommandService,
	registrationService *AgentRegistrationService,
) *AgentFacade {
	return &AgentFacade{
		queryService:        queryService,
		commandService:      commandService,
		registrationService: registrationService,
	}
}

func (service *AgentFacade) ListAgents(ctx context.Context, input AgentListQueryInput) (*AgentListResult, error) {
	if service == nil || service.queryService == nil {
		return nil, fmt.Errorf("agent query service is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	return service.queryService.ListAgents(ctx, input)
}

func (service *AgentFacade) ListFilterOptions(ctx context.Context, field string) ([]agentdomain.FilterOption, error) {
	return service.queryService.ListFilterOptions(ctx, field)
}

func (service *AgentFacade) GetAgent(ctx context.Context, id int) (*agentdomain.Agent, error) {
	if service == nil || service.queryService == nil {
		return nil, fmt.Errorf("agent query service is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	agent, err := service.queryService.GetAgent(ctx, id)
	if err != nil {
		if errors.Is(err, ErrAgentNotFound) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}
	return agent, nil
}

func (service *AgentFacade) UpdateAgentConfig(ctx context.Context, id int, update agentdomain.AgentConfigUpdate) (*agentdomain.Agent, error) {
	agent, err := service.commandService.UpdateAgentConfig(ctx, id, update)
	if err != nil {
		if errors.Is(err, ErrAgentNotFound) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}
	return agent, nil
}

func (service *AgentFacade) DeleteAgent(ctx context.Context, id int) error {
	err := service.commandService.DeleteAgent(ctx, id)
	if err != nil {
		if errors.Is(err, ErrAgentNotFound) {
			return ErrAgentNotFound
		}
		return err
	}
	return nil
}

func (service *AgentFacade) CreateRegistrationToken(ctx context.Context) (*agentdomain.RegistrationToken, error) {
	return service.registrationService.CreateRegistrationToken(ctx)
}

func (service *AgentFacade) GetRegistrationToken(ctx context.Context, id int) (*agentdomain.RegistrationTokenResource, error) {
	return service.registrationService.GetRegistrationToken(ctx, id)
}

func (service *AgentFacade) ValidateRegistrationToken(ctx context.Context, token string) error {
	err := service.registrationService.ValidateRegistrationToken(ctx, token)
	if err != nil {
		if errors.Is(err, ErrRegistrationTokenInvalid) {
			return ErrRegistrationTokenInvalid
		}
		return err
	}
	return nil
}

func (service *AgentFacade) RegisterAgent(ctx context.Context, token, observedHostname, agentVersion string, options agentdomain.AgentRegistrationOptions) (*agentdomain.Agent, error) {
	agent, err := service.registrationService.RegisterAgent(ctx, token, observedHostname, agentVersion, options)
	if err != nil {
		if errors.Is(err, ErrRegistrationTokenInvalid) {
			return nil, ErrRegistrationTokenInvalid
		}
		return nil, err
	}
	return agent, nil
}
