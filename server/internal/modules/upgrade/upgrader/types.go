// Package upgrader contains the host-side, deployment-scoped upgrade boundary.
//
// The package deliberately keeps the request surface small. A Server may hand
// off an operation identity, a fixed action, and the digest it already
// verified; all executable material is resolved by the host process from its
// fixed deployment root.
package upgrader

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/blang/semver"
)

const (
	JournalDirectory     = ".lunafox/upgrade"
	CurrentStateFile     = "current.json"
	HistoryDirectory     = "operations"
	ReceiptDirectory     = "receipts"
	ManifestDirectory    = "manifests"
	CompositionDirectory = "compositions"
	PlanDirectory        = "plans"
	LockFile             = "lock"
	SocketFile           = "upgrader.sock"
	OverrideFileSuffix   = ".compose.json"
	// JournalSchema and RequestSchema intentionally retain their legacy v1
	// values. Existing deployments may resume these records indefinitely, so a
	// new binary must not reinterpret an omitted scope as frontend-only.
	JournalSchema = 1
	// ScopedJournalSchema and ScopedRequestSchema retain the wire schema-v2
	// value. Their names describe the scope evidence introduced at that version
	// without weakening compatibility for persisted records or host messages.
	ScopedJournalSchema = 2
	RequestSchema       = 1
	ScopedRequestSchema = 2
	// RequestSchemaV3 adds a read-only candidate-inventory availability probe.
	// It intentionally does not add a new executable or journal schema, so v2
	// plan-bound execution remains stable for existing Server/host rollouts.
	RequestSchemaV3      = 3
	ScopePlanSchema      = 1
	ConfirmedStateSchema = 1

	MaxDiagnosticBytes = 4096
	MaxRequestBytes    = 16 * 1024
)

const (
	MigrationStatusNotStarted = "not_started"
	MigrationStatusRunning    = "running"
	MigrationStatusSucceeded  = "succeeded"
	MigrationStatusFailed     = "failed"
	MigrationStatusUnknown    = "unknown"
)

type Stage string

const (
	StageQueued         Stage = "queued"
	StageStopping       Stage = "stopping"
	StagePreflight      Stage = "preflight"
	StageUpdating       Stage = "updating"
	StageMigrating      Stage = "migrating"
	StageRestarting     Stage = "restarting"
	StageAgentVerifying Stage = "agent_verifying"
	StageVerifying      Stage = "verifying"
	StageSucceeded      Stage = "succeeded"
	StageFailed         Stage = "failed"
	StageNeedsRecovery  Stage = "needs_recovery"
	StageNeedsAttention Stage = "needs_attention"
)

type Action string

const (
	ActionStart  Action = "start"
	ActionResume Action = "resume"
	// ActionStop cancels the currently executing operation. It is intentionally
	// separate from repair so a stop can never reset a terminal journal.
	ActionStop Action = "stop"
	// ActionRepair is an explicit operator-authorized retry of a terminal
	// operation. Resume remains a read/idempotent continuation and must never
	// reset a terminal journal implicitly.
	ActionRepair Action = "repair"
	// ActionCapabilities is an explicit v2 negotiation request. It has no
	// operation identity because it performs no deployment work.
	ActionCapabilities Action = "capabilities"
	// ActionPlan asks the host to create a read-only, persisted scope plan.
	// The returned plan digest is later required by v2 start and confirmation.
	ActionPlan Action = "plan"
	// ActionConfirm atomically advances host-owned confirmed deployment state
	// only after the Server has completed its non-host verification gates.
	ActionConfirm Action = "confirm"
	// ActionDeploymentState returns the last fully confirmed deployment without
	// requiring an operation identity. It is a read-only v2 projection used by
	// update availability checks; schema-v1 deliberately has no equivalent.
	ActionDeploymentState Action = "state"
	// ActionCandidateAvailability is a schema-v3 read-only comparison between
	// one cached candidate and the host-owned confirmed deployment. It never
	// persists a scope plan or creates executable work.
	ActionCandidateAvailability Action = "availability"
)

var (
	operationIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	digestPattern      = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

// Request is the complete wire contract accepted over the local socket.
// Do not add paths, image references, shell commands or arbitrary arguments
// here: those would turn the privileged host boundary into a command proxy.
type Request struct {
	SchemaVersion  int    `json:"schemaVersion"`
	OperationID    string `json:"operationId"`
	Action         Action `json:"action"`
	ManifestDigest string `json:"manifestDigest"`
	// The following fields are required only by a v2 plan-bound start, resume,
	// repair, or confirmation. A v1 request must leave all of them empty.
	ExecutionMode              ExecutionMode `json:"executionMode,omitempty"`
	PlanDigest                 string        `json:"planDigest,omitempty"`
	BaselineStateDigest        string        `json:"baselineStateDigest,omitempty"`
	TouchedServices            []string      `json:"touchedServices,omitempty"`
	ConfirmedDeploymentVersion string        `json:"confirmedDeploymentVersion,omitempty"`
	// RequireFull is accepted only by the v2 read-only planning request. It is
	// a Server-owned safety ceiling for migration/compatibility gates; it can
	// only remove the fast path and is never persisted as execution evidence.
	RequireFull bool `json:"requireFull,omitempty"`
}

type Journal struct {
	SchemaVersion  int    `json:"schemaVersion"`
	OperationID    string `json:"operationId"`
	ManifestDigest string `json:"manifestDigest"`
	Stage          Stage  `json:"stage"`
	// Scope evidence is written only by v2 operations. Keeping it beside the
	// lifecycle checkpoint lets a restart reconstruct the same sealed scope
	// rather than accidentally widening a previously non-disruptive operation.
	ExecutionMode              ExecutionMode `json:"executionMode,omitempty"`
	PlanDigest                 string        `json:"planDigest,omitempty"`
	BaselineStateDigest        string        `json:"baselineStateDigest,omitempty"`
	TouchedServices            []string      `json:"touchedServices,omitempty"`
	ConfirmedDeploymentVersion string        `json:"confirmedDeploymentVersion,omitempty"`
	// RepairStage records the first safe stage for an explicit repair. A
	// migration or post-migration failure resumes at verification instead of
	// rerunning an operation whose database outcome may already be committed.
	RepairStage       Stage     `json:"repairStage,omitempty"`
	MigrationID       string    `json:"migrationId,omitempty"`
	MigrationChecksum string    `json:"migrationChecksum,omitempty"`
	MigrationStatus   string    `json:"migrationStatus,omitempty"`
	StartedAt         time.Time `json:"startedAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	// StageUpdatedAt is independent from UpdatedAt: progress heartbeats may
	// refresh the latter without manufacturing a new lifecycle checkpoint.
	StageUpdatedAt time.Time       `json:"stageUpdatedAt,omitempty"`
	CompletedAt    *time.Time      `json:"completedAt,omitempty"`
	ExitCode       *int            `json:"exitCode,omitempty"`
	Diagnostic     string          `json:"diagnostic,omitempty"`
	ProgressEvents []ProgressEvent `json:"progressEvents,omitempty"`
}

// Receipt is a host-only deployment proof. It records that the Compose
// command completed, but it deliberately carries no Operation success state;
// Server/Agent/migration verification remains authoritative elsewhere.
type Receipt struct {
	SchemaVersion              int               `json:"schemaVersion"`
	OperationID                string            `json:"operationId"`
	ManifestDigest             string            `json:"manifestDigest"`
	CompletedAt                time.Time         `json:"completedAt"`
	Services                   []string          `json:"services"`
	ObservedImages             map[string]string `json:"observedImages"`
	ExecutionMode              ExecutionMode     `json:"executionMode,omitempty"`
	PlanDigest                 string            `json:"planDigest,omitempty"`
	BaselineStateDigest        string            `json:"baselineStateDigest,omitempty"`
	TouchedServices            []string          `json:"touchedServices,omitempty"`
	ConfirmedDeploymentVersion string            `json:"confirmedDeploymentVersion,omitempty"`
}

func (receipt Receipt) Validate() error {
	if !validJournalSchema(receipt.SchemaVersion) {
		return fmt.Errorf("unsupported receipt schema version %d", receipt.SchemaVersion)
	}
	if err := validateOperationID(receipt.OperationID); err != nil {
		return err
	}
	if err := validateDigest(receipt.ManifestDigest); err != nil {
		return err
	}
	if receipt.CompletedAt.IsZero() {
		return fmt.Errorf("receipt completedAt is required")
	}
	if len(receipt.Services) == 0 {
		return fmt.Errorf("receipt services are required")
	}
	seenServices := make(map[string]struct{}, len(receipt.Services))
	for _, service := range receipt.Services {
		if !validComposeService(service) {
			return fmt.Errorf("receipt contains unsupported service %q", service)
		}
		if _, duplicate := seenServices[service]; duplicate {
			return fmt.Errorf("receipt contains duplicate service %q", service)
		}
		seenServices[service] = struct{}{}
	}
	for service, digest := range receipt.ObservedImages {
		if _, expected := seenServices[service]; !expected {
			return fmt.Errorf("receipt contains unsupported service %q", service)
		}
		if err := validateDigest(digest); err != nil {
			return fmt.Errorf("receipt observed image %s: %w", service, err)
		}
	}
	for service := range seenServices {
		if _, observed := receipt.ObservedImages[service]; !observed {
			return fmt.Errorf("receipt is missing observed image %q", service)
		}
	}
	if receipt.SchemaVersion == JournalSchema {
		return validateEmptyScopeEvidence(receipt.ExecutionMode, receipt.PlanDigest, receipt.BaselineStateDigest, receipt.TouchedServices, receipt.ConfirmedDeploymentVersion)
	}
	if err := validateScopeEvidence(receipt.ExecutionMode, receipt.PlanDigest, receipt.BaselineStateDigest, receipt.TouchedServices, receipt.ConfirmedDeploymentVersion); err != nil {
		return fmt.Errorf("receipt scope evidence: %w", err)
	}
	if receipt.ExecutionMode == ExecutionModeFrontendOnly && !sameStringSlice(receipt.Services, receipt.TouchedServices) {
		return fmt.Errorf("frontend-only receipt services must match its touched services")
	}
	return nil
}

func validComposeService(service string) bool {
	switch service {
	case "server", "frontend", "nginx", "agent":
		return true
	default:
		return false
	}
}

type Response struct {
	SchemaVersion            int                       `json:"schemaVersion"`
	Accepted                 bool                      `json:"accepted"`
	Replayed                 bool                      `json:"replayed"`
	Repaired                 bool                      `json:"repaired,omitempty"`
	Journal                  Journal                   `json:"journal"`
	Capabilities             *HostCapabilities         `json:"capabilities,omitempty"`
	ScopePlan                *ScopePlan                `json:"scopePlan,omitempty"`
	ConfirmedDeploymentState *ConfirmedDeploymentState `json:"confirmedDeploymentState,omitempty"`
	CandidateAvailability    *CandidateAvailability    `json:"candidateAvailability,omitempty"`
	Error                    string                    `json:"error,omitempty"`
}

// Validate checks the response envelope before a caller acts on it.  The
// socket is local, but it still crosses a process boundary; accepting a
// malformed or mismatched journal here could make the Server believe a
// different operation was handed off.
func (response Response) Validate() error {
	if !validRequestSchema(response.SchemaVersion) {
		return fmt.Errorf("unsupported upgrader response schema version %d", response.SchemaVersion)
	}
	if !response.Accepted {
		if response.Replayed || response.Repaired {
			return fmt.Errorf("rejected upgrader response cannot be replayed or repaired")
		}
		if strings.TrimSpace(response.Error) == "" {
			return fmt.Errorf("rejected upgrader response must include an error")
		}
		if response.Capabilities != nil || response.ScopePlan != nil || response.ConfirmedDeploymentState != nil || response.CandidateAvailability != nil {
			return fmt.Errorf("rejected upgrader response cannot contain result data")
		}
		return nil
	}
	if response.Error != "" {
		return fmt.Errorf("accepted upgrader response contains an error")
	}
	if response.Capabilities != nil {
		if response.ScopePlan != nil || response.ConfirmedDeploymentState != nil || response.CandidateAvailability != nil || response.Replayed || response.Repaired || !isZeroJournal(response.Journal) {
			return fmt.Errorf("capability response shape is invalid")
		}
		return response.Capabilities.Validate()
	}
	if response.ScopePlan != nil {
		if response.ConfirmedDeploymentState != nil || response.CandidateAvailability != nil || response.Replayed || response.Repaired || !isZeroJournal(response.Journal) {
			return fmt.Errorf("scope-plan response shape is invalid")
		}
		return response.ScopePlan.Validate()
	}
	if response.CandidateAvailability != nil {
		if response.ConfirmedDeploymentState != nil || response.Replayed || response.Repaired || !isZeroJournal(response.Journal) {
			return fmt.Errorf("candidate availability response shape is invalid")
		}
		return response.CandidateAvailability.Validate()
	}
	if response.ConfirmedDeploymentState != nil {
		if response.Replayed || response.Repaired {
			return fmt.Errorf("deployment state response cannot be replayed or repaired")
		}
		if !isZeroJournal(response.Journal) {
			if err := response.Journal.Validate(); err != nil {
				return fmt.Errorf("invalid deployment state response journal: %w", err)
			}
		}
		return response.ConfirmedDeploymentState.Validate()
	}
	if err := response.Journal.Validate(); err != nil {
		return fmt.Errorf("invalid upgrader response journal: %w", err)
	}
	return nil
}

// ValidateFor verifies the action-specific response shape before a caller uses
// it. Generic envelope validation alone cannot prove that a scope-plan reply
// belongs to the operation that requested it or that a v2 start kept its exact
// persisted execution scope.
func (response Response) ValidateFor(request Request) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("invalid upgrader request binding: %w", err)
	}
	if err := response.Validate(); err != nil {
		return err
	}
	if response.SchemaVersion != request.SchemaVersion {
		return fmt.Errorf("upgrader response schema does not match request schema")
	}
	if !response.Accepted {
		return nil
	}
	if request.SchemaVersion == RequestSchema {
		if response.Capabilities != nil || response.ScopePlan != nil || response.ConfirmedDeploymentState != nil || response.CandidateAvailability != nil {
			return fmt.Errorf("schema-v1 response contains v2 result data")
		}
		return validateResponseJournalBinding(response.Journal, request)
	}
	if request.SchemaVersion == ScopedRequestSchema && response.Capabilities != nil &&
		(response.Capabilities.CandidateInventoryAvailability || containsSchema(response.Capabilities.SchemaVersions, RequestSchemaV3)) {
		return fmt.Errorf("schema-v2 capability response contains schema-v3 data")
	}
	switch request.Action {
	case ActionCapabilities:
		if response.Capabilities == nil {
			return fmt.Errorf("capability response is required")
		}
		return nil
	case ActionPlan:
		if response.ScopePlan == nil {
			return fmt.Errorf("scope-plan response is required")
		}
		if response.ScopePlan.OperationID != request.OperationID || response.ScopePlan.ManifestDigest != request.ManifestDigest {
			return ErrReplayDigestMismatch
		}
		return nil
	case ActionConfirm:
		if response.ConfirmedDeploymentState == nil {
			return fmt.Errorf("deployment confirmation response is required")
		}
		if isZeroJournal(response.Journal) {
			return fmt.Errorf("deployment confirmation journal is required")
		}
		if err := validateResponseJournalBinding(response.Journal, request); err != nil {
			return err
		}
		state := response.ConfirmedDeploymentState
		if state.OperationID != request.OperationID || state.ManifestDigest != request.ManifestDigest {
			return ErrReplayDigestMismatch
		}
		return nil
	case ActionDeploymentState:
		if response.ConfirmedDeploymentState == nil {
			return fmt.Errorf("confirmed deployment state response is required")
		}
		if !isZeroJournal(response.Journal) {
			return fmt.Errorf("state response must not contain an operation journal")
		}
		return nil
	case ActionCandidateAvailability:
		if response.CandidateAvailability == nil {
			return fmt.Errorf("candidate availability response is required")
		}
		if response.CandidateAvailability.ManifestDigest != request.ManifestDigest {
			return ErrReplayDigestMismatch
		}
		return nil
	case ActionStart, ActionResume, ActionStop, ActionRepair:
		return validateResponseJournalBinding(response.Journal, request)
	default:
		return fmt.Errorf("unsupported upgrader action %q", request.Action)
	}
}

func validateResponseJournalBinding(journal Journal, request Request) error {
	if journal.OperationID != request.OperationID || journal.ManifestDigest != request.ManifestDigest {
		return ErrReplayDigestMismatch
	}
	if request.SchemaVersion == ScopedRequestSchema && !sameScopeEvidence(journal.ExecutionMode, journal.PlanDigest, journal.BaselineStateDigest, journal.TouchedServices, journal.ConfirmedDeploymentVersion, request.ExecutionMode, request.PlanDigest, request.BaselineStateDigest, request.TouchedServices, request.ConfirmedDeploymentVersion) {
		return fmt.Errorf("upgrader response journal scope does not match request")
	}
	return nil
}

func isZeroJournal(journal Journal) bool {
	return journal.SchemaVersion == 0 && journal.OperationID == "" && journal.ManifestDigest == "" && journal.Stage == ""
}

func validRequestSchema(schemaVersion int) bool {
	return schemaVersion == RequestSchema || schemaVersion == ScopedRequestSchema || schemaVersion == RequestSchemaV3
}

func validJournalSchema(schemaVersion int) bool {
	return schemaVersion == JournalSchema || schemaVersion == ScopedJournalSchema
}

func validateEmptyScopeEvidence(mode ExecutionMode, planDigest, baselineStateDigest string, touchedServices []string, confirmedDeploymentVersion string) error {
	if mode != "" || planDigest != "" || baselineStateDigest != "" || len(touchedServices) != 0 || confirmedDeploymentVersion != "" {
		return fmt.Errorf("schema-v1 record cannot contain scope evidence")
	}
	return nil
}

func validateScopeEvidence(mode ExecutionMode, planDigest, baselineStateDigest string, touchedServices []string, confirmedDeploymentVersion string) error {
	if !mode.Valid() {
		return fmt.Errorf("execution mode is required")
	}
	if err := validateDigest(planDigest); err != nil {
		return fmt.Errorf("plan digest: %w", err)
	}
	if err := validateTouchedServices(mode, touchedServices); err != nil {
		return err
	}
	if mode == ExecutionModeFrontendOnly {
		if err := validateDigest(baselineStateDigest); err != nil {
			return fmt.Errorf("baseline state digest: %w", err)
		}
		if confirmedDeploymentVersion == "" || confirmedDeploymentVersion != strings.TrimSpace(confirmedDeploymentVersion) {
			return fmt.Errorf("confirmed deployment version is required")
		}
		return nil
	}
	if baselineStateDigest != "" || confirmedDeploymentVersion != "" {
		return fmt.Errorf("full scope cannot contain selective baseline evidence")
	}
	return nil
}

func sameScopeEvidence(leftMode ExecutionMode, leftPlan, leftBaseline string, leftServices []string, leftVersion string, rightMode ExecutionMode, rightPlan, rightBaseline string, rightServices []string, rightVersion string) bool {
	return leftMode == rightMode && leftPlan == rightPlan && leftBaseline == rightBaseline && leftVersion == rightVersion && sameStringSlice(leftServices, rightServices)
}

func sameStringSlice(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// JournalEvent is the host-to-control-plane observation boundary. It carries
// only validated operation identity, stage and bounded diagnostics. Receipt is
// a deployment proof and never implies overall Operation success.
type JournalEvent struct {
	OperationID       string          `json:"operationId"`
	ManifestDigest    string          `json:"manifestDigest"`
	Stage             Stage           `json:"stage"`
	UpdatedAt         time.Time       `json:"updatedAt"`
	StageUpdatedAt    time.Time       `json:"stageUpdatedAt,omitempty"`
	Diagnostic        string          `json:"diagnostic,omitempty"`
	MigrationID       string          `json:"migrationId,omitempty"`
	MigrationChecksum string          `json:"migrationChecksum,omitempty"`
	MigrationStatus   string          `json:"migrationStatus,omitempty"`
	ProgressEvents    []ProgressEvent `json:"progressEvents,omitempty"`
	Receipt           *Receipt        `json:"receipt,omitempty"`
}

func (event JournalEvent) Validate() error {
	if err := validateOperationID(event.OperationID); err != nil {
		return err
	}
	if err := validateDigest(event.ManifestDigest); err != nil {
		return err
	}
	if !validStage(event.Stage) {
		// Receipt-only events may be emitted before a current checkpoint is
		// readable; they still carry a valid receipt identity and are accepted.
		if event.Receipt == nil {
			return fmt.Errorf("unsupported journal event stage %q", event.Stage)
		}
	} else if event.UpdatedAt.IsZero() {
		return fmt.Errorf("journal event updatedAt is required")
	}
	if !event.StageUpdatedAt.IsZero() && event.StageUpdatedAt.After(event.UpdatedAt) {
		return fmt.Errorf("journal event stageUpdatedAt is newer than updatedAt")
	}
	if err := validateProgressEvents(event.ProgressEvents); err != nil {
		return err
	}
	for _, progress := range event.ProgressEvents {
		if progress.Timestamp.After(event.UpdatedAt) {
			return fmt.Errorf("journal event progress timestamp is newer than updatedAt")
		}
	}
	if err := ValidateDiagnostic(event.Diagnostic); err != nil {
		return err
	}
	if event.MigrationID != "" && !validJournalToken(event.MigrationID) {
		return fmt.Errorf("journal event migrationId is not canonical")
	}
	if event.MigrationChecksum != "" && !digestPattern.MatchString(event.MigrationChecksum) {
		return fmt.Errorf("journal event migrationChecksum is not canonical")
	}
	if event.MigrationStatus != "" {
		switch event.MigrationStatus {
		case MigrationStatusNotStarted, MigrationStatusRunning, MigrationStatusSucceeded, MigrationStatusFailed, MigrationStatusUnknown:
		default:
			return fmt.Errorf("unsupported journal event migrationStatus %q", event.MigrationStatus)
		}
		if event.MigrationStatus != MigrationStatusNotStarted && (event.MigrationID == "" || event.MigrationChecksum == "") {
			return fmt.Errorf("journal event migration identity and checksum are required for status %q", event.MigrationStatus)
		}
	}
	if event.Receipt != nil {
		if err := event.Receipt.Validate(); err != nil {
			return err
		}
		if event.Receipt.OperationID != event.OperationID || event.Receipt.ManifestDigest != event.ManifestDigest {
			return ErrReceiptMismatch
		}
	}
	return nil
}

// EventSink receives best-effort host observations. Persistence remains the
// source of truth: a sink failure must never turn a durable checkpoint into a
// failed upgrade or block the privileged executor.
type EventSink interface {
	Publish(context.Context, JournalEvent) error
}

// EventSinkFunc adapts a callback to EventSink.
type EventSinkFunc func(context.Context, JournalEvent) error

func (function EventSinkFunc) Publish(ctx context.Context, event JournalEvent) error {
	if function == nil {
		return nil
	}
	return function(ctx, event)
}

func (request Request) Validate() error {
	if !validRequestSchema(request.SchemaVersion) {
		return fmt.Errorf("unsupported upgrader request schema version %d", request.SchemaVersion)
	}
	if request.SchemaVersion == RequestSchema {
		if err := validateOperationID(request.OperationID); err != nil {
			return err
		}
		if request.Action != ActionStart && request.Action != ActionResume && request.Action != ActionStop && request.Action != ActionRepair {
			return fmt.Errorf("unsupported upgrader action %q", request.Action)
		}
		if err := validateDigest(request.ManifestDigest); err != nil {
			return err
		}
		if request.RequireFull {
			return fmt.Errorf("schema-v1 request cannot require full scope planning")
		}
		return validateEmptyScopeEvidence(request.ExecutionMode, request.PlanDigest, request.BaselineStateDigest, request.TouchedServices, request.ConfirmedDeploymentVersion)
	}
	if request.SchemaVersion == RequestSchemaV3 {
		switch request.Action {
		case ActionCapabilities:
			if request.OperationID != "" || request.ManifestDigest != "" || request.RequireFull {
				return fmt.Errorf("schema-v3 capability request cannot contain operation or planning fields")
			}
			return validateEmptyScopeEvidence(request.ExecutionMode, request.PlanDigest, request.BaselineStateDigest, request.TouchedServices, request.ConfirmedDeploymentVersion)
		case ActionCandidateAvailability:
			if request.OperationID != "" {
				return fmt.Errorf("candidate availability request cannot contain an operation identity")
			}
			if err := validateDigest(request.ManifestDigest); err != nil {
				return err
			}
			if request.RequireFull {
				return fmt.Errorf("candidate availability request cannot require full scope planning")
			}
			return validateEmptyScopeEvidence(request.ExecutionMode, request.PlanDigest, request.BaselineStateDigest, request.TouchedServices, request.ConfirmedDeploymentVersion)
		default:
			return fmt.Errorf("unsupported schema-v3 upgrader action %q", request.Action)
		}
	}
	switch request.Action {
	case ActionCapabilities:
		if request.OperationID != "" || request.ManifestDigest != "" {
			return fmt.Errorf("capability request cannot contain operation identity")
		}
		if request.RequireFull {
			return fmt.Errorf("capability request cannot require full scope planning")
		}
		return validateEmptyScopeEvidence(request.ExecutionMode, request.PlanDigest, request.BaselineStateDigest, request.TouchedServices, request.ConfirmedDeploymentVersion)
	case ActionPlan:
		if err := validateOperationID(request.OperationID); err != nil {
			return err
		}
		if err := validateDigest(request.ManifestDigest); err != nil {
			return err
		}
		return validateEmptyScopeEvidence(request.ExecutionMode, request.PlanDigest, request.BaselineStateDigest, request.TouchedServices, request.ConfirmedDeploymentVersion)
	case ActionStart, ActionResume, ActionStop, ActionRepair, ActionConfirm:
		if err := validateOperationID(request.OperationID); err != nil {
			return err
		}
		if err := validateDigest(request.ManifestDigest); err != nil {
			return err
		}
		if request.RequireFull {
			return fmt.Errorf("execution request cannot require full scope planning")
		}
		return validateScopeEvidence(request.ExecutionMode, request.PlanDigest, request.BaselineStateDigest, request.TouchedServices, request.ConfirmedDeploymentVersion)
	case ActionDeploymentState:
		if request.OperationID != "" || request.ManifestDigest != "" {
			return fmt.Errorf("deployment state request cannot contain operation identity")
		}
		if request.RequireFull {
			return fmt.Errorf("deployment state request cannot require full scope planning")
		}
		return validateEmptyScopeEvidence(request.ExecutionMode, request.PlanDigest, request.BaselineStateDigest, request.TouchedServices, request.ConfirmedDeploymentVersion)
	default:
		return fmt.Errorf("unsupported schema-v2 upgrader action %q", request.Action)
	}
}

// CandidateAvailabilityDecision is the host's bounded answer to a read-only
// candidate inventory comparison. A fallback deliberately carries no baseline
// evidence, so the Server can preserve its legacy semantic-version behavior.
type CandidateAvailabilityDecision string

const (
	CandidateAvailabilityAvailable      CandidateAvailabilityDecision = "available"
	CandidateAvailabilityAlreadyApplied CandidateAvailabilityDecision = "already_applied"
	CandidateAvailabilityNotNewer       CandidateAvailabilityDecision = "not_newer"
	CandidateAvailabilityFallback       CandidateAvailabilityDecision = "fallback"
	CandidateAvailabilityConflict       CandidateAvailabilityDecision = "conflict"
)

// CandidateAvailability is the narrow v3 availability response. It does not
// disclose the deployment inventory, but definitive answers prove that the
// host compared against one validated confirmed state.
type CandidateAvailability struct {
	ManifestDigest             string                        `json:"manifestDigest"`
	Decision                   CandidateAvailabilityDecision `json:"decision"`
	BaselineStateDigest        string                        `json:"baselineStateDigest,omitempty"`
	ConfirmedDeploymentVersion string                        `json:"confirmedDeploymentVersion,omitempty"`
}

func (availability CandidateAvailability) Validate() error {
	if err := validateDigest(availability.ManifestDigest); err != nil {
		return err
	}
	switch availability.Decision {
	case CandidateAvailabilityFallback:
		if availability.BaselineStateDigest != "" || availability.ConfirmedDeploymentVersion != "" {
			return fmt.Errorf("fallback candidate availability cannot contain confirmed deployment evidence")
		}
		return nil
	case CandidateAvailabilityAvailable, CandidateAvailabilityAlreadyApplied, CandidateAvailabilityNotNewer, CandidateAvailabilityConflict:
		if err := validateDigest(availability.BaselineStateDigest); err != nil {
			return fmt.Errorf("candidate availability baseline state digest: %w", err)
		}
		if availability.ConfirmedDeploymentVersion == "" || availability.ConfirmedDeploymentVersion != strings.TrimSpace(availability.ConfirmedDeploymentVersion) {
			return fmt.Errorf("candidate availability confirmed deployment version is required")
		}
		if _, err := semver.Parse(availability.ConfirmedDeploymentVersion); err != nil {
			return fmt.Errorf("candidate availability confirmed deployment version is invalid: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported candidate availability decision %q", availability.Decision)
	}
}

func (journal Journal) Validate() error {
	if !validJournalSchema(journal.SchemaVersion) {
		return fmt.Errorf("unsupported journal schema version %d", journal.SchemaVersion)
	}
	if err := validateOperationID(journal.OperationID); err != nil {
		return err
	}
	if err := validateDigest(journal.ManifestDigest); err != nil {
		return err
	}
	if journal.SchemaVersion == JournalSchema {
		if err := validateEmptyScopeEvidence(journal.ExecutionMode, journal.PlanDigest, journal.BaselineStateDigest, journal.TouchedServices, journal.ConfirmedDeploymentVersion); err != nil {
			return err
		}
	} else if err := validateScopeEvidence(journal.ExecutionMode, journal.PlanDigest, journal.BaselineStateDigest, journal.TouchedServices, journal.ConfirmedDeploymentVersion); err != nil {
		return fmt.Errorf("journal scope evidence: %w", err)
	}
	if !validStage(journal.Stage) {
		return fmt.Errorf("unsupported journal stage %q", journal.Stage)
	}
	if journal.RepairStage != "" {
		if !validStage(journal.RepairStage) || IsTerminal(journal.RepairStage) {
			return fmt.Errorf("unsupported journal repairStage %q", journal.RepairStage)
		}
		if !IsTerminal(journal.Stage) {
			return fmt.Errorf("non-terminal journal cannot have repairStage")
		}
	}
	if journal.MigrationStatus != "" {
		switch journal.MigrationStatus {
		case MigrationStatusNotStarted, MigrationStatusRunning, MigrationStatusSucceeded, MigrationStatusFailed, MigrationStatusUnknown:
		default:
			return fmt.Errorf("unsupported journal migrationStatus %q", journal.MigrationStatus)
		}
	}
	if journal.MigrationID != "" && !validJournalToken(journal.MigrationID) {
		return fmt.Errorf("journal migrationId is not canonical")
	}
	if journal.MigrationChecksum != "" && !digestPattern.MatchString(journal.MigrationChecksum) {
		return fmt.Errorf("journal migrationChecksum is not canonical")
	}
	if journal.MigrationStatus != "" && journal.MigrationStatus != MigrationStatusNotStarted && (journal.MigrationID == "" || journal.MigrationChecksum == "") {
		return fmt.Errorf("journal migration identity and checksum are required for status %q", journal.MigrationStatus)
	}
	if journal.StartedAt.IsZero() || journal.UpdatedAt.IsZero() {
		return fmt.Errorf("journal timestamps are required")
	}
	if journal.UpdatedAt.Before(journal.StartedAt) {
		return fmt.Errorf("journal updatedAt precedes startedAt")
	}
	if !journal.StageUpdatedAt.IsZero() {
		if journal.StageUpdatedAt.Before(journal.StartedAt) || journal.StageUpdatedAt.After(journal.UpdatedAt) {
			return fmt.Errorf("journal stageUpdatedAt is outside journal timestamps")
		}
	}
	if err := validateProgressEvents(journal.ProgressEvents); err != nil {
		return err
	}
	for _, progress := range journal.ProgressEvents {
		if progress.Timestamp.Before(journal.StartedAt) || progress.Timestamp.After(journal.UpdatedAt) {
			return fmt.Errorf("journal progress timestamp is outside journal timestamps")
		}
	}
	if journal.CompletedAt != nil && journal.CompletedAt.Before(journal.StartedAt) {
		return fmt.Errorf("journal completedAt precedes startedAt")
	}
	if IsTerminal(journal.Stage) && journal.CompletedAt == nil {
		return fmt.Errorf("terminal journal completedAt is required")
	}
	if !IsTerminal(journal.Stage) && journal.CompletedAt != nil {
		return fmt.Errorf("non-terminal journal cannot have completedAt")
	}
	if err := ValidateDiagnostic(journal.Diagnostic); err != nil {
		return err
	}
	return nil
}

func validStage(stage Stage) bool {
	switch stage {
	case StageQueued, StageStopping, StagePreflight, StageUpdating,
		StageMigrating, StageRestarting, StageAgentVerifying, StageVerifying,
		StageSucceeded, StageFailed, StageNeedsRecovery, StageNeedsAttention:
		return true
	default:
		return false
	}
}

func IsTerminal(stage Stage) bool {
	switch stage {
	case StageSucceeded, StageFailed, StageNeedsRecovery, StageNeedsAttention:
		return true
	default:
		return false
	}
}

func ValidateStageTransition(from, to Stage) error {
	if !validStage(from) || !validStage(to) {
		return fmt.Errorf("unsupported journal stage transition %q -> %q", from, to)
	}
	if IsTerminal(from) && from != to {
		return fmt.Errorf("terminal journal cannot transition from %q to %q", from, to)
	}
	if from == to {
		return nil
	}
	if to == StageNeedsRecovery || to == StageNeedsAttention {
		return nil
	}
	if stageRank(to) < stageRank(from) {
		return fmt.Errorf("journal stage cannot move backwards from %q to %q", from, to)
	}
	return nil
}

func stageRank(stage Stage) int {
	switch stage {
	case StageQueued:
		return 0
	case StageStopping:
		return 1
	case StagePreflight:
		return 2
	case StageUpdating:
		return 3
	case StageMigrating:
		return 4
	case StageRestarting:
		return 5
	case StageAgentVerifying:
		return 6
	case StageVerifying:
		return 7
	case StageSucceeded, StageFailed:
		return 8
	case StageNeedsRecovery, StageNeedsAttention:
		return 9
	default:
		return -1
	}
}

// repairStageFor maps a terminal safety outcome to the earliest stage that is
// safe to execute again. Once migration or post-migration verification has
// started, repair skips mutation and migration and only re-runs verification.
func repairStageFor(stage Stage) Stage {
	switch stage {
	case StageMigrating, StageRestarting, StageAgentVerifying, StageVerifying:
		return StageRestarting
	case StageUpdating:
		return StageUpdating
	case StagePreflight, StageStopping, StageQueued:
		return StageQueued
	default:
		return StageQueued
	}
}

func validateOperationID(operationID string) error {
	if operationID != strings.TrimSpace(operationID) || operationID == "" || !operationIDPattern.MatchString(operationID) {
		return fmt.Errorf("operationId must be a canonical opaque identifier")
	}
	return nil
}

func validateDigest(digest string) error {
	if !digestPattern.MatchString(digest) {
		return fmt.Errorf("manifestDigest must be a canonical sha256 digest")
	}
	return nil
}

// ValidateDiagnostic rejects values that could accidentally persist a secret.
// Diagnostics are intentionally a single bounded line; callers should provide
// a fixed safe message rather than copying command output into the journal.
func ValidateDiagnostic(diagnostic string) error {
	if len([]byte(diagnostic)) > MaxDiagnosticBytes {
		return fmt.Errorf("diagnostic exceeds %d bytes", MaxDiagnosticBytes)
	}
	if !utf8.ValidString(diagnostic) || strings.ContainsAny(diagnostic, "\x00\r\n") {
		return fmt.Errorf("diagnostic contains invalid control characters")
	}
	lower := strings.ToLower(diagnostic)
	for _, secretWord := range []string{
		"authorization", "bearer ", "jwt", "password", "passwd", "token", "private key", "secret",
	} {
		if strings.Contains(lower, secretWord) {
			return fmt.Errorf("diagnostic contains a forbidden credential marker")
		}
	}
	return nil
}
