package domain

import "time"

// AgentHeartbeatUpdate describes runtime fields persisted on heartbeat updates.
type AgentHeartbeatUpdate struct {
	LastHeartbeat time.Time
	AgentVersion  string
	WorkerVersion string
	Hostname      string
	HealthState   string
	HealthReason  string
	HealthMessage string
	HealthSince   *time.Time
	HasHealth     bool
}

// AgentInboundEventType describes events sent by agent runtime.
type AgentInboundEventType string

const (
	AgentInboundHeartbeat AgentInboundEventType = "heartbeat"
)

// AgentInboundEvent represents a typed inbound runtime event.
type AgentInboundEvent struct {
	Type      AgentInboundEventType
	Heartbeat *AgentHeartbeatEvent
}

// AgentHeartbeatEvent is a value object extracted from heartbeat payload.
type AgentHeartbeatEvent struct {
	CPU           float64
	Mem           float64
	Disk          float64
	Tasks         int
	AgentVersion  string
	WorkerVersion string
	Hostname      string
	Uptime        int64
	Health        *AgentHealthEvent
}

// AgentHealthEvent is the domain representation of health state from runtime.
type AgentHealthEvent struct {
	State   string
	Reason  string
	Message string
	Since   *time.Time
}

// AgentOutboundEventType describes events emitted to agents.
type AgentOutboundEventType string

const (
	AgentOutboundConfigUpdate   AgentOutboundEventType = "config_update"
	AgentOutboundUpdateRequired AgentOutboundEventType = "update_required"
	AgentOutboundTaskCancel     AgentOutboundEventType = "task_cancel"
)

// AgentOutboundEvent is a lightweight typed envelope used by application services.
type AgentOutboundEvent struct {
	Type    AgentOutboundEventType
	AgentID int
}
