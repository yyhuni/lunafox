package application

import (
	"context"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

// AgentLicense is the safe current entitlement view shared by activation and summary.
type AgentLicense = agentdomain.AgentLicense

// AgentLicenseReader reads entitlements at the caller's single observation instant.
type AgentLicenseReader interface {
	CurrentAt(ctx context.Context, now time.Time) (AgentLicense, error)
}

// AgentLicenseStore atomically replaces a verified Key without changing signed dates.
type AgentLicenseStore interface {
	AgentLicenseReader
	ActivateAt(ctx context.Context, key string, now time.Time) (AgentLicense, error)
}
