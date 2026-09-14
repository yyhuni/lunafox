package application

import (
	"context"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

// ManifestSource is the server-owned release inventory boundary. It has no
// request parameters so a browser cannot choose a manifest path or image.
type ManifestSource interface {
	Load() (*releasemanifest.Manifest, error)
}

// MigrationPolicySource reads the policy selected by the deployment. The
// policy is evaluated again for every create/retry operation.
type MigrationPolicySource interface {
	Load(context.Context) (domain.MigrationPolicy, error)
}

// ActiveSuperuserAuthorizer resolves the current database-backed authority;
// JWT usernames and client role fields are intentionally not accepted here.
type ActiveSuperuserAuthorizer interface {
	IsActiveSuperuser(context.Context, int) (bool, error)
}

// HostUpgradeRequest is the complete server-to-host handoff contract. Keep
// this type deliberately narrow: paths, image refs and shell commands belong
// to the independently managed host upgrader, never to an HTTP request.
type HostUpgradeRequest struct {
	OperationID    string
	ManifestDigest string
	Action         HostUpgradeAction
}

type HostUpgradeAction string

const (
	HostUpgradeActionStart  HostUpgradeAction = "start"
	HostUpgradeActionResume HostUpgradeAction = "resume"
	// Repair is an explicit operator-confirmed retry of a terminal host
	// operation. Resume is reserved for non-terminal handoff recovery and must
	// not reset a terminal journal.
	HostUpgradeActionRepair HostUpgradeAction = "repair"
)

// HostUpgradeDispatcher accepts a handoff and returns after the host process
// has accepted it. The host process owns the long-running Compose lifecycle.
type HostUpgradeDispatcher interface {
	Dispatch(context.Context, HostUpgradeRequest) error
}

// AgentUpgradeTarget is the immutable Agent release projection used by the
// existing update_required control-plane event. ImageRef is resolved from the
// server-owned Manifest and never comes from an HTTP request.
type AgentUpgradeTarget struct {
	Version  string
	Digest   string
	ImageRef string
}

// AgentUpgradeSource owns the narrow read/notify boundary for Agent upgrade
// reconciliation. Snapshot must be read-only: it may inspect persisted
// heartbeat evidence but must not claim work or mutate task state.
type AgentUpgradeSource interface {
	Snapshot(context.Context, AgentUpgradeTarget) ([]domain.AgentExpectation, error)
	NotifyUpdateRequired(context.Context, AgentUpgradeTarget, []domain.AgentExpectation) error
}

// AgentUpgradeReconciler is an optional read-only extension implemented by
// production sources. It refreshes only the Agent IDs captured at operation
// creation; newly registered Agents cannot silently expand an in-flight
// operation's scope.
type AgentUpgradeReconciler interface {
	Reconcile(context.Context, AgentUpgradeTarget, []domain.AgentExpectation) ([]domain.AgentExpectation, error)
}

// VerificationEvidence is the cross-process evidence collected by the host
// journal and the Server-side health verifier. Receipt data remains
// supplemental and cannot be interpreted as a successful Operation alone.
type VerificationEvidence struct {
	ObservedDigests map[string]string
	ExpectedDigests map[string]string
}

type VerificationResult struct {
	Passed          bool
	ObservedDigests map[string]string
	Diagnostic      string
}

// UpgradeVerifier checks the final service, dependency, API and public-edge
// evidence. Keeping it behind a port prevents the upgrade state machine from
// depending on HTTP clients, Redis, Loki, or Compose implementations.
type UpgradeVerifier interface {
	Verify(context.Context, *domain.Operation, VerificationEvidence) (VerificationResult, error)
}

// RequestLookupRepository is an optional fast path for idempotent request
// replays. It lets the application return the already persisted Operation
// before loading a newer manifest from disk; a retry/replay must never be
// mistaken for permission to switch release targets.
type RequestLookupRepository interface {
	GetByRequest(context.Context, string) (*domain.Operation, error)
}

// PreparationResult records work cancellation committed before host handoff.
type PreparationResult struct {
	CancelledScanCount int
	CancelledTaskCount int
}

// PreDispatchCoordinator pauses new work and commits cancellation of active
// scans/tasks before the independent host upgrader is dispatched.
type PreDispatchCoordinator interface {
	Prepare(context.Context) (PreparationResult, error)
}

// UpgradePreDispatchCoordinator is the operation-aware extension used by the
// production coordinator. The legacy Prepare method remains available for
// narrow callers/tests, while this method lets cancellation records carry the
// durable operation identity.
type UpgradePreDispatchCoordinator interface {
	PreDispatchCoordinator
	PrepareForUpgrade(context.Context, string) (PreparationResult, error)
}

// MigrationPolicyFunc adapts the existing file loader and deterministic test
// policies without making application code depend on filesystem details.
type MigrationPolicyFunc func(context.Context) (domain.MigrationPolicy, error)

func (function MigrationPolicyFunc) Load(ctx context.Context) (domain.MigrationPolicy, error) {
	if function == nil {
		return domain.MigrationPolicy{}, domain.WrapMigrationMetadataMissing(nil)
	}
	return function(ctx)
}

// StaticMigrationPolicySource is useful for bootstrap wiring and contract
// tests where the policy has already been loaded and validated.
type StaticMigrationPolicySource struct {
	Policy domain.MigrationPolicy
}

func (source StaticMigrationPolicySource) Load(context.Context) (domain.MigrationPolicy, error) {
	return source.Policy, nil
}

// ManifestSourceFunc adapts a fixed manifest loader in tests or bootstrap.
type ManifestSourceFunc func() (*releasemanifest.Manifest, error)

func (function ManifestSourceFunc) Load() (*releasemanifest.Manifest, error) {
	if function == nil {
		return nil, domain.WrapManifestInvalid(nil)
	}
	return function()
}
