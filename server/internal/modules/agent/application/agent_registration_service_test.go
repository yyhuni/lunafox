package application

import (
	"context"
	"errors"
	"strings"
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
	created   []*agentdomain.Agent
	getByID   *agentdomain.Agent
	getErr    error
	createErr error
	delErr    error
}

func (repo *agentRepoStub) Create(_ context.Context, agent *agentdomain.Agent) error {
	if repo.createErr != nil {
		return repo.createErr
	}
	repo.created = append(repo.created, agent)
	return nil
}
func (repo *agentRepoStub) GetByID(_ context.Context, _ int) (*agentdomain.Agent, error) {
	return repo.getByID, repo.getErr
}
func (repo *agentRepoStub) FindByAuthenticationToken(_ context.Context, _ string) (*agentdomain.Agent, error) {
	return nil, nil
}
func (repo *agentRepoStub) List(_ context.Context, _, _ int, _, _ string) ([]*agentdomain.Agent, int64, error) {
	return nil, 0, nil
}
func (repo *agentRepoStub) ListFilterOptions(_ context.Context, _ string) ([]agentdomain.FilterOption, error) {
	return nil, nil
}
func (repo *agentRepoStub) FindStaleOnline(_ context.Context, _ time.Time) ([]*agentdomain.Agent, error) {
	return nil, nil
}
func (repo *agentRepoStub) Update(_ context.Context, _ *agentdomain.Agent) error { return nil }
func (repo *agentRepoStub) RecordConnection(_ context.Context, agentID int, connectionIP string, _ time.Time) (agentdomain.AgentConnectionObservation, error) {
	return agentdomain.AgentConnectionObservation{AgentID: agentID, ConnectionIP: connectionIP, SourceIP: connectionIP}, nil
}
func (repo *agentRepoStub) ClearConnectionIP(_ context.Context, _ int) error      { return nil }
func (repo *agentRepoStub) UpdateStatus(_ context.Context, _ int, _ string) error { return nil }
func (repo *agentRepoStub) UpdateHeartbeat(_ context.Context, _ int, _ agentdomain.AgentHeartbeatUpdate) error {
	return nil
}
func (repo *agentRepoStub) Delete(_ context.Context, _ int) error { return repo.delErr }

type tokenRepoStub struct {
	created        []*agentdomain.RegistrationToken
	createdID      int
	findResult     *agentdomain.RegistrationToken
	findErr        error
	resourceResult *agentdomain.RegistrationTokenResource
	resourceErr    error
	deleteCutoffs  chan time.Time
	deleteErr      error
}

func (repo *tokenRepoStub) Create(_ context.Context, token *agentdomain.RegistrationToken) error {
	repo.created = append(repo.created, token)
	if repo.createdID > 0 {
		token.ID = repo.createdID
	}
	return nil
}
func (repo *tokenRepoStub) FindValid(_ context.Context, _ string, _ time.Time) (*agentdomain.RegistrationToken, error) {
	return repo.findResult, repo.findErr
}
func (repo *tokenRepoStub) GetResourceByID(_ context.Context, _ int) (*agentdomain.RegistrationTokenResource, error) {
	return repo.resourceResult, repo.resourceErr
}
func (repo *tokenRepoStub) DeleteNeverAttributedBefore(_ context.Context, cutoff time.Time) error {
	if repo.deleteCutoffs != nil {
		repo.deleteCutoffs <- cutoff
	}
	return repo.deleteErr
}

func TestAgentRegistrationServiceCreateToken(t *testing.T) {
	agentRepo := &agentRepoStub{}
	tokenRepo := &tokenRepoStub{createdID: 17}
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
	if token.Token != "abcd1234" {
		t.Fatalf("expected token abcd1234, got %s", token.Token)
	}
	if token.ExpiresAt.Sub(service.clock.NowUTC()) != time.Hour {
		t.Fatalf("expected 1h ttl")
	}
	if token.ID != 17 {
		t.Fatalf("expected persistent resource identity 17, got %d", token.ID)
	}
}

func TestAgentRegistrationServiceCreateTokenDoesNotSynchronouslyCleanExpiredTokens(t *testing.T) {
	tokenRepo := &tokenRepoStub{createdID: 18, deleteCutoffs: make(chan time.Time, 1)}
	service := NewAgentRegistrationService(
		&agentRepoStub{},
		tokenRepo,
		fixedClock{now: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)},
		&tokenGenStub{values: []string{"abcd1234"}},
	)
	if _, err := service.CreateRegistrationToken(context.Background()); err != nil {
		t.Fatalf("CreateRegistrationToken error: %v", err)
	}
	select {
	case cutoff := <-tokenRepo.deleteCutoffs:
		t.Fatalf("token creation synchronously triggered cleanup at %s", cutoff)
	default:
	}
}

func TestAgentRegistrationServiceCreateTokenRequiresPersistentResourceIdentity(t *testing.T) {
	service := NewAgentRegistrationService(
		&agentRepoStub{},
		&tokenRepoStub{},
		fixedClock{now: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)},
		&tokenGenStub{values: []string{"abcd1234"}},
	)
	if _, err := service.CreateRegistrationToken(context.Background()); err == nil || !strings.Contains(err.Error(), "persistent identity") {
		t.Fatalf("missing resource identity error = %v", err)
	}
}

func TestAgentRegistrationServiceGetsNonSecretTokenResource(t *testing.T) {
	want := &agentdomain.RegistrationTokenResource{
		ID:        23,
		ExpiresAt: time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC),
		Agents:    []*agentdomain.Agent{{ID: 7, DisplayName: "node-7"}},
	}
	service := NewAgentRegistrationService(
		&agentRepoStub{},
		&tokenRepoStub{resourceResult: want},
		fixedClock{now: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)},
		&tokenGenStub{},
	)
	got, err := service.GetRegistrationToken(context.Background(), 23)
	if err != nil {
		t.Fatalf("GetRegistrationToken error: %v", err)
	}
	if got != want || got.StateAt(service.clock.NowUTC()) != agentdomain.RegistrationTokenStateActive {
		t.Fatalf("registration token resource = %#v", got)
	}
}

func TestAgentRegistrationServiceRegisterInvalidToken(t *testing.T) {
	agentRepo := &agentRepoStub{}
	tokenRepo := &tokenRepoStub{}
	service := NewAgentRegistrationService(agentRepo, tokenRepo, fixedClock{now: time.Now().UTC()}, &tokenGenStub{})

	_, err := service.RegisterAgent(context.Background(), "", "host", "1.0", agentdomain.AgentRegistrationOptions{})
	if !errors.Is(err, ErrRegistrationTokenInvalid) {
		t.Fatalf("expected ErrRegistrationTokenInvalid, got %v", err)
	}
}

func TestAgentRegistrationServiceRegisterPersistsAgentIdentity(t *testing.T) {
	agentRepo := &agentRepoStub{}
	tokenRepo := &tokenRepoStub{
		findResult: &agentdomain.RegistrationToken{ID: 11, Token: "abcd1234"},
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
		agentdomain.AgentRegistrationOptions{},
	)
	if err != nil {
		t.Fatalf("RegisterAgent error: %v", err)
	}
	if len(agentRepo.created) != 1 {
		t.Fatalf("expected one created agent, got %d", len(agentRepo.created))
	}
	if agentRepo.created[0].AgentVersion != "1.0.0" {
		t.Fatalf("expected Agent version persisted on create, got %q", agentRepo.created[0].AgentVersion)
	}
	if agent.AgentVersion != "1.0.0" {
		t.Fatalf("expected returned Agent version, got %q", agent.AgentVersion)
	}
	if agent.RegistrationTokenID != 11 {
		t.Fatalf("expected stable token attribution 11, got %d", agent.RegistrationTokenID)
	}
	if agent.ObservedSourceIP != "" {
		t.Fatalf("registration must not set observed source, got %q", agent.ObservedSourceIP)
	}
}

func TestAgentRegistrationServiceRegisterInitializesIdentityAndDefaultDisplayName(t *testing.T) {
	agentRepo := &agentRepoStub{}
	tokenRepo := &tokenRepoStub{
		findResult: &agentdomain.RegistrationToken{ID: 12, Token: "abcd1234"},
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
		"scanner-a",
		"1.0.0",
		agentdomain.AgentRegistrationOptions{},
	)
	if err != nil {
		t.Fatalf("RegisterAgent error: %v", err)
	}
	if agent.InstanceID == "" {
		t.Fatalf("expected instance ID assigned, got %#v", agent)
	}
	if agent.DisplayName == "" {
		t.Fatalf("expected display name assigned, got %#v", agent)
	}
	if !strings.HasPrefix(agent.DisplayName, "scanner-a-") {
		t.Fatalf("expected default display name derived from hostname, got %q", agent.DisplayName)
	}
}

func TestAgentRegistrationServiceRegisterUsesFallbackWhenHostnameUnavailable(t *testing.T) {
	agentRepo := &agentRepoStub{}
	tokenRepo := &tokenRepoStub{
		findResult: &agentdomain.RegistrationToken{ID: 13, Token: "abcd1234"},
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
		"unknown",
		"1.0.0",
		agentdomain.AgentRegistrationOptions{},
	)
	if err != nil {
		t.Fatalf("RegisterAgent error: %v", err)
	}
	if !strings.HasPrefix(agent.DisplayName, "agent-") {
		t.Fatalf("expected fallback display name, got %q", agent.DisplayName)
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

	_, err := service.RegisterAgent(context.Background(), "abc123", "host", "1.0", agentdomain.AgentRegistrationOptions{})
	if !errors.Is(err, ErrRegistrationTokenInvalid) {
		t.Fatalf("expected ErrRegistrationTokenInvalid, got %v", err)
	}
}

func TestAgentRegistrationServiceMapsTransactionExpiryToInvalidToken(t *testing.T) {
	agentRepo := &agentRepoStub{createErr: agentdomain.ErrRegistrationTokenInvalid}
	tokenRepo := &tokenRepoStub{findResult: &agentdomain.RegistrationToken{ID: 11, Token: "abcd1234"}}
	service := NewAgentRegistrationService(
		agentRepo,
		tokenRepo,
		fixedClock{now: time.Now().UTC()},
		&tokenGenStub{values: []string{"deadbeef"}},
	)
	_, err := service.RegisterAgent(context.Background(), "abcd1234", "host", "1.0", agentdomain.AgentRegistrationOptions{})
	if !errors.Is(err, ErrRegistrationTokenInvalid) {
		t.Fatalf("transaction expiry error = %v, want ErrRegistrationTokenInvalid", err)
	}
}

func TestRegistrationTokenCleanupJobRunsAsynchronouslyWithRetentionCutoff(t *testing.T) {
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	cutoffs := make(chan time.Time, 1)
	job := NewRegistrationTokenCleanupJob(
		&tokenRepoStub{deleteCutoffs: cutoffs},
		fixedClock{now: now},
		time.Hour,
	)
	ctx, cancel := context.WithCancel(context.Background())
	job.Start(ctx)
	select {
	case cutoff := <-cutoffs:
		if !cutoff.Equal(now.Add(-RegistrationTokenNeverAttributedRetention)) {
			t.Fatalf("cleanup cutoff = %s", cutoff)
		}
	case <-time.After(time.Second):
		t.Fatal("asynchronous cleanup did not start")
	}
	cancel()
}
