package domain

import (
	"errors"
	"testing"
	"time"
)

func TestScanTaskApplyAgentResult(t *testing.T) {
	completedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	succeeding := &ScanTask{Status: TaskStatusRunning}
	if err := succeeding.ApplyAgentResult(TaskStatusSucceeded, "", completedAt); err != nil {
		t.Fatalf("expected succeeded transition to succeed, got %v", err)
	}
	if succeeding.Status != TaskStatusSucceeded || succeeding.CompletedAt == nil || !succeeding.CompletedAt.Equal(completedAt) {
		t.Fatalf("expected task to be marked succeeded at completion time, got %+v", succeeding)
	}

	failing := &ScanTask{Status: TaskStatusRunning}
	if err := failing.ApplyAgentResult(TaskStatusFailed, "", completedAt); !errors.Is(err, ErrFailureMessageMissing) {
		t.Fatalf("expected ErrFailureMessageMissing, got %v", err)
	}

	cancelling := &ScanTask{Status: TaskStatusPending}
	if err := cancelling.ApplyAgentResult(TaskStatusCancelled, "", completedAt); !errors.Is(err, ErrInvalidStatusChange) {
		t.Fatalf("expected ErrInvalidStatusChange, got %v", err)
	}
}

func TestParseTaskStatus(t *testing.T) {
	status, ok := ParseTaskStatus("blocked")
	if !ok || status != TaskStatusBlocked {
		t.Fatalf("expected blocked parse success, got status=%q ok=%v", status, ok)
	}

	status, ok = ParseTaskStatus("skipped")
	if !ok || status != TaskStatusSkipped {
		t.Fatalf("expected skipped parse success, got status=%q ok=%v", status, ok)
	}

	if !IsTerminalTaskStatus(TaskStatusFailed) {
		t.Fatalf("failed should be terminal")
	}
	if !IsTerminalTaskStatus(TaskStatusSkipped) {
		t.Fatalf("skipped should be terminal")
	}
	if IsTerminalTaskStatus(TaskStatusRunning) {
		t.Fatalf("running should not be terminal")
	}

	_, ok = ParseTaskStatus("unknown")
	if ok {
		t.Fatalf("unknown status should fail parse")
	}
}
