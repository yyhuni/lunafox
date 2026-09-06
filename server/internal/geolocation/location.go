package geolocation

import (
	"fmt"
	"time"
)

const FreeIPAPIProviderKey = "freeipapi"

// Location is the normalized allowlisted result of one successful lookup.
type Location struct {
	ObservedIP       string
	Latitude         float64
	Longitude        float64
	AccuracyRadiusKM *float64
	ProviderKey      string
	ResolvedAt       time.Time
}

// FailureClass is a bounded machine-readable terminal lookup category.
type FailureClass string

const (
	FailureInvalidTarget  FailureClass = "invalid_target"
	FailureCanceled       FailureClass = "canceled"
	FailureTimeout        FailureClass = "timeout"
	FailureNetwork        FailureClass = "network"
	FailureRedirect       FailureClass = "redirect"
	FailureHTTPStatus     FailureClass = "http_status"
	FailureRateLimited    FailureClass = "rate_limited"
	FailureBodyTooLarge   FailureClass = "body_too_large"
	FailureInvalidPayload FailureClass = "invalid_payload"
)

// LookupError reports only bounded failure data needed by scheduling policy.
type LookupError struct {
	Class        FailureClass
	RetryAfterAt time.Time
}

func (failure *LookupError) Error() string {
	if failure == nil {
		return "geolocation lookup failed"
	}
	return fmt.Sprintf("geolocation lookup failed: %s", failure.Class)
}
