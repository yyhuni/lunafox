package model

import (
	"time"
)

// Agent represents a persistent agent service running on remote VPS.
type Agent struct {
	ID     int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name   string `gorm:"type:varchar(100);not null" json:"name"`
	APIKey string `gorm:"type:varchar(8);not null;uniqueIndex" json:"apiKey"`
	Status string `gorm:"type:varchar(20);default:'offline'" json:"status"`

	Hostname      string `gorm:"type:varchar(255)" json:"hostname"`
	IPAddress     string `gorm:"type:varchar(45)" json:"ipAddress"`
	AgentVersion  string `gorm:"column:agent_version;type:varchar(20)" json:"agentVersion"`
	WorkerVersion string `gorm:"column:worker_version;type:varchar(64)" json:"workerVersion"`

	MaxTasks      int `gorm:"default:5" json:"maxTasks"`
	CPUThreshold  int `gorm:"default:85" json:"cpuThreshold"`
	MemThreshold  int `gorm:"default:85" json:"memThreshold"`
	DiskThreshold int `gorm:"default:90" json:"diskThreshold"`

	HealthState   string     `gorm:"type:varchar(20);default:'ok'" json:"healthState"`
	HealthReason  string     `gorm:"type:varchar(64)" json:"healthReason,omitempty"`
	HealthMessage string     `gorm:"type:varchar(256)" json:"healthMessage,omitempty"`
	HealthSince   *time.Time `gorm:"type:timestamptz" json:"healthSince,omitempty"`

	RegistrationToken string `gorm:"type:varchar(8)" json:"registrationToken,omitempty"`

	ConnectedAt   *time.Time `gorm:"type:timestamptz" json:"connectedAt,omitempty"`
	LastHeartbeat *time.Time `gorm:"type:timestamptz" json:"lastHeartbeat,omitempty"`
	CreatedAt     time.Time  `gorm:"type:timestamptz;default:now()" json:"createdAt"`
	UpdatedAt     time.Time  `gorm:"type:timestamptz;default:now()" json:"updatedAt"`
}

func (Agent) TableName() string {
	return "agent"
}

// RegistrationToken represents a token for agent self-registration.
type RegistrationToken struct {
	ID        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Token     string    `gorm:"type:varchar(8);not null;uniqueIndex" json:"token"`
	ExpiresAt time.Time `gorm:"type:timestamptz;not null;default:now() + interval '1 hour'" json:"expiresAt"`
	CreatedAt time.Time `gorm:"type:timestamptz;default:now()" json:"createdAt"`
}

func (RegistrationToken) TableName() string {
	return "registration_token"
}
