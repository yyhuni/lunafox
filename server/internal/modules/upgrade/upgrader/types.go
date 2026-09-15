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
)

const (
	JournalDirectory   = ".lunafox/upgrade"
	CurrentStateFile   = "current.json"
	HistoryDirectory   = "operations"
	ReceiptDirectory   = "receipts"
	ManifestDirectory  = "manifests"
	LockFile           = "lock"
	SocketFile         = "upgrader.sock"
	OverrideFileSuffix = ".compose.json"
	JournalSchema      = 1
	RequestSchema      = 1

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
	// ActionRepair is an explicit operator-authorized retry of a terminal
	// operation. Resume remains a read/idempotent continuation and must never
	// reset a terminal journal implicitly.
	ActionRepair Action = "repair"
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
}

type Journal struct {
	SchemaVersion  int    `json:"schemaVersion"`
	OperationID    string `json:"operationId"`
	ManifestDigest string `json:"manifestDigest"`
	Stage          Stage  `json:"stage"`
	// RepairStage records the first safe stage for an explicit repair. A
	// migration or post-migration failure resumes at verification instead of
	// rerunning an operation whose database outcome may already be committed.
	RepairStage       Stage      `json:"repairStage,omitempty"`
	MigrationID       string     `json:"migrationId,omitempty"`
	MigrationChecksum string     `json:"migrationChecksum,omitempty"`
	MigrationStatus   string     `json:"migrationStatus,omitempty"`
	StartedAt         time.Time  `json:"startedAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	CompletedAt       *time.Time `json:"completedAt,omitempty"`
	ExitCode          *int       `json:"exitCode,omitempty"`
	Diagnostic        string     `json:"diagnostic,omitempty"`
}

// Receipt is a host-only deployment proof. It records that the Compose
// command completed, but it deliberately carries no Operation success state;
// Server/Agent/migration verification remains authoritative elsewhere.
type Receipt struct {
	SchemaVersion  int               `json:"schemaVersion"`
	OperationID    string            `json:"operationId"`
	ManifestDigest string            `json:"manifestDigest"`
	CompletedAt    time.Time         `json:"completedAt"`
	Services       []string          `json:"services"`
	ObservedImages map[string]string `json:"observedImages"`
}

func (receipt Receipt) Validate() error {
	if receipt.SchemaVersion != JournalSchema {
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
	SchemaVersion int     `json:"schemaVersion"`
	Accepted      bool    `json:"accepted"`
	Replayed      bool    `json:"replayed"`
	Repaired      bool    `json:"repaired,omitempty"`
	Journal       Journal `json:"journal"`
	Error         string  `json:"error,omitempty"`
}

// Validate checks the response envelope before a caller acts on it.  The
// socket is local, but it still crosses a process boundary; accepting a
// malformed or mismatched journal here could make the Server believe a
// different operation was handed off.
func (response Response) Validate() error {
	if response.SchemaVersion != RequestSchema {
		return fmt.Errorf("unsupported upgrader response schema version %d", response.SchemaVersion)
	}
	if response.Accepted {
		if response.Error != "" {
			return fmt.Errorf("accepted upgrader response contains an error")
		}
		if err := response.Journal.Validate(); err != nil {
			return fmt.Errorf("invalid upgrader response journal: %w", err)
		}
		return nil
	}
	if response.Replayed || response.Repaired {
		return fmt.Errorf("rejected upgrader response cannot be replayed or repaired")
	}
	if strings.TrimSpace(response.Error) == "" {
		return fmt.Errorf("rejected upgrader response must include an error")
	}
	return nil
}

// JournalEvent is the host-to-control-plane observation boundary. It carries
// only validated operation identity, stage and bounded diagnostics. Receipt is
// a deployment proof and never implies overall Operation success.
type JournalEvent struct {
	OperationID       string    `json:"operationId"`
	ManifestDigest    string    `json:"manifestDigest"`
	Stage             Stage     `json:"stage"`
	UpdatedAt         time.Time `json:"updatedAt"`
	Diagnostic        string    `json:"diagnostic,omitempty"`
	MigrationID       string    `json:"migrationId,omitempty"`
	MigrationChecksum string    `json:"migrationChecksum,omitempty"`
	MigrationStatus   string    `json:"migrationStatus,omitempty"`
	Receipt           *Receipt  `json:"receipt,omitempty"`
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
	if request.SchemaVersion != RequestSchema {
		return fmt.Errorf("unsupported upgrader request schema version %d", request.SchemaVersion)
	}
	if err := validateOperationID(request.OperationID); err != nil {
		return err
	}
	if request.Action != ActionStart && request.Action != ActionResume && request.Action != ActionRepair {
		return fmt.Errorf("unsupported upgrader action %q", request.Action)
	}
	if err := validateDigest(request.ManifestDigest); err != nil {
		return err
	}
	return nil
}

func (journal Journal) Validate() error {
	if journal.SchemaVersion != JournalSchema {
		return fmt.Errorf("unsupported journal schema version %d", journal.SchemaVersion)
	}
	if err := validateOperationID(journal.OperationID); err != nil {
		return err
	}
	if err := validateDigest(journal.ManifestDigest); err != nil {
		return err
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
