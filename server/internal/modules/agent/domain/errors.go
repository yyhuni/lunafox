package domain

import "errors"

var (
	ErrRegistrationTokenInvalid  = errors.New("invalid or expired registration token")
	ErrRegistrationTokenNotFound = errors.New("registration token not found")
	ErrAgentNotFound             = errors.New("agent not found")
	ErrStaleAgentHeartbeat       = errors.New("stale agent heartbeat")
)
