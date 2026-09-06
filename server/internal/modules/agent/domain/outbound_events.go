package domain

// AgentInboundEventType describes events sent by the agent control plane.
type AgentInboundEventType string

const (
	AgentInboundHeartbeat AgentInboundEventType = "heartbeat"
)

// AgentInboundEvent represents a typed inbound control-plane event.
type AgentInboundEvent struct {
	Type      AgentInboundEventType
	Heartbeat *AgentHeartbeatEvent
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
