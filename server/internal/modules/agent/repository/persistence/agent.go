package model

import (
	"time"

	"gorm.io/datatypes"
)

// Agent represents a persistent agent service running on remote VPS.
type Agent struct {
	ID                  int    `gorm:"primaryKey;autoIncrement;index:idx_agent_created_at_id,priority:2" json:"id"`
	InstanceID          string `gorm:"column:instance_id;type:varchar(64);not null;uniqueIndex" json:"instanceId"`
	DisplayName         string `gorm:"column:display_name;type:varchar(100);not null" json:"displayName"`
	AuthenticationToken string `gorm:"type:varchar(8);not null;uniqueIndex" json:"agentAuthenticationToken"`
	// Status is a coarse-grained summary used for filtering and display.
	// It is reconciled asynchronously from runtime heartbeats and may lag behind runtime facts.
	Status string `gorm:"type:varchar(20);default:'offline';index:idx_agent_status" json:"status"`

	MaxTasks      int `gorm:"default:5" json:"maxTasks"`
	CPUThreshold  int `gorm:"default:85" json:"cpuThreshold"`
	MemThreshold  int `gorm:"default:85" json:"memThreshold"`
	DiskThreshold int `gorm:"default:90" json:"diskThreshold"`

	RegistrationTokenID int `gorm:"column:registration_token_id;not null;index:idx_agent_registration_token_id" json:"registrationTokenId"`

	RuntimeStatus *AgentRuntimeStatus `gorm:"foreignKey:AgentID;references:ID" json:"-"`
	Location      *AgentLocation      `gorm:"foreignKey:AgentID;references:ID" json:"-"`

	CreatedAt time.Time `gorm:"type:timestamptz;default:now();index:idx_agent_created_at_id,priority:1" json:"createdAt"`
	UpdatedAt time.Time `gorm:"type:timestamptz;default:now()" json:"updatedAt"`
}

func (Agent) TableName() string {
	return "agent"
}

type AgentRuntimeStatus struct {
	AgentID                  int            `gorm:"column:agent_id;primaryKey;autoIncrement:false;index:idx_agent_runtime_status_health_state_agent_id,priority:2" json:"agentId"`
	SessionID                string         `gorm:"column:session_id;type:varchar(64)" json:"sessionId,omitempty"`
	SessionEpoch             int64          `gorm:"column:session_epoch" json:"sessionEpoch,omitempty"`
	ObservedHostname         string         `gorm:"column:observed_hostname;type:varchar(255)" json:"observedHostname,omitempty"`
	ConnectionIP             string         `gorm:"column:connection_ip;type:varchar(45);not null;default:''" json:"connectionIp,omitempty"`
	ObservedSourceIP         string         `gorm:"column:observed_source_ip;type:varchar(45);not null;default:''" json:"observedSourceIp,omitempty"`
	ObservedIPGeneration     int64          `gorm:"column:observed_ip_generation;not null;default:0" json:"observedIpGeneration"`
	AgentVersion             string         `gorm:"column:agent_version;type:varchar(20)" json:"agentVersion,omitempty"`
	OperatingSystem          string         `gorm:"column:operating_system;type:varchar(32)" json:"operatingSystem,omitempty"`
	Architecture             string         `gorm:"column:architecture;type:varchar(32)" json:"architecture,omitempty"`
	ContainerRuntimeReady    bool           `gorm:"column:container_runtime_ready;not null;default:false" json:"containerRuntimeReady"`
	SupportedEngineAPIMajors datatypes.JSON `gorm:"column:supported_engine_api_majors;type:jsonb;not null;default:'[]'" json:"supportedEngineApiMajors"`
	ConnectedAt              *time.Time     `gorm:"column:connected_at;type:timestamptz" json:"connectedAt,omitempty"`
	LastHeartbeat            *time.Time     `gorm:"column:last_heartbeat;type:timestamptz" json:"lastHeartbeat,omitempty"`
	// HealthState is the runtime health signal reported by agent heartbeats,
	// such as healthy or paused. It is distinct from Agent.Status.
	HealthState   string     `gorm:"column:health_state;type:varchar(20);default:'healthy';index:idx_agent_runtime_status_health_state_agent_id,priority:1" json:"healthState"`
	HealthReason  string     `gorm:"column:health_reason;type:varchar(64)" json:"healthReason,omitempty"`
	HealthMessage string     `gorm:"column:health_message;type:varchar(256)" json:"healthMessage,omitempty"`
	HealthSince   *time.Time `gorm:"column:health_since;type:timestamptz" json:"healthSince,omitempty"`
	CPUUsage      float64    `gorm:"column:cpu_usage;default:0" json:"cpuUsage"`
	MemUsage      float64    `gorm:"column:mem_usage;default:0" json:"memUsage"`
	DiskUsage     float64    `gorm:"column:disk_usage;default:0" json:"diskUsage"`
	RunningTasks  int        `gorm:"column:running_tasks;default:0" json:"runningTasks"`
	TaskSlotsUsed int        `gorm:"column:task_slots_used;default:0" json:"taskSlotsUsed"`
	UptimeSeconds int64      `gorm:"column:uptime_seconds;default:0" json:"uptimeSeconds"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;type:timestamptz;default:now()" json:"updatedAt"`
}

func (AgentRuntimeStatus) TableName() string {
	return "agent_runtime_status"
}

// RegistrationToken represents a token for agent self-registration.
type RegistrationToken struct {
	ID               int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Token            string     `gorm:"type:varchar(8);not null;uniqueIndex" json:"token"`
	ExpiresAt        time.Time  `gorm:"type:timestamptz;not null;default:now() + interval '1 hour'" json:"expiresAt"`
	EverAttributedAt *time.Time `gorm:"column:ever_attributed_at;type:timestamptz" json:"everAttributedAt,omitempty"`
	CreatedAt        time.Time  `gorm:"type:timestamptz;default:now()" json:"createdAt"`
	Agents           []Agent    `gorm:"foreignKey:RegistrationTokenID;references:ID" json:"-"`
}

func (RegistrationToken) TableName() string {
	return "registration_token"
}
