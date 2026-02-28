package server

import "fmt"

// HTTPError represents an HTTP response failure used by legacy-compatible callers.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

// IsRetryable returns true when the failure is likely transient.
func (e *HTTPError) IsRetryable() bool {
	return e.StatusCode >= 500
}
