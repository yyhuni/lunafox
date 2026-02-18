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

// HealthStatus captures agent health state.
type HealthStatus struct {
	State   string     `json:"state"`
	Reason  string     `json:"reason,omitempty"`
	Message string     `json:"message,omitempty"`
	Since   *time.Time `json:"since,omitempty"`
}
