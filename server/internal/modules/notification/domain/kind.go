// Package domain defines the canonical notification vocabulary and validation rules.
package domain

import "fmt"

// Kind is the canonical, versioned notification event classification.
type Kind string

const (
	KindScanSucceeded          Kind = "scan-succeeded"
	KindScanFailed             Kind = "scan-failed"
	KindVulnerabilityObserved  Kind = "vulnerability-observed"
	KindAgentOffline           Kind = "agent-offline"
	KindNucleiPOCSyncSucceeded Kind = "nuclei-poc-sync-succeeded"
	KindNucleiPOCSyncFailed    Kind = "nuclei-poc-sync-failed"
)

// Category is fixed inbox taxonomy for a kind. It is neither a personal
// subscription control nor a substitute for a destination's exact-kind
// subscription allowlist.
type Category string

const (
	CategoryScan          Category = "scan"
	CategoryVulnerability Category = "vulnerability"
	CategorySystem        Category = "system"
)

// Priority describes attention priority independently from domain severity.
type Priority string

const (
	PriorityNormal   Priority = "normal"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// Locale is a persisted notification rendering locale.
type Locale string

const (
	LocaleChinese Locale = "zh"
	LocaleEnglish Locale = "en"
)

// VulnerabilitySeverity is the canonical severity vocabulary accepted by the
// notification threshold. It deliberately stays separate from Priority.
type VulnerabilitySeverity string

const (
	VulnerabilitySeverityUnknown  VulnerabilitySeverity = "unknown"
	VulnerabilitySeverityInfo     VulnerabilitySeverity = "info"
	VulnerabilitySeverityLow      VulnerabilitySeverity = "low"
	VulnerabilitySeverityMedium   VulnerabilitySeverity = "medium"
	VulnerabilitySeverityHigh     VulnerabilitySeverity = "high"
	VulnerabilitySeverityCritical VulnerabilitySeverity = "critical"
)

var inboxSupportedKinds = [...]Kind{
	KindScanSucceeded,
	KindScanFailed,
	KindVulnerabilityObserved,
	KindAgentOffline,
	KindNucleiPOCSyncSucceeded,
	KindNucleiPOCSyncFailed,
}

var externallyDeliverableKinds = [...]Kind{
	KindScanSucceeded,
	KindScanFailed,
	KindVulnerabilityObserved,
	KindAgentOffline,
}

// InboxSupportedKinds returns the full closed vocabulary accepted by the
// inbox decoder and materializer, including inbox-only kinds.
func InboxSupportedKinds() []Kind {
	return append([]Kind(nil), inboxSupportedKinds[:]...)
}

// ExternallyDeliverableKinds returns only kinds administrators can opt into
// for installation-owned destinations.
func ExternallyDeliverableKinds() []Kind {
	return append([]Kind(nil), externallyDeliverableKinds[:]...)
}

// SupportedKinds remains the inbox vocabulary for existing decoder callers.
// New destination code must use ExternallyDeliverableKinds explicitly.
func SupportedKinds() []Kind {
	return InboxSupportedKinds()
}

// IsInboxSupportedKind reports whether a kind has an exact inbox decoder.
func IsInboxSupportedKind(kind Kind) bool {
	switch kind {
	case KindScanSucceeded,
		KindScanFailed,
		KindVulnerabilityObserved,
		KindAgentOffline,
		KindNucleiPOCSyncSucceeded,
		KindNucleiPOCSyncFailed:
		return true
	default:
		return false
	}
}

// IsExternallyDeliverableKind reports whether a kind may appear in a
// destination allowlist or create provider delivery work.
func IsExternallyDeliverableKind(kind Kind) bool {
	switch kind {
	case KindScanSucceeded, KindScanFailed, KindVulnerabilityObserved, KindAgentOffline:
		return true
	default:
		return false
	}
}

// IsSupportedKind remains the inbox decoder predicate for existing callers.
func IsSupportedKind(kind Kind) bool {
	return IsInboxSupportedKind(kind)
}

// CategoryForKind derives the only valid inbox category for a kind.
func CategoryForKind(kind Kind) (Category, error) {
	switch kind {
	case KindScanSucceeded, KindScanFailed:
		return CategoryScan, nil
	case KindVulnerabilityObserved:
		return CategoryVulnerability, nil
	case KindAgentOffline:
		return CategorySystem, nil
	case KindNucleiPOCSyncSucceeded, KindNucleiPOCSyncFailed:
		return CategorySystem, nil
	default:
		return "", fmt.Errorf("unsupported notification kind %q", kind)
	}
}

// PriorityForKind derives the fixed attention priority for kinds whose
// priority is independent of payload data. Vulnerability priority remains
// severity-derived and is handled by PriorityForVulnerabilitySeverity.
func PriorityForKind(kind Kind) (Priority, error) {
	switch kind {
	case KindScanSucceeded, KindNucleiPOCSyncSucceeded:
		return PriorityNormal, nil
	case KindScanFailed, KindAgentOffline, KindNucleiPOCSyncFailed:
		return PriorityHigh, nil
	case KindVulnerabilityObserved:
		return "", fmt.Errorf("notification kind %q requires payload-derived priority", kind)
	default:
		return "", fmt.Errorf("unsupported notification kind %q", kind)
	}
}

// ValidatePriority keeps attention priority closed. In particular, low is not
// a compatibility default because it would silently weaken notification policy.
func ValidatePriority(priority Priority) error {
	switch priority {
	case PriorityNormal, PriorityHigh, PriorityCritical:
		return nil
	default:
		return fmt.Errorf("unsupported notification priority %q", priority)
	}
}

// ValidateLocale rejects values that could make asynchronous rendering depend
// on a request or browser-specific locale outside the persisted contract.
func ValidateLocale(locale Locale) error {
	switch locale {
	case LocaleChinese, LocaleEnglish:
		return nil
	default:
		return fmt.Errorf("unsupported notification locale %q", locale)
	}
}

// ValidateVulnerabilityThreshold rejects unknown and non-actionable values so
// a deployment cannot silently broaden or narrow notification fanout.
func ValidateVulnerabilityThreshold(threshold VulnerabilitySeverity) error {
	switch threshold {
	case VulnerabilitySeverityInfo,
		VulnerabilitySeverityLow,
		VulnerabilitySeverityMedium,
		VulnerabilitySeverityHigh,
		VulnerabilitySeverityCritical:
		return nil
	default:
		return fmt.Errorf("unsupported vulnerability notification threshold %q", threshold)
	}
}

// ValidateVulnerabilitySeverity accepts unknown as a persisted business fact,
// but unknown can never be eligible for a notification occurrence.
func ValidateVulnerabilitySeverity(severity VulnerabilitySeverity) error {
	switch severity {
	case VulnerabilitySeverityUnknown,
		VulnerabilitySeverityInfo,
		VulnerabilitySeverityLow,
		VulnerabilitySeverityMedium,
		VulnerabilitySeverityHigh,
		VulnerabilitySeverityCritical:
		return nil
	default:
		return fmt.Errorf("unsupported vulnerability severity %q", severity)
	}
}

// MeetsVulnerabilityThreshold reports notification eligibility without
// treating unknown observations as an implicit low-severity notification.
func MeetsVulnerabilityThreshold(severity, threshold VulnerabilitySeverity) (bool, error) {
	if err := ValidateVulnerabilitySeverity(severity); err != nil {
		return false, err
	}
	if err := ValidateVulnerabilityThreshold(threshold); err != nil {
		return false, err
	}
	if severity == VulnerabilitySeverityUnknown {
		return false, nil
	}
	return vulnerabilitySeverityRank(severity) >= vulnerabilitySeverityRank(threshold), nil
}

// PriorityForVulnerabilitySeverity maps eligible business severity into the
// closed attention-priority vocabulary without exposing a low priority value.
func PriorityForVulnerabilitySeverity(severity VulnerabilitySeverity) (Priority, error) {
	if err := ValidateVulnerabilitySeverity(severity); err != nil {
		return "", err
	}
	switch severity {
	case VulnerabilitySeverityCritical:
		return PriorityCritical, nil
	case VulnerabilitySeverityHigh:
		return PriorityHigh, nil
	case VulnerabilitySeverityInfo, VulnerabilitySeverityLow, VulnerabilitySeverityMedium:
		return PriorityNormal, nil
	default:
		return "", fmt.Errorf("unknown vulnerability severity cannot create a notification")
	}
}

func vulnerabilitySeverityRank(severity VulnerabilitySeverity) int {
	switch severity {
	case VulnerabilitySeverityInfo:
		return 1
	case VulnerabilitySeverityLow:
		return 2
	case VulnerabilitySeverityMedium:
		return 3
	case VulnerabilitySeverityHigh:
		return 4
	case VulnerabilitySeverityCritical:
		return 5
	default:
		return 0
	}
}
