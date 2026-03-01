package application

import (
	"context"
	"errors"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"gorm.io/gorm"
)

type agentQueryStoreStub struct {
	agentByID map[int]*agentdomain.Agent
	listItems []*agentdomain.Agent
	total     int64
	getErr    error
	listErr   error
}

func (stub *agentQueryStoreStub) Create(context.Context, *agentdomain.Agent) error { return nil }
func (stub *agentQueryStoreStub) FindByAPIKey(context.Context, string) (*agentdomain.Agent, error) {
	return nil, nil
}
func (stub *agentQueryStoreStub) FindStaleOnline(context.Context, time.Time) ([]*agentdomain.Agent, error) {
	return nil, nil
}
func (stub *agentQueryStoreStub) Update(context.Context, *agentdomain.Agent) error { return nil }
func (stub *agentQueryStoreStub) UpdateStatus(context.Context, int, string) error  { return nil }
func (stub *agentQueryStoreStub) UpdateHeartbeat(context.Context, int, agentdomain.AgentHeartbeatUpdate) error {
	return nil
}
func (stub *agentQueryStoreStub) Delete(context.Context, int) error { return nil }

func (stub *agentQueryStoreStub) GetByID(ctx context.Context, id int) (*agentdomain.Agent, error) {
	_ = ctx
	if stub.getErr != nil {
		return nil, stub.getErr
	}
	agent, ok := stub.agentByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyAgent := *agent
	return &copyAgent, nil
}

func (stub *agentQueryStoreStub) List(ctx context.Context, page, pageSize int, status string) ([]*agentdomain.Agent, int64, error) {
	_ = ctx
	_ = page
	_ = pageSize
	_ = status
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	results := make([]*agentdomain.Agent, 0, len(stub.listItems))
	for _, item := range stub.listItems {
		copyItem := *item
		results = append(results, &copyItem)
	}
	return results, stub.total, nil
}

type agentCommandStoreStub struct {
	agentByID map[int]*agentdomain.Agent
	getErr    error
	updateErr error
	deleteErr error
	updated   *agentdomain.Agent
	deletedID int
}

func (stub *agentCommandStoreStub) Create(ctx context.Context, agent *agentdomain.Agent) error {
	_ = ctx
	_ = agent
	return nil
}

func (stub *agentCommandStoreStub) GetByID(ctx context.Context, id int) (*agentdomain.Agent, error) {
	_ = ctx
	if stub.getErr != nil {
		return nil, stub.getErr
	}
	agent, ok := stub.agentByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyAgent := *agent
	return &copyAgent, nil
}

func (stub *agentCommandStoreStub) FindByAPIKey(context.Context, string) (*agentdomain.Agent, error) {
	return nil, nil
}
func (stub *agentCommandStoreStub) List(context.Context, int, int, string) ([]*agentdomain.Agent, int64, error) {
	return nil, 0, nil
}
func (stub *agentCommandStoreStub) FindStaleOnline(context.Context, time.Time) ([]*agentdomain.Agent, error) {
	return nil, nil
}

func (stub *agentCommandStoreStub) Update(ctx context.Context, agent *agentdomain.Agent) error {
	_ = ctx
	if stub.updateErr != nil {
		return stub.updateErr
	}
	copyAgent := *agent
	stub.updated = &copyAgent
	return nil
}

func (stub *agentCommandStoreStub) UpdateStatus(context.Context, int, string) error { return nil }
func (stub *agentCommandStoreStub) UpdateHeartbeat(context.Context, int, agentdomain.AgentHeartbeatUpdate) error {
	return nil
}

func (stub *agentCommandStoreStub) Delete(ctx context.Context, id int) error {
	_ = ctx
	if stub.deleteErr != nil {
		return stub.deleteErr
	}
	stub.deletedID = id
	return nil
}

func TestAgentQueryServiceGetAgentNotFound(t *testing.T) {
	service := NewAgentQueryService(&agentQueryStoreStub{agentByID: map[int]*agentdomain.Agent{}})

	_, err := service.GetAgent(context.Background(), 7)
	if !errors.Is(err, ErrAgentNotFound) {
		t.Fatalf("expected ErrAgentNotFound, got %v", err)
	}
}

func TestAgentCommandServiceUpdateAndDeleteNotFound(t *testing.T) {
	t.Run("update missing agent maps to domain error", func(t *testing.T) {
		service := NewAgentCommandService(&agentCommandStoreStub{agentByID: map[int]*agentdomain.Agent{}})
		update := agentdomain.AgentConfigUpdate{}

		_, err := service.UpdateAgentConfig(context.Background(), 9, update)
		if !errors.Is(err, ErrAgentNotFound) {
			t.Fatalf("expected ErrAgentNotFound, got %v", err)
		}
	})

	t.Run("delete missing agent maps to domain error", func(t *testing.T) {
		service := NewAgentCommandService(&agentCommandStoreStub{deleteErr: gorm.ErrRecordNotFound})

		err := service.DeleteAgent(context.Background(), 9)
		if !errors.Is(err, ErrAgentNotFound) {
			t.Fatalf("expected ErrAgentNotFound, got %v", err)
		}
	})
}
