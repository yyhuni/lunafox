package domain

import "errors"

var (
	ErrRegistrationTokenInvalid = errors.New("invalid or expired registration token")
	ErrAgentNotFound            = errors.New("agent not found")
)
