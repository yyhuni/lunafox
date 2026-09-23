package application

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

var upgradeObservedDigestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

// reconcileAfterHostEvent is called for every durable host checkpoint at or
// beyond the Agent verification boundary. It is intentionally idempotent: a
// recovery poll may observe the same journal stage many times while Agents
// reconnect or while the Server-side health probes become available.
func (service *Service) reconcileAfterHostEvent(ctx context.Context, operation *domain.Operation) (*domain.Operation, error) {
	if service == nil || operation == nil {
		return operation, nil
	}
	if operation.MigrationStatus == domain.MigrationStatusFailed || operation.MigrationStatus == domain.MigrationStatusUnknown {
		return service.markVerificationRecovery(ctx, operation, "database migration outcome requires recovery")
	}
	if operation.Status != domain.StatusAgentVerifying && operation.Status != domain.StatusVerifying {
		return operation, nil
	}
	if operation.EffectiveExecutionMode() == domain.ExecutionModeFrontendOnly {
		// A frontend-only plan has no Agent snapshot or lifecycle dependency.
		// Do not call Agent target parsing, notification, reconciliation, or its
		// timeout path merely because it shares the final verifying stage.
		if operation.Status != domain.StatusVerifying {
			return operation, nil
		}
		manifest, err := service.loadTargetManifest(operation.ManifestDigest)
		if err != nil || manifest == nil || manifest.Digest() != operation.ManifestDigest {
			return service.recordVerificationDiagnostic(ctx, operation, "release manifest could not be reloaded")
		}
		return service.verifyDeployment(ctx, operation, manifest)
	}

	// The target is re-read from the immutable digest cache. The fixed channel
	// may advance while an Operation is running and must never redirect its
	// Agent or final verification target.
	manifest, err := service.loadTargetManifest(operation.ManifestDigest)
	if err != nil {
		return service.recordVerificationDiagnostic(ctx, operation, "release manifest could not be reloaded")
	}
	target, err := service.agentTarget(manifest)
	if err != nil || target.Version != operation.AgentDesiredVersion || target.Digest != operation.AgentTargetDigest || manifest.Digest() != operation.ManifestDigest {
		return service.markVerificationRecovery(ctx, operation, "release manifest target no longer matches the Upgrade Operation")
	}

	// A restarted Server may first observe the later verifying checkpoint. The
	// notification remains idempotent, so both verification states must cover
	// remote Agents whose update_required event was missed during downtime.
	if err := service.notifyAgentsOnce(ctx, operation, target); err != nil {
		// Notification is best effort. Keep the operation alive so a healthy
		// Agent that reconnects later can still satisfy the deadline.
		operation.Diagnostic = sanitizeUpgradeDiagnostic(err.Error())
	}

	refreshed, refreshErr := service.refreshAgentExpectations(ctx, target, operation.AgentExpectations)
	if refreshErr != nil {
		return service.recordVerificationDiagnostic(ctx, operation, "Agent verification data is temporarily unavailable")
	}
	operation.AgentExpectations = refreshed
	operation.AgentSummary = summarizeAgentExpectations(refreshed)
	operation.UpdatedAt = service.now().UTC()

	if !allAgentsReady(refreshed) {
		deadline := operation.AgentVerificationDeadline
		if deadline != nil && !service.now().UTC().Before(deadline.UTC()) {
			return service.markVerificationAttention(ctx, operation, "one or more Agents did not reconnect and become ready before the verification deadline")
		}
		if err := service.repository.Update(ctx, operation); err != nil {
			return nil, err
		}
		return operation, nil
	}

	if operation.Status == domain.StatusAgentVerifying {
		previous := operation.Status
		now := service.now().UTC()
		operation.Status = domain.StatusVerifying
		operation.UpdatedAt = now
		if operation.StageTimes == nil {
			operation.StageTimes = map[domain.Status]time.Time{}
		}
		operation.StageTimes[domain.StatusVerifying] = now
		if _, err := service.persistReconciledOperation(ctx, operation, previous, false); err != nil {
			return nil, err
		}
	}

	return service.verifyDeployment(ctx, operation, manifest)
}

// verifyDeployment performs only the final service/digest evidence check. It
// is shared by full and frontend-only operations after their distinct lifecycle
// prerequisites have been satisfied by their respective branches above.
func (service *Service) verifyDeployment(ctx context.Context, operation *domain.Operation, manifest *releasemanifest.Manifest) (*domain.Operation, error) {
	if operation == nil || manifest == nil {
		return service.recordVerificationDiagnostic(ctx, operation, "release manifest could not be reloaded")
	}
	if service.verifier == nil {
		operation.Diagnostic = "waiting for service and dependency verification evidence"
		operation.UpdatedAt = service.now().UTC()
		if err := service.repository.Update(ctx, operation); err != nil {
			return nil, err
		}
		return operation, nil
	}
	expectedDigests, digestErr := manifest.RuntimeImageDigests()
	if digestErr != nil {
		return service.markVerificationRecovery(ctx, operation, "release manifest component digests could not be resolved")
	}
	verification, verifyErr := service.verifier.Verify(ctx, operation, VerificationEvidence{
		ObservedDigests: cloneStringMap(operation.ObservedDigests),
		ExpectedDigests: expectedDigests,
	})
	if verifyErr != nil {
		return service.recordVerificationDiagnostic(ctx, operation, "service verification is temporarily unavailable")
	}
	mergeObservedDigests(operation, verification.ObservedDigests)
	if !verification.Passed {
		diagnostic := sanitizeUpgradeDiagnostic(verification.Diagnostic)
		if diagnostic == "" {
			diagnostic = "one or more upgrade verification checks are not complete"
		}
		operation.Diagnostic = diagnostic
		operation.UpdatedAt = service.now().UTC()
		if err := service.repository.Update(ctx, operation); err != nil {
			return nil, err
		}
		return operation, nil
	}
	if operation.ExecutionMode != "" {
		// A v2 operation is not complete until the host atomically records the
		// fully verified deployment as its next baseline. Keep verifying on a
		// transient confirmation failure: turning it terminal would either claim
		// an unconfirmed frontend-only deployment or force a new full lifecycle.
		if err := service.confirmDeployment(ctx, operation); err != nil {
			return service.recordVerificationDiagnostic(ctx, operation, "host deployment confirmation is temporarily unavailable")
		}
	}

	now := service.now().UTC()
	previous := operation.Status
	operation.Status = domain.StatusSucceeded
	operation.Diagnostic = ""
	operation.CompletedAt = &now
	operation.UpdatedAt = now
	if operation.StageTimes == nil {
		operation.StageTimes = map[domain.Status]time.Time{}
	}
	operation.StageTimes[domain.StatusSucceeded] = now
	if _, err := service.persistReconciledOperation(ctx, operation, previous, false); err != nil {
		return nil, err
	}
	return operation, nil
}

func (service *Service) confirmDeployment(ctx context.Context, operation *domain.Operation) error {
	if service == nil || service.dispatcher == nil || operation == nil {
		return domain.ErrUpgradeHostUnavailable
	}
	if operation.ExecutionMode == "" {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	confirmCtx, cancel := context.WithTimeout(ctx, hostDispatchTimeout)
	defer cancel()
	if err := service.dispatcher.Dispatch(confirmCtx, HostUpgradeRequest{
		OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest, Action: HostUpgradeActionConfirm,
		ExecutionMode: operation.ExecutionMode, PlanDigest: operation.PlanDigest,
		BaselineDeploymentDigest:   operation.BaselineDeploymentDigest,
		TouchedServices:            append([]string(nil), operation.PlanSummary.TouchedServices...),
		ConfirmedDeploymentVersion: operation.ConfirmedDeploymentVersion,
	}); err != nil {
		return fmt.Errorf("confirm host deployment baseline: %w", err)
	}
	return nil
}

func (service *Service) refreshAgentExpectations(ctx context.Context, target AgentUpgradeTarget, expected []domain.AgentExpectation) ([]domain.AgentExpectation, error) {
	if len(expected) == 0 {
		return []domain.AgentExpectation{}, nil
	}
	if service.agentSource == nil {
		return expected, nil
	}
	if reconciler, ok := service.agentSource.(AgentUpgradeReconciler); ok {
		return reconciler.Reconcile(ctx, target, expected)
	}
	current, err := service.agentSource.Snapshot(ctx, target)
	if err != nil {
		return nil, err
	}
	byID := make(map[int]domain.AgentExpectation, len(current))
	for _, item := range current {
		byID[item.AgentID] = item
	}
	result := make([]domain.AgentExpectation, 0, len(expected))
	for _, wanted := range expected {
		if observed, ok := byID[wanted.AgentID]; ok {
			observed.DesiredVersion = wanted.DesiredVersion
			observed.TargetDigest = wanted.TargetDigest
			result = append(result, observed)
			continue
		}
		wanted.Connected = false
		wanted.Healthy = false
		wanted.ClaimReady = false
		wanted.ObservedVersion = ""
		wanted.ObservedDigest = ""
		wanted.LastObservedAt = nil
		wanted.Diagnostic = "Agent is no longer registered"
		result = append(result, wanted)
	}
	return result, nil
}

func (service *Service) notifyAgentsOnce(ctx context.Context, operation *domain.Operation, target AgentUpgradeTarget) error {
	if service.agentSource == nil || operation == nil || len(operation.AgentExpectations) == 0 {
		return nil
	}
	service.agentNotifyMu.Lock()
	if _, sent := service.agentNotifiedOperations[operation.OperationID]; sent {
		service.agentNotifyMu.Unlock()
		return nil
	}
	service.agentNotifyMu.Unlock()
	if err := service.agentSource.NotifyUpdateRequired(ctx, target, operation.AgentExpectations); err != nil {
		// A failed best-effort delivery must remain retryable. Marking it sent
		// before the publisher accepts the batch would strand Agents that were
		// offline during the first post-restart reconciliation.
		return err
	}
	service.agentNotifyMu.Lock()
	service.agentNotifiedOperations[operation.OperationID] = struct{}{}
	service.agentNotifyMu.Unlock()
	return nil
}

func (service *Service) markVerificationAttention(ctx context.Context, operation *domain.Operation, diagnostic string) (*domain.Operation, error) {
	return service.moveVerificationTerminal(ctx, operation, domain.StatusNeedsAttention, diagnostic)
}

func (service *Service) markVerificationRecovery(ctx context.Context, operation *domain.Operation, diagnostic string) (*domain.Operation, error) {
	return service.moveVerificationTerminal(ctx, operation, domain.StatusNeedsRecovery, diagnostic)
}

func (service *Service) moveVerificationTerminal(ctx context.Context, operation *domain.Operation, status domain.Status, diagnostic string) (*domain.Operation, error) {
	if operation == nil {
		return nil, fmt.Errorf("upgrade operation is required")
	}
	if operation.Status.IsTerminal() {
		return operation, nil
	}
	previous := operation.Status
	now := service.now().UTC()
	operation.Status = status
	operation.Diagnostic = sanitizeUpgradeDiagnostic(diagnostic)
	operation.CompletedAt = &now
	operation.UpdatedAt = now
	if operation.StageTimes == nil {
		operation.StageTimes = map[domain.Status]time.Time{}
	}
	operation.StageTimes[status] = now
	if _, err := service.persistReconciledOperation(ctx, operation, previous, false); err != nil {
		return nil, err
	}
	return operation, nil
}

func (service *Service) recordVerificationDiagnostic(ctx context.Context, operation *domain.Operation, diagnostic string) (*domain.Operation, error) {
	if operation == nil {
		return nil, fmt.Errorf("upgrade operation is required")
	}
	operation.Diagnostic = sanitizeUpgradeDiagnostic(diagnostic)
	operation.UpdatedAt = service.now().UTC()
	if err := service.repository.Update(ctx, operation); err != nil {
		return nil, err
	}
	return operation, nil
}

func allAgentsReady(expectations []domain.AgentExpectation) bool {
	for _, expectation := range expectations {
		if !expectation.Ready() {
			return false
		}
	}
	return true
}

func mergeObservedDigests(operation *domain.Operation, values map[string]string) {
	if operation == nil || len(values) == 0 {
		return
	}
	allowed := map[string]struct{}{
		"server": {}, "frontend": {}, "nginx": {}, "postgres": {}, "redis": {}, "loki": {}, "api": {}, "publicFrontend": {},
	}
	if operation.EffectiveExecutionMode() == domain.ExecutionModeFrontendOnly {
		// Scoped verification proves only the replacement frontend and its public
		// response. Do not let a verifier or future adapter widen the durable
		// evidence map with untouched service identities.
		allowed = map[string]struct{}{"frontend": {}}
	}
	if operation.ObservedDigests == nil {
		operation.ObservedDigests = map[string]string{}
	}
	for key, value := range values {
		if _, ok := allowed[key]; !ok || !upgradeObservedDigestPattern.MatchString(strings.TrimSpace(value)) {
			continue
		}
		operation.ObservedDigests[key] = strings.TrimSpace(value)
	}
}
