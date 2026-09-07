package domain

import "errors"

// ErrInvalidInputSource marks a missing or unrecognized execution-input source.
// Callers must reject it rather than selecting a compatibility default.
var ErrInvalidInputSource = errors.New("scan input source is required and must be scanSnapshot or targetInventory")

// InputSource determines where a Scan reads each execution input.
// It is independent from the workflow, Agent assignment, and trigger type.
type InputSource string

const (
	InputSourceScanSnapshot    InputSource = "scanSnapshot"
	InputSourceTargetInventory InputSource = "targetInventory"
)

// Valid reports whether the source belongs to the closed Scan vocabulary.
func (source InputSource) Valid() bool {
	switch source {
	case InputSourceScanSnapshot, InputSourceTargetInventory:
		return true
	default:
		return false
	}
}

// ParseInputSource validates an HTTP or domain source value.
func ParseInputSource(value string) (InputSource, bool) {
	source := InputSource(value)
	return source, source.Valid()
}

// DatabaseValue maps a valid source to its persisted representation.
func (source InputSource) DatabaseValue() (string, bool) {
	switch source {
	case InputSourceScanSnapshot:
		return "scan_snapshot", true
	case InputSourceTargetInventory:
		return "target_inventory", true
	default:
		return "", false
	}
}

// ParseDatabaseInputSource validates a persisted source value.
func ParseDatabaseInputSource(value string) (InputSource, bool) {
	switch value {
	case "scan_snapshot":
		return InputSourceScanSnapshot, true
	case "target_inventory":
		return InputSourceTargetInventory, true
	default:
		return "", false
	}
}
