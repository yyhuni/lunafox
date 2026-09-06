package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	nucleipocdomain "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
)

// ScanSucceededPayload contains canonical scan terminal facts only.
type ScanSucceededPayload struct {
	ScanID     int    `json:"scanId"`
	TargetID   int    `json:"targetId"`
	TargetName string `json:"targetName"`
}

// ScanFailedPayload preserves the canonical failure classification and message.
type ScanFailedPayload struct {
	ScanID         int    `json:"scanId"`
	TargetID       int    `json:"targetId"`
	TargetName     string `json:"targetName"`
	FailureKind    string `json:"failureKind"`
	FailureMessage string `json:"failureMessage"`
}

// VulnerabilityObservedPayload identifies one observation only within its Scan.
type VulnerabilityObservedPayload struct {
	ScanID     int    `json:"scanId"`
	TargetID   int    `json:"targetId"`
	TargetName string `json:"targetName"`
	URL        string `json:"url"`
	VulnType   string `json:"vulnType"`
	Severity   string `json:"severity"`
}

// AgentOfflinePayload captures the authoritative monitor transition context.
type AgentOfflinePayload struct {
	AgentID       int       `json:"agentId"`
	DisplayName   string    `json:"displayName"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
}

// NucleiPOCSyncSucceededPayload preserves only the terminal catalog facts
// needed to render a durable, redacted inbox snapshot.
type NucleiPOCSyncSucceededPayload struct {
	TaskName          string `json:"taskName"`
	SourceType        string `json:"sourceType"`
	CommitSHA         string `json:"commitSha"`
	CommittedPOCCount int64  `json:"committedPocCount"`
}

// NucleiPOCSyncFailedPayload preserves only the terminal failure facts that
// have already been sanitized by the Nuclei persistence boundary.
type NucleiPOCSyncFailedPayload struct {
	TaskName       string `json:"taskName"`
	SourceType     string `json:"sourceType"`
	FailureCode    string `json:"failureCode"`
	FailureSummary string `json:"failureSummary"`
}

// DecodeOccurrence performs exact kind/version decoding. Unknown fields are
// rejected intentionally so future envelope changes never acquire accidental
// backward compatibility before a decoder and policy are explicitly added.
func DecodeOccurrence(occurrence Occurrence) (ValidatedEvent, error) {
	category, err := CategoryForKind(occurrence.Kind)
	if err != nil {
		return ValidatedEvent{}, err
	}
	if occurrence.PayloadVersion != PayloadVersionOne {
		return ValidatedEvent{}, fmt.Errorf("unsupported notification payload version %d for kind %q", occurrence.PayloadVersion, occurrence.Kind)
	}

	var payload any
	switch occurrence.Kind {
	case KindScanSucceeded:
		var value ScanSucceededPayload
		err = decodeStrictPayload(occurrence.Payload, &value)
		if err == nil {
			err = validateScanSucceededPayload(value)
		}
		payload = value
	case KindScanFailed:
		var value ScanFailedPayload
		err = decodeStrictPayload(occurrence.Payload, &value)
		if err == nil {
			err = validateScanFailedPayload(value)
		}
		payload = value
	case KindVulnerabilityObserved:
		var value VulnerabilityObservedPayload
		err = decodeStrictPayload(occurrence.Payload, &value)
		if err == nil {
			err = validateVulnerabilityObservedPayload(value)
		}
		payload = value
	case KindAgentOffline:
		var value AgentOfflinePayload
		err = decodeStrictPayload(occurrence.Payload, &value)
		if err == nil {
			err = validateAgentOfflinePayload(value)
		}
		payload = value
	case KindNucleiPOCSyncSucceeded:
		var value NucleiPOCSyncSucceededPayload
		err = decodeStrictPayload(occurrence.Payload, &value)
		if err == nil {
			err = validateNucleiPOCSyncSucceededPayload(occurrence, value)
		}
		payload = value
	case KindNucleiPOCSyncFailed:
		var value NucleiPOCSyncFailedPayload
		err = decodeStrictPayload(occurrence.Payload, &value)
		if err == nil {
			err = validateNucleiPOCSyncFailedPayload(occurrence, value)
		}
		payload = value
	default:
		return ValidatedEvent{}, fmt.Errorf("unsupported notification kind %q", occurrence.Kind)
	}
	if err != nil {
		return ValidatedEvent{}, err
	}
	if err := validateOccurrencePriority(occurrence, payload); err != nil {
		return ValidatedEvent{}, err
	}
	return ValidatedEvent{Occurrence: occurrence, Category: category, Payload: payload}, nil
}

func decodeStrictPayload(raw json.RawMessage, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("invalid notification payload: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("invalid notification payload: multiple JSON values")
		}
		return fmt.Errorf("invalid notification payload: %w", err)
	}
	return nil
}

func validateScanSucceededPayload(payload ScanSucceededPayload) error {
	if payload.ScanID <= 0 || payload.TargetID <= 0 || strings.TrimSpace(payload.TargetName) == "" {
		return fmt.Errorf("scan-succeeded payload has missing required fields")
	}
	return nil
}

func validateScanFailedPayload(payload ScanFailedPayload) error {
	if payload.ScanID <= 0 || payload.TargetID <= 0 || strings.TrimSpace(payload.TargetName) == "" || strings.TrimSpace(payload.FailureKind) == "" || strings.TrimSpace(payload.FailureMessage) == "" {
		return fmt.Errorf("scan-failed payload has missing required fields")
	}
	return nil
}

func validateVulnerabilityObservedPayload(payload VulnerabilityObservedPayload) error {
	if payload.ScanID <= 0 || payload.TargetID <= 0 || strings.TrimSpace(payload.TargetName) == "" || payload.URL == "" || strings.TrimSpace(payload.VulnType) == "" {
		return fmt.Errorf("vulnerability-observed payload has missing required fields")
	}
	severity := VulnerabilitySeverity(payload.Severity)
	if _, err := PriorityForVulnerabilitySeverity(severity); err != nil {
		return fmt.Errorf("vulnerability-observed payload has unsupported severity %q: %w", payload.Severity, err)
	}
	return nil
}

func validateAgentOfflinePayload(payload AgentOfflinePayload) error {
	if payload.AgentID <= 0 || strings.TrimSpace(payload.DisplayName) == "" || payload.LastHeartbeat.IsZero() {
		return fmt.Errorf("agent-offline payload has missing required fields")
	}
	return nil
}

func validateNucleiPOCSyncSucceededPayload(occurrence Occurrence, payload NucleiPOCSyncSucceededPayload) error {
	taskID, ok := canonicalNucleiPOCSyncTaskID(payload.TaskName)
	if !ok || !validNucleiSourceType(payload.SourceType) || !isNucleiCommitSHA(payload.CommitSHA) || payload.CommittedPOCCount <= 0 {
		return fmt.Errorf("nuclei-poc-sync-succeeded payload has missing or invalid required fields")
	}
	if occurrence.Subject != payload.TaskName || occurrence.EventID != nucleiPOCSyncEventID(taskID, "succeeded") {
		return fmt.Errorf("nuclei-poc-sync-succeeded occurrence identity is invalid")
	}
	return nil
}

func validateNucleiPOCSyncFailedPayload(occurrence Occurrence, payload NucleiPOCSyncFailedPayload) error {
	taskID, ok := canonicalNucleiPOCSyncTaskID(payload.TaskName)
	if !ok || !validNucleiSourceType(payload.SourceType) || !validNucleiFailureCode(payload.FailureCode) || !safeNucleiFailureSummary(payload.FailureCode, payload.FailureSummary) {
		return fmt.Errorf("nuclei-poc-sync-failed payload has missing or invalid required fields")
	}
	if occurrence.Subject != payload.TaskName || occurrence.EventID != nucleiPOCSyncEventID(taskID, "failed") {
		return fmt.Errorf("nuclei-poc-sync-failed occurrence identity is invalid")
	}
	return nil
}

func validateOccurrencePriority(occurrence Occurrence, payload any) error {
	var expected Priority
	var err error
	switch value := payload.(type) {
	case VulnerabilityObservedPayload:
		expected, err = PriorityForVulnerabilitySeverity(VulnerabilitySeverity(value.Severity))
	default:
		expected, err = PriorityForKind(occurrence.Kind)
	}
	if err != nil {
		return err
	}
	if occurrence.Priority != expected {
		return fmt.Errorf("notification kind %q requires priority %q", occurrence.Kind, expected)
	}
	return nil
}

// NewScanSucceededOccurrence creates the one stable envelope for a Scan's
// persisted success transition.
func NewScanSucceededOccurrence(scanID, targetID int, targetName string, occurredAt time.Time) (Occurrence, error) {
	payload := ScanSucceededPayload{ScanID: scanID, TargetID: targetID, TargetName: strings.TrimSpace(targetName)}
	return newOccurrence("scan:"+strconv.Itoa(scanID)+":succeeded", KindScanSucceeded, "scans/"+strconv.Itoa(scanID), occurredAt, PriorityNormal, payload)
}

// NewScanFailedOccurrence creates the one stable envelope for a Scan's
// persisted failure transition.
func NewScanFailedOccurrence(scanID, targetID int, targetName, failureKind, failureMessage string, occurredAt time.Time) (Occurrence, error) {
	payload := ScanFailedPayload{ScanID: scanID, TargetID: targetID, TargetName: strings.TrimSpace(targetName), FailureKind: strings.TrimSpace(failureKind), FailureMessage: strings.TrimSpace(failureMessage)}
	return newOccurrence("scan:"+strconv.Itoa(scanID)+":failed", KindScanFailed, "scans/"+strconv.Itoa(scanID), occurredAt, PriorityHigh, payload)
}

// NewVulnerabilityObservedOccurrence creates a Scan-scoped stable event ID.
// The canonical observation fingerprint deliberately includes the Scan so the
// same finding in a later Scan is a new observation rather than a global reuse.
func NewVulnerabilityObservedOccurrence(payload VulnerabilityObservedPayload, occurredAt time.Time) (Occurrence, error) {
	severity := VulnerabilitySeverity(strings.TrimSpace(payload.Severity))
	priority, err := PriorityForVulnerabilitySeverity(severity)
	if err != nil {
		return Occurrence{}, err
	}
	payload.Severity = string(severity)
	canonical := strings.Join([]string{
		strconv.Itoa(payload.ScanID),
		payload.URL,
		strings.TrimSpace(payload.VulnType),
		strings.TrimSpace(payload.Severity),
	}, "\x00")
	digest := sha256.Sum256([]byte(canonical))
	eventID := "vulnerability:" + strconv.Itoa(payload.ScanID) + ":" + hex.EncodeToString(digest[:])
	subject := "scans/" + strconv.Itoa(payload.ScanID) + "/vulnerabilityObservations/" + hex.EncodeToString(digest[:16])
	return newOccurrence(eventID, KindVulnerabilityObserved, subject, occurredAt, priority, payload)
}

// NewAgentOfflineOccurrence creates an idempotent envelope for one persisted
// online-to-offline transition timestamp.
func NewAgentOfflineOccurrence(agentID int, displayName string, lastHeartbeat, occurredAt time.Time) (Occurrence, error) {
	payload := AgentOfflinePayload{AgentID: agentID, DisplayName: strings.TrimSpace(displayName), LastHeartbeat: lastHeartbeat.UTC()}
	eventID := "agent:" + strconv.Itoa(agentID) + ":offline:" + strconv.FormatInt(occurredAt.UTC().UnixNano(), 10)
	return newOccurrence(eventID, KindAgentOffline, "agents/"+strconv.Itoa(agentID), occurredAt, PriorityHigh, payload)
}

// NewNucleiPOCSyncSucceededOccurrence creates the stable success envelope for
// one terminal Nuclei sync task.
func NewNucleiPOCSyncSucceededOccurrence(taskID uuid.UUID, sourceType, commitSHA string, committedPOCCount int64, occurredAt time.Time) (Occurrence, error) {
	taskName := nucleiPOCSyncTaskName(taskID)
	payload := NucleiPOCSyncSucceededPayload{
		TaskName:          taskName,
		SourceType:        sourceType,
		CommitSHA:         commitSHA,
		CommittedPOCCount: committedPOCCount,
	}
	return newOccurrence(nucleiPOCSyncEventID(taskID, "succeeded"), KindNucleiPOCSyncSucceeded, taskName, occurredAt, PriorityNormal, payload)
}

// NewNucleiPOCSyncFailedOccurrence creates the stable failure envelope for
// one terminal Nuclei sync task.
func NewNucleiPOCSyncFailedOccurrence(taskID uuid.UUID, sourceType, failureCode, failureSummary string, occurredAt time.Time) (Occurrence, error) {
	taskName := nucleiPOCSyncTaskName(taskID)
	payload := NucleiPOCSyncFailedPayload{
		TaskName:       taskName,
		SourceType:     sourceType,
		FailureCode:    failureCode,
		FailureSummary: failureSummary,
	}
	return newOccurrence(nucleiPOCSyncEventID(taskID, "failed"), KindNucleiPOCSyncFailed, taskName, occurredAt, PriorityHigh, payload)
}

func nucleiPOCSyncTaskName(taskID uuid.UUID) string {
	return "nucleiPocSyncTasks/" + taskID.String()
}

func nucleiPOCSyncEventID(taskID uuid.UUID, terminal string) string {
	return "nuclei-poc-sync:" + taskID.String() + ":" + terminal
}

func canonicalNucleiPOCSyncTaskID(taskName string) (uuid.UUID, bool) {
	const prefix = "nucleiPocSyncTasks/"
	if !strings.HasPrefix(taskName, prefix) || strings.Count(taskName, "/") != 1 {
		return uuid.Nil, false
	}
	taskID, err := uuid.Parse(strings.TrimPrefix(taskName, prefix))
	if err != nil || taskID == uuid.Nil || taskName != nucleiPOCSyncTaskName(taskID) {
		return uuid.Nil, false
	}
	return taskID, true
}

func validNucleiSourceType(sourceType string) bool {
	switch sourceType {
	case "git", "gitee", "custom":
		return true
	default:
		return false
	}
}

func isNucleiCommitSHA(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') && (character < 'A' || character > 'F') {
			return false
		}
	}
	return true
}

func validNucleiFailureCode(value string) bool {
	canonicalCode, _ := nucleipocdomain.CanonicalTerminalFailure(value)
	return value == canonicalCode
}

func safeNucleiFailureSummary(code, value string) bool {
	return nucleipocdomain.IsCanonicalTerminalFailure(code, value)
}

func newOccurrence(eventID string, kind Kind, subject string, occurredAt time.Time, priority Priority, payload any) (Occurrence, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return Occurrence{}, err
	}
	occurrence := Occurrence{
		EventID:        eventID,
		Kind:           kind,
		PayloadVersion: PayloadVersionOne,
		Subject:        subject,
		OccurredAt:     occurredAt.UTC(),
		Priority:       priority,
		Payload:        encoded,
	}
	if err := ValidateOccurrence(occurrence); err != nil {
		return Occurrence{}, err
	}
	return occurrence, nil
}
