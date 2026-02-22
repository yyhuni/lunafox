package agentproto

import (
	"encoding/json"
	"time"
)

const (
	MessageTypeHeartbeat      = "heartbeat"
	MessageTypeConfigUpdate   = "config_update"
	MessageTypeTaskCancel     = "task_cancel"
	MessageTypeUpdateRequired = "update_required"
	MessageTypeLogOpen        = "log_open"
	MessageTypeLogCancel      = "log_cancel"
	MessageTypeLogStarted     = "log_started"
	MessageTypeLogChunk       = "log_chunk"
	MessageTypeLogEnd         = "log_end"
	MessageTypeLogError       = "log_error"
)

// Message is the WebSocket message envelope used by agents.
type Message struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

// HeartbeatPayload is sent by agents over WebSocket.
type HeartbeatPayload struct {
	CPU      float64       `json:"cpu"`
	Mem      float64       `json:"mem"`
	Disk     float64       `json:"disk"`
	Tasks    int           `json:"tasks"`
	Version  string        `json:"version"`
	Hostname string        `json:"hostname"`
	Uptime   int64         `json:"uptime"`
	Health   *HealthStatus `json:"health,omitempty"`
}

// ConfigUpdatePayload is sent by the server to update agent scheduling config.
type ConfigUpdatePayload struct {
	MaxTasks      *int `json:"maxTasks"`
	CPUThreshold  *int `json:"cpuThreshold"`
	MemThreshold  *int `json:"memThreshold"`
	DiskThreshold *int `json:"diskThreshold"`
}

// UpdateRequiredPayload instructs agents to self-update.
type UpdateRequiredPayload struct {
	Version  string `json:"version"`
	ImageRef string `json:"imageRef"`
}

// TaskCancelPayload instructs agent to cancel a task.
type TaskCancelPayload struct {
	TaskID int `json:"taskId"`
}

// LogOpenPayload requests an agent to open a container log stream.
type LogOpenPayload struct {
	RequestID  string     `json:"requestId"`
	Container  string     `json:"container"`
	Tail       int        `json:"tail"`
	Follow     bool       `json:"follow"`
	Since      *time.Time `json:"since,omitempty"`
	Timestamps bool       `json:"timestamps"`
}

// LogCancelPayload asks an agent to cancel a running log stream.
type LogCancelPayload struct {
	RequestID string `json:"requestId"`
}

// LogStartedPayload indicates a log stream has started.
type LogStartedPayload struct {
	RequestID string `json:"requestId"`
}

// LogChunkPayload carries one log line from an agent.
type LogChunkPayload struct {
	RequestID string    `json:"requestId"`
	TS        time.Time `json:"ts"`
	Stream    string    `json:"stream"`
	Line      string    `json:"line"`
	Truncated bool      `json:"truncated"`
}

// LogEndPayload marks end-of-stream for a request.
type LogEndPayload struct {
	RequestID string `json:"requestId"`
	Reason    string `json:"reason,omitempty"`
}

// LogErrorPayload carries a structured stream error.
type LogErrorPayload struct {
	RequestID string `json:"requestId"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

// HealthStatus captures agent health state.
type HealthStatus struct {
	State   string     `json:"state"`
	Reason  string     `json:"reason,omitempty"`
	Message string     `json:"message,omitempty"`
	Since   *time.Time `json:"since,omitempty"`
}
