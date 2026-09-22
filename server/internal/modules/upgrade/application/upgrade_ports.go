package application

import (
	"context"
	"errors"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

// ErrHostUpgradeScopePlanningUnsupported means the host explicitly reported a
// compatible legacy/full-only protocol. It is intentionally distinct from a
// malformed v2 reply, which must fail before Server-side cancellation begins.
var ErrHostUpgradeScopePlanningUnsupported = errors.New("host upgrade scope planning is not supported")

// ErrHostUpgradeScopePlanningFullOnly means a v2-capable host cannot bind this
// candidate to composition evidence. The Server must retain the established v1
// full path instead of persisting a v2 full plan that can never establish the
// required confirmed deployment baseline.
var ErrHostUpgradeScopePlanningFullOnly = errors.New("host upgrade scope planning is full-only for this candidate")

// ErrHostDeploymentStateUnsupported means the host predates the v2 read-only
// confirmed-state projection. It is a compatibility signal, not evidence that
// the deployment is current.
var ErrHostDeploymentStateUnsupported = errors.New("host deployment state projection is not supported")

// ErrHostDeploymentStateUnavailable means no confirmed baseline has been
// established yet. Availability checks may use the running Server version in
// this case, while any malformed state remains a hard error.
var ErrHostDeploymentStateUnavailable = errors.New("host confirmed deployment state is unavailable")

// ErrHostCandidateAvailabilityUnsupported means the host does not implement
// the isolated schema-v3 inventory comparison. The Server may retain the
// established confirmed-state and running-version fallback in that case; a
// malformed advertised v3 response is deliberately not this compatibility case.
var ErrHostCandidateAvailabilityUnsupported = errors.New("host candidate availability projection is not supported")

// ErrHostDeploymentStateConflict means the candidate has the same release
// version as the confirmed deployment but a different manifest identity.
// Treating that combination as an update would make a mutable/rebuilt release
// silently replace an already confirmed version.
var ErrHostDeploymentStateConflict = errors.New("candidate conflicts with confirmed host deployment state")

// ManifestSource is the server-owned release inventory boundary. It has no
// request parameters so a browser cannot choose a manifest path or image.
type ManifestSource interface {
	Load() (*releasemanifest.Manifest, error)
}

// TargetManifestSource reloads immutable bytes for an already persisted
// operation. A mutable release-channel alias must never change a retry target.
type TargetManifestSource interface {
	ManifestSource
	LoadTarget(string) (*releasemanifest.Manifest, error)
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
	OperationID                string
	ManifestDigest             string
	Action                     HostUpgradeAction
	ExecutionMode              domain.ExecutionMode
	PlanDigest                 string
	BaselineDeploymentDigest   string
	TouchedServices            []string
	ConfirmedDeploymentVersion string
}

type HostUpgradeAction string

const (
	HostUpgradeActionStart  HostUpgradeAction = "start"
	HostUpgradeActionResume HostUpgradeAction = "resume"
	// Stop requests cancellation of the host execution context. It never
	// implies rollback of a migration or already-applied Compose changes.
	HostUpgradeActionStop HostUpgradeAction = "stop"
	// Repair is an explicit operator-confirmed retry of a terminal host
	// operation. Resume is reserved for non-terminal handoff recovery and must
	// not reset a terminal journal.
	HostUpgradeActionRepair HostUpgradeAction = "repair"
	// Confirm is issued only after Server-side service, migration and Agent
	// verification succeeds. It asks a v2 host to atomically record its own
	// confirmed deployment baseline; it is never an HTTP-selectable action.
	HostUpgradeActionConfirm HostUpgradeAction = "confirm"
)

// HostUpgradeDispatcher accepts a handoff and returns after the host process
// has accepted it. The host process owns the long-running Compose lifecycle.
type HostUpgradeDispatcher interface {
	Dispatch(context.Context, HostUpgradeRequest) error
}

// HostUpgradeScopePlanRequest is deliberately read-only. The host is the only
// process that can reconcile the candidate composition against the confirmed
// baseline and live Compose/Docker identities, so the Server supplies no
// client-selected scope or service list.
type HostUpgradeScopePlanRequest struct {
	OperationID    string
	ManifestDigest string
	// RequireFull is set only by Server-owned eligibility gates such as a
	// reviewed database migration. It can narrow a host decision to full but
	// can never be supplied by the browser or widen a frontend-only plan.
	RequireFull bool
}

// HostUpgradeScopePlan is the auditable result of host reconciliation. The
// Server persists it verbatim after strict validation and later returns its
// identity in the plan-bound start request; it never recalculates a service
// diff from release manifest fields.
type HostUpgradeScopePlan struct {
	ExecutionMode              domain.ExecutionMode
	PlanDigest                 string
	BaselineDeploymentDigest   string
	TouchedServices            []string
	ConfirmedDeploymentVersion string
}

// HostUpgradeScopePlanner is an optional v2 extension. A dispatcher that does
// not implement it is an explicitly legacy/full route; a dispatcher that does
// implement it but returns malformed data is a pre-side-effect failure.
type HostUpgradeScopePlanner interface {
	PlanUpgradeScope(context.Context, HostUpgradeScopePlanRequest) (HostUpgradeScopePlan, error)
}

// HostDeploymentState is the minimal host-owned projection needed for update
// availability. The complete component inventory remains private to the host;
// the manifest identity is enough to avoid re-offering an already applied
// frontend-only release while the Server binary is still older.
type HostDeploymentState struct {
	ReleaseVersion string
	ManifestDigest string
	StateDigest    string
}

type HostDeploymentStateSource interface {
	ConfirmedDeploymentState(context.Context) (HostDeploymentState, error)
}

// HostCandidateAvailabilityDecision is the bounded result of the host's
// complete candidate-versus-confirmed inventory comparison. The Server never
// receives the inventory itself and cannot derive this decision from a manifest.
type HostCandidateAvailabilityDecision string

const (
	HostCandidateAvailabilityAvailable      HostCandidateAvailabilityDecision = "available"
	HostCandidateAvailabilityAlreadyApplied HostCandidateAvailabilityDecision = "already_applied"
	HostCandidateAvailabilityNotNewer       HostCandidateAvailabilityDecision = "not_newer"
	HostCandidateAvailabilityFallback       HostCandidateAvailabilityDecision = "fallback"
)

// HostCandidateAvailability contains only the confirmed evidence required to
// render update state. Fallback has no evidence and preserves the legacy
// running-Server-version behavior.
type HostCandidateAvailability struct {
	Decision                   HostCandidateAvailabilityDecision
	BaselineDeploymentDigest   string
	ConfirmedDeploymentVersion string
}

// HostCandidateAvailabilitySource is an optional schema-v3 extension. It is
// intentionally separate from planning and dispatch so update checks cannot
// create an Operation, reserve the deployment lock, or mutate Compose state.
type HostCandidateAvailabilitySource interface {
	CandidateAvailability(context.Context, string) (HostCandidateAvailability, error)
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
