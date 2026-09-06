package job

import (
	"context"
	"errors"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	pkg "github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type fakeAgentRepo struct {
	agents             []*agentdomain.Agent
	sessions           []agentdomain.AgentRuntimeSession
	updated            []int
	markOfflineResults []bool
}

func (f *fakeAgentRepo) Create(ctx context.Context, agent *agentdomain.Agent) error {
	return nil
}

func (f *fakeAgentRepo) GetByID(ctx context.Context, id int) (*agentdomain.Agent, error) {
	return nil, nil
}

func (f *fakeAgentRepo) FindByAuthenticationToken(ctx context.Context, authenticationToken string) (*agentdomain.Agent, error) {
	return nil, nil
}

func (f *fakeAgentRepo) List(ctx context.Context, page, pageSize int, filter, orderBy string) ([]*agentdomain.Agent, int64, error) {
	return nil, 0, nil
}

func (f *fakeAgentRepo) ListFilterOptions(ctx context.Context, field string) ([]agentdomain.FilterOption, error) {
	return nil, nil
}

func (f *fakeAgentRepo) FindStaleOnline(ctx context.Context, before time.Time) ([]*agentdomain.Agent, error) {
	return f.agents, nil
}

func (f *fakeAgentRepo) FindOnlineRuntimeSessions(ctx context.Context) ([]agentdomain.AgentRuntimeSession, error) {
	return f.sessions, nil
}

func (f *fakeAgentRepo) Update(ctx context.Context, agent *agentdomain.Agent) error {
	return nil
}

func (f *fakeAgentRepo) UpdateStatus(ctx context.Context, id int, status string) error {
	f.updated = append(f.updated, id)
	return nil
}

func (f *fakeAgentRepo) MarkOfflineFromOnline(ctx context.Context, id int) (bool, error) {
	transitioned := true
	if len(f.markOfflineResults) > 0 {
		transitioned = f.markOfflineResults[0]
		f.markOfflineResults = f.markOfflineResults[1:]
	}
	if transitioned {
		f.updated = append(f.updated, id)
	}
	return transitioned, nil
}

func (f *fakeAgentRepo) UpdateHeartbeat(ctx context.Context, id int, update agentdomain.AgentHeartbeatUpdate) error {
	return nil
}

func (f *fakeAgentRepo) Delete(ctx context.Context, id int) error {
	return nil
}

type fakeScanTaskLeaseRecovery struct {
	recovered           []int
	recoveredScanIDs    []int
	recoveredSessions   []recoveredSessionCall
	recoveredSessionIDs []int
	recalculatedScanID  []int
	recalculateErrs     []error
	terminalReconciles  int
	terminalErr         error
}

type recoveredSessionCall struct {
	agentID             int
	currentSessionEpoch int64
}

func (f *fakeScanTaskLeaseRecovery) FailTasksForOfflineAgent(ctx context.Context, agentID int) ([]int, error) {
	f.recovered = append(f.recovered, agentID)
	return f.recoveredScanIDs, nil
}

func (f *fakeScanTaskLeaseRecovery) FailTasksForSupersededAgentSession(ctx context.Context, agentID int, currentSessionEpoch int64) ([]int, error) {
	f.recoveredSessions = append(f.recoveredSessions, recoveredSessionCall{agentID: agentID, currentSessionEpoch: currentSessionEpoch})
	return f.recoveredSessionIDs, nil
}

func (f *fakeScanTaskLeaseRecovery) RecalculateScanStatus(ctx context.Context, scanID int) error {
	f.recalculatedScanID = append(f.recalculatedScanID, scanID)
	if len(f.recalculateErrs) > 0 {
		err := f.recalculateErrs[0]
		f.recalculateErrs = f.recalculateErrs[1:]
		return err
	}
	return nil
}

func (f *fakeScanTaskLeaseRecovery) ReconcilePersistedTerminalTasks(context.Context) error {
	f.terminalReconciles++
	return f.terminalErr
}

func TestAgentMonitorMarksOfflineAndRecovers(t *testing.T) {
	agentRepo := &fakeAgentRepo{
		agents: []*agentdomain.Agent{{ID: 1}, {ID: 2}},
	}
	taskLeaseRecovery := &fakeScanTaskLeaseRecovery{}

	monitor := NewAgentMonitor(agentRepo, taskLeaseRecovery, taskLeaseRecovery, time.Minute, 2*time.Minute)
	monitor.check(context.Background())

	if len(agentRepo.updated) != 2 {
		t.Fatalf("expected 2 agents updated, got %d", len(agentRepo.updated))
	}
	if len(taskLeaseRecovery.recovered) != 2 {
		t.Fatalf("expected 2 agents recovered, got %d", len(taskLeaseRecovery.recovered))
	}
}

func TestAgentMonitorRecalculatesRecoveredScanStatuses(t *testing.T) {
	agentRepo := &fakeAgentRepo{
		agents: []*agentdomain.Agent{{ID: 7}},
	}
	taskLeaseRecovery := &fakeScanTaskLeaseRecovery{recoveredScanIDs: []int{101, 102}}

	monitor := NewAgentMonitor(agentRepo, taskLeaseRecovery, taskLeaseRecovery, time.Minute, 2*time.Minute)
	monitor.check(context.Background())

	if len(taskLeaseRecovery.recalculatedScanID) != 2 {
		t.Fatalf("expected 2 scans recalculated, got %d", len(taskLeaseRecovery.recalculatedScanID))
	}
	if taskLeaseRecovery.recalculatedScanID[0] != 101 || taskLeaseRecovery.recalculatedScanID[1] != 102 {
		t.Fatalf("unexpected recalculated scans: %+v", taskLeaseRecovery.recalculatedScanID)
	}
}

func TestAgentMonitorRecoversSupersededSessionTasksForOnlineAgent(t *testing.T) {
	agentRepo := &fakeAgentRepo{
		sessions: []agentdomain.AgentRuntimeSession{{AgentID: 7, SessionEpoch: 3}},
	}
	taskLeaseRecovery := &fakeScanTaskLeaseRecovery{recoveredSessionIDs: []int{201}}

	monitor := NewAgentMonitor(agentRepo, taskLeaseRecovery, taskLeaseRecovery, time.Minute, 2*time.Minute)
	monitor.check(context.Background())

	if len(taskLeaseRecovery.recoveredSessions) != 1 {
		t.Fatalf("expected superseded session recovery once, got %+v", taskLeaseRecovery.recoveredSessions)
	}
	if taskLeaseRecovery.recoveredSessions[0].agentID != 7 || taskLeaseRecovery.recoveredSessions[0].currentSessionEpoch != 3 {
		t.Fatalf("unexpected superseded session recovery call: %+v", taskLeaseRecovery.recoveredSessions[0])
	}
	if len(taskLeaseRecovery.recalculatedScanID) != 1 || taskLeaseRecovery.recalculatedScanID[0] != 201 {
		t.Fatalf("expected recovered scan recalculated, got %+v", taskLeaseRecovery.recalculatedScanID)
	}
}

func TestAgentMonitorKeepsStaleAgentOnlineUntilLeaseAndScanRecoveryConverge(t *testing.T) {
	agentRepo := &fakeAgentRepo{agents: []*agentdomain.Agent{{ID: 7}}}
	recovery := &fakeScanTaskLeaseRecovery{
		recoveredScanIDs: []int{101},
		recalculateErrs:  []error{errors.New("scan write unavailable"), nil},
	}
	monitor := NewAgentMonitor(agentRepo, recovery, recovery, time.Minute, 2*time.Minute)

	monitor.check(context.Background())
	if len(agentRepo.updated) != 0 {
		t.Fatalf("agent was marked offline before recovery converged: %v", agentRepo.updated)
	}
	monitor.check(context.Background())
	if len(agentRepo.updated) != 1 || agentRepo.updated[0] != 7 {
		t.Fatalf("agent was not marked offline after retry convergence: %v", agentRepo.updated)
	}
	if len(recovery.recovered) != 2 || len(recovery.recalculatedScanID) != 2 {
		t.Fatalf("offline recovery was not retried: leases=%v scans=%v", recovery.recovered, recovery.recalculatedScanID)
	}
	if recovery.terminalReconciles != 4 {
		t.Fatalf("durable terminal replay runs = %d, want start+end of each check", recovery.terminalReconciles)
	}
}

func TestAgentMonitorDoesNotTreatRepeatedOfflineCheckAsNewTransition(t *testing.T) {
	agentRepo := &fakeAgentRepo{
		agents:             []*agentdomain.Agent{{ID: 7}},
		markOfflineResults: []bool{true, false},
	}
	recovery := &fakeScanTaskLeaseRecovery{}
	monitor := NewAgentMonitor(agentRepo, recovery, recovery, time.Minute, 2*time.Minute)

	monitor.check(context.Background())
	monitor.check(context.Background())

	if len(agentRepo.updated) != 1 || agentRepo.updated[0] != 7 {
		t.Fatalf("offline transitions = %v, want one persisted transition", agentRepo.updated)
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
	taskLeaseRecovery := &fakeScanTaskLeaseRecovery{}
	monitor := NewAgentMonitor(agentRepo, taskLeaseRecovery, taskLeaseRecovery, time.Minute, 2*time.Minute)

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
