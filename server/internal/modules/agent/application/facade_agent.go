package application

import (
	"context"
	"errors"

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
	agentStore AgentStore,
	tokenStore RegistrationTokenStore,
	clock Clock,
	tokenGenerator TokenGenerator,
) *AgentFacade {
	return &AgentFacade{
		queryService:        NewAgentQueryService(agentStore),
		commandService:      NewAgentCommandService(agentStore),
		registrationService: NewAgentRegistrationService(agentStore, tokenStore, clock, tokenGenerator),
	}
}

func (service *AgentFacade) ListAgents(ctx context.Context, page, pageSize int, status string) ([]*agentdomain.Agent, int64, error) {
	return service.queryService.ListAgents(ctx, page, pageSize, status)
}

func (service *AgentFacade) GetAgent(ctx context.Context, id int) (*agentdomain.Agent, error) {
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

func (service *AgentFacade) RegisterAgent(ctx context.Context, token, hostname, agentVersion, workerVersion, ipAddress string, options agentdomain.AgentRegistrationOptions) (*agentdomain.Agent, error) {
	agent, err := service.registrationService.RegisterAgent(ctx, token, hostname, agentVersion, workerVersion, ipAddress, options)
	if err != nil {
		if errors.Is(err, ErrRegistrationTokenInvalid) {
			return nil, ErrRegistrationTokenInvalid
		}
		return nil, err
	}
	return agent, nil
}
