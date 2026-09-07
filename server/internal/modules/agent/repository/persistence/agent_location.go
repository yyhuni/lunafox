package model

import "time"

// AgentLocation is the complete last-successful GeoIP snapshot for one Agent.
type AgentLocation struct {
	AgentID          int       `gorm:"column:agent_id;primaryKey;autoIncrement:false" json:"agentId"`
	Latitude         float64   `gorm:"column:latitude;not null" json:"latitude"`
	Longitude        float64   `gorm:"column:longitude;not null" json:"longitude"`
	AccuracyRadiusKM *float64  `gorm:"column:accuracy_radius_km" json:"accuracyRadiusKm,omitempty"`
	SourceObservedIP string    `gorm:"column:source_observed_ip;type:varchar(45);not null" json:"sourceObservedIp"`
	ProviderKey      string    `gorm:"column:provider_key;type:varchar(32);not null" json:"providerKey"`
	ResolvedAt       time.Time `gorm:"column:resolved_at;type:timestamptz;not null" json:"resolvedAt"`
	ForcedExpired    bool      `gorm:"column:forced_expired;not null;default:false" json:"forcedExpired"`
	UpdatedAt        time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updatedAt"`
}

func (AgentLocation) TableName() string {
	return "agent_location"
}
