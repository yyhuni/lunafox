package job

import (
	"context"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

// AgentRepository defines behavior required by AgentMonitor.
type AgentRepository interface {
	FindStaleOnline(ctx context.Context, before time.Time) ([]*agentdomain.Agent, error)
	MarkOfflineFromOnline(ctx context.Context, id int) (bool, error)
}

type agentRuntimeSessionRepository interface {
	FindOnlineRuntimeSessions(ctx context.Context) ([]agentdomain.AgentRuntimeSession, error)
}

// ScanTaskLeaseRecovery defines scan task lease cleanup behavior required by AgentMonitor.
type ScanTaskLeaseRecovery interface {
	FailTasksForOfflineAgent(ctx context.Context, agentID int) ([]int, error)
	FailTasksForSupersededAgentSession(ctx context.Context, agentID int, currentSessionEpoch int64) ([]int, error)
}

// ScanStatusRecalculator defines scan aggregate reconciliation required by AgentMonitor.
type ScanStatusRecalculator interface {
	RecalculateScanStatus(ctx context.Context, scanID int) error
	ReconcilePersistedTerminalTasks(ctx context.Context) error
}

// AgentMonitor marks stale agents offline and recovers their tasks.
type AgentMonitor struct {
	agentRepo         AgentRepository
	taskLeaseRecovery ScanTaskLeaseRecovery
	scanStatus        ScanStatusRecalculator
	interval          time.Duration
	timeout           time.Duration
}

// NewAgentMonitor creates a new AgentMonitor.
func NewAgentMonitor(agentRepo AgentRepository, taskLeaseRecovery ScanTaskLeaseRecovery, scanStatus ScanStatusRecalculator, interval, timeout time.Duration) *AgentMonitor {
	return &AgentMonitor{agentRepo: agentRepo, taskLeaseRecovery: taskLeaseRecovery, scanStatus: scanStatus, interval: interval, timeout: timeout}
}

// Run starts the monitor loop.
func (m *AgentMonitor) Run(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	m.check(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.check(ctx)
		}
	}
}

func (m *AgentMonitor) check(ctx context.Context) {
	m.reconcilePersistedTerminalTasks(ctx)
	defer m.reconcilePersistedTerminalTasks(ctx)

	cutoff := time.Now().UTC().Add(-m.timeout)

	agents, err := m.agentRepo.FindStaleOnline(ctx, cutoff)
	if err != nil {
		pkg.Warn("Failed to query stale agents", zap.Error(err))
		return
	}
	if len(agents) > 0 {
		pkg.Info("Stale agents detected", zap.Int("count", len(agents)), zap.Time("cutoff", cutoff))
	}

	for _, agent := range agents {
		lastHeartbeat := time.Time{}
		if agent.LastHeartbeat != nil {
			lastHeartbeat = *agent.LastHeartbeat
		}
		pkg.Info("Marking agent offline due to stale heartbeat",
			zap.Int("agent.id", agent.ID),
			zap.Time("agent.last_heartbeat", lastHeartbeat),
			zap.Duration("timeout", m.timeout),
		)
		affectedScanIDs, err := m.taskLeaseRecovery.FailTasksForOfflineAgent(ctx, agent.ID)
		if err != nil {
			pkg.Warn("Failed to fail tasks for offline agent", zap.Int("agent.id", agent.ID), zap.Error(err))
			continue
		}
		reconciled := true
		for _, scanID := range affectedScanIDs {
			if m.scanStatus == nil {
				pkg.Warn("Skipped scan status recalculation after offline-agent recovery", zap.Int("scan.id", scanID), zap.Int("agent.id", agent.ID))
				reconciled = false
				continue
			}
			// Task recovery can terminalize rows outside the normal agent result
			// path, so scan aggregation must run through the application resolver.
			if err := m.scanStatus.RecalculateScanStatus(ctx, scanID); err != nil {
				pkg.Warn("Failed to recalculate scan status after offline-agent recovery", zap.Int("scan.id", scanID), zap.Int("agent.id", agent.ID), zap.Error(err))
				reconciled = false
			}
		}
		if !reconciled {
			continue
		}
		transitioned, err := m.agentRepo.MarkOfflineFromOnline(ctx, agent.ID)
		if err != nil {
			pkg.Warn("Failed to mark agent offline", zap.Int("agent.id", agent.ID), zap.Error(err))
			continue
		}
		if !transitioned {
			pkg.Info("Skipped duplicate agent offline transition", zap.Int("agent.id", agent.ID))
		}
	}

	sessionRepo, ok := m.agentRepo.(agentRuntimeSessionRepository)
	if !ok {
		return
	}
	sessions, err := sessionRepo.FindOnlineRuntimeSessions(ctx)
	if err != nil {
		pkg.Warn("Failed to query online agent sessions", zap.Error(err))
		return
	}
	for _, session := range sessions {
		if session.AgentID <= 0 || session.SessionEpoch <= 0 {
			continue
		}
		affectedScanIDs, err := m.taskLeaseRecovery.FailTasksForSupersededAgentSession(ctx, session.AgentID, session.SessionEpoch)
		if err != nil {
			pkg.Warn("Failed to fail tasks for superseded agent session", zap.Int("agent.id", session.AgentID), zap.Int64("agent.session_epoch", session.SessionEpoch), zap.Error(err))
			continue
		}
		for _, scanID := range affectedScanIDs {
			if m.scanStatus == nil {
				pkg.Warn("Skipped scan status recalculation after superseded-session recovery", zap.Int("scan.id", scanID), zap.Int("agent.id", session.AgentID))
				continue
			}
			// Control-plane fencing rejects terminal results from older session
			// epochs, so the monitor must close those stale execution leases.
			if err := m.scanStatus.RecalculateScanStatus(ctx, scanID); err != nil {
				pkg.Warn("Failed to recalculate scan status after superseded-session recovery", zap.Int("scan.id", scanID), zap.Int("agent.id", session.AgentID), zap.Error(err))
			}
		}
	}
}

func (m *AgentMonitor) reconcilePersistedTerminalTasks(ctx context.Context) {
	if m == nil || m.scanStatus == nil {
		return
	}
	if err := m.scanStatus.ReconcilePersistedTerminalTasks(ctx); err != nil {
		pkg.Warn("Failed to reconcile persisted terminal tasks", zap.Error(err))
	}
}
