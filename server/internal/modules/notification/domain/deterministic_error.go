package domain

import "fmt"

// DeterministicError marks malformed persisted notification state that cannot
// be healed by retrying the same immutable envelope.
type DeterministicError struct {
	Code string
	Err  error
}

func (err *DeterministicError) Error() string {
	if err == nil || err.Err == nil {
		return "deterministic notification materialization error"
	}
	return err.Err.Error()
}

func (err *DeterministicError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Err
}

// NewDeterministicError records a stable terminal code without serializing a
// potentially unsafe low-level error into notification persistence or logs.
func NewDeterministicError(code string, err error) error {
	if code == "" {
		code = "deterministic_materialization_error"
	}
	return &DeterministicError{Code: code, Err: fmt.Errorf("%s: %w", code, err)}
}
