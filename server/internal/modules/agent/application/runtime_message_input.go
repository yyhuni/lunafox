package application

import "time"

const (
	RuntimeMessageTypeHeartbeat  = "heartbeat"
	RuntimeMessageTypeLogStarted = "log_started"
	RuntimeMessageTypeLogChunk   = "log_chunk"
	RuntimeMessageTypeLogEnd     = "log_end"
	RuntimeMessageTypeLogError   = "log_error"
)

// RuntimeMessageInput is the application-level input for agent runtime messages.
type RuntimeMessageInput struct {
	Type       string
	Heartbeat  *HeartbeatItem
	LogStarted *LogStartedItem
	LogChunk   *LogChunkItem
	LogEnd     *LogEndItem
	LogError   *LogErrorItem
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

type LogStartedItem struct {
	RequestID string
}

type LogChunkItem struct {
	RequestID string
	TS        time.Time
	Stream    string
	Line      string
	Truncated bool
}

type LogEndItem struct {
	RequestID string
	Reason    string
}

type LogErrorItem struct {
	RequestID string
	Code      string
	Message   string
}
