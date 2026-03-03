package application

import (
	"context"
	"strings"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

// AgentRegistrationService handles agent registration and token lifecycle.
type AgentRegistrationService struct {
	agentRepo       AgentCommandStore
	tokenRepo       RegistrationTokenStore
	clock           Clock
	tokenGenerator  TokenGenerator
	tokenTTL        time.Duration
	tokenByteLength int
}

func NewAgentRegistrationService(
	agentRepo AgentCommandStore,
	tokenRepo RegistrationTokenStore,
	clock Clock,
	tokenGenerator TokenGenerator,
) *AgentRegistrationService {
	if clock == nil {
		panic("clock is required")
	}
	if tokenGenerator == nil {
		panic("tokenGenerator is required")
	}

	return &AgentRegistrationService{
		agentRepo:       agentRepo,
		tokenRepo:       tokenRepo,
		clock:           clock,
		tokenGenerator:  tokenGenerator,
		tokenTTL:        1 * time.Hour,
		tokenByteLength: 4,
	}
}

func (service *AgentRegistrationService) CreateRegistrationToken(ctx context.Context) (*agentdomain.RegistrationToken, error) {
	now := service.clock.NowUTC()
	if err := service.tokenRepo.DeleteExpired(ctx, now); err != nil {
		return nil, err
	}

	token, err := service.tokenGenerator.GenerateHex(service.tokenByteLength)
	if err != nil {
		return nil, err
	}

	registration := agentdomain.NewRegistrationToken(token, now.Add(service.tokenTTL))
	if err := service.tokenRepo.Create(ctx, registration); err != nil {
		return nil, err
	}
	return registration, nil
}

func (service *AgentRegistrationService) ValidateRegistrationToken(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return ErrRegistrationTokenInvalid
	}
	registration, err := service.tokenRepo.FindValid(ctx, token, service.clock.NowUTC())
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrRegistrationTokenInvalid
		}
		return err
	}
	if registration == nil {
		return ErrRegistrationTokenInvalid
	}
	return nil
}

func (service *AgentRegistrationService) RegisterAgent(ctx context.Context, token, hostname, agentVersion, workerVersion, ipAddress string, options agentdomain.AgentRegistrationOptions) (*agentdomain.Agent, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrRegistrationTokenInvalid
	}

	registration, err := service.tokenRepo.FindValid(ctx, token, service.clock.NowUTC())
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrRegistrationTokenInvalid
		}
		return nil, err
	}
	if registration == nil {
		return nil, ErrRegistrationTokenInvalid
	}

	apiKey, err := service.tokenGenerator.GenerateHex(service.tokenByteLength)
	if err != nil {
		return nil, err
	}

	agent := agentdomain.NewRegisteredAgent(token, hostname, agentVersion, workerVersion, ipAddress, apiKey, options)
	if err := service.agentRepo.Create(ctx, agent); err != nil {
		return nil, err
	}
	return agent, nil
}
