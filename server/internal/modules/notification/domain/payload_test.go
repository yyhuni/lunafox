package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDecodeOccurrenceRejectsUnknownPayloadFields(t *testing.T) {
	payload := []byte(`{"scanId":1,"targetId":2,"targetName":"example.test","unexpected":true}`)
	_, err := DecodeOccurrence(Occurrence{
		EventID:        "scan:1:succeeded",
		Kind:           KindScanSucceeded,
		PayloadVersion: PayloadVersionOne,
		Subject:        "scans/1",
		OccurredAt:     time.Now().UTC(),
		Priority:       PriorityNormal,
		Payload:        payload,
	})
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("DecodeOccurrence error = %v, want unknown field rejection", err)
	}
}

func TestValidatePriorityRejectsLow(t *testing.T) {
	if err := ValidatePriority(Priority("low")); err == nil {
		t.Fatal("ValidatePriority accepted low")
	}
}

func TestVulnerabilityOccurrenceIsScanScopedAndDeterministic(t *testing.T) {
	payload := VulnerabilityObservedPayload{
		ScanID: 1, TargetID: 2, TargetName: "example.test", URL: "https://example.test", VulnType: "xss", Severity: "high",
	}
	first, err := NewVulnerabilityObservedOccurrence(payload, time.Unix(1, 0).UTC())
	if err != nil {
		t.Fatalf("NewVulnerabilityObservedOccurrence first: %v", err)
	}
	second, err := NewVulnerabilityObservedOccurrence(payload, time.Unix(2, 0).UTC())
	if err != nil {
		t.Fatalf("NewVulnerabilityObservedOccurrence second: %v", err)
	}
	if first.EventID != second.EventID {
		t.Fatalf("event ids differ: %q != %q", first.EventID, second.EventID)
	}
	payload.ScanID = 3
	third, err := NewVulnerabilityObservedOccurrence(payload, time.Unix(2, 0).UTC())
	if err != nil {
		t.Fatalf("NewVulnerabilityObservedOccurrence third: %v", err)
	}
	if first.EventID == third.EventID {
		t.Fatal("cross-scan observation reused event identity")
	}
	if !json.Valid(first.Payload) {
		t.Fatalf("payload is not JSON: %s", first.Payload)
	}
}

func TestVulnerabilityOccurrenceKeepsRawURLBytesInEventIdentity(t *testing.T) {
	base := VulnerabilityObservedPayload{
		ScanID: 1, TargetID: 2, TargetName: "example.test", URL: "https://example.test/path", VulnType: "xss", Severity: "high",
	}
	first, err := NewVulnerabilityObservedOccurrence(base, time.Unix(1, 0).UTC())
	if err != nil {
		t.Fatalf("NewVulnerabilityObservedOccurrence first: %v", err)
	}
	base.URL += " "
	second, err := NewVulnerabilityObservedOccurrence(base, time.Unix(1, 0).UTC())
	if err != nil {
		t.Fatalf("NewVulnerabilityObservedOccurrence second: %v", err)
	}
	if first.EventID == second.EventID {
		t.Fatalf("raw URL variants collapsed to one event identity: %q", first.EventID)
	}
}

func TestNucleiPOCSyncOccurrencesUseStableTaskIdentityAndClosedMappings(t *testing.T) {
	taskID := uuid.MustParse("6dd1f0fd-5b34-4cbb-9c4f-3b9c6c0f6a12")
	commitSHA := "0123456789abcdef0123456789abcdef01234567"
	succeeded, err := NewNucleiPOCSyncSucceededOccurrence(taskID, "git", commitSHA, 184, time.Unix(1, 0).UTC())
	if err != nil {
		t.Fatalf("NewNucleiPOCSyncSucceededOccurrence: %v", err)
	}
	if succeeded.EventID != "nuclei-poc-sync:6dd1f0fd-5b34-4cbb-9c4f-3b9c6c0f6a12:succeeded" || succeeded.Subject != "nucleiPocSyncTasks/6dd1f0fd-5b34-4cbb-9c4f-3b9c6c0f6a12" || succeeded.Priority != PriorityNormal {
		t.Fatalf("success occurrence = %#v, want canonical task-scoped identity and normal priority", succeeded)
	}
	validatedSuccess, err := DecodeOccurrence(succeeded)
	if err != nil || validatedSuccess.Category != CategorySystem {
		t.Fatalf("DecodeOccurrence success = %#v, %v; want system category", validatedSuccess, err)
	}
	payload := validatedSuccess.Payload.(NucleiPOCSyncSucceededPayload)
	if payload.CommitSHA != commitSHA || payload.CommittedPOCCount != 184 {
		t.Fatalf("success payload = %#v, want full SHA and count", payload)
	}

	failed, err := NewNucleiPOCSyncFailedOccurrence(taskID, "custom", "TEMPLATE_INVALID", "One or more Nuclei templates failed validation.", time.Unix(2, 0).UTC())
	if err != nil {
		t.Fatalf("NewNucleiPOCSyncFailedOccurrence: %v", err)
	}
	if failed.EventID != "nuclei-poc-sync:6dd1f0fd-5b34-4cbb-9c4f-3b9c6c0f6a12:failed" || failed.Priority != PriorityHigh {
		t.Fatalf("failure occurrence = %#v, want canonical task-scoped identity and high priority", failed)
	}
	validatedFailure, err := DecodeOccurrence(failed)
	if err != nil || validatedFailure.Category != CategorySystem {
		t.Fatalf("DecodeOccurrence failure = %#v, %v; want system category", validatedFailure, err)
	}
}

func TestNucleiPOCSyncPayloadRejectsUnsafeAndUnknownFields(t *testing.T) {
	taskID := uuid.MustParse("6dd1f0fd-5b34-4cbb-9c4f-3b9c6c0f6a12")
	occurrence, err := NewNucleiPOCSyncFailedOccurrence(taskID, "git", "TEMPLATE_INVALID", "One or more Nuclei templates failed validation.", time.Now().UTC())
	if err != nil {
		t.Fatalf("NewNucleiPOCSyncFailedOccurrence: %v", err)
	}
	occurrence.Payload = []byte(`{"taskName":"nucleiPocSyncTasks/6dd1f0fd-5b34-4cbb-9c4f-3b9c6c0f6a12","sourceType":"git","failureCode":"TEMPLATE_INVALID","failureSummary":"https://unsafe.example/repository","repoUrl":"https://unsafe.example"}`)
	if _, err := DecodeOccurrence(occurrence); err == nil || (!strings.Contains(err.Error(), "unknown field") && !strings.Contains(err.Error(), "missing or invalid")) {
		t.Fatalf("DecodeOccurrence unsafe failure payload error = %v, want strict rejection", err)
	}

	// A harmless-looking arbitrary sentence is still rejected. Only the
	// Nuclei-owned code/summary pairs may become durable display snapshots.
	occurrence.Payload = []byte(`{"taskName":"nucleiPocSyncTasks/6dd1f0fd-5b34-4cbb-9c4f-3b9c6c0f6a12","sourceType":"git","failureCode":"TEMPLATE_INVALID","failureSummary":"The template import did not complete."}`)
	if _, err := DecodeOccurrence(occurrence); err == nil || !strings.Contains(err.Error(), "missing or invalid") {
		t.Fatalf("DecodeOccurrence arbitrary failure summary error = %v, want closed summary rejection", err)
	}

	occurrence.Payload = []byte(`{"taskName":"nucleiPocSyncTasks/6dd1f0fd-5b34-4cbb-9c4f-3b9c6c0f6a12","sourceType":"git","failureCode":"UNREVIEWED_INTERNAL_FAILURE","failureSummary":"The Nuclei POC sync could not be completed."}`)
	if _, err := DecodeOccurrence(occurrence); err == nil || !strings.Contains(err.Error(), "missing or invalid") {
		t.Fatalf("DecodeOccurrence arbitrary failure code error = %v, want closed code rejection", err)
	}

	occurrence.Payload = []byte(`{"taskName":"nucleiPocSyncTasks/6dd1f0fd-5b34-4cbb-9c4f-3b9c6c0f6a12","sourceType":"git","failureCode":"TEMPLATE_INVALID","failureSummary":"One or more Nuclei templates failed validation."}`)
	occurrence.Priority = PriorityNormal
	if _, err := DecodeOccurrence(occurrence); err == nil || !strings.Contains(err.Error(), "requires priority") {
		t.Fatalf("DecodeOccurrence wrong priority error = %v, want fixed priority rejection", err)
	}
}

func TestNucleiKindsAreInboxOnly(t *testing.T) {
	for _, kind := range []Kind{KindNucleiPOCSyncSucceeded, KindNucleiPOCSyncFailed} {
		if !IsInboxSupportedKind(kind) || IsExternallyDeliverableKind(kind) {
			t.Fatalf("kind %q inbox/destination membership is wrong", kind)
		}
	}
	if len(InboxSupportedKinds()) != len(ExternallyDeliverableKinds())+2 {
		t.Fatalf("kind registries = inbox %#v external %#v, want exactly two inbox-only kinds", InboxSupportedKinds(), ExternallyDeliverableKinds())
	}
}
