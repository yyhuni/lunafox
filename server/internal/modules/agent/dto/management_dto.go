package dto

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/agentproto"
)

type HealthStatus = agentproto.HealthStatus

type AgentHeartbeatResponse struct {
	CPU       float64       `json:"cpu"`
	Mem       float64       `json:"mem"`
	Disk      float64       `json:"disk"`
	Tasks     int           `json:"tasks"`
	Uptime    int64         `json:"uptime"`
	UpdatedAt time.Time     `json:"updatedAt"`
	Health    *HealthStatus `json:"health,omitempty"`
}

type AgentResponse struct {
	ID            int                     `json:"id"`
	Name          string                  `json:"name"`
	Status        string                  `json:"status"`
	Hostname      string                  `json:"hostname,omitempty"`
	IPAddress     string                  `json:"ipAddress,omitempty"`
	Version       string                  `json:"version,omitempty"`
	MaxTasks      int                     `json:"maxTasks"`
	CPUThreshold  int                     `json:"cpuThreshold"`
	MemThreshold  int                     `json:"memThreshold"`
	DiskThreshold int                     `json:"diskThreshold"`
	ConnectedAt   *time.Time              `json:"connectedAt,omitempty"`
	LastHeartbeat *time.Time              `json:"lastHeartbeat,omitempty"`
	Health        HealthStatus            `json:"health"`
	Heartbeat     *AgentHeartbeatResponse `json:"heartbeat,omitempty"`
	CreatedAt     time.Time               `json:"createdAt"`
}

type AgentListResponse struct {
	Results  []AgentResponse `json:"results"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
}

type UpdateAgentConfigRequest struct {
	MaxTasks      *int `json:"maxTasks" binding:"omitempty,min=1,max=20"`
	CPUThreshold  *int `json:"cpuThreshold" binding:"omitempty,min=1,max=100"`
	MemThreshold  *int `json:"memThreshold" binding:"omitempty,min=1,max=100"`
	DiskThreshold *int `json:"diskThreshold" binding:"omitempty,min=1,max=100"`
}
