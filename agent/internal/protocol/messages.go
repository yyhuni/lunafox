package protocol

import (
	"time"

	"github.com/yyhuni/lunafox/agent/internal/domain"
)

const (
	MessageTypeHeartbeat      = "heartbeat"
	MessageTypeTaskAvailable  = "task_available"
	MessageTypeTaskCancel     = "task_cancel"
	MessageTypeConfigUpdate   = "config_update"
	MessageTypeUpdateRequired = "update_required"
	MessageTypeLogOpen        = "log_open"
	MessageTypeLogCancel      = "log_cancel"
	MessageTypeLogStarted     = "log_started"
	MessageTypeLogChunk       = "log_chunk"
	MessageTypeLogEnd         = "log_end"
	MessageTypeLogError       = "log_error"
)

type Message struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
}

type HealthStatus = domain.HealthStatus

type HeartbeatPayload struct {
	CPU      float64      `json:"cpu"`
	Mem      float64      `json:"mem"`
	Disk     float64      `json:"disk"`
	Tasks    int          `json:"tasks"`
	Version  string       `json:"version"`
	Hostname string       `json:"hostname"`
	Uptime   int64        `json:"uptime"`
	Health   HealthStatus `json:"health"`
}

type ConfigUpdatePayload = domain.ConfigUpdate

type UpdateRequiredPayload = domain.UpdateRequiredPayload

type TaskCancelPayload struct {
	TaskID int `json:"taskId"`
}

type LogOpenPayload struct {
	RequestID  string     `json:"requestId"`
	Container  string     `json:"container"`
	Tail       int        `json:"tail"`
	Follow     bool       `json:"follow"`
	Since      *time.Time `json:"since,omitempty"`
	Timestamps bool       `json:"timestamps"`
}

type LogCancelPayload struct {
	RequestID string `json:"requestId"`
}

type LogStartedPayload struct {
	RequestID string `json:"requestId"`
}

type LogChunkPayload struct {
	RequestID string    `json:"requestId"`
	TS        time.Time `json:"ts"`
	Stream    string    `json:"stream"`
	Line      string    `json:"line"`
	Truncated bool      `json:"truncated"`
}

type LogEndPayload struct {
	RequestID string `json:"requestId"`
	Reason    string `json:"reason,omitempty"`
}

type LogErrorPayload struct {
	RequestID string `json:"requestId"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}
