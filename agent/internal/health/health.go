package health

import (
	"sync"
	"time"

	"github.com/yyhuni/lunafox/agent/internal/domain"
)

// Status represents the agent health state reported in heartbeats.
type Status = domain.HealthStatus

// Manager stores current health status.
type Manager struct {
	mu     sync.RWMutex
	status Status
}

// NewManager initializes the manager with ok status.
func NewManager() *Manager {
	return &Manager{
		status: Status{State: "ok"},
	}
}

// Get returns a snapshot of current status.
func (m *Manager) Get() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// Set updates health status and timestamps transitions.
func (m *Manager) Set(state, reason, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status.State != state {
		now := time.Now().UTC()
		m.status.Since = &now
	}

	m.status.State = state
	m.status.Reason = reason
	m.status.Message = message
	if state == "ok" {
		m.status.Since = nil
		m.status.Reason = ""
		m.status.Message = ""
	}
}
