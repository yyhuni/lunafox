package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/cache"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type runtimeRepoStub struct {
	agentRepoStub
	heartbeats []agentdomain.AgentHeartbeatUpdate
	statuses   []string
	updated    []*agentdomain.Agent
	updateErr  error
}

func (repo *runtimeRepoStub) Update(_ context.Context, agent *agentdomain.Agent) error {
	if repo.updateErr != nil {
		return repo.updateErr
	}
	copied := *agent
	repo.updated = append(repo.updated, &copied)
	return nil
}

func (repo *runtimeRepoStub) UpdateHeartbeat(_ context.Context, _ int, update agentdomain.AgentHeartbeatUpdate) error {
	repo.heartbeats = append(repo.heartbeats, update)
	return nil
}
func (repo *runtimeRepoStub) UpdateStatus(_ context.Context, _ int, status string) error {
	repo.statuses = append(repo.statuses, status)
	return nil
}

type cacheStub struct {
	setCalled bool
	setErr    error
	deleted   bool
	lastSet   *cache.HeartbeatData
}

func (cacheStore *cacheStub) Set(_ context.Context, _ int, data *cache.HeartbeatData) error {
	cacheStore.setCalled = true
	if data != nil {
		copied := *data
		cacheStore.lastSet = &copied
	}
	return cacheStore.setErr
}
func (cacheStore *cacheStub) Get(_ context.Context, _ int) (*cache.HeartbeatData, error) {
	return nil, nil
}
func (cacheStore *cacheStub) Delete(_ context.Context, _ int) error {
	cacheStore.deleted = true
	return nil
}

type publisherStub struct {
	configSent        bool
	updateSent        bool
	updateSendSuccess bool
	lastUpdatePayload agentdomain.UpdateRequiredPayload
}

func (publisher *publisherStub) SendConfigUpdate(int, agentdomain.ConfigUpdatePayload) {
	publisher.configSent = true
}
func (publisher *publisherStub) SendUpdateRequired(_ int, payload agentdomain.UpdateRequiredPayload) bool {
	publisher.updateSent = true
	publisher.lastUpdatePayload = payload
	return publisher.updateSendSuccess
}
func (publisher *publisherStub) SendTaskCancel(int, int) {}

func TestAgentRuntimeServiceHeartbeatAndUpdateRequired(t *testing.T) {
	repo := &runtimeRepoStub{}
	cacheStore := &cacheStub{}
	publisher := &publisherStub{updateSendSuccess: true}
	service := NewAgentRuntimeService(
		repo,
		cacheStore,
		publisher,
		fixedClock{now: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)},
		"2.0.0",
		"img",
		"wimg",
		"3.1.4",
	)

	message := RuntimeMessageInput{
		Type: RuntimeMessageTypeHeartbeat,
		Heartbeat: &HeartbeatItem{
			AgentVersion:  "1.0.0",
			WorkerVersion: "1.0.0",
			Hostname:      "node1",
			CPU:           1,
			Mem:           2,
			Disk:          3,
			Tasks:         1,
			Uptime:        10,
		},
	}

	err := service.HandleMessage(context.Background(), 1, message)
	if err != nil {
		t.Fatalf("HandleMessage error: %v", err)
	}
	if len(repo.heartbeats) != 1 {
		t.Fatalf("expected 1 heartbeat update")
	}
	if repo.heartbeats[0].AgentVersion != "1.0.0" {
		t.Fatalf("expected agent version persisted, got %q", repo.heartbeats[0].AgentVersion)
	}
	if repo.heartbeats[0].WorkerVersion != "1.0.0" {
		t.Fatalf("expected worker version persisted, got %q", repo.heartbeats[0].WorkerVersion)
	}
	if !cacheStore.setCalled {
		t.Fatalf("expected heartbeat cache set")
	}
	if cacheStore.lastSet == nil || cacheStore.lastSet.WorkerVersion != "1.0.0" {
		t.Fatalf("expected worker version cached, got %#v", cacheStore.lastSet)
	}
	if !publisher.updateSent {
		t.Fatalf("expected update_required notification")
	}
	if publisher.lastUpdatePayload.AgentVersion != "2.0.0" {
		t.Fatalf("expected update payload version 2.0.0, got %q", publisher.lastUpdatePayload.AgentVersion)
	}
	if publisher.lastUpdatePayload.AgentImageRef != "img" {
		t.Fatalf("expected update payload imageRef img, got %q", publisher.lastUpdatePayload.AgentImageRef)
	}
	if publisher.lastUpdatePayload.WorkerImageRef != "wimg" {
		t.Fatalf("expected worker image ref wimg, got %q", publisher.lastUpdatePayload.WorkerImageRef)
	}
	if publisher.lastUpdatePayload.WorkerVersion != "3.1.4" {
		t.Fatalf("expected worker version 3.1.4, got %q", publisher.lastUpdatePayload.WorkerVersion)
	}
}

func TestAgentRuntimeServiceOnDisconnected(t *testing.T) {
	repo := &runtimeRepoStub{}
	cacheStore := &cacheStub{}
	service := NewAgentRuntimeService(repo, cacheStore, &publisherStub{}, fixedClock{now: time.Now().UTC()}, "", "", "", "")
	if err := service.OnDisconnected(context.Background(), 1); err != nil {
		t.Fatalf("OnDisconnected error: %v", err)
	}
	if len(repo.statuses) != 1 || repo.statuses[0] != "offline" {
		t.Fatalf("expected offline status update")
	}
	if !cacheStore.deleted {
		t.Fatalf("expected cache deletion")
	}
}

func TestAgentRuntimeServiceOnConnectedUsesProvidedIPAddress(t *testing.T) {
	repo := &runtimeRepoStub{}
	now := time.Date(2026, 2, 28, 18, 0, 0, 0, time.UTC)
	service := NewAgentRuntimeService(repo, nil, &publisherStub{}, fixedClock{now: now}, "", "", "", "")
	agent := &agentdomain.Agent{ID: 1}

	if err := service.OnConnected(context.Background(), agent, "203.0.113.10"); err != nil {
		t.Fatalf("OnConnected error: %v", err)
	}
	if agent.IPAddress != "203.0.113.10" {
		t.Fatalf("expected ip updated, got %q", agent.IPAddress)
	}
	if len(repo.updated) != 1 || repo.updated[0].IPAddress != "203.0.113.10" {
		t.Fatalf("expected persisted ip update, got %#v", repo.updated)
	}
}

func TestAgentRuntimeServiceOnConnectedDoesNotOverwriteIPAddressWhenEmpty(t *testing.T) {
	repo := &runtimeRepoStub{}
	now := time.Date(2026, 2, 28, 18, 5, 0, 0, time.UTC)
	service := NewAgentRuntimeService(repo, nil, &publisherStub{}, fixedClock{now: now}, "", "", "", "")
	agent := &agentdomain.Agent{ID: 2, IPAddress: "198.51.100.2"}

	if err := service.OnConnected(context.Background(), agent, ""); err != nil {
		t.Fatalf("OnConnected error: %v", err)
	}
	if agent.IPAddress != "198.51.100.2" {
		t.Fatalf("expected existing ip preserved, got %q", agent.IPAddress)
	}
	if len(repo.updated) != 1 || repo.updated[0].IPAddress != "198.51.100.2" {
		t.Fatalf("expected persisted ip unchanged, got %#v", repo.updated)
	}
}

func TestAgentRuntimeServiceCacheFailureNonBlocking(t *testing.T) {
	repo := &runtimeRepoStub{}
	cacheStore := &cacheStub{setErr: errors.New("boom")}
	service := NewAgentRuntimeService(repo, cacheStore, &publisherStub{}, fixedClock{now: time.Now().UTC()}, "", "", "", "")
	message := RuntimeMessageInput{
		Type: RuntimeMessageTypeHeartbeat,
		Heartbeat: &HeartbeatItem{
			AgentVersion: "1.0.0",
			Hostname:     "node1",
		},
	}

	if err := service.HandleMessage(context.Background(), 1, message); err != nil {
		t.Fatalf("expected non-blocking cache write, got %v", err)
	}
}

func TestAgentRuntimeServiceHandleMessageUnknownTypeIgnored(t *testing.T) {
	repo := &runtimeRepoStub{}
	service := NewAgentRuntimeService(repo, nil, &publisherStub{}, fixedClock{now: time.Now().UTC()}, "", "", "", "")
	message := RuntimeMessageInput{Type: "unknown"}

	if err := service.HandleMessage(context.Background(), 1, message); err != nil {
		t.Fatalf("expected unknown message type to be ignored, got %v", err)
	}
}

func TestAgentRuntimeServiceHandleMessageInvalidHeartbeatPayload(t *testing.T) {
	repo := &runtimeRepoStub{}
	service := NewAgentRuntimeService(repo, nil, &publisherStub{}, fixedClock{now: time.Now().UTC()}, "", "", "", "")
	message := RuntimeMessageInput{
		Type:      RuntimeMessageTypeHeartbeat,
		Heartbeat: nil,
	}

	if err := service.HandleMessage(context.Background(), 1, message); err == nil {
		t.Fatalf("expected heartbeat payload decode error")
	}
}
