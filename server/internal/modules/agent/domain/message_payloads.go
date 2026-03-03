package domain

import "time"

// ConfigUpdatePayload carries scheduler config updates sent to agents.
type ConfigUpdatePayload struct {
	MaxTasks      *int
	CPUThreshold  *int
	MemThreshold  *int
	DiskThreshold *int
}

// UpdateRequiredPayload carries self-update instructions sent to agents.
type UpdateRequiredPayload struct {
	AgentVersion   string
	AgentImageRef  string
	WorkerImageRef string
	WorkerVersion  string
}

// HealthStatus is the shared agent health shape used by API/cache projections.
type HealthStatus struct {
	State   string     `json:"state"`
	Reason  string     `json:"reason,omitempty"`
	Message string     `json:"message,omitempty"`
	Since   *time.Time `json:"since,omitempty"`
}
