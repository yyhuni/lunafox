package domain

import (
	"errors"
	"time"
)

// AgentLicense is the current entitlement projection, never the signed Key itself.
type AgentLicense struct {
	State      string
	AgentLimit int
	Months     *int
	ExpiresAt  *time.Time
	LicenseID  *string
}

var (
	// ErrAgentLicenseKeyRequired means activation was submitted without a Key.
	ErrAgentLicenseKeyRequired = errors.New("agent license key is required")
	// ErrAgentLicenseKeyInvalid means a Key failed its format or signature check.
	ErrAgentLicenseKeyInvalid = errors.New("agent license key is invalid")
	// ErrAgentLicenseKeyExpired means the signed fixed expiry has been reached.
	ErrAgentLicenseKeyExpired = errors.New("agent license key has expired")
	// ErrAgentLicenseKeyNotYetValid means the signed issuance time is in the future.
	ErrAgentLicenseKeyNotYetValid = errors.New("agent license key is not yet valid")
)
