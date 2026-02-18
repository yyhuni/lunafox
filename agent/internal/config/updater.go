package config

import (
	"sync"

	"github.com/yyhuni/lunafox/agent/internal/domain"
)

// Update holds optional configuration updates.
type Update = domain.ConfigUpdate

// Updater manages runtime configuration changes.
type Updater struct {
	mu  sync.RWMutex
	cfg Config
}

// NewUpdater creates an updater with initial config.
func NewUpdater(cfg Config) *Updater {
	return &Updater{cfg: cfg}
}

// Apply updates the configuration and returns the new snapshot.
func (u *Updater) Apply(update Update) Config {
	u.mu.Lock()
	defer u.mu.Unlock()

	if update.MaxTasks != nil && *update.MaxTasks > 0 {
		u.cfg.MaxTasks = *update.MaxTasks
	}
	if update.CPUThreshold != nil && *update.CPUThreshold > 0 {
		u.cfg.CPUThreshold = *update.CPUThreshold
	}
	if update.MemThreshold != nil && *update.MemThreshold > 0 {
		u.cfg.MemThreshold = *update.MemThreshold
	}
	if update.DiskThreshold != nil && *update.DiskThreshold > 0 {
		u.cfg.DiskThreshold = *update.DiskThreshold
	}

	return u.cfg
}

// Snapshot returns a copy of current config.
func (u *Updater) Snapshot() Config {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.cfg
}
