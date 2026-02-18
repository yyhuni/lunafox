package domain

import (
	"errors"
	"testing"
	"time"
)

func TestScanTransition(t *testing.T) {
	now := time.Now()
	scan := &Scan{TargetID: 1, Mode: ScanModeQuick, Status: ScanStatusPending, CreatedAt: now}

	if err := scan.MarkRunning(); err != nil {
		t.Fatalf("MarkRunning failed: %v", err)
	}
	if scan.Status != ScanStatusRunning {
		t.Fatalf("expected running, got %s", scan.Status)
	}

	if err := scan.MarkCompleted(); err != nil {
		t.Fatalf("MarkCompleted failed: %v", err)
	}
	if scan.Status != ScanStatusCompleted {
		t.Fatalf("expected completed, got %s", scan.Status)
	}
	if scan.Progress != 100 {
		t.Fatalf("expected progress 100, got %d", scan.Progress)
	}

	if err := scan.Stop(time.Now()); !errors.Is(err, ErrScanCannotStop) {
		t.Fatalf("expected ErrScanCannotStop after completion, got %v", err)
	}
}

func TestScanFailValidation(t *testing.T) {
	now := time.Now()
	scan := &Scan{TargetID: 1, Mode: ScanModeFull, Status: ScanStatusPending, CreatedAt: now}
	if err := scan.MarkRunning(); err != nil {
		t.Fatalf("MarkRunning failed: %v", err)
	}

	if err := scan.MarkFailed("", time.Now()); !errors.Is(err, ErrFailureMessageMissing) {
		t.Fatalf("expected ErrFailureMessageMissing, got %v", err)
	}

	if err := scan.MarkFailed("worker timeout", time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scan.Status != ScanStatusFailed {
		t.Fatalf("expected failed status, got %s", scan.Status)
	}
	if scan.StoppedAt == nil {
		t.Fatalf("expected stoppedAt to be set")
	}
}

func TestParseScanStatusAndActive(t *testing.T) {
	status, ok := ParseScanStatus("running")
	if !ok || status != ScanStatusRunning {
		t.Fatalf("expected running parse success, got status=%q ok=%v", status, ok)
	}
	if !IsActiveScanStatus(status) {
		t.Fatalf("running should be active")
	}

	_, ok = ParseScanStatus("unknown")
	if ok {
		t.Fatalf("unknown status should fail parse")
	}
}

func TestResolveScanStatusFromTaskCounts(t *testing.T) {
	status, shouldUpdate := ResolveScanStatusFromTaskCounts(1, 0, 0, 0)
	if shouldUpdate {
		t.Fatalf("pending tasks should not trigger terminal status, got %s", status)
	}

	status, shouldUpdate = ResolveScanStatusFromTaskCounts(0, 0, 2, 0)
	if !shouldUpdate || status != ScanStatusFailed {
		t.Fatalf("expected failed terminal status, got %s update=%v", status, shouldUpdate)
	}

	status, shouldUpdate = ResolveScanStatusFromTaskCounts(0, 0, 0, 1)
	if !shouldUpdate || status != ScanStatusCancelled {
		t.Fatalf("expected cancelled terminal status, got %s update=%v", status, shouldUpdate)
	}

	status, shouldUpdate = ResolveScanStatusFromTaskCounts(0, 0, 0, 0)
	if !shouldUpdate || status != ScanStatusCompleted {
		t.Fatalf("expected completed terminal status, got %s update=%v", status, shouldUpdate)
	}
}
