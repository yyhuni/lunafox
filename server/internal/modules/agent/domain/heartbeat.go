package domain

import "time"

// AgentHeartbeatUpdate describes runtime fields persisted on heartbeat updates.
type AgentHeartbeatUpdate struct {
	LastHeartbeat time.Time
	InstanceID    string
	// SessionID is the agent-generated process session identifier carried by
	// heartbeats for correlation and stale-session detection.
	SessionID string
	// SessionEpoch is the server-issued fencing token of the active runtime
	// session observed when the heartbeat was recorded.
	SessionEpoch             int64
	ObservedHostname         string
	AgentVersion             string
	OperatingSystem          string
	Architecture             string
	ContainerRuntimeReady    bool
	SupportedEngineAPIMajors []uint32
	CPU                      float64
	Mem                      float64
	Disk                     float64
	RunningTasks             int
	TaskSlotsUsed            int
	Uptime                   int64
	HealthState              string
	HealthReason             string
	HealthMessage            string
	HealthSince              *time.Time
	HasHealth                bool
}

// AgentHeartbeatEvent is a value object extracted from heartbeat payload.
type AgentHeartbeatEvent struct {
	InstanceID string
	// SessionID is the agent-generated process session identifier attached to
	// the runtime connection that emitted this heartbeat.
	SessionID string
	// SessionEpoch is the server-issued fencing token returned when the runtime
	// session became active.
	SessionEpoch             int64
	ObservedHostname         string
	CPU                      float64
	Mem                      float64
	Disk                     float64
	RunningTasks             int
	TaskSlotsUsed            int
	AgentVersion             string
	OperatingSystem          string
	Architecture             string
	ContainerRuntimeReady    bool
	SupportedEngineAPIMajors []uint32
	Uptime                   int64
	Health                   *AgentHealthEvent
}

// AgentExecutionCapabilitySnapshot is the authenticated, session-bound view
// used only for scheduler compatibility and node readiness checks.
type AgentExecutionCapabilitySnapshot struct {
	AgentVersion             string
	OperatingSystem          string
	Architecture             string
	ContainerRuntimeReady    bool
	SupportedEngineAPIMajors []uint32
	RunningTasks             int
	TaskSlotsUsed            int
	ObservedAt               time.Time
}

func (snapshot AgentExecutionCapabilitySnapshot) Clone() AgentExecutionCapabilitySnapshot {
	snapshot.SupportedEngineAPIMajors = append([]uint32(nil), snapshot.SupportedEngineAPIMajors...)
	return snapshot
}

// AgentRuntimeSession is the persisted active runtime session projection used
// by recovery jobs to close stale task execution leases.
type AgentRuntimeSession struct {
	AgentID      int
	SessionEpoch int64
}

// AgentHealthEvent is the domain representation of health state from runtime.
type AgentHealthEvent struct {
	State   string
	Reason  string
	Message string
	Since   *time.Time
}
