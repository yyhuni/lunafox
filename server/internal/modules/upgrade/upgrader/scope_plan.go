package upgrader

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/releasemanifest"
)

const (
	runtimeServerComponent    = "runtime.server"
	runtimeFrontendComponent  = "runtime.frontend"
	runtimeNginxComponent     = "runtime.nginx"
	runtimeAgentComponent     = "runtime.agent"
	runtimeBootstrapComponent = "runtime.bootstrap"
)

var (
	componentIDPattern = regexp.MustCompile(`^(?:runtime\.(?:server|frontend|nginx|agent|bootstrap)|engine\.[a-z0-9][a-z0-9._-]*)$`)
	// Runtime observations may contain the complete confirmed deployment
	// inventory.  Engine entries are represented as runtime/package pairs, but
	// the observer still rejects every other arbitrary component id.
	runtimeObservationComponentIDPattern = regexp.MustCompile(`^engine\.[a-z0-9][a-z0-9._-]*\.(?:runtime|package)$`)
	fullScopeServices                    = []string{"agent", "bootstrap", "engine", "engine_package", "engine_runtime", "frontend", "migration", "nginx", "server"}
	residentRuntimeIDs                   = []string{runtimeServerComponent, runtimeFrontendComponent, runtimeNginxComponent, runtimeAgentComponent}
)

// ErrCandidateReleaseOlderThanConfirmed prevents a stale release from being
// represented as either a selective or a full plan after the host has already
// confirmed a newer deployment. A full plan is not a downgrade escape hatch.
var ErrCandidateReleaseOlderThanConfirmed = errors.New("candidate release is older than confirmed deployment")

// ErrCandidateReleaseConflictsWithConfirmed prevents a same-version candidate
// with a different manifest identity from being interpreted as a new selective
// deployment. The release identity is immutable at this boundary.
var ErrCandidateReleaseConflictsWithConfirmed = errors.New("candidate release conflicts with confirmed deployment")

// ExecutionMode is selected by the host from its own deployment observation.
// It is durable evidence, not an operator-provided switch.
type ExecutionMode string

const (
	ExecutionModeFull         ExecutionMode = "full"
	ExecutionModeFrontendOnly ExecutionMode = "frontend_only"
)

func (mode ExecutionMode) Valid() bool {
	return mode == ExecutionModeFull || mode == ExecutionModeFrontendOnly
}

// HostCapabilities is the explicit versioned negotiation response. A client
// may use frontend-only only when all fields required by SupportsV2Planning
// are true; missing fields are never treated as a compatibility fallback.
type HostCapabilities struct {
	SchemaVersions          []int `json:"schemaVersions"`
	ScopePlanning           bool  `json:"scopePlanning"`
	PlanBoundStart          bool  `json:"planBoundStart"`
	DeploymentConfirmation  bool  `json:"deploymentConfirmation"`
	DynamicFrontendUpstream bool  `json:"dynamicFrontendUpstream"`
	// CandidateInventoryAvailability is intentionally emitted only in a v3
	// capability response. Older Servers strictly reject unknown JSON fields.
	CandidateInventoryAvailability bool `json:"candidateInventoryAvailability,omitempty"`
}

func DefaultHostCapabilities() HostCapabilities {
	return HostCapabilities{
		SchemaVersions:                 []int{RequestSchema, ScopedRequestSchema, RequestSchemaV3},
		ScopePlanning:                  true,
		PlanBoundStart:                 true,
		DeploymentConfirmation:         true,
		DynamicFrontendUpstream:        true,
		CandidateInventoryAvailability: true,
	}
}

func (capabilities HostCapabilities) Validate() error {
	if len(capabilities.SchemaVersions) == 0 {
		return fmt.Errorf("host capability schemaVersions are required")
	}
	previous := 0
	for _, schema := range capabilities.SchemaVersions {
		if !validRequestSchema(schema) {
			return fmt.Errorf("host capability schema version %d is unsupported", schema)
		}
		if schema <= previous {
			return fmt.Errorf("host capability schemaVersions must be sorted and unique")
		}
		previous = schema
	}
	if capabilities.CandidateInventoryAvailability && !containsSchema(capabilities.SchemaVersions, RequestSchemaV3) {
		return fmt.Errorf("candidate inventory availability requires schema-v3 support")
	}
	return nil
}

func (capabilities HostCapabilities) SupportsV2Planning() bool {
	if capabilities.Validate() != nil {
		return false
	}
	return containsSchema(capabilities.SchemaVersions, ScopedRequestSchema) &&
		capabilities.ScopePlanning &&
		capabilities.PlanBoundStart &&
		capabilities.DeploymentConfirmation &&
		capabilities.DynamicFrontendUpstream
}

// SupportsV3CandidateInventoryAvailability verifies support for the isolated
// read-only availability protocol. It is intentionally independent from plan
// execution: a caller must never infer selective execution from this result.
func (capabilities HostCapabilities) SupportsV3CandidateInventoryAvailability() bool {
	if capabilities.Validate() != nil {
		return false
	}
	return containsSchema(capabilities.SchemaVersions, RequestSchemaV3) && capabilities.CandidateInventoryAvailability
}

func projectHostCapabilitiesForSchema(capabilities HostCapabilities, schemaVersion int) HostCapabilities {
	projected := cloneHostCapabilities(capabilities)
	if schemaVersion == RequestSchemaV3 {
		return projected
	}
	// Preserve the exact v1/v2 JSON shape: old Server decoders use
	// DisallowUnknownFields, so even a false v3 field would break gradual rollout.
	projected.CandidateInventoryAvailability = false
	projected.SchemaVersions = projected.SchemaVersions[:0]
	for _, schema := range capabilities.SchemaVersions {
		if schema <= ScopedRequestSchema {
			projected.SchemaVersions = append(projected.SchemaVersions, schema)
		}
	}
	return projected
}

// DeploymentCapabilities are release/deployment facts, not host binary
// capabilities. The first release that introduces dynamic frontend resolution
// changes nginx and therefore cannot qualify as frontend-only; only a
// confirmed baseline reporting this capability can do so.
type DeploymentCapabilities struct {
	DynamicFrontendUpstream bool `json:"dynamicFrontendUpstream"`
}

// DeploymentComponent is a stable deployment inventory entry. Its ID comes
// from trusted release composition; its digest is used only to bind immutable
// runtime identity, never to infer source-code change by itself.
type DeploymentComponent struct {
	ID     string `json:"id"`
	Digest string `json:"digest"`
}

func (component DeploymentComponent) Validate() error {
	if component.ID != strings.TrimSpace(component.ID) || !componentIDPattern.MatchString(component.ID) {
		return fmt.Errorf("deployment component id %q is not canonical", component.ID)
	}
	if err := validateDigest(component.Digest); err != nil {
		return fmt.Errorf("deployment component %s: %w", component.ID, err)
	}
	return nil
}

// CandidateDeployment is the host-trusted projection of one immutable release
// composition. CompositionDigest is optional for full compatibility fallback,
// but mandatory before the planner can produce frontend_only.
type CandidateDeployment struct {
	ReleaseVersion    string                 `json:"releaseVersion"`
	CompositionDigest string                 `json:"compositionDigest,omitempty"`
	Components        []DeploymentComponent  `json:"components"`
	Capabilities      DeploymentCapabilities `json:"capabilities"`
}

func (candidate CandidateDeployment) Validate() error {
	if candidate.ReleaseVersion != strings.TrimSpace(candidate.ReleaseVersion) || candidate.ReleaseVersion == "" {
		return fmt.Errorf("candidate deployment release version is required")
	}
	if candidate.CompositionDigest != "" {
		if err := validateDigest(candidate.CompositionDigest); err != nil {
			return fmt.Errorf("candidate composition digest: %w", err)
		}
	}
	return validateDeploymentComponents(candidate.Components)
}

func (candidate CandidateDeployment) componentDigest(componentID string) string {
	for _, component := range candidate.Components {
		if component.ID == componentID {
			return component.Digest
		}
	}
	return ""
}

// RuntimeObservation must come from Docker container inspection, not from
// image inspect of a target reference. The observer is deliberately injected
// so host scope logic can be tested without a Docker daemon and cannot mistake
// a locally pulled image for the running container identity.
type RuntimeObservation struct {
	Images            map[string]string `json:"images"`
	NginxHealthy      bool              `json:"nginxHealthy"`
	NginxConfigDigest string            `json:"nginxConfigDigest"`
	ObservedAt        time.Time         `json:"observedAt"`
}

func (observation RuntimeObservation) Validate() error {
	if observation.ObservedAt.IsZero() {
		return fmt.Errorf("runtime observation timestamp is required")
	}
	if len(observation.Images) < len(residentRuntimeIDs) {
		return fmt.Errorf("runtime observation must contain every resident runtime component")
	}
	for _, componentID := range residentRuntimeIDs {
		digest, exists := observation.Images[componentID]
		if !exists {
			return fmt.Errorf("runtime observation is missing %s", componentID)
		}
		if err := validateDigest(digest); err != nil {
			return fmt.Errorf("runtime observation %s: %w", componentID, err)
		}
	}
	for componentID := range observation.Images {
		if !isRuntimeObservationComponent(componentID) {
			return fmt.Errorf("runtime observation contains unsupported component %q", componentID)
		}
		if err := validateDigest(observation.Images[componentID]); err != nil {
			return fmt.Errorf("runtime observation %s: %w", componentID, err)
		}
	}
	if !observation.NginxHealthy {
		return fmt.Errorf("runtime observation nginx is not healthy")
	}
	if err := validateDigest(observation.NginxConfigDigest); err != nil {
		return fmt.Errorf("runtime observation nginx configuration: %w", err)
	}
	return nil
}

type RuntimeObserver interface {
	ObserveRuntime(context.Context) (RuntimeObservation, error)
}

type RuntimeObserverFunc func(context.Context) (RuntimeObservation, error)

func (function RuntimeObserverFunc) ObserveRuntime(ctx context.Context) (RuntimeObservation, error) {
	if function == nil {
		return RuntimeObservation{}, fmt.Errorf("runtime observer is not configured")
	}
	return function(ctx)
}

// CandidateDeploymentSource resolves the release-owned inventory from a fixed
// host source. It must not accept paths, refs, or component lists from the
// socket request.
type CandidateDeploymentSource interface {
	LoadCandidateDeployment(context.Context, string) (CandidateDeployment, error)
}

type CandidateDeploymentSourceFunc func(context.Context, string) (CandidateDeployment, error)

func (function CandidateDeploymentSourceFunc) LoadCandidateDeployment(ctx context.Context, manifestDigest string) (CandidateDeployment, error) {
	if function == nil {
		return CandidateDeployment{}, fmt.Errorf("candidate deployment source is not configured")
	}
	return function(ctx, manifestDigest)
}

// ManifestCandidateDeploymentSource provides the legacy-compatible full
// fallback source. It deliberately leaves CompositionDigest empty: a manifest
// alone cannot prove the source-level component diff needed for frontend-only.
type ManifestCandidateDeploymentSource struct {
	store *JournalStore
}

func NewManifestCandidateDeploymentSource(store *JournalStore) *ManifestCandidateDeploymentSource {
	return &ManifestCandidateDeploymentSource{store: store}
}

func (source *ManifestCandidateDeploymentSource) LoadCandidateDeployment(_ context.Context, manifestDigest string) (CandidateDeployment, error) {
	if source == nil || source.store == nil {
		return CandidateDeployment{}, fmt.Errorf("candidate deployment source is not configured")
	}
	path, err := source.store.ManifestPath(manifestDigest)
	if err != nil {
		return CandidateDeployment{}, err
	}
	manifest, err := loadManifestWithLegacyCompatibility(path)
	if err != nil {
		return CandidateDeployment{}, fmt.Errorf("load candidate release manifest: %w", err)
	}
	if manifest.Digest() != manifestDigest {
		return CandidateDeployment{}, ErrManifestMismatch
	}
	return CandidateDeploymentFromManifest(manifest)
}

func CandidateDeploymentFromManifest(manifest *releasemanifest.Manifest) (CandidateDeployment, error) {
	if manifest == nil {
		return CandidateDeployment{}, fmt.Errorf("release manifest is required")
	}
	components := make([]DeploymentComponent, 0, len(manifest.RuntimeImages)+len(manifest.EnginePackages))
	for _, runtimeName := range []string{"server", "frontend", "nginx", "agent", "bootstrap"} {
		digest, err := manifest.RuntimeImageDigest(runtimeName)
		if err != nil {
			return CandidateDeployment{}, err
		}
		components = append(components, DeploymentComponent{ID: "runtime." + runtimeName, Digest: digest})
	}
	for index, enginePackage := range manifest.EnginePackages {
		if len(enginePackage.Refs) == 0 {
			return CandidateDeployment{}, fmt.Errorf("engine package %d has no immutable reference", index)
		}
		ref, err := ociartifact.ParseDigestReference(enginePackage.Refs[0])
		if err != nil {
			return CandidateDeployment{}, fmt.Errorf("parse engine package %d: %w", index, err)
		}
		components = append(components, DeploymentComponent{ID: fmt.Sprintf("engine.package.%03d", index), Digest: ref.Digest})
	}
	sortDeploymentComponents(components)
	candidate := CandidateDeployment{ReleaseVersion: manifest.ReleaseVersion, Components: components}
	if err := candidate.Validate(); err != nil {
		return CandidateDeployment{}, err
	}
	return candidate, nil
}

// ScopePlan binds a single candidate, confirmed baseline, and live runtime
// observation. PlanDigest deliberately excludes GeneratedAt so host can repeat
// the same observation under the deployment lock and compare the exact plan.
type ScopePlan struct {
	SchemaVersion              int                 `json:"schemaVersion"`
	OperationID                string              `json:"operationId"`
	ManifestDigest             string              `json:"manifestDigest"`
	ExecutionMode              ExecutionMode       `json:"executionMode"`
	PlanDigest                 string              `json:"planDigest"`
	BaselineStateDigest        string              `json:"baselineStateDigest,omitempty"`
	TouchedServices            []string            `json:"touchedServices"`
	ConfirmedDeploymentVersion string              `json:"confirmedDeploymentVersion,omitempty"`
	Candidate                  CandidateDeployment `json:"candidate"`
	ObservedRuntimeImages      map[string]string   `json:"observedRuntimeImages,omitempty"`
	ObservedNginxHealthy       bool                `json:"observedNginxHealthy,omitempty"`
	ObservedNginxConfigDigest  string              `json:"observedNginxConfigDigest,omitempty"`
	GeneratedAt                time.Time           `json:"generatedAt"`
	Diagnostic                 string              `json:"diagnostic,omitempty"`
}

func (plan ScopePlan) Validate() error {
	if plan.SchemaVersion != ScopePlanSchema {
		return fmt.Errorf("unsupported scope plan schema version %d", plan.SchemaVersion)
	}
	if err := validateOperationID(plan.OperationID); err != nil {
		return err
	}
	if err := validateDigest(plan.ManifestDigest); err != nil {
		return err
	}
	if !plan.ExecutionMode.Valid() {
		return fmt.Errorf("scope plan execution mode is required")
	}
	if err := plan.Candidate.Validate(); err != nil {
		return fmt.Errorf("scope plan candidate: %w", err)
	}
	if err := validateTouchedServices(plan.ExecutionMode, plan.TouchedServices); err != nil {
		return err
	}
	if plan.GeneratedAt.IsZero() {
		return fmt.Errorf("scope plan generatedAt is required")
	}
	if err := ValidateDiagnostic(plan.Diagnostic); err != nil {
		return err
	}
	if plan.ExecutionMode == ExecutionModeFrontendOnly {
		if err := validateDigest(plan.BaselineStateDigest); err != nil {
			return fmt.Errorf("scope plan baseline state digest: %w", err)
		}
		if plan.ConfirmedDeploymentVersion == "" || plan.ConfirmedDeploymentVersion != strings.TrimSpace(plan.ConfirmedDeploymentVersion) {
			return fmt.Errorf("scope plan confirmed deployment version is required")
		}
		if err := validateDigest(plan.Candidate.CompositionDigest); err != nil {
			return fmt.Errorf("frontend-only scope plan composition digest: %w", err)
		}
		observation := RuntimeObservation{
			Images:            plan.ObservedRuntimeImages,
			NginxHealthy:      plan.ObservedNginxHealthy,
			NginxConfigDigest: plan.ObservedNginxConfigDigest,
			ObservedAt:        plan.GeneratedAt,
		}
		if err := observation.Validate(); err != nil {
			return fmt.Errorf("frontend-only scope plan observation: %w", err)
		}
	} else if plan.BaselineStateDigest != "" || plan.ConfirmedDeploymentVersion != "" || len(plan.ObservedRuntimeImages) != 0 || plan.ObservedNginxHealthy || plan.ObservedNginxConfigDigest != "" {
		return fmt.Errorf("full scope plan must not contain selective baseline evidence")
	}
	if err := validateDigest(plan.PlanDigest); err != nil {
		return fmt.Errorf("scope plan digest: %w", err)
	}
	derived, err := plan.derivedDigest()
	if err != nil {
		return err
	}
	if plan.PlanDigest != derived {
		return fmt.Errorf("scope plan digest does not match canonical plan")
	}
	return nil
}

func (plan ScopePlan) derivedDigest() (string, error) {
	payload := struct {
		SchemaVersion              int                 `json:"schemaVersion"`
		OperationID                string              `json:"operationId"`
		ManifestDigest             string              `json:"manifestDigest"`
		ExecutionMode              ExecutionMode       `json:"executionMode"`
		BaselineStateDigest        string              `json:"baselineStateDigest,omitempty"`
		TouchedServices            []string            `json:"touchedServices"`
		ConfirmedDeploymentVersion string              `json:"confirmedDeploymentVersion,omitempty"`
		Candidate                  CandidateDeployment `json:"candidate"`
		ObservedRuntimeImages      map[string]string   `json:"observedRuntimeImages,omitempty"`
		ObservedNginxHealthy       bool                `json:"observedNginxHealthy,omitempty"`
		ObservedNginxConfigDigest  string              `json:"observedNginxConfigDigest,omitempty"`
		Diagnostic                 string              `json:"diagnostic,omitempty"`
	}{
		SchemaVersion:              plan.SchemaVersion,
		OperationID:                plan.OperationID,
		ManifestDigest:             plan.ManifestDigest,
		ExecutionMode:              plan.ExecutionMode,
		BaselineStateDigest:        plan.BaselineStateDigest,
		TouchedServices:            append([]string(nil), plan.TouchedServices...),
		ConfirmedDeploymentVersion: plan.ConfirmedDeploymentVersion,
		Candidate:                  cloneCandidateDeployment(plan.Candidate),
		ObservedRuntimeImages:      cloneStringMap(plan.ObservedRuntimeImages),
		ObservedNginxHealthy:       plan.ObservedNginxHealthy,
		ObservedNginxConfigDigest:  plan.ObservedNginxConfigDigest,
		Diagnostic:                 plan.Diagnostic,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode canonical scope plan: %w", err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum[:]), nil
}

// ScopePlanner is the host-side planning contract. It deliberately returns a
// full plan for incomplete facts but returns an error for corrupt protected
// state: unknown data can preserve old full behavior, damaged state cannot.
type ScopePlanner interface {
	Capabilities() HostCapabilities
	Plan(context.Context, string, string, bool) (ScopePlan, error)
	Revalidate(context.Context, ScopePlan) error
}

// CandidateAvailabilityPlanner is deliberately separate from ScopePlanner's
// persisted-plan API. Its result may decide whether to show an update, but it
// cannot create an Operation, reserve the deployment lock, or mutate Compose.
type CandidateAvailabilityPlanner interface {
	CandidateAvailability(context.Context, string) (CandidateAvailability, error)
}

// FullDeploymentConfirmation is implemented by the host planner that owns the
// Docker observer. A full v2 confirmation must reread the live deployment under
// the deployment lock; the persisted candidate and receipt are not substitutes
// for bootstrap/Engine inventory evidence.
type FullDeploymentConfirmation interface {
	ObserveFullDeployment(context.Context, ScopePlan) (RuntimeObservation, error)
}

type HostScopePlanner struct {
	store        *JournalStore
	candidate    CandidateDeploymentSource
	observer     RuntimeObserver
	capabilities HostCapabilities
	now          func() time.Time
}

func NewHostScopePlanner(store *JournalStore, candidate CandidateDeploymentSource, observer RuntimeObserver) *HostScopePlanner {
	if candidate == nil {
		candidate = NewManifestCandidateDeploymentSource(store)
	}
	return &HostScopePlanner{
		store:        store,
		candidate:    candidate,
		observer:     observer,
		capabilities: DefaultHostCapabilities(),
		now:          func() time.Time { return time.Now().UTC() },
	}
}

func (planner *HostScopePlanner) Capabilities() HostCapabilities {
	if planner == nil {
		return HostCapabilities{}
	}
	capabilities := cloneHostCapabilities(planner.capabilities)
	// Complete inventory comparison additionally requires immutable manifests
	// for both candidate and confirmed state. Development/legacy stores have one
	// mutable manifest path and must retain the semantic-version fallback.
	if planner.store == nil || !planner.store.manifestCache {
		capabilities.CandidateInventoryAvailability = false
	}
	return capabilities
}

var ErrCandidateAvailabilityFallback = errors.New("candidate inventory availability is unavailable")

// CandidateAvailability compares a composition-bound candidate with the
// complete confirmed deployment inventory without creating a scope plan. A
// missing composition, baseline, or immutable baseline manifest is explicitly
// returned as fallback; corrupt evidence remains a hard error.
func (planner *HostScopePlanner) CandidateAvailability(ctx context.Context, manifestDigest string) (CandidateAvailability, error) {
	if planner == nil || planner.store == nil || planner.candidate == nil {
		return CandidateAvailability{}, fmt.Errorf("candidate availability planner is not configured")
	}
	if err := validateDigest(manifestDigest); err != nil {
		return CandidateAvailability{}, err
	}
	fallback := CandidateAvailability{
		ManifestDigest: manifestDigest,
		Decision:       CandidateAvailabilityFallback,
	}
	if !planner.Capabilities().SupportsV3CandidateInventoryAvailability() {
		return fallback, nil
	}
	candidate, err := planner.candidate.LoadCandidateDeployment(ctx, manifestDigest)
	if err != nil {
		// A digest-addressed candidate manifest can legitimately be absent until
		// the Server has refreshed its immutable cache. That is incomplete
		// availability evidence, not corruption; malformed bytes still fail closed.
		if errors.Is(err, ErrCompositionUnavailable) || errors.Is(err, os.ErrNotExist) {
			return fallback, nil
		}
		return CandidateAvailability{}, fmt.Errorf("load candidate deployment for availability: %w", err)
	}
	if err := candidate.Validate(); err != nil {
		return CandidateAvailability{}, fmt.Errorf("validate candidate deployment for availability: %w", err)
	}
	if err := validateDigest(candidate.CompositionDigest); err != nil {
		return fallback, nil
	}
	state, err := planner.store.LoadConfirmedDeploymentState()
	if err != nil {
		if errors.Is(err, ErrConfirmedStateNotFound) {
			return fallback, nil
		}
		return CandidateAvailability{}, fmt.Errorf("load confirmed deployment state for availability: %w", err)
	}
	candidateVersion, err := semver.Parse(candidate.ReleaseVersion)
	if err != nil {
		return CandidateAvailability{}, fmt.Errorf("candidate deployment release version is invalid: %w", err)
	}
	confirmedVersion, err := semver.Parse(state.ReleaseVersion)
	if err != nil {
		return CandidateAvailability{}, fmt.Errorf("confirmed deployment release version is invalid: %w", err)
	}
	result := func(decision CandidateAvailabilityDecision) CandidateAvailability {
		return CandidateAvailability{
			ManifestDigest:             manifestDigest,
			Decision:                   decision,
			BaselineStateDigest:        state.StateDigest,
			ConfirmedDeploymentVersion: state.ReleaseVersion,
		}
	}
	if candidateVersion.LT(confirmedVersion) {
		return result(CandidateAvailabilityNotNewer), nil
	}
	if candidateVersion.EQ(confirmedVersion) {
		if manifestDigest != state.ManifestDigest {
			return result(CandidateAvailabilityConflict), nil
		}
		return result(CandidateAvailabilityAlreadyApplied), nil
	}
	if !sameDeploymentComponents(state.Components, candidate.Components) || state.Capabilities != candidate.Capabilities {
		return result(CandidateAvailabilityAvailable), nil
	}
	confirmedMigration, err := planner.availabilityMigrationIdentity(state.ManifestDigest, state.CompositionDigest)
	if err != nil {
		if errors.Is(err, ErrCandidateAvailabilityFallback) {
			return fallback, nil
		}
		return CandidateAvailability{}, err
	}
	candidateMigration, err := planner.availabilityMigrationIdentity(manifestDigest, candidate.CompositionDigest)
	if err != nil {
		if errors.Is(err, ErrCandidateAvailabilityFallback) {
			return fallback, nil
		}
		return CandidateAvailability{}, err
	}
	if confirmedMigration != candidateMigration {
		return result(CandidateAvailabilityAvailable), nil
	}
	return result(CandidateAvailabilityAlreadyApplied), nil
}

type deploymentMigrationIdentity struct {
	HasDatabaseMigration bool
	MigrationType        string
	MigrationID          string
	Checksum             string
	PolicyVersion        int
}

func (planner *HostScopePlanner) availabilityMigrationIdentity(manifestDigest, compositionDigest string) (deploymentMigrationIdentity, error) {
	if planner == nil || planner.store == nil || !planner.store.manifestCache {
		return deploymentMigrationIdentity{}, ErrCandidateAvailabilityFallback
	}
	path, err := planner.store.ManifestPath(manifestDigest)
	if err != nil {
		return deploymentMigrationIdentity{}, fmt.Errorf("resolve availability manifest: %w", err)
	}
	manifest, err := loadManifestWithLegacyCompatibility(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return deploymentMigrationIdentity{}, ErrCandidateAvailabilityFallback
		}
		return deploymentMigrationIdentity{}, fmt.Errorf("load availability manifest: %w", err)
	}
	if manifest.Digest() != manifestDigest {
		return deploymentMigrationIdentity{}, ErrManifestMismatch
	}
	if manifest.RuntimeComposition.SHA256 != compositionDigest {
		return deploymentMigrationIdentity{}, fmt.Errorf("availability manifest composition does not match confirmed deployment evidence")
	}
	migration := manifest.Upgrade.DatabaseMigration
	return deploymentMigrationIdentity{
		HasDatabaseMigration: migration.HasDatabaseMigration,
		MigrationType:        migration.MigrationType,
		MigrationID:          migration.MigrationID,
		Checksum:             migration.Checksum,
		PolicyVersion:        migration.PolicyVersion,
	}, nil
}

func (planner *HostScopePlanner) Plan(ctx context.Context, operationID, manifestDigest string, requireFull bool) (ScopePlan, error) {
	if planner == nil || planner.store == nil || planner.candidate == nil {
		return ScopePlan{}, fmt.Errorf("host scope planner is not configured")
	}
	if err := validateOperationID(operationID); err != nil {
		return ScopePlan{}, err
	}
	if err := validateDigest(manifestDigest); err != nil {
		return ScopePlan{}, err
	}
	candidate, err := planner.candidate.LoadCandidateDeployment(ctx, manifestDigest)
	if err != nil {
		// A missing digest-addressed composition cache is an incomplete fast-path
		// fact, not permission to reject an otherwise valid full upgrade. Reload
		// the same immutable manifest through the fixed fallback source so the
		// resulting plan has no composition evidence and therefore cannot become
		// frontend_only. Malformed/tampered composition evidence remains a hard
		// error and must never be silently downgraded.
		if errors.Is(err, ErrCompositionUnavailable) {
			fallback := NewManifestCandidateDeploymentSource(planner.store)
			candidate, fallbackErr := fallback.LoadCandidateDeployment(ctx, manifestDigest)
			if fallbackErr != nil {
				return ScopePlan{}, fmt.Errorf("load candidate deployment after composition fallback: %w", fallbackErr)
			}
			return planner.planFromCandidate(ctx, operationID, manifestDigest, candidate, requireFull)
		}
		return ScopePlan{}, fmt.Errorf("load candidate deployment: %w", err)
	}
	if err := candidate.Validate(); err != nil {
		return ScopePlan{}, fmt.Errorf("validate candidate deployment: %w", err)
	}
	return planner.planFromCandidate(ctx, operationID, manifestDigest, candidate, requireFull)
}

func (planner *HostScopePlanner) planFromCandidate(ctx context.Context, operationID, manifestDigest string, candidate CandidateDeployment, requireFull bool) (ScopePlan, error) {
	full := func(diagnostic string) (ScopePlan, error) {
		plan := ScopePlan{
			SchemaVersion:   ScopePlanSchema,
			OperationID:     operationID,
			ManifestDigest:  manifestDigest,
			ExecutionMode:   ExecutionModeFull,
			TouchedServices: fullTouchedServices(),
			Candidate:       cloneCandidateDeployment(candidate),
			GeneratedAt:     planner.now().UTC(),
			Diagnostic:      diagnostic,
		}
		digest, err := plan.derivedDigest()
		if err != nil {
			return ScopePlan{}, err
		}
		plan.PlanDigest = digest
		return plan, plan.Validate()
	}
	candidateVersion, err := semver.Parse(candidate.ReleaseVersion)
	if err != nil {
		return ScopePlan{}, fmt.Errorf("candidate deployment release version is invalid: %w", err)
	}
	state, stateErr := planner.store.LoadConfirmedDeploymentState()
	if stateErr != nil && !errors.Is(stateErr, ErrConfirmedStateNotFound) {
		return ScopePlan{}, fmt.Errorf("load confirmed deployment state: %w", stateErr)
	}
	if stateErr == nil {
		confirmedVersion, parseErr := semver.Parse(state.ReleaseVersion)
		if parseErr != nil {
			return ScopePlan{}, fmt.Errorf("confirmed deployment release version is invalid: %w", parseErr)
		}
		if candidateVersion.LT(confirmedVersion) {
			return ScopePlan{}, fmt.Errorf("%w: candidate %s is older than confirmed %s", ErrCandidateReleaseOlderThanConfirmed, candidate.ReleaseVersion, state.ReleaseVersion)
		}
		if candidateVersion.EQ(confirmedVersion) {
			if manifestDigest != state.ManifestDigest {
				return ScopePlan{}, fmt.Errorf("%w: candidate manifest %s differs from confirmed manifest %s", ErrCandidateReleaseConflictsWithConfirmed, manifestDigest, state.ManifestDigest)
			}
			// Equal release identities are already confirmed. Even if an
			// inconsistent candidate source claims a frontend-only component
			// difference, never grant selective execution for that identity.
			return full("candidate release is already confirmed")
		}
	}
	if requireFull {
		return full("full scope is required by a Server-owned eligibility gate")
	}
	if !planner.capabilities.SupportsV2Planning() {
		return full("host does not support selective upgrade planning")
	}
	if candidate.CompositionDigest == "" {
		return full("candidate composition evidence is unavailable")
	}
	if errors.Is(stateErr, ErrConfirmedStateNotFound) {
		return full("confirmed deployment state is unavailable")
	}
	if !state.Capabilities.DynamicFrontendUpstream || !candidate.Capabilities.DynamicFrontendUpstream {
		return full("dynamic frontend upstream is not confirmed")
	}
	if !onlyFrontendComponentChanged(state.Components, candidate.Components) {
		return full("candidate deployment change is not frontend-only")
	}
	if planner.observer == nil {
		return full("runtime observation is unavailable")
	}
	observation, err := planner.observer.ObserveRuntime(ctx)
	if err != nil {
		return full("runtime observation is unavailable")
	}
	if err := observation.Validate(); err != nil {
		return full("runtime observation is incomplete")
	}
	if !runtimeObservationMatchesState(observation, state) {
		return full("runtime containers drift from confirmed deployment state")
	}
	plan := ScopePlan{
		SchemaVersion:              ScopePlanSchema,
		OperationID:                operationID,
		ManifestDigest:             manifestDigest,
		ExecutionMode:              ExecutionModeFrontendOnly,
		BaselineStateDigest:        state.StateDigest,
		TouchedServices:            []string{FrontendOnlyService},
		ConfirmedDeploymentVersion: state.ReleaseVersion,
		Candidate:                  cloneCandidateDeployment(candidate),
		ObservedRuntimeImages:      cloneStringMap(observation.Images),
		ObservedNginxHealthy:       observation.NginxHealthy,
		ObservedNginxConfigDigest:  observation.NginxConfigDigest,
		GeneratedAt:                planner.now().UTC(),
	}
	digest, err := plan.derivedDigest()
	if err != nil {
		return ScopePlan{}, err
	}
	plan.PlanDigest = digest
	return plan, plan.Validate()
}

func (planner *HostScopePlanner) Revalidate(ctx context.Context, plan ScopePlan) error {
	if planner == nil || planner.store == nil || planner.candidate == nil {
		return fmt.Errorf("host scope planner is not configured")
	}
	if err := plan.Validate(); err != nil {
		return err
	}
	if plan.ExecutionMode != ExecutionModeFrontendOnly {
		return nil
	}
	candidate, err := planner.candidate.LoadCandidateDeployment(ctx, plan.ManifestDigest)
	if err != nil {
		return fmt.Errorf("%w: candidate deployment is unavailable", ErrScopePlanStale)
	}
	if !sameCandidateDeployment(candidate, plan.Candidate) {
		return fmt.Errorf("%w: candidate deployment changed", ErrScopePlanStale)
	}
	state, err := planner.store.LoadConfirmedDeploymentState()
	if err != nil || state.StateDigest != plan.BaselineStateDigest {
		return fmt.Errorf("%w: confirmed deployment state changed", ErrScopePlanStale)
	}
	if planner.observer == nil {
		return fmt.Errorf("%w: runtime observation is unavailable", ErrScopePlanStale)
	}
	observation, err := planner.observer.ObserveRuntime(ctx)
	if err != nil || observation.Validate() != nil || !runtimeObservationMatchesState(observation, state) || !sameStringMap(observation.Images, plan.ObservedRuntimeImages) || observation.NginxHealthy != plan.ObservedNginxHealthy || observation.NginxConfigDigest != plan.ObservedNginxConfigDigest {
		return fmt.Errorf("%w: runtime containers changed", ErrScopePlanStale)
	}
	return nil
}

// ObserveFullDeployment validates the live component inventory used to create
// a confirmed baseline. It intentionally compares the complete candidate set,
// including bootstrap and Engine runtime/package pairs, so a partial observer
// can never establish a baseline that later unlocks frontend-only execution.
func (planner *HostScopePlanner) ObserveFullDeployment(ctx context.Context, plan ScopePlan) (RuntimeObservation, error) {
	if planner == nil || planner.candidate == nil || planner.observer == nil {
		return RuntimeObservation{}, fmt.Errorf("full deployment observation is unavailable")
	}
	if err := plan.Validate(); err != nil {
		return RuntimeObservation{}, err
	}
	if plan.ExecutionMode != ExecutionModeFull {
		return RuntimeObservation{}, fmt.Errorf("full deployment observation requires a full scope plan")
	}
	observation, err := planner.observer.ObserveRuntime(ctx)
	if err != nil {
		return RuntimeObservation{}, fmt.Errorf("observe live deployment: %w", err)
	}
	if err := observation.Validate(); err != nil {
		return RuntimeObservation{}, fmt.Errorf("validate live deployment observation: %w", err)
	}
	if !runtimeObservationMatchesComponents(observation, plan.Candidate.Components) {
		return RuntimeObservation{}, fmt.Errorf("live deployment inventory does not match candidate components")
	}
	return observation, nil
}

func fullTouchedServices() []string {
	return append([]string(nil), fullScopeServices...)
}

func validTouchedService(service string) bool {
	for _, allowed := range fullScopeServices {
		if service == allowed {
			return true
		}
	}
	return false
}

func validateTouchedServices(mode ExecutionMode, services []string) error {
	if !mode.Valid() {
		return fmt.Errorf("execution mode is required")
	}
	if len(services) == 0 {
		return fmt.Errorf("touched services are required")
	}
	previous := ""
	for _, service := range services {
		if !validTouchedService(service) {
			return fmt.Errorf("unsupported touched service %q", service)
		}
		if service <= previous {
			return fmt.Errorf("touched services must be sorted and unique")
		}
		previous = service
	}
	if mode == ExecutionModeFrontendOnly {
		if len(services) != 1 || services[0] != FrontendOnlyService {
			return fmt.Errorf("frontend-only scope must touch only frontend")
		}
		return nil
	}
	if len(services) != len(fullScopeServices) {
		return fmt.Errorf("full scope must include every protected deployment surface")
	}
	for index, service := range fullScopeServices {
		if services[index] != service {
			return fmt.Errorf("full scope must include every protected deployment surface")
		}
	}
	return nil
}

func validateDeploymentComponents(components []DeploymentComponent) error {
	if len(components) == 0 {
		return fmt.Errorf("deployment components are required")
	}
	previous := ""
	seenRuntime := make(map[string]struct{}, 5)
	for _, component := range components {
		if err := component.Validate(); err != nil {
			return err
		}
		if component.ID <= previous {
			return fmt.Errorf("deployment components must be sorted and unique")
		}
		previous = component.ID
		if strings.HasPrefix(component.ID, "runtime.") {
			seenRuntime[component.ID] = struct{}{}
		}
	}
	for _, componentID := range []string{runtimeServerComponent, runtimeFrontendComponent, runtimeNginxComponent, runtimeAgentComponent, runtimeBootstrapComponent} {
		if _, exists := seenRuntime[componentID]; !exists {
			return fmt.Errorf("deployment components are missing %s", componentID)
		}
	}
	return nil
}

func sortDeploymentComponents(components []DeploymentComponent) {
	sort.Slice(components, func(left, right int) bool { return components[left].ID < components[right].ID })
}

func onlyFrontendComponentChanged(baseline, candidate []DeploymentComponent) bool {
	if len(baseline) != len(candidate) {
		return false
	}
	changes := 0
	for index := range baseline {
		if baseline[index].ID != candidate[index].ID {
			return false
		}
		if baseline[index].Digest != candidate[index].Digest {
			if baseline[index].ID != runtimeFrontendComponent {
				return false
			}
			changes++
		}
	}
	return changes == 1
}

func runtimeObservationMatchesState(observation RuntimeObservation, state ConfirmedDeploymentState) bool {
	// A selective plan is safe only when the live observation covers the exact
	// component set recorded in the confirmed baseline.  Comparing resident
	// containers alone would let an unobserved bootstrap or engine drift pass.
	if len(observation.Images) != len(state.Components) {
		return false
	}
	for _, component := range state.Components {
		if observation.Images[component.ID] != component.Digest {
			return false
		}
	}
	return observation.NginxConfigDigest == state.NginxConfigDigest
}

func runtimeObservationMatchesComponents(observation RuntimeObservation, components []DeploymentComponent) bool {
	if len(observation.Images) != len(components) {
		return false
	}
	for _, component := range components {
		if observation.Images[component.ID] != component.Digest {
			return false
		}
	}
	return true
}

func componentDigestFromInventory(components []DeploymentComponent, componentID string) string {
	for _, component := range components {
		if component.ID == componentID {
			return component.Digest
		}
	}
	return ""
}

func isResidentRuntimeComponent(componentID string) bool {
	for _, resident := range residentRuntimeIDs {
		if resident == componentID {
			return true
		}
	}
	return false
}

func isRuntimeObservationComponent(componentID string) bool {
	if isResidentRuntimeComponent(componentID) || componentID == runtimeBootstrapComponent {
		return true
	}
	return runtimeObservationComponentIDPattern.MatchString(componentID)
}

func cloneCandidateDeployment(candidate CandidateDeployment) CandidateDeployment {
	return CandidateDeployment{
		ReleaseVersion:    candidate.ReleaseVersion,
		CompositionDigest: candidate.CompositionDigest,
		Components:        append([]DeploymentComponent(nil), candidate.Components...),
		Capabilities:      candidate.Capabilities,
	}
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func sameStringMap(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func sameCandidateDeployment(left, right CandidateDeployment) bool {
	if left.ReleaseVersion != right.ReleaseVersion || left.CompositionDigest != right.CompositionDigest || left.Capabilities != right.Capabilities || len(left.Components) != len(right.Components) {
		return false
	}
	for index := range left.Components {
		if left.Components[index] != right.Components[index] {
			return false
		}
	}
	return true
}

func cloneHostCapabilities(capabilities HostCapabilities) HostCapabilities {
	capabilities.SchemaVersions = append([]int(nil), capabilities.SchemaVersions...)
	return capabilities
}

func containsSchema(schemas []int, wanted int) bool {
	for _, schema := range schemas {
		if schema == wanted {
			return true
		}
	}
	return false
}
