package domain

import (
	"strings"
	"time"
)

// Agent represents an execution node in the domain layer.
type Agent struct {
	ID                       int
	InstanceID               string
	DisplayName              string
	AuthenticationToken      string
	Status                   string
	ObservedHostname         string
	ConnectionIP             string
	ObservedSourceIP         string
	ObservedIPGeneration     int64
	AgentVersion             string
	OperatingSystem          string
	Architecture             string
	ContainerRuntimeReady    bool
	SupportedEngineAPIMajors []uint32
	RunningTasks             int
	TaskSlotsUsed            int
	SessionID                string
	SessionEpoch             int64
	MaxTasks                 int
	CPUThreshold             int
	MemThreshold             int
	DiskThreshold            int
	HealthState              string
	HealthReason             string
	HealthMessage            string
	HealthSince              *time.Time
	RegistrationTokenID      int
	Location                 *AgentLocationSnapshot
	ConnectedAt              *time.Time
	LastHeartbeat            *time.Time
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func NewRegisteredAgent(registrationTokenID int, instanceID, observedHostname, agentVersion, authenticationToken string, options AgentRegistrationOptions) *Agent {
	normalizedHost := NormalizeObservedHostname(observedHostname)
	resolvedDisplayName := BuildDefaultAgentDisplayName(normalizedHost, instanceID)

	agent := &Agent{
		InstanceID:          strings.TrimSpace(instanceID),
		DisplayName:         resolvedDisplayName,
		AuthenticationToken: authenticationToken,
		Status:              "offline",
		HealthState:         "healthy",
		ObservedHostname:    normalizedHost,
		AgentVersion:        strings.TrimSpace(agentVersion),
		RegistrationTokenID: registrationTokenID,
	}
	agent.ApplyConfigUpdate(AgentConfigUpdate(ApplyRegistrationDefaults(options)))
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
