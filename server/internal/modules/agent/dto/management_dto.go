package dto

import (
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type HealthStatus = agentdomain.HealthStatus

type AgentHeartbeatResponse struct {
	CPU                      float64       `json:"cpu"`
	Mem                      float64       `json:"mem"`
	Disk                     float64       `json:"disk"`
	RunningTasks             int           `json:"runningTasks"`
	TaskSlotsUsed            int           `json:"taskSlotsUsed"`
	Uptime                   int64         `json:"uptime"`
	AgentVersion             string        `json:"agentVersion,omitempty"`
	OperatingSystem          string        `json:"operatingSystem,omitempty"`
	Architecture             string        `json:"architecture,omitempty"`
	ContainerRuntimeReady    bool          `json:"containerRuntimeReady"`
	SupportedEngineAPIMajors []uint32      `json:"supportedEngineApiMajors"`
	UpdatedAt                time.Time     `json:"updatedAt"`
	Health                   *HealthStatus `json:"health,omitempty"`
}

type AgentResponse struct {
	ID               int                     `json:"id"`
	Name             string                  `json:"name"`
	InstanceID       string                  `json:"instanceId"`
	DisplayName      string                  `json:"displayName"`
	Status           string                  `json:"status"`
	ObservedHostname string                  `json:"observedHostname,omitempty"`
	ConnectionIP     string                  `json:"connectionIp,omitempty"`
	AgentVersion     string                  `json:"agentVersion,omitempty"`
	MaxTasks         int                     `json:"maxTasks"`
	CPUThreshold     int                     `json:"cpuThreshold"`
	MemThreshold     int                     `json:"memThreshold"`
	DiskThreshold    int                     `json:"diskThreshold"`
	ConnectedAt      *time.Time              `json:"connectedAt,omitempty"`
	LastHeartbeat    *time.Time              `json:"lastHeartbeat,omitempty"`
	Health           HealthStatus            `json:"health"`
	Heartbeat        *AgentHeartbeatResponse `json:"heartbeat,omitempty"`
	CreatedAt        time.Time               `json:"createdAt"`
}

type AgentLocationResponse struct {
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	AccuracyRadiusKM *float64  `json:"accuracyRadiusKm"`
	SourceObservedIP string    `json:"sourceObservedIp"`
	ProviderKey      string    `json:"providerKey"`
	ResolvedAt       time.Time `json:"resolvedAt"`
}

type AgentDetailResponse struct {
	AgentResponse
	ObservedSourceIP     string                 `json:"observedSourceIp,omitempty"`
	ObservedIPGeneration int64                  `json:"observedIpGeneration"`
	LocationState        string                 `json:"locationState"`
	Location             *AgentLocationResponse `json:"location"`
}

type AgentListQuery struct {
	PageSize  int    `form:"pageSize" binding:"omitempty,min=1,max=1000"`
	PageToken string `form:"pageToken" binding:"omitempty"`
	Filter    string `form:"filter" binding:"omitempty"`
	OrderBy   string `form:"orderBy" binding:"omitempty"`
}

func (q *AgentListQuery) GetPageSize() int {
	if q.PageSize <= 0 {
		return 20
	}
	return q.PageSize
}

type UpdateAgentConfigRequest struct {
	Name          string `json:"name" binding:"required"`
	UpdateMask    string `json:"updateMask" binding:"required"`
	MaxTasks      *int   `json:"maxTasks" binding:"omitempty,min=1"`
	CPUThreshold  *int   `json:"cpuThreshold" binding:"omitempty,min=1,max=100"`
	MemThreshold  *int   `json:"memThreshold" binding:"omitempty,min=1,max=100"`
	DiskThreshold *int   `json:"diskThreshold" binding:"omitempty,min=1,max=100"`
}
