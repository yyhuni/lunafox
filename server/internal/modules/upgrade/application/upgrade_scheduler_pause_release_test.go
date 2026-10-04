package application

import (
	"context"
	"errors"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

type schedulerResumerStub struct{ calls int }

func (stub *schedulerResumerStub) Resume() { stub.calls++ }

func fullUpgradeOperationInput(t *testing.T) CreateUpgradeOperationInput {
	t.Helper()
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	return CreateUpgradeOperationInput{
		RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID,
		ManifestDigest: manifest.Digest(), Confirmed: true,
	}
}

// A failed preparation is committed by the process that keeps serving, so the
// pause it acquired must be released before the failure is returned.
func TestUpgradeServiceReleasesSchedulerPauseAfterFailedPreparation(t *testing.T) {
	dispatcher := &upgradeDispatcherStub{}
	service, _ := newUpgradeServiceForTest(t, dispatcher)
	resumer := &schedulerResumerStub{}
	service.resumer = resumer
	service.coordinator = preDispatchCoordinatorStub{prepare: func(context.Context) (PreparationResult, error) {
		return PreparationResult{}, errors.New("cancel commit failed")
	}}

	operation, _, err := service.CreateOperation(context.Background(), 7, fullUpgradeOperationInput(t))
	if err == nil {
		t.Fatal("CreateOperation() must fail when preparation fails")
	}
	if operation.Status != domain.StatusFailed || operation.WorkDisposition != domain.WorkDispositionCancellationFailed {
		t.Fatalf("operation outcome = %s/%s; want failed/cancellation_failed", operation.Status, operation.WorkDisposition)
	}
	if resumer.calls != 1 {
		t.Fatalf("resume calls after failed preparation = %d, want 1", resumer.calls)
	}
}

// A host handoff that was never accepted leaves this process serving; the
// pause must go even though the cancellation itself committed.
func TestUpgradeServiceReleasesSchedulerPauseWhenHostHandoffNotAccepted(t *testing.T) {
	dispatcher := &upgradeDispatcherStub{err: errors.New("host daemon unavailable")}
	service, _ := newUpgradeServiceForTest(t, dispatcher)
	resumer := &schedulerResumerStub{}
	service.resumer = resumer
	service.coordinator = preDispatchCoordinatorStub{prepare: func(context.Context) (PreparationResult, error) {
		return PreparationResult{CancelledScanCount: 1}, nil
	}}

	operation, _, err := service.CreateOperation(context.Background(), 7, fullUpgradeOperationInput(t))
	if err == nil {
		t.Fatal("CreateOperation() must fail when the host rejects the handoff")
	}
	if !operation.Status.IsTerminal() {
		t.Fatalf("handoff-rejected status = %s; want terminal", operation.Status)
	}
	if resumer.calls != 1 {
		t.Fatalf("resume calls after rejected handoff = %d, want 1", resumer.calls)
	}
}

// A stop request the host never accepted also ends in a terminal outcome that
// this process commits, so the pause must go.
func TestUpgradeServiceReleasesSchedulerPauseWhenStopHandoffNotAccepted(t *testing.T) {
	dispatcher := &upgradeDispatcherStub{}
	service, _ := newUpgradeServiceForTest(t, dispatcher)
	resumer := &schedulerResumerStub{}
	service.resumer = resumer
	service.coordinator = preDispatchCoordinatorStub{prepare: func(context.Context) (PreparationResult, error) {
		return PreparationResult{}, nil
	}}

	operation, _, err := service.CreateOperation(context.Background(), 7, fullUpgradeOperationInput(t))
	if err != nil {
		t.Fatal(err)
	}
	dispatcher.err = errors.New("host daemon unavailable")
	stopped, err := service.StopOperation(context.Background(), 7, operation.OperationID, true)
	if err == nil {
		t.Fatal("StopOperation() must fail when the host rejects the stop")
	}
	if !stopped.Status.IsTerminal() {
		t.Fatalf("stop-unavailable status = %s; want terminal", stopped.Status)
	}
	if resumer.calls != 1 {
		t.Fatalf("resume calls after rejected stop = %d, want 1", resumer.calls)
	}
}

// The recovery watchdog closes host-less operations in the surviving process;
// every such terminal reconcile must hand scheduled claiming back.
func TestUpgradeServiceReleasesSchedulerPauseOnTerminalRecoveryReconcile(t *testing.T) {
	dispatcher := &upgradeDispatcherStub{}
	service, _ := newUpgradeServiceForTest(t, dispatcher)
	resumer := &schedulerResumerStub{}
	service.resumer = resumer
	service.coordinator = preDispatchCoordinatorStub{prepare: func(context.Context) (PreparationResult, error) {
		return PreparationResult{}, nil
	}}

	operation, _, err := service.CreateOperation(context.Background(), 7, fullUpgradeOperationInput(t))
	if err != nil {
		t.Fatal(err)
	}
	if resumer.calls != 0 {
		t.Fatalf("resume calls while the operation is active = %d, want 0", resumer.calls)
	}
	reconciled, err := service.ReconcileJournalUnavailable(context.Background(), "host upgrade journal is unreadable")
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.OperationID != operation.OperationID || !reconciled.Status.IsTerminal() {
		t.Fatalf("reconciled = %s/%s; want terminal %s", reconciled.OperationID, reconciled.Status, operation.OperationID)
	}
	if resumer.calls != 1 {
		t.Fatalf("resume calls after terminal recovery reconcile = %d, want 1", resumer.calls)
	}
}

// Frontend-only operations never pause the scheduler, so their terminal
// outcomes must not touch pause state at all.
func TestUpgradeServiceFrontendOnlyTerminalNeverTouchesSchedulerPause(t *testing.T) {
	dispatcher := &scopePlanningDispatcherStub{plan: HostUpgradeScopePlan{
		ExecutionMode: domain.ExecutionModeFrontendOnly,
		PlanDigest:    "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		BaselineDeploymentDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		TouchedServices:          []string{"frontend"}, ConfirmedDeploymentVersion: "1.0.0",
	}}
	dispatcher.err = errors.New("host daemon unavailable")
	service, _ := newUpgradeServiceForTest(t, dispatcher)
	resumer := &schedulerResumerStub{}
	service.resumer = resumer

	operation, _, err := service.CreateOperation(context.Background(), 7, fullUpgradeOperationInput(t))
	if err == nil {
		t.Fatal("CreateOperation() must fail when the host rejects the handoff")
	}
	if !operation.Status.IsTerminal() {
		t.Fatalf("frontend-only dispatch failure status = %s; want terminal", operation.Status)
	}
	if resumer.calls != 0 {
		t.Fatalf("frontend-only terminal touched the scheduler pause: resume calls = %d", resumer.calls)
	}
}
