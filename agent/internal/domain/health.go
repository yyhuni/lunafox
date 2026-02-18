package domain

import "time"

type HealthStatus struct {
	State   string     `json:"state"`
	Reason  string     `json:"reason,omitempty"`
	Message string     `json:"message,omitempty"`
	Since   *time.Time `json:"since,omitempty"`
}
