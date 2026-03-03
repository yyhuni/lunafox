package dto

import "time"

type RegistrationTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type AgentRegistrationRequest struct {
	Token         string `json:"token" binding:"required,len=8"`
	Hostname      string `json:"hostname" binding:"required"`
	AgentVersion  string `json:"agentVersion" binding:"required"`
	WorkerVersion string `json:"workerVersion" binding:"required"`
	MaxTasks      *int   `json:"maxTasks" binding:"omitempty,min=1,max=20"`
	CPUThreshold  *int   `json:"cpuThreshold" binding:"omitempty,min=1,max=100"`
	MemThreshold  *int   `json:"memThreshold" binding:"omitempty,min=1,max=100"`
	DiskThreshold *int   `json:"diskThreshold" binding:"omitempty,min=1,max=100"`
}

type AgentRegistrationResponse struct {
	AgentID int    `json:"agentId"`
	APIKey  string `json:"apiKey"`
}
