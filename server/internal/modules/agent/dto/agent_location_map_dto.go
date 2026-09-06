package dto

import "time"

type AgentLocationMapLocationResponse struct {
	State            string    `json:"state"`
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	AccuracyRadiusKM *float64  `json:"accuracyRadiusKm"`
	SourceObservedIP string    `json:"sourceObservedIp"`
	ProviderKey      string    `json:"providerKey"`
	ResolvedAt       time.Time `json:"resolvedAt"`
}

type AgentLocationMapAgentResponse struct {
	Name          string                           `json:"name"`
	DisplayName   string                           `json:"displayName"`
	Status        string                           `json:"status"`
	HealthState   string                           `json:"healthState"`
	TaskSlotsUsed *int                             `json:"taskSlotsUsed"`
	Location      AgentLocationMapLocationResponse `json:"location"`
}

type ServerLocationMapResponse struct {
	State            string    `json:"state"`
	ObservedEgressIP string    `json:"observedEgressIp"`
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	AccuracyRadiusKM *float64  `json:"accuracyRadiusKm"`
	ProviderKey      string    `json:"providerKey"`
	ResolvedAt       time.Time `json:"resolvedAt"`
}

type AgentLocationMapResponse struct {
	Name           string                          `json:"name"`
	GeneratedAt    time.Time                       `json:"generatedAt"`
	ServerLocation *ServerLocationMapResponse      `json:"serverLocation"`
	Agents         []AgentLocationMapAgentResponse `json:"agents"`
}
