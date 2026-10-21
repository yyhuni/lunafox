package dto

import "time"

// AgentLicenseActivationRequest imports an author-issued signed entitlement.
type AgentLicenseActivationRequest struct {
	Key string `json:"key"`
}

// AgentLicenseResponse is the fixed current authorization resource without its Key.
type AgentLicenseResponse struct {
	Name       string     `json:"name"`
	State      string     `json:"state"`
	AgentLimit int        `json:"agentLimit"`
	Months     *int       `json:"months"`
	ExpiresAt  *time.Time `json:"expiresAt"`
	LicenseID  *string    `json:"licenseId"`
}
