package application

import (
	"context"
	"errors"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"gorm.io/gorm"
)

type fixedClock struct{ now time.Time }

func (clock fixedClock) NowUTC() time.Time { return clock.now }

type tokenGenStub struct {
	values []string
	index  int
	err    error
}

func (generator *tokenGenStub) GenerateHex(int) (string, error) {
	if generator.err != nil {
		return "", generator.err
	}
	if generator.index >= len(generator.values) {
		return "", errors.New("no token")
	}
	value := generator.values[generator.index]
	generator.index++
	return value, nil
}

type agentRepoStub struct {
	created []*agentdomain.Agent
	getByID *agentdomain.Agent
	getErr  error
	delErr  error
}

func (repo *agentRepoStub) Create(_ context.Context, agent *agentdomain.Agent) error {
	repo.created = append(repo.created, agent)
	return nil
}
func (repo *agentRepoStub) GetByID(_ context.Context, _ int) (*agentdomain.Agent, error) {
	return repo.getByID, repo.getErr
}
func (repo *agentRepoStub) FindByAPIKey(_ context.Context, _ string) (*agentdomain.Agent, error) {
	return nil, nil
}
func (repo *agentRepoStub) List(_ context.Context, _, _ int, _ string) ([]*agentdomain.Agent, int64, error) {
	return nil, 0, nil
}
func (repo *agentRepoStub) FindStaleOnline(_ context.Context, _ time.Time) ([]*agentdomain.Agent, error) {
	return nil, nil
}
func (repo *agentRepoStub) Update(_ context.Context, _ *agentdomain.Agent) error  { return nil }
func (repo *agentRepoStub) UpdateStatus(_ context.Context, _ int, _ string) error { return nil }
func (repo *agentRepoStub) UpdateHeartbeat(_ context.Context, _ int, _ agentdomain.AgentHeartbeatUpdate) error {
	return nil
}
func (repo *agentRepoStub) Delete(_ context.Context, _ int) error { return repo.delErr }

type tokenRepoStub struct {
	created      []*agentdomain.RegistrationToken
	findResult   *agentdomain.RegistrationToken
	findErr      error
	deleteCalled bool
}

func (repo *tokenRepoStub) Create(_ context.Context, token *agentdomain.RegistrationToken) error {
	repo.created = append(repo.created, token)
	return nil
}
func (repo *tokenRepoStub) FindValid(_ context.Context, _ string, _ time.Time) (*agentdomain.RegistrationToken, error) {
	return repo.findResult, repo.findErr
}
func (repo *tokenRepoStub) DeleteExpired(_ context.Context, _ time.Time) error {
	repo.deleteCalled = true
	return nil
}

func TestAgentRegistrationServiceCreateToken(t *testing.T) {
	agentRepo := &agentRepoStub{}
	tokenRepo := &tokenRepoStub{}
	service := NewAgentRegistrationService(
		agentRepo,
		tokenRepo,
		fixedClock{now: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)},
		&tokenGenStub{values: []string{"abcd1234"}},
	)

	token, err := service.CreateRegistrationToken(context.Background())
	if err != nil {
		t.Fatalf("CreateRegistrationToken error: %v", err)
	}
	if !tokenRepo.deleteCalled {
		t.Fatalf("expected DeleteExpired called")
	}
	if token.Token != "abcd1234" {
		t.Fatalf("expected token abcd1234, got %s", token.Token)
	}
	if token.ExpiresAt.Sub(service.clock.NowUTC()) != time.Hour {
		t.Fatalf("expected 1h ttl")
	}
}

func TestAgentRegistrationServiceRegisterInvalidToken(t *testing.T) {
	agentRepo := &agentRepoStub{}
	tokenRepo := &tokenRepoStub{}
	service := NewAgentRegistrationService(agentRepo, tokenRepo, fixedClock{now: time.Now().UTC()}, &tokenGenStub{})

	_, err := service.RegisterAgent(context.Background(), "", "host", "1.0", "1.0", "127.0.0.1", agentdomain.AgentRegistrationOptions{})
	if !errors.Is(err, ErrRegistrationTokenInvalid) {
		t.Fatalf("expected ErrRegistrationTokenInvalid, got %v", err)
	}
}

func TestAgentRegistrationServiceRegisterPersistsWorkerVersion(t *testing.T) {
	agentRepo := &agentRepoStub{}
	tokenRepo := &tokenRepoStub{
		findResult: &agentdomain.RegistrationToken{Token: "abcd1234"},
	}
	service := NewAgentRegistrationService(
		agentRepo,
		tokenRepo,
		fixedClock{now: time.Now().UTC()},
		&tokenGenStub{values: []string{"deadbeef"}},
	)

	agent, err := service.RegisterAgent(
		context.Background(),
		"abcd1234",
		"host",
		"1.0.0",
		"1.0.0",
		"127.0.0.1",
		agentdomain.AgentRegistrationOptions{},
	)
	if err != nil {
		t.Fatalf("RegisterAgent error: %v", err)
	}
	if len(agentRepo.created) != 1 {
		t.Fatalf("expected one created agent, got %d", len(agentRepo.created))
	}
	if agentRepo.created[0].WorkerVersion != "1.0.0" {
		t.Fatalf("expected worker version persisted on create, got %q", agentRepo.created[0].WorkerVersion)
	}
	if agent.WorkerVersion != "1.0.0" {
		t.Fatalf("expected returned agent worker version, got %q", agent.WorkerVersion)
	}
}

func TestAgentRegistrationServiceValidateTokenRecordNotFound(t *testing.T) {
	agentRepo := &agentRepoStub{}
	tokenRepo := &tokenRepoStub{findErr: gorm.ErrRecordNotFound}
	service := NewAgentRegistrationService(agentRepo, tokenRepo, fixedClock{now: time.Now().UTC()}, &tokenGenStub{})

	err := service.ValidateRegistrationToken(context.Background(), "abc123")
	if !errors.Is(err, ErrRegistrationTokenInvalid) {
		t.Fatalf("expected ErrRegistrationTokenInvalid, got %v", err)
	}
}

func TestAgentRegistrationServiceRegisterTokenRecordNotFound(t *testing.T) {
	agentRepo := &agentRepoStub{}
	tokenRepo := &tokenRepoStub{findErr: gorm.ErrRecordNotFound}
	service := NewAgentRegistrationService(agentRepo, tokenRepo, fixedClock{now: time.Now().UTC()}, &tokenGenStub{})

	_, err := service.RegisterAgent(context.Background(), "abc123", "host", "1.0", "1.0", "127.0.0.1", agentdomain.AgentRegistrationOptions{})
	if !errors.Is(err, ErrRegistrationTokenInvalid) {
		t.Fatalf("expected ErrRegistrationTokenInvalid, got %v", err)
	}
}
