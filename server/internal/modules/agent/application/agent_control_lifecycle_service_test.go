package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/cache"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type runtimeRepoStub struct {
	agentRepoStub
	heartbeats            []agentdomain.AgentHeartbeatUpdate
	statuses              []string
	clearedConnectionIDs  []int
	connections           []agentdomain.AgentConnectionObservation
	connectionObservation agentdomain.AgentConnectionObservation
	connectionErr         error
	heartbeatErr          error
}

func (repo *runtimeRepoStub) Update(_ context.Context, agent *agentdomain.Agent) error {
	return nil
}

func (repo *runtimeRepoStub) RecordConnection(_ context.Context, agentID int, connectionIP string, _ time.Time) (agentdomain.AgentConnectionObservation, error) {
	observation := repo.connectionObservation
	if observation.AgentID == 0 {
		observation = agentdomain.AgentConnectionObservation{AgentID: agentID, ConnectionIP: connectionIP, SourceIP: connectionIP}
	}
	repo.connections = append(repo.connections, observation)
	return observation, repo.connectionErr
}

func (repo *runtimeRepoStub) ClearConnectionIP(_ context.Context, agentID int) error {
	repo.clearedConnectionIDs = append(repo.clearedConnectionIDs, agentID)
	return nil
}

func (repo *runtimeRepoStub) UpdateHeartbeat(_ context.Context, _ int, update agentdomain.AgentHeartbeatUpdate) error {
	repo.heartbeats = append(repo.heartbeats, update)
	return repo.heartbeatErr
}
func (repo *runtimeRepoStub) UpdateStatus(_ context.Context, _ int, status string) error {
	repo.statuses = append(repo.statuses, status)
	return nil
}

func (repo *runtimeRepoStub) MarkOfflineFromOnline(context.Context, int) (bool, error) {
	return false, nil
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
func (publisher *publisherStub) SendTaskCancel(int, int, int) {}

func (publisher *publisherStub) TrySendTaskCancel(int, int, int) bool { return true }

func TestAgentControlLifecycleServiceRecordHeartbeatAndUpdateRequired(t *testing.T) {
	repo := &runtimeRepoStub{}
	cacheStore := &cacheStub{}
	publisher := &publisherStub{updateSendSuccess: true}
	service := NewAgentControlLifecycleService(
		repo,
		cacheStore,
		publisher,
		fixedClock{now: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)},
		"2.0.0",
		"img",
	)

	err := service.RecordHeartbeat(context.Background(), 1, agentdomain.AgentHeartbeatEvent{
		InstanceID:               "agt-1",
		SessionID:                "session-1",
		SessionEpoch:             4,
		ObservedHostname:         "node1",
		AgentVersion:             "1.0.0",
		OperatingSystem:          "linux",
		Architecture:             "arm64",
		ContainerRuntimeReady:    true,
		SupportedEngineAPIMajors: []uint32{2},
		CPU:                      1,
		Mem:                      2,
		Disk:                     3,
		RunningTasks:             1,
		TaskSlotsUsed:            2,
		Uptime:                   10,
		Health: &agentdomain.AgentHealthEvent{
			State: "healthy",
		},
	})
	if err != nil {
		t.Fatalf("RecordHeartbeat error: %v", err)
	}
	if len(repo.heartbeats) != 1 {
		t.Fatalf("expected 1 heartbeat update")
	}
	if repo.heartbeats[0].ObservedHostname != "node1" {
		t.Fatalf("expected observed hostname persisted, got %q", repo.heartbeats[0].ObservedHostname)
	}
	if repo.heartbeats[0].SessionID != "session-1" || repo.heartbeats[0].SessionEpoch != 4 || cacheStore.lastSet == nil || cacheStore.lastSet.SessionEpoch != 4 {
		t.Fatalf("session epoch was not persisted/cached coherently: update=%#v cache=%#v", repo.heartbeats[0], cacheStore.lastSet)
	}
	if repo.heartbeats[0].OperatingSystem != "linux" || repo.heartbeats[0].Architecture != "arm64" || !repo.heartbeats[0].ContainerRuntimeReady || len(repo.heartbeats[0].SupportedEngineAPIMajors) != 1 || repo.heartbeats[0].SupportedEngineAPIMajors[0] != 2 || repo.heartbeats[0].RunningTasks != 1 || repo.heartbeats[0].TaskSlotsUsed != 2 {
		t.Fatalf("execution snapshot was not persisted coherently: %#v", repo.heartbeats[0])
	}
	if !cacheStore.setCalled {
		t.Fatalf("expected heartbeat cache set")
	}
	if cacheStore.lastSet == nil || cacheStore.lastSet.Health == nil || cacheStore.lastSet.Health.State != "healthy" {
		t.Fatalf("expected healthy health state cached, got %#v", cacheStore.lastSet)
	}
	if cacheStore.lastSet.RunningTasks != 1 {
		t.Fatalf("expected running tasks cached, got %d", cacheStore.lastSet.RunningTasks)
	}
	if cacheStore.lastSet.TaskSlotsUsed != 2 {
		t.Fatalf("expected task slots used cached, got %d", cacheStore.lastSet.TaskSlotsUsed)
	}
	if cacheStore.lastSet.OperatingSystem != "linux" || cacheStore.lastSet.Architecture != "arm64" || !cacheStore.lastSet.ContainerRuntimeReady || len(cacheStore.lastSet.SupportedEngineAPIMajors) != 1 || cacheStore.lastSet.SupportedEngineAPIMajors[0] != 2 {
		t.Fatalf("execution snapshot was not cached coherently: %#v", cacheStore.lastSet)
	}
	if !publisher.updateSent {
		t.Fatalf("expected update_required notification")
	}
}

func TestAgentControlLifecycleServiceDoesNotCacheRejectedStaleHeartbeat(t *testing.T) {
	repository := &runtimeRepoStub{heartbeatErr: agentdomain.ErrStaleAgentHeartbeat}
	cacheStore := &cacheStub{}
	service := NewAgentControlLifecycleService(repository, cacheStore, nil, fixedClock{now: time.Now().UTC()}, "", "")
	err := service.RecordHeartbeat(context.Background(), 1, agentdomain.AgentHeartbeatEvent{SessionID: "stale", SessionEpoch: 1})
	if !errors.Is(err, agentdomain.ErrStaleAgentHeartbeat) {
		t.Fatalf("RecordHeartbeat() error = %v, want stale heartbeat", err)
	}
	if cacheStore.setCalled {
		t.Fatal("rejected stale heartbeat must not overwrite current cache snapshot")
	}
}

func TestAgentControlLifecycleServiceRemovesGenericMessageEntry(t *testing.T) {
	service := NewAgentControlLifecycleService(&runtimeRepoStub{}, nil, &publisherStub{}, fixedClock{now: time.Now().UTC()}, "", "")
	if reflect.ValueOf(service).MethodByName("HandleMessage").IsValid() {
		t.Fatalf("expected generic HandleMessage to be removed in favor of explicit lifecycle actions")
	}
}

func TestAgentControlLifecycleServiceOnDisconnected(t *testing.T) {
	repo := &runtimeRepoStub{}
	cacheStore := &cacheStub{}
	service := NewAgentControlLifecycleService(repo, cacheStore, &publisherStub{}, fixedClock{now: time.Now().UTC()}, "", "")
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

func TestAgentControlLifecycleServiceOnControlConnectionDetachedClearsAddressWithoutChangingLiveness(t *testing.T) {
	repo := &runtimeRepoStub{}
	cacheStore := &cacheStub{}
	service := NewAgentControlLifecycleService(repo, cacheStore, &publisherStub{}, fixedClock{now: time.Now().UTC()}, "", "")

	if err := service.OnControlConnectionDetached(context.Background(), 7); err != nil {
		t.Fatalf("OnControlConnectionDetached error: %v", err)
	}
	if len(repo.clearedConnectionIDs) != 1 || repo.clearedConnectionIDs[0] != 7 {
		t.Fatalf("clear connection calls = %#v, want [7]", repo.clearedConnectionIDs)
	}
	if len(repo.statuses) != 0 || cacheStore.deleted {
		t.Fatalf("transport detach must preserve liveness while recovery remains possible: statuses=%#v cacheDeleted=%t", repo.statuses, cacheStore.deleted)
	}
}

func TestAgentControlLifecycleServiceOnConnectedRecordsConnectionAndObservedSource(t *testing.T) {
	repo := &runtimeRepoStub{connectionObservation: agentdomain.AgentConnectionObservation{AgentID: 1, ConnectionIP: "8.8.8.8", SourceIP: "8.8.8.8", Generation: 3, SourceChanged: true}}
	now := time.Date(2026, 2, 28, 18, 0, 0, 0, time.UTC)
	service := NewAgentControlLifecycleService(repo, nil, &publisherStub{}, fixedClock{now: now}, "", "")
	agent := &agentdomain.Agent{ID: 1, Location: &agentdomain.AgentLocationSnapshot{SourceObservedIP: "8.8.8.8"}}

	if err := service.OnConnected(context.Background(), agent, "8.8.8.8"); err != nil {
		t.Fatalf("OnConnected error: %v", err)
	}
	if agent.ConnectionIP != "8.8.8.8" || agent.ObservedSourceIP != "8.8.8.8" || agent.ObservedIPGeneration != 3 {
		t.Fatalf("expected persisted connection copied to Agent, got connection=%q source=%q generation=%d", agent.ConnectionIP, agent.ObservedSourceIP, agent.ObservedIPGeneration)
	}
	if len(repo.connections) != 1 || repo.connections[0].SourceIP != "8.8.8.8" {
		t.Fatalf("expected connection observation write, got %#v", repo.connections)
	}
	if !agent.Location.ForcedExpired {
		t.Fatal("source generation change must expire the request-local prior success")
	}
}

func TestAgentControlLifecycleServiceOnConnectedPersistsUnknownSource(t *testing.T) {
	repo := &runtimeRepoStub{connectionObservation: agentdomain.AgentConnectionObservation{AgentID: 2, ConnectionIP: "", SourceIP: "", Generation: 4, SourceChanged: true}}
	now := time.Date(2026, 2, 28, 18, 5, 0, 0, time.UTC)
	service := NewAgentControlLifecycleService(repo, nil, &publisherStub{}, fixedClock{now: now}, "", "")
	agent := &agentdomain.Agent{ID: 2, ObservedSourceIP: "8.8.8.8", ObservedIPGeneration: 3}

	if err := service.OnConnected(context.Background(), agent, ""); err != nil {
		t.Fatalf("OnConnected error: %v", err)
	}
	if agent.ObservedSourceIP != "" || agent.ObservedIPGeneration != 4 {
		t.Fatalf("expected public-to-unknown observation, got source=%q generation=%d", agent.ObservedSourceIP, agent.ObservedIPGeneration)
	}
	if len(repo.connections) != 1 || repo.connections[0].SourceIP != "" {
		t.Fatalf("expected unknown source to be persisted, got %#v", repo.connections)
	}
}

func TestAgentControlLifecycleServiceOnConnectedRejectsMalformedSource(t *testing.T) {
	repo := &runtimeRepoStub{}
	service := NewAgentControlLifecycleService(repo, nil, nil, fixedClock{now: time.Now().UTC()}, "", "")
	if err := service.OnConnected(context.Background(), &agentdomain.Agent{ID: 3}, "not-an-ip"); err == nil {
		t.Fatal("expected malformed observed source to fail fast")
	}
	if len(repo.connections) != 0 {
		t.Fatalf("malformed source must not reach persistence, got %#v", repo.connections)
	}
}

func TestAgentControlLifecycleServiceCacheFailureNonBlocking(t *testing.T) {
	repo := &runtimeRepoStub{}
	cacheStore := &cacheStub{setErr: errors.New("boom")}
	service := NewAgentControlLifecycleService(repo, cacheStore, &publisherStub{}, fixedClock{now: time.Now().UTC()}, "", "")
	if err := service.RecordHeartbeat(context.Background(), 1, agentdomain.AgentHeartbeatEvent{
		InstanceID:       "agt-1",
		SessionID:        "session-1",
		ObservedHostname: "node1",
		AgentVersion:     "1.0.0",
	}); err != nil {
		t.Fatalf("expected non-blocking cache write, got %v", err)
	}
}
