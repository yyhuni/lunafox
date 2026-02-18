package application

import "time"

const (
	RuntimeMessageTypeHeartbeat = "heartbeat"
)

// RuntimeMessageInput is the application-level input for agent runtime messages.
type RuntimeMessageInput struct {
	Type      string
	Heartbeat *HeartbeatItem
}

// HeartbeatItem carries heartbeat metrics used by runtime processing.
type HeartbeatItem struct {
	CPU      float64
	Mem      float64
	Disk     float64
	Tasks    int
	Version  string
	Hostname string
	Uptime   int64
	Health   *HeartbeatHealthItem
}

type HeartbeatHealthItem struct {
	State   string
	Reason  string
	Message string
	Since   *time.Time
}
