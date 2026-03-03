package model

import (
	"time"
)

// Agent represents a persistent agent service running on remote VPS.
type Agent struct {
	ID     int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name   string `gorm:"type:varchar(100);not null" json:"name"`
	APIKey string `gorm:"type:varchar(8);not null;uniqueIndex" json:"api_key"`
	Status string `gorm:"type:varchar(20);default:'offline'" json:"status"`

	Hostname      string `gorm:"type:varchar(255)" json:"hostname"`
	IPAddress     string `gorm:"type:varchar(45)" json:"ip_address"`
	AgentVersion  string `gorm:"column:agent_version;type:varchar(20)" json:"agent_version"`
	WorkerVersion string `gorm:"column:worker_version;type:varchar(64)" json:"worker_version"`

	MaxTasks      int `gorm:"default:5" json:"max_tasks"`
	CPUThreshold  int `gorm:"default:85" json:"cpu_threshold"`
	MemThreshold  int `gorm:"default:85" json:"mem_threshold"`
	DiskThreshold int `gorm:"default:90" json:"disk_threshold"`

	HealthState   string     `gorm:"type:varchar(20);default:'ok'" json:"health_state"`
	HealthReason  string     `gorm:"type:varchar(64)" json:"health_reason,omitempty"`
	HealthMessage string     `gorm:"type:varchar(256)" json:"health_message,omitempty"`
	HealthSince   *time.Time `gorm:"type:timestamptz" json:"health_since,omitempty"`

	RegistrationToken string `gorm:"type:varchar(8)" json:"registration_token,omitempty"`

	ConnectedAt   *time.Time `gorm:"type:timestamptz" json:"connected_at,omitempty"`
	LastHeartbeat *time.Time `gorm:"type:timestamptz" json:"last_heartbeat,omitempty"`
	CreatedAt     time.Time  `gorm:"type:timestamptz;default:now()" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"type:timestamptz;default:now()" json:"updated_at"`
}

func (Agent) TableName() string {
	return "agent"
}

// RegistrationToken represents a token for agent self-registration.
type RegistrationToken struct {
	ID        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Token     string    `gorm:"type:varchar(8);not null;uniqueIndex" json:"token"`
	ExpiresAt time.Time `gorm:"type:timestamptz;not null;default:now() + interval '1 hour'" json:"expires_at"`
	CreatedAt time.Time `gorm:"type:timestamptz;default:now()" json:"created_at"`
}

func (RegistrationToken) TableName() string {
	return "registration_token"
}
