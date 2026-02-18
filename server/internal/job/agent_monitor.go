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
	UpdateStatus(ctx context.Context, id int, status string) error
}

// ScanTaskRepository defines behavior required by AgentMonitor.
type ScanTaskRepository interface {
	FailTasksForOfflineAgent(ctx context.Context, agentID int) error
}

// AgentMonitor marks stale agents offline and recovers their tasks.
type AgentMonitor struct {
	agentRepo    AgentRepository
	scanTaskRepo ScanTaskRepository
	interval     time.Duration
	timeout      time.Duration
}

// NewAgentMonitor creates a new AgentMonitor.
func NewAgentMonitor(agentRepo AgentRepository, scanTaskRepo ScanTaskRepository, interval, timeout time.Duration) *AgentMonitor {
	return &AgentMonitor{agentRepo: agentRepo, scanTaskRepo: scanTaskRepo, interval: interval, timeout: timeout}
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
			zap.Int("agent_id", agent.ID),
			zap.Time("lastHeartbeat", lastHeartbeat),
			zap.Duration("timeout", m.timeout),
		)
		if err := m.agentRepo.UpdateStatus(ctx, agent.ID, "offline"); err != nil {
			pkg.Warn("Failed to mark agent offline", zap.Int("agent_id", agent.ID), zap.Error(err))
			continue
		}
		if err := m.scanTaskRepo.FailTasksForOfflineAgent(ctx, agent.ID); err != nil {
			pkg.Warn("Failed to fail tasks for offline agent", zap.Int("agent_id", agent.ID), zap.Error(err))
		}
	}
}
