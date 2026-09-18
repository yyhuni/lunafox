package dto

import (
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
