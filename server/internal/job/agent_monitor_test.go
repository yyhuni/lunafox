package job

import (
	"context"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	scanmodel "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
	pkg "github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type fakeAgentRepo struct {
	agents  []*agentdomain.Agent
	updated []int
}

func (f *fakeAgentRepo) Create(ctx context.Context, agent *agentdomain.Agent) error {
	return nil
}

func (f *fakeAgentRepo) GetByID(ctx context.Context, id int) (*agentdomain.Agent, error) {
	return nil, nil
}

func (f *fakeAgentRepo) FindByAPIKey(ctx context.Context, apiKey string) (*agentdomain.Agent, error) {
	return nil, nil
}

func (f *fakeAgentRepo) List(ctx context.Context, page, pageSize int, status string) ([]*agentdomain.Agent, int64, error) {
	return nil, 0, nil
}

func (f *fakeAgentRepo) FindStaleOnline(ctx context.Context, before time.Time) ([]*agentdomain.Agent, error) {
	return f.agents, nil
}

func (f *fakeAgentRepo) Update(ctx context.Context, agent *agentdomain.Agent) error {
	return nil
}

func (f *fakeAgentRepo) UpdateStatus(ctx context.Context, id int, status string) error {
	f.updated = append(f.updated, id)
	return nil
}

func (f *fakeAgentRepo) UpdateHeartbeat(ctx context.Context, id int, update agentdomain.AgentHeartbeatUpdate) error {
	return nil
}

func (f *fakeAgentRepo) Delete(ctx context.Context, id int) error {
	return nil
}

type fakeScanTaskRepo struct {
	recovered []int
}

func (f *fakeScanTaskRepo) FindByID(ctx context.Context, id int) (*scanmodel.ScanTask, error) {
	return nil, nil
}

func (f *fakeScanTaskRepo) PullTask(ctx context.Context, agentID int) (*scanmodel.ScanTask, error) {
	return nil, nil
}

func (f *fakeScanTaskRepo) UpdateStatus(ctx context.Context, id int, status string, errorMessage string) error {
	return nil
}

func (f *fakeScanTaskRepo) GetStatusCountsByScanID(ctx context.Context, scanID int) (pending, running, completed, failed, cancelled int, err error) {
	return 0, 0, 1, 0, 0, nil
}

func (f *fakeScanTaskRepo) CountActiveByScanAndStage(ctx context.Context, scanID, stage int) (int, error) {
	return 0, nil
}

func (f *fakeScanTaskRepo) UnlockNextStage(ctx context.Context, scanID, stage int) (int64, error) {
	return 0, nil
}

func (f *fakeScanTaskRepo) CancelTasksByScanID(ctx context.Context, scanID int) ([]scanrepo.CancelledTaskInfo, error) {
	return nil, nil
}

func (f *fakeScanTaskRepo) FailTasksForOfflineAgent(ctx context.Context, agentID int) error {
	f.recovered = append(f.recovered, agentID)
	return nil
}

func TestAgentMonitorMarksOfflineAndRecovers(t *testing.T) {
	agentRepo := &fakeAgentRepo{
		agents: []*agentdomain.Agent{{ID: 1}, {ID: 2}},
	}
	taskRepo := &fakeScanTaskRepo{}

	monitor := NewAgentMonitor(agentRepo, taskRepo, time.Minute, 2*time.Minute)
	monitor.check(context.Background())

	if len(agentRepo.updated) != 2 {
		t.Fatalf("expected 2 agents updated, got %d", len(agentRepo.updated))
	}
	if len(taskRepo.recovered) != 2 {
		t.Fatalf("expected 2 agents recovered, got %d", len(taskRepo.recovered))
	}
}

func TestAgentMonitorLogsSemanticLastHeartbeat(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	previousLogger := pkg.Logger
	previousSugar := pkg.Sugar
	pkg.Logger = logger
	pkg.Sugar = logger.Sugar()
	defer func() {
		pkg.Logger = previousLogger
		pkg.Sugar = previousSugar
	}()

	heartbeat := time.Now().UTC().Add(-time.Minute)
	agentRepo := &fakeAgentRepo{agents: []*agentdomain.Agent{{ID: 1, LastHeartbeat: &heartbeat}}}
	taskRepo := &fakeScanTaskRepo{}
	monitor := NewAgentMonitor(agentRepo, taskRepo, time.Minute, 2*time.Minute)

	monitor.check(context.Background())

	entries := logs.FilterMessage("Marking agent offline due to stale heartbeat").All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 stale heartbeat log, got %d", len(entries))
	}
	ctx := entries[0].ContextMap()
	if _, ok := ctx["agent.last_heartbeat"]; !ok {
		t.Fatalf("expected agent.last_heartbeat field, got %v", ctx)
	}
	if _, ok := ctx["agent.id"]; !ok {
		t.Fatalf("expected agent.id field, got %v", ctx)
	}
	if _, ok := ctx["last_heartbeat"]; ok {
		t.Fatalf("expected legacy last_heartbeat field removed, got %v", ctx)
	}
	if _, ok := ctx["agent_id"]; ok {
		t.Fatalf("expected legacy agent_id field removed, got %v", ctx)
	}
}
