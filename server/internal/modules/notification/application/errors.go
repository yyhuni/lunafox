package application

import "errors"

var (
	// ErrInvalidInboxPageToken identifies malformed opaque inbox pagination.
	ErrInvalidInboxPageToken = errors.New("invalid notification page token")
	// ErrInvalidInboxPageSize identifies out-of-contract page size values.
	ErrInvalidInboxPageSize = errors.New("invalid notification pageSize")
	// ErrNotificationPermissionDenied prevents credential-bearing settings reads.
	ErrNotificationPermissionDenied = errors.New("notification permission denied")
)
