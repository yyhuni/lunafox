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
	agentByID   map[int]*agentdomain.Agent
	listItems   []*agentdomain.Agent
	options     []agentdomain.FilterOption
	total       int64
	lastPage    int
	lastSize    int
	lastFilter  string
	lastOrderBy string
	getErr      error
	listErr     error
}

func (stub *agentQueryStoreStub) Create(context.Context, *agentdomain.Agent) error { return nil }
func (stub *agentQueryStoreStub) FindByAuthenticationToken(context.Context, string) (*agentdomain.Agent, error) {
	return nil, nil
}
func (stub *agentQueryStoreStub) FindStaleOnline(context.Context, time.Time) ([]*agentdomain.Agent, error) {
	return nil, nil
}
func (stub *agentQueryStoreStub) Update(context.Context, *agentdomain.Agent) error { return nil }
func (stub *agentQueryStoreStub) ClearConnectionIP(context.Context, int) error     { return nil }
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

func (stub *agentQueryStoreStub) List(ctx context.Context, page, pageSize int, filter, orderBy string) ([]*agentdomain.Agent, int64, error) {
	_ = ctx
	stub.lastPage = page
	stub.lastSize = pageSize
	stub.lastFilter = filter
	stub.lastOrderBy = orderBy
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

func (stub *agentQueryStoreStub) ListFilterOptions(ctx context.Context, field string) ([]agentdomain.FilterOption, error) {
	_ = ctx
	_ = field
	return stub.options, nil
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

func (stub *agentCommandStoreStub) FindByAuthenticationToken(context.Context, string) (*agentdomain.Agent, error) {
	return nil, nil
}
func (stub *agentCommandStoreStub) List(context.Context, int, int, string, string) ([]*agentdomain.Agent, int64, error) {
	return nil, 0, nil
}
func (stub *agentCommandStoreStub) ListFilterOptions(context.Context, string) ([]agentdomain.FilterOption, error) {
	return nil, nil
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
func (stub *agentCommandStoreStub) ClearConnectionIP(context.Context, int) error    { return nil }
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

func TestAgentQueryServiceListUsesCanonicalQueryShape(t *testing.T) {
	store := &agentQueryStoreStub{
		listItems: []*agentdomain.Agent{{ID: 1, DisplayName: "edge-01"}, {ID: 2, DisplayName: "edge-02"}},
		total:     3,
	}
	service := NewAgentQueryService(store)

	result, err := service.ListAgents(context.Background(), AgentListQueryInput{
		PageSize: 2,
		Filter:   `(displayName="edge" || observedHostname="edge" || connectionIp="edge") && status=="online"`,
		OrderBy:  "createdAt desc",
	})
	if err != nil {
		t.Fatalf("list agents failed: %v", err)
	}
	if store.lastPage != 1 || store.lastSize != 2 {
		t.Fatalf("unexpected pagination page=%d size=%d", store.lastPage, store.lastSize)
	}
	if store.lastFilter != `(displayName="edge" || observedHostname="edge" || connectionIp="edge") && status=="online"` {
		t.Fatalf("unexpected filter %q", store.lastFilter)
	}
	if store.lastOrderBy != "createdAt desc" {
		t.Fatalf("unexpected orderBy %q", store.lastOrderBy)
	}
	if result.TotalSize != 3 || result.NextPageToken == "" {
		t.Fatalf("expected total and next token, got %+v", result)
	}

	_, err = service.ListAgents(context.Background(), AgentListQueryInput{
		PageSize:  2,
		PageToken: result.NextPageToken,
		Filter:    `displayName="other"`,
		OrderBy:   "createdAt desc",
	})
	if !errors.Is(err, ErrInvalidAgentPageToken) {
		t.Fatalf("expected ErrInvalidAgentPageToken, got %v", err)
	}
}

func TestAgentQueryServiceListPreservesIndependentLastHeartbeat(t *testing.T) {
	lastHeartbeat := time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC)
	service := NewAgentQueryService(&agentQueryStoreStub{
		listItems: []*agentdomain.Agent{{
			ID:            1,
			LastHeartbeat: &lastHeartbeat,
			RunningTasks:  99,
			TaskSlotsUsed: 99,
		}},
		total: 1,
	})

	result, err := service.ListAgents(context.Background(), AgentListQueryInput{})
	if err != nil {
		t.Fatalf("ListAgents() error = %v", err)
	}
	if len(result.Agents) != 1 || result.Agents[0].LastHeartbeat == nil || !result.Agents[0].LastHeartbeat.Equal(lastHeartbeat) {
		t.Fatalf("ListAgents() lost the independent lastHeartbeat projection: %#v", result.Agents)
	}
}

func TestAgentQueryServiceRejectsUnsupportedAgentFilterAndOrderBy(t *testing.T) {
	service := NewAgentQueryService(&agentQueryStoreStub{})

	if _, err := service.ListAgents(context.Background(), AgentListQueryInput{Filter: `created_at="x"`}); !errors.Is(err, ErrUnsupportedAgentFilter) {
		t.Fatalf("expected ErrUnsupportedAgentFilter, got %v", err)
	}
	if _, err := service.ListAgents(context.Background(), AgentListQueryInput{Filter: `ipAddress="8.8.8.8"`}); !errors.Is(err, ErrUnsupportedAgentFilter) {
		t.Fatalf("legacy ipAddress filter must be rejected, got %v", err)
	}
	if _, err := service.ListAgents(context.Background(), AgentListQueryInput{OrderBy: "lastHeartbeat desc"}); !errors.Is(err, ErrUnsupportedAgentOrderBy) {
		t.Fatalf("expected ErrUnsupportedAgentOrderBy, got %v", err)
	}
	if _, err := service.ListFilterOptions(context.Background(), "displayName"); !errors.Is(err, ErrUnsupportedAgentFilter) {
		t.Fatalf("expected ErrUnsupportedAgentFilter for filter options, got %v", err)
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
