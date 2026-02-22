package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/agentproto"
	"github.com/yyhuni/lunafox/server/internal/cache"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type runtimeRepoStub struct {
	agentRepoStub
	heartbeats []agentdomain.AgentHeartbeatUpdate
	statuses   []string
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
}

func (cacheStore *cacheStub) Set(_ context.Context, _ int, _ *cache.HeartbeatData) error {
	cacheStore.setCalled = true
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
	lastUpdatePayload agentproto.UpdateRequiredPayload
}

func (publisher *publisherStub) SendConfigUpdate(int, agentproto.ConfigUpdatePayload) {
	publisher.configSent = true
}
func (publisher *publisherStub) SendUpdateRequired(_ int, payload agentproto.UpdateRequiredPayload) bool {
	publisher.updateSent = true
	publisher.lastUpdatePayload = payload
	return publisher.updateSendSuccess
}
func (publisher *publisherStub) SendTaskCancel(int, int)                             {}
func (publisher *publisherStub) SendLogOpen(int, agentproto.LogOpenPayload) bool     { return true }
func (publisher *publisherStub) SendLogCancel(int, agentproto.LogCancelPayload) bool { return true }

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
	)

	message := RuntimeMessageInput{
		Type: RuntimeMessageTypeHeartbeat,
		Heartbeat: &HeartbeatItem{
			Version:  "1.0.0",
			Hostname: "node1",
			CPU:      1,
			Mem:      2,
			Disk:     3,
			Tasks:    1,
			Uptime:   10,
		},
	}

	err := service.HandleMessage(context.Background(), 1, message)
	if err != nil {
		t.Fatalf("HandleMessage error: %v", err)
	}
	if len(repo.heartbeats) != 1 {
		t.Fatalf("expected 1 heartbeat update")
	}
	if !cacheStore.setCalled {
		t.Fatalf("expected heartbeat cache set")
	}
	if !publisher.updateSent {
		t.Fatalf("expected update_required notification")
	}
	if publisher.lastUpdatePayload.Version != "2.0.0" {
		t.Fatalf("expected update payload version 2.0.0, got %q", publisher.lastUpdatePayload.Version)
	}
	if publisher.lastUpdatePayload.ImageRef != "img" {
		t.Fatalf("expected update payload imageRef img, got %q", publisher.lastUpdatePayload.ImageRef)
	}
}

func TestAgentRuntimeServiceOnDisconnected(t *testing.T) {
	repo := &runtimeRepoStub{}
	cacheStore := &cacheStub{}
	service := NewAgentRuntimeService(repo, cacheStore, &publisherStub{}, fixedClock{now: time.Now().UTC()}, "", "")
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

func TestAgentRuntimeServiceCacheFailureNonBlocking(t *testing.T) {
	repo := &runtimeRepoStub{}
	cacheStore := &cacheStub{setErr: errors.New("boom")}
	service := NewAgentRuntimeService(repo, cacheStore, &publisherStub{}, fixedClock{now: time.Now().UTC()}, "", "")
	message := RuntimeMessageInput{
		Type: RuntimeMessageTypeHeartbeat,
		Heartbeat: &HeartbeatItem{
			Version:  "1.0.0",
			Hostname: "node1",
		},
	}

	if err := service.HandleMessage(context.Background(), 1, message); err != nil {
		t.Fatalf("expected non-blocking cache write, got %v", err)
	}
}

func TestAgentRuntimeServiceHandleMessageUnknownTypeIgnored(t *testing.T) {
	repo := &runtimeRepoStub{}
	service := NewAgentRuntimeService(repo, nil, &publisherStub{}, fixedClock{now: time.Now().UTC()}, "", "")
	message := RuntimeMessageInput{Type: "unknown"}

	if err := service.HandleMessage(context.Background(), 1, message); err != nil {
		t.Fatalf("expected unknown message type to be ignored, got %v", err)
	}
}

func TestAgentRuntimeServiceHandleMessageInvalidHeartbeatPayload(t *testing.T) {
	repo := &runtimeRepoStub{}
	service := NewAgentRuntimeService(repo, nil, &publisherStub{}, fixedClock{now: time.Now().UTC()}, "", "")
	message := RuntimeMessageInput{
		Type:      RuntimeMessageTypeHeartbeat,
		Heartbeat: nil,
	}

	if err := service.HandleMessage(context.Background(), 1, message); err == nil {
		t.Fatalf("expected heartbeat payload decode error")
	}
}
