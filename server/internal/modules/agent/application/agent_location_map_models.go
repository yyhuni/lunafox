package application

import "time"

type LocationFreshness string

const (
	LocationFreshnessCurrent LocationFreshness = "current"
	LocationFreshnessExpired LocationFreshness = "expired"
)

type AgentLocationMapLocation struct {
	State            LocationFreshness
	Latitude         float64
	Longitude        float64
	AccuracyRadiusKM *float64
	SourceObservedIP string
	ProviderKey      string
	ResolvedAt       time.Time
}

type AgentLocationMapAgent struct {
	AgentID       int
	DisplayName   string
	Status        string
	HealthState   string
	TaskSlotsUsed *int
	Location      AgentLocationMapLocation
}

type AgentLocationMap struct {
	GeneratedAt    time.Time
	ServerLocation *ServerLocationRead
	Agents         []AgentLocationMapAgent
}
