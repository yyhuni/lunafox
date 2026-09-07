package persistence

import "time"

// ServerLocationSnapshot stores the one supported Server's last successful location.
type ServerLocationSnapshot struct {
	SingletonID      int16     `gorm:"column:singleton_id;primaryKey;autoIncrement:false"`
	ObservedEgressIP string    `gorm:"column:observed_egress_ip;type:varchar(45);not null"`
	Latitude         float64   `gorm:"column:latitude;not null"`
	Longitude        float64   `gorm:"column:longitude;not null"`
	AccuracyRadiusKM *float64  `gorm:"column:accuracy_radius_km"`
	ProviderKey      string    `gorm:"column:provider_key;type:varchar(32);not null"`
	ResolvedAt       time.Time `gorm:"column:resolved_at;type:timestamptz;not null"`
	ForcedExpired    bool      `gorm:"column:forced_expired;not null;default:false"`
	UpdatedAt        time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()"`
}

func (ServerLocationSnapshot) TableName() string {
	return "server_location_snapshot"
}
