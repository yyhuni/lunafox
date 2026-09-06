package domain

// ScanTriggerType records the trusted producer that created a Scan.
// It is persisted at creation and must not be inferred from mutable schedule state.
type ScanTriggerType string

const (
	ScanTriggerTypeManual    ScanTriggerType = "manual"
	ScanTriggerTypeScheduled ScanTriggerType = "scheduled"
	ScanTriggerTypeAI        ScanTriggerType = "ai"
)

// Valid reports whether the trigger type belongs to the closed Scan vocabulary.
func (triggerType ScanTriggerType) Valid() bool {
	switch triggerType {
	case ScanTriggerTypeManual, ScanTriggerTypeScheduled, ScanTriggerTypeAI:
		return true
	default:
		return false
	}
}

// ParseScanTriggerType validates a persisted or boundary trigger value.
func ParseScanTriggerType(value string) (ScanTriggerType, bool) {
	triggerType := ScanTriggerType(value)
	return triggerType, triggerType.Valid()
}
