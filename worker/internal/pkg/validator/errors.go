package validator

import "errors"

var (
	// ErrEmptyDomain is returned when domain is empty
	ErrEmptyDomain = errors.New("domain cannot be empty")

	// ErrInvalidDomain is returned when domain format is invalid
	ErrInvalidDomain = errors.New("invalid domain format")
)
