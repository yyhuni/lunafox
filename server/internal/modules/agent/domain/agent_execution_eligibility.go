package domain

const (
	// AgentExecutionEligible permits this identity to claim new tasks under the current entitlement.
	AgentExecutionEligible = "eligible"
	// AgentExecutionLicenseRestricted keeps existing work connected while withholding new claims.
	AgentExecutionLicenseRestricted = "license_restricted"
)
