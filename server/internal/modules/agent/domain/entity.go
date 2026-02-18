package domain

import (
	"strings"
	"time"
)

// Agent represents a worker node in domain layer.
type Agent struct {
	ID                int
	Name              string
	APIKey            string
	Status            string
	Hostname          string
	IPAddress         string
	Version           string
	MaxTasks          int
	CPUThreshold      int
	MemThreshold      int
	DiskThreshold     int
	HealthState       string
	HealthReason      string
	HealthMessage     string
	HealthSince       *time.Time
	RegistrationToken string
	ConnectedAt       *time.Time
	LastHeartbeat     *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// RegistrationToken represents registration capability for agent onboarding.
type RegistrationToken struct {
	ID        int
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

func NewRegistrationToken(token string, expiresAt time.Time) *RegistrationToken {
	return &RegistrationToken{Token: token, ExpiresAt: expiresAt}
}

func NewRegisteredAgent(token, hostname, version, ipAddress, apiKey string, options AgentRegistrationOptions) *Agent {
	normalizedHost := NormalizeAgentHostname(hostname)

	agent := &Agent{
		Name:              BuildAgentName(normalizedHost),
		APIKey:            apiKey,
		Status:            "offline",
		Hostname:          normalizedHost,
		IPAddress:         strings.TrimSpace(ipAddress),
		Version:           strings.TrimSpace(version),
		RegistrationToken: token,
	}
	agent.ApplyConfigUpdate(AgentConfigUpdate(options))
	return agent
}

func (agent *Agent) ApplyConfigUpdate(update AgentConfigUpdate) {
	config := ApplyAgentConfig(AgentConfig{
		MaxTasks:      agent.MaxTasks,
		CPUThreshold:  agent.CPUThreshold,
		MemThreshold:  agent.MemThreshold,
		DiskThreshold: agent.DiskThreshold,
	}, update)

	agent.MaxTasks = config.MaxTasks
	agent.CPUThreshold = config.CPUThreshold
	agent.MemThreshold = config.MemThreshold
	agent.DiskThreshold = config.DiskThreshold
}
