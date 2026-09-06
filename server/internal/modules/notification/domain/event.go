package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const PayloadVersionOne = 1

// Occurrence is the immutable envelope written by a business producer and
// later consumed by the notification outbox worker.
type Occurrence struct {
	EventID        string
	Kind           Kind
	PayloadVersion int
	Subject        string
	OccurredAt     time.Time
	Priority       Priority
	Payload        json.RawMessage
}

// ValidatedEvent is an exact kind/version envelope after strict payload decode.
type ValidatedEvent struct {
	Occurrence
	Category Category
	Payload  any
}

// ValidateOccurrence fails before persistence instead of allowing workers to
// infer missing fields from rendered text or arbitrary payload members.
func ValidateOccurrence(occurrence Occurrence) error {
	if strings.TrimSpace(occurrence.EventID) == "" {
		return fmt.Errorf("notification event id is required")
	}
	if len(occurrence.EventID) > 255 {
		return fmt.Errorf("notification event id is too long")
	}
	if strings.TrimSpace(string(occurrence.Kind)) == "" {
		return fmt.Errorf("notification kind is required")
	}
	if occurrence.PayloadVersion <= 0 {
		return fmt.Errorf("notification payload version is required")
	}
	if strings.TrimSpace(occurrence.Subject) == "" {
		return fmt.Errorf("notification subject is required")
	}
	if occurrence.OccurredAt.IsZero() {
		return fmt.Errorf("notification occurrence time is required")
	}
	if err := ValidatePriority(occurrence.Priority); err != nil {
		return err
	}
	if len(occurrence.Payload) == 0 {
		return fmt.Errorf("notification payload is required")
	}
	_, err := DecodeOccurrence(occurrence)
	return err
}
