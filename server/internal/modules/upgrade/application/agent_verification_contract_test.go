package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

type agentVerificationSourceStub struct {
	expectations   []domain.AgentExpectation
	notifyCalls    int
	reconcileCalls int
	notifiedTarget AgentUpgradeTarget
}

func (stub *agentVerificationSourceStub) Snapshot(context.Context, AgentUpgradeTarget) ([]domain.AgentExpectation, error) {
	return append([]domain.AgentExpectation(nil), stub.expectations...), nil
}

func (stub *agentVerificationSourceStub) NotifyUpdateRequired(_ context.Context, target AgentUpgradeTarget, _ []domain.AgentExpectation) error {
	stub.notifyCalls++
	stub.notifiedTarget = target
	return nil
}

func (stub *agentVerificationSourceStub) Reconcile(_ context.Context, _ AgentUpgradeTarget, _ []domain.AgentExpectation) ([]domain.AgentExpectation, error) {
	stub.reconcileCalls++
	return append([]domain.AgentExpectation(nil), stub.expectations...), nil
}

type agentVerificationVerifierStub struct {
	calls  int
	passed bool
}

func (stub *agentVerificationVerifierStub) Verify(context.Context, *domain.Operation, VerificationEvidence) (VerificationResult, error) {
	stub.calls++
	if !stub.passed {
		return VerificationResult{Diagnostic: "service evidence is not ready"}, nil
	}
	return VerificationResult{Passed: true}, nil
}

func TestAgentVerificationRequiresReconnectBeforeSucceededAndNotifiesOnce(t *testing.T) {
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	targetDigest := runtimeAgentDigest(manifest)
	source := &agentVerificationSourceStub{}
	verifier := &agentVerificationVerifierStub{passed: true}
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	service.agentSource = source
	service.verifier = verifier
	service.now = func() time.Time { return now }

	deadline := now.Add(time.Hour)
	operation := &domain.Operation{
		OperationID:         "77777777-7777-4777-8777-777777777777",
		RequestID:           "88888888-8888-4888-8888-888888888888",
		OperatorID:          7,
		ManifestID:          manifest.Upgrade.ManifestID,
		ManifestDigest:      manifest.Digest(),
		ReleaseVersion:      manifest.ReleaseVersion,
		CompatibilityRange:  manifest.Upgrade.CompatibilityRange,
		Status:              domain.StatusRestarting,
		MigrationStatus:     domain.MigrationStatusNotStarted,
		MigrationType:       "none",
		AgentDesiredVersion: manifest.ReleaseVersion,
		AgentTargetDigest:   targetDigest,
		AgentExpectations: []domain.AgentExpectation{{
			AgentID: 42, DesiredVersion: manifest.ReleaseVersion, TargetDigest: targetDigest,
		}},
		AgentSummary:              domain.AgentSummary{Expected: 1, Missing: 1},
		AgentVerificationDeadline: &deadline,
		ObservedDigests:           map[string]string{},
		StageTimes:                map[domain.Status]time.Time{domain.StatusRestarting: now},
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}
	repository.byID[operation.OperationID] = operation
	repository.byRequest[operation.RequestID] = operation
	repository.active = operation

	// The Agent is offline at first. The operation must remain in the
	// verification phase, and the final verifier must not run prematurely.
	source.expectations = []domain.AgentExpectation{{
		AgentID: 42, DesiredVersion: manifest.ReleaseVersion, TargetDigest: targetDigest,
		ObservedVersion: "", Connected: false, Healthy: false, ClaimReady: false,
		Diagnostic: "Agent is offline",
	}}
	first, err := service.ReconcileHostEvent(context.Background(), HostUpgradeEvent{
		OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest,
		Stage: string(domain.StatusAgentVerifying),
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != domain.StatusAgentVerifying || first.AgentSummary.Ready != 0 || verifier.calls != 0 {
		t.Fatalf("offline Agent advanced unexpectedly: operation=%#v verifierCalls=%d", first, verifier.calls)
	}
	if source.notifyCalls != 1 {
		t.Fatalf("update_required calls after first reconcile=%d, want 1", source.notifyCalls)
	}

	// A later authenticated heartbeat satisfies every readiness fence. Replaying
	// the same host checkpoint may advance the operation, but must not publish a
	// duplicate update_required event.
	source.expectations = []domain.AgentExpectation{{
		AgentID: 42, DesiredVersion: manifest.ReleaseVersion, TargetDigest: targetDigest,
		ObservedVersion: manifest.ReleaseVersion, Connected: true, Healthy: true, ClaimReady: true,
	}}
	second, err := service.ReconcileHostEvent(context.Background(), HostUpgradeEvent{
		OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest,
		Stage: string(domain.StatusAgentVerifying),
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != domain.StatusSucceeded || second.AgentSummary.Ready != 1 || verifier.calls != 1 {
		t.Fatalf("ready Agent did not complete verification: operation=%#v verifierCalls=%d", second, verifier.calls)
	}
	if source.notifyCalls != 1 || source.reconcileCalls != 2 {
		t.Fatalf("reconciliation was not idempotent: notify=%d reconcile=%d", source.notifyCalls, source.reconcileCalls)
	}
	if source.notifiedTarget.Version != manifest.ReleaseVersion || source.notifiedTarget.Digest != targetDigest || strings.TrimSpace(source.notifiedTarget.ImageRef) == "" {
		t.Fatalf("update_required target was not manifest-bound: %#v", source.notifiedTarget)
	}

	// A late duplicate completion cannot reopen or re-notify a succeeded
	// operation.
	if _, err := service.ReconcileHostEvent(context.Background(), HostUpgradeEvent{
		OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest,
		Stage: string(domain.StatusSucceeded),
	}); err != nil {
		t.Fatal(err)
	}
	if source.notifyCalls != 1 || verifier.calls != 1 {
		t.Fatalf("late host event caused duplicate verification: notify=%d verifier=%d", source.notifyCalls, verifier.calls)
	}
}

func TestAgentVerificationDeadlineProducesNeedsAttention(t *testing.T) {
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 13, 13, 0, 0, 0, time.UTC)
	targetDigest := runtimeAgentDigest(manifest)
	source := &agentVerificationSourceStub{expectations: []domain.AgentExpectation{{
		AgentID: 43, DesiredVersion: manifest.ReleaseVersion, TargetDigest: targetDigest,
		Connected: false, Healthy: false, ClaimReady: false,
	}}}
	verifier := &agentVerificationVerifierStub{passed: true}
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	service.agentSource = source
	service.verifier = verifier
	service.now = func() time.Time { return now }
	deadline := now
	operation := &domain.Operation{
		OperationID:         "99999999-9999-4999-8999-999999999999",
		RequestID:           "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		OperatorID:          7,
		ManifestID:          manifest.Upgrade.ManifestID,
		ManifestDigest:      manifest.Digest(),
		ReleaseVersion:      manifest.ReleaseVersion,
		CompatibilityRange:  manifest.Upgrade.CompatibilityRange,
		Status:              domain.StatusAgentVerifying,
		MigrationStatus:     domain.MigrationStatusNotStarted,
		MigrationType:       "none",
		AgentDesiredVersion: manifest.ReleaseVersion,
		AgentTargetDigest:   targetDigest,
		AgentExpectations: []domain.AgentExpectation{{
			AgentID: 43, DesiredVersion: manifest.ReleaseVersion, TargetDigest: targetDigest,
		}},
		AgentSummary:              domain.AgentSummary{Expected: 1, Missing: 1},
		AgentVerificationDeadline: &deadline,
		ObservedDigests:           map[string]string{},
		StageTimes:                map[domain.Status]time.Time{domain.StatusAgentVerifying: now},
		CreatedAt:                 now.Add(-time.Minute),
		UpdatedAt:                 now.Add(-time.Minute),
	}
	repository.byID[operation.OperationID] = operation
	repository.byRequest[operation.RequestID] = operation
	repository.active = operation

	updated, err := service.ReconcileHostEvent(context.Background(), HostUpgradeEvent{
		OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest,
		Stage: string(domain.StatusAgentVerifying),
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.StatusNeedsAttention || updated.CompletedAt == nil {
		t.Fatalf("expired Agent verification = %#v, want needs_attention terminal state", updated)
	}
	if verifier.calls != 0 {
		t.Fatalf("final verifier ran despite missing Agent readiness: %d calls", verifier.calls)
	}
}

var _ AgentUpgradeSource = (*agentVerificationSourceStub)(nil)
var _ AgentUpgradeReconciler = (*agentVerificationSourceStub)(nil)
var _ UpgradeVerifier = (*agentVerificationVerifierStub)(nil)
