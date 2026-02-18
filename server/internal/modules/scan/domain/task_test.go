package domain

import (
	"errors"
	"testing"
	"time"
)

func TestScanTaskTransition(t *testing.T) {
	startedAt := time.Now()
	task := &ScanTask{ID: 1, ScanID: 2, Name: "subdomain", Stage: 1, Status: TaskStatusRunning, StartedAt: &startedAt}

	completedAt := startedAt.Add(time.Minute)
	if err := task.Complete(completedAt); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if task.Status != TaskStatusCompleted {
		t.Fatalf("expected completed status, got %s", task.Status)
	}
}

func TestScanTaskFailValidation(t *testing.T) {
	task := &ScanTask{ID: 3, ScanID: 4, Name: "nuclei", Stage: 2, Status: TaskStatusRunning}

	if err := task.Fail("", time.Now()); !errors.Is(err, ErrFailureMessageMissing) {
		t.Fatalf("expected ErrFailureMessageMissing, got %v", err)
	}

	if err := task.Fail("panic", time.Now()); err != nil {
		t.Fatalf("unexpected fail error: %v", err)
	}
	if task.Status != TaskStatusFailed {
		t.Fatalf("expected failed status, got %s", task.Status)
	}
}

func TestScanTaskCancelValidation(t *testing.T) {
	task := &ScanTask{ID: 5, ScanID: 6, Status: TaskStatusCompleted}
	if err := task.Cancel(time.Now()); !errors.Is(err, ErrInvalidStatusChange) {
		t.Fatalf("expected ErrInvalidStatusChange, got %v", err)
	}
}

func TestApplyAgentResult(t *testing.T) {
	task := &ScanTask{Status: TaskStatusRunning}
	if err := task.ApplyAgentResult(TaskStatusCompleted, "", time.Now()); err != nil {
		t.Fatalf("expected completed transition to succeed, got %v", err)
	}

	task = &ScanTask{Status: TaskStatusRunning}
	if err := task.ApplyAgentResult(TaskStatusFailed, "", time.Now()); !errors.Is(err, ErrFailureMessageMissing) {
		t.Fatalf("expected ErrFailureMessageMissing, got %v", err)
	}

	task = &ScanTask{Status: TaskStatusPending}
	if err := task.ApplyAgentResult(TaskStatusCancelled, "", time.Now()); !errors.Is(err, ErrInvalidStatusChange) {
		t.Fatalf("expected ErrInvalidStatusChange, got %v", err)
	}
}

func TestParseTaskStatus(t *testing.T) {
	status, ok := ParseTaskStatus("blocked")
	if !ok || status != TaskStatusBlocked {
		t.Fatalf("expected blocked parse success, got status=%q ok=%v", status, ok)
	}

	if !IsTerminalTaskStatus(TaskStatusFailed) {
		t.Fatalf("failed should be terminal")
	}
	if IsTerminalTaskStatus(TaskStatusRunning) {
		t.Fatalf("running should not be terminal")
	}

	_, ok = ParseTaskStatus("unknown")
	if ok {
		t.Fatalf("unknown status should fail parse")
	}
}
