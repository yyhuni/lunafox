package domain

import "time"

type RegistrationTokenState string

const (
	RegistrationTokenStateActive  RegistrationTokenState = "active"
	RegistrationTokenStateExpired RegistrationTokenState = "expired"
)

// RegistrationToken represents registration capability for agent onboarding.
type RegistrationToken struct {
	ID               int
	Token            string
	ExpiresAt        time.Time
	EverAttributedAt *time.Time
	CreatedAt        time.Time
}

func NewRegistrationToken(token string, expiresAt time.Time) *RegistrationToken {
	return &RegistrationToken{Token: token, ExpiresAt: expiresAt}
}

// RegistrationTokenResource is the non-secret management projection for one
// registration capability and every Agent attributed to it.
type RegistrationTokenResource struct {
	ID        int
	ExpiresAt time.Time
	Agents    []*Agent
}

func (resource *RegistrationTokenResource) StateAt(now time.Time) RegistrationTokenState {
	if resource == nil || !now.UTC().Before(resource.ExpiresAt.UTC()) {
		return RegistrationTokenStateExpired
	}
	return RegistrationTokenStateActive
}
