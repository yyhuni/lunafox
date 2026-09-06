// Package application coordinates blacklist policy resource behavior without
// exposing repository models to HTTP or other module boundaries.
package application

import "time"

// BlacklistPolicy is the application projection of one global or Target-local
// singleton resource.
type BlacklistPolicy struct {
	Name       string
	Patterns   []string
	ETag       string
	UpdateTime time.Time
}

// ReplaceBlacklistPolicyInput contains the complete local policy replacement.
// HTTP presence validation happens at the transport boundary before this input
// is constructed.
type ReplaceBlacklistPolicyInput struct {
	ETag     string
	Patterns []string
}

// BlacklistPolicyScope identifies the owner of a repository-neutral policy
// record passed through the application store port.
type BlacklistPolicyScope string

const (
	BlacklistPolicyScopeGlobal BlacklistPolicyScope = "global"
	BlacklistPolicyScopeTarget BlacklistPolicyScope = "target"
)

// BlacklistPolicyRecord is the persistence-neutral local policy value returned
// by the application store. Database IDs deliberately do not cross this port.
type BlacklistPolicyRecord struct {
	Scope     BlacklistPolicyScope
	TargetID  *int
	Patterns  []string
	UpdatedAt time.Time
}
