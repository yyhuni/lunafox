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
