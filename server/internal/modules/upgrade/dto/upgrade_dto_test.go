package dto

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

func TestUpgradeLogsAreOrderedAndDiagnosticIsSafe(t *testing.T) {
	base := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	operation := &domain.Operation{
		OperationID: "11111111-1111-4111-8111-111111111111",
		Status:      domain.StatusNeedsAttention,
		StageTimes: map[domain.Status]time.Time{
			domain.StatusQueued:   base,
			domain.StatusUpdating: base.Add(3 * time.Minute),
			domain.StatusStopping: base.Add(1 * time.Minute),
		},
		CancelledScanCount: 2,
		CancelledTaskCount: 1,
		CreatedAt:          base,
		UpdatedAt:          base.Add(4 * time.Minute),
		Diagnostic:         "executor failed at /srv/lunafox/.env with TOKEN=secret",
	}

	logs := upgradeLogs(operation)
	if len(logs) == 0 || len(logs) > 32 {
		t.Fatalf("log count = %d, want 1..32", len(logs))
	}
	for index := 1; index < len(logs); index++ {
		if logs[index].Timestamp.Before(logs[index-1].Timestamp) {
			t.Fatalf("logs are not ordered: %#v", logs)
		}
	}
	var sawCurrentStage, sawStopped, sawDiagnostic bool
	for _, entry := range logs {
		if entry.Stage == string(domain.StatusNeedsAttention) {
			sawCurrentStage = true
		}
		if entry.MessageKey == "workStopped" {
			sawStopped = true
		}
		if entry.MessageKey == "diagnostic" {
			sawDiagnostic = true
			if strings.Contains(entry.Message, "/srv") || strings.Contains(strings.ToLower(entry.Message), "token") {
				t.Fatalf("unsafe diagnostic leaked into log: %q", entry.Message)
			}
		}
	}
	if !sawCurrentStage || !sawStopped || !sawDiagnostic {
		t.Fatalf("logs missing current/stopped/diagnostic entries: %#v", logs)
	}
}

func TestUpgradeOperationResponseProjectsCurrentVersion(t *testing.T) {
	operation := &domain.Operation{
		OperationID:     "11111111-1111-4111-8111-111111111111",
		RequestID:       "22222222-2222-4222-8222-222222222222",
		OperatorID:      7,
		ManifestID:      "release-1.1.0",
		ManifestDigest:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ReleaseVersion:  "1.1.0",
		Status:          domain.StatusQueued,
		MigrationStatus: domain.MigrationStatusNotStarted,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
		StageTimes:      map[domain.Status]time.Time{},
	}
	response := NewUpgradeOperationResponse(operation, "1.0.0")
	if response.CurrentVersion != "1.0.0" {
		t.Fatalf("currentVersion = %q, want 1.0.0", response.CurrentVersion)
	}
}

func TestFullUpgradeOperationResponseAddsScopeWithoutChangingBasicShape(t *testing.T) {
	now := time.Now().UTC()
	operation := &domain.Operation{
		OperationID: "11111111-1111-4111-8111-111111111111", RequestID: "22222222-2222-4222-8222-222222222222", OperatorID: 7,
		ManifestID: "release-1.1.0", ManifestDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ReleaseVersion: "1.1.0", CompatibilityRange: "*", Status: domain.StatusQueued,
		MigrationStatus: domain.MigrationStatusNotStarted, MigrationType: "none", CreatedAt: now, UpdatedAt: now,
		StageTimes: map[domain.Status]time.Time{domain.StatusQueued: now}, ObservedDigests: map[string]string{},
		ExecutionMode: domain.ExecutionModeFrontendOnly, WorkDisposition: domain.WorkDispositionNotRequired,
		PlanSummary:                domain.PlanSummary{TouchedServices: []string{"frontend"}},
		PlanDigest:                 "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		BaselineDeploymentDigest:   "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		ConfirmedDeploymentVersion: "1.0.0",
	}
	basicBytes, err := json.Marshal(NewUpgradeOperationResponse(operation, "1.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	var basic map[string]json.RawMessage
	if err := json.Unmarshal(basicBytes, &basic); err != nil {
		t.Fatal(err)
	}
	if _, found := basic["executionMode"]; found {
		t.Fatal("BASIC response unexpectedly contains executionMode")
	}
	full := NewFullUpgradeOperationResponse(operation, "1.0.0")
	if full.ExecutionMode != "frontend_only" || full.WorkDisposition != "not_required" || len(full.PlanSummary.TouchedServices) != 1 || full.PlanSummary.TouchedServices[0] != "frontend" || full.ConfirmedDeploymentVersion != "1.0.0" {
		t.Fatalf("FULL scope projection = %#v", full)
	}
	operation.Status = domain.StatusSucceeded
	completed := NewFullUpgradeOperationResponse(operation, "1.0.0")
	if completed.ConfirmedDeploymentVersion != operation.ReleaseVersion {
		t.Fatalf("completed frontend-only version = %q, want target %q", completed.ConfirmedDeploymentVersion, operation.ReleaseVersion)
	}
	if operation.ConfirmedDeploymentVersion != "1.0.0" {
		t.Fatalf("FULL projection rewrote persisted baseline = %q", operation.ConfirmedDeploymentVersion)
	}
}

func TestUpgradeLogsProjectsBoundedProgressEventsAndRedactsUnsafeDiagnostic(t *testing.T) {
	base := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	operation := &domain.Operation{
		OperationID: "11111111-1111-4111-8111-111111111111",
		Status:      domain.StatusUpdating,
		StageTimes:  map[domain.Status]time.Time{domain.StatusQueued: base, domain.StatusUpdating: base.Add(time.Minute)},
		ProgressEvents: []domain.ProgressEvent{
			{Timestamp: base.Add(2 * time.Minute), Stage: domain.StatusUpdating, MessageKey: "pullImagesStarted", Message: "Pulling release images", Metadata: map[string]string{"scope": "release"}},
			{Timestamp: base.Add(3 * time.Minute), Stage: domain.StatusUpdating, MessageKey: "servicesUpdateStarted", Message: "Updating core services", Metadata: map[string]string{}},
		},
		CreatedAt: base, UpdatedAt: base.Add(3 * time.Minute),
		Diagnostic: "compose failed at /deployment/.env with TOKEN=secret",
	}

	logs := upgradeLogs(operation)
	if len(logs) < 4 || len(logs) > domain.MaxProgressEvents {
		t.Fatalf("log count = %d, want progress entries plus bounded lifecycle evidence", len(logs))
	}
	seen := map[string]bool{}
	var pullEntry UpgradeLogEntry
	for index, entry := range logs {
		if index > 0 && entry.Timestamp.Before(logs[index-1].Timestamp) {
			t.Fatalf("logs are not ordered: %#v", logs)
		}
		seen[entry.MessageKey] = true
		if entry.MessageKey == "pullImagesStarted" {
			pullEntry = entry
		}
		if strings.Contains(entry.Message, "/deployment") || strings.Contains(strings.ToLower(entry.Message), "token") {
			t.Fatalf("unsafe log text leaked: %#v", entry)
		}
	}
	if !seen["pullImagesStarted"] || !seen["servicesUpdateStarted"] || !seen["diagnostic"] {
		t.Fatalf("progress or diagnostic entries missing: %#v", logs)
	}
	if pullEntry.Metadata["scope"] != "release" {
		t.Fatalf("progress metadata = %#v, want scope", pullEntry.Metadata)
	}
}

func TestUpgradeLogsRetainsNewestBoundedProgressWindow(t *testing.T) {
	base := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	events := make([]domain.ProgressEvent, 0, domain.MaxProgressEvents)
	for index := 0; index < domain.MaxProgressEvents; index++ {
		events = append(events, domain.ProgressEvent{
			Timestamp: base.Add(time.Duration(index+1) * time.Minute), Stage: domain.StatusUpdating,
			MessageKey: "heartbeat-" + strconv.Itoa(index+1), Message: "Progress checkpoint reached",
			Metadata: map[string]string{},
		})
	}
	operation := &domain.Operation{
		OperationID: "11111111-1111-4111-8111-111111111111", Status: domain.StatusUpdating,
		StageTimes: map[domain.Status]time.Time{domain.StatusQueued: base, domain.StatusUpdating: base}, ProgressEvents: events,
		CreatedAt: base, UpdatedAt: base.Add(time.Duration(domain.MaxProgressEvents+1) * time.Minute),
	}
	logs := upgradeLogs(operation)
	if len(logs) != domain.MaxProgressEvents {
		t.Fatalf("log count = %d, want %d", len(logs), domain.MaxProgressEvents)
	}
	if logs[len(logs)-1].MessageKey != "heartbeat-32" {
		t.Fatalf("newest log = %#v, want heartbeat-32", logs[len(logs)-1])
	}
}
