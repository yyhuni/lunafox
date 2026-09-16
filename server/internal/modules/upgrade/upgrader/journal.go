package upgrader

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrJournalNotFound           = os.ErrNotExist
	ErrJournalCorrupt            = errors.New("upgrader journal is missing, truncated, or invalid")
	ErrReceiptMismatch           = errors.New("deployment receipt does not match the operation journal")
	ErrRepairNotAllowed          = errors.New("upgrade operation cannot be repaired from its current journal")
	ErrMigrationIdentityMismatch = errors.New("migration identity does not match the operation journal")
	ErrMigrationTransition       = errors.New("migration status transition is not allowed")
)

// Receipt timestamps are written by the host, but they are also read after a
// restart and may arrive through a best-effort observation path. Allow a small
// clock skew without accepting an obviously fabricated future receipt.
const maxReceiptFutureSkew = 5 * time.Minute

type JournalStore struct {
	deploymentRoot string
	root           string
	manifestCache  bool
	// writeMu serializes read/modify/write transitions.  Atomic rename keeps
	// individual files intact, but without this lock two concurrent checkpoint
	// writers could both read the same stage and lose one migration outcome.
	writeMu     sync.Mutex
	eventMu     sync.RWMutex
	eventSink   EventSink
	eventQueue  chan JournalEvent
	eventWorker sync.Once
}

func NewJournalStore(deploymentRoot string) (*JournalStore, error) {
	return newJournalStore(deploymentRoot, false)
}

// NewPublicJournalStore uses immutable digest-addressed manifests shared with
// Server. The legacy host layout retains its fixed root manifest for existing
// development tooling.
func NewPublicJournalStore(deploymentRoot string) (*JournalStore, error) {
	return newJournalStore(deploymentRoot, true)
}

func newJournalStore(deploymentRoot string, manifestCache bool) (*JournalStore, error) {
	if strings.TrimSpace(deploymentRoot) == "" {
		return nil, fmt.Errorf("deployment root is required")
	}
	abs, err := filepath.Abs(deploymentRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve deployment root: %w", err)
	}
	abs = filepath.Clean(abs)
	info, err := os.Lstat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat deployment root: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("deployment root must be a regular directory")
	}
	// Check each private path component independently. Checking only the final
	// directory would allow a pre-created `.lunafox` symlink to redirect the
	// privileged journal outside the deployment root.
	privateRoot := filepath.Join(abs, ".lunafox")
	if err := ensurePrivateDirectory(privateRoot); err != nil {
		return nil, err
	}
	root := filepath.Join(privateRoot, "upgrade")
	if err := ensurePrivateDirectory(root); err != nil {
		return nil, err
	}
	history := filepath.Join(root, HistoryDirectory)
	if err := ensurePrivateDirectory(history); err != nil {
		return nil, err
	}
	receipts := filepath.Join(root, ReceiptDirectory)
	if err := ensurePrivateDirectory(receipts); err != nil {
		return nil, err
	}
	if manifestCache {
		if err := ensurePrivateDirectory(filepath.Join(root, ManifestDirectory)); err != nil {
			return nil, err
		}
	}
	return &JournalStore{deploymentRoot: abs, root: root, manifestCache: manifestCache}, nil
}

func (store *JournalStore) DeploymentRoot() string {
	if store == nil {
		return ""
	}
	return store.deploymentRoot
}

func (store *JournalStore) Directory() string {
	if store == nil {
		return ""
	}
	return store.root
}

func (store *JournalStore) SocketPath() string {
	if store == nil {
		return ""
	}
	return filepath.Join(store.root, SocketFile)
}

func (store *JournalStore) LockPath() string {
	if store == nil {
		return ""
	}
	return filepath.Join(store.root, LockFile)
}

// ManifestPath resolves the only manifest location accepted for a request.
// Public deployments address immutable cached bytes by digest; callers cannot
// provide a path across the privileged socket boundary.
func (store *JournalStore) ManifestPath(digest string) (string, error) {
	if store == nil {
		return "", fmt.Errorf("journal store is nil")
	}
	if !store.manifestCache {
		return filepath.Join(store.deploymentRoot, defaultManifestName), nil
	}
	if err := validateDigest(digest); err != nil {
		return "", err
	}
	return filepath.Join(store.root, ManifestDirectory, strings.TrimPrefix(digest, "sha256:")+".yaml"), nil
}

// SetEventSink installs a best-effort observation callback. The callback is
// intentionally separate from persistence so a disconnected Server cannot
// prevent the host from recording a checkpoint or continuing an upgrade.
func (store *JournalStore) SetEventSink(sink EventSink) {
	if store == nil {
		return
	}
	store.eventMu.Lock()
	store.eventSink = sink
	if sink != nil {
		if store.eventQueue == nil {
			// A bounded queue preserves checkpoint order while ensuring a stalled
			// Server cannot block fsync/rename or consume unbounded memory.
			store.eventQueue = make(chan JournalEvent, 128)
		}
		queue := store.eventQueue
		store.eventWorker.Do(func() { go store.deliverEvents(queue) })
	}
	store.eventMu.Unlock()
}

func (store *JournalStore) publish(event JournalEvent) {
	if store == nil {
		return
	}
	if err := event.Validate(); err != nil {
		return
	}
	store.eventMu.Lock()
	defer store.eventMu.Unlock()
	if store.eventSink == nil {
		return
	}
	if store.eventQueue == nil {
		store.eventQueue = make(chan JournalEvent, 128)
	}
	queue := store.eventQueue
	store.eventWorker.Do(func() { go store.deliverEvents(queue) })
	// Event delivery is deliberately detached from the write path. If the
	// bounded queue is full, the durable journal remains the source of truth and
	// this best-effort observation is dropped. Enqueue while holding eventMu so
	// callers that serialize their journal writes cannot publish observations in
	// a different order.
	select {
	case queue <- event:
	default:
	}
}

func (store *JournalStore) deliverEvents(queue <-chan JournalEvent) {
	for event := range queue {
		store.eventMu.RLock()
		sink := store.eventSink
		store.eventMu.RUnlock()
		if sink == nil {
			continue
		}
		// The observation path is intentionally outside the privileged decision
		// path. A buggy control-plane adapter must not panic the host upgrader or
		// turn a durable checkpoint into an implicit execution failure.
		func() {
			defer func() { _ = recover() }()
			_ = sink.Publish(context.Background(), event)
		}()
	}
}

func (store *JournalStore) Save(journal Journal) error {
	if store == nil {
		return fmt.Errorf("journal store is nil")
	}
	if err := journal.Validate(); err != nil {
		return fmt.Errorf("validate journal: %w", err)
	}
	store.writeMu.Lock()
	err := store.saveLocked(journal)
	if err == nil {
		store.publish(journalEventFor(journal))
	}
	store.writeMu.Unlock()
	return err
}

func (store *JournalStore) saveLocked(journal Journal) error {
	if err := store.validatePrivateLayout(); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return fmt.Errorf("encode journal: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := atomicWrite(store.currentPath(), encoded, 0o600); err != nil {
		return fmt.Errorf("write current journal: %w", err)
	}
	if err := atomicWrite(store.historyPath(journal.OperationID), encoded, 0o600); err != nil {
		return fmt.Errorf("write operation journal: %w", err)
	}
	return nil
}

func journalEventFor(journal Journal) JournalEvent {
	return JournalEvent{
		OperationID:       journal.OperationID,
		ManifestDigest:    journal.ManifestDigest,
		Stage:             journal.Stage,
		UpdatedAt:         journal.UpdatedAt,
		Diagnostic:        journal.Diagnostic,
		MigrationID:       journal.MigrationID,
		MigrationChecksum: journal.MigrationChecksum,
		MigrationStatus:   journal.MigrationStatus,
	}
}

// Checkpoint updates one operation while preserving its original start time.
// It is the only convenience transition used by host executors, which keeps
// phase timestamps and operation/digest binding consistent across retries.
func (store *JournalStore) Checkpoint(operationID, manifestDigest string, stage Stage, diagnostic string, exitCode *int) (Journal, error) {
	if err := validateOperationID(operationID); err != nil {
		return Journal{}, err
	}
	if err := validateDigest(manifestDigest); err != nil {
		return Journal{}, err
	}
	store.writeMu.Lock()
	current, err := store.LoadCurrent()
	if errors.Is(err, ErrJournalNotFound) {
		now := time.Now().UTC()
		current = Journal{SchemaVersion: JournalSchema, OperationID: operationID, ManifestDigest: manifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}
	} else if err != nil {
		store.writeMu.Unlock()
		return Journal{}, err
	}
	if current.OperationID != operationID || current.ManifestDigest != manifestDigest {
		store.writeMu.Unlock()
		return Journal{}, ErrReplayDigestMismatch
	}
	if err := ValidateStageTransition(current.Stage, stage); err != nil {
		store.writeMu.Unlock()
		return Journal{}, err
	}
	if err := ValidateDiagnostic(diagnostic); err != nil {
		store.writeMu.Unlock()
		return Journal{}, err
	}
	previousStage := current.Stage
	now := time.Now().UTC()
	current.Stage = stage
	current.UpdatedAt = now
	current.Diagnostic = diagnostic
	current.ExitCode = exitCode
	if IsTerminal(stage) {
		current.RepairStage = repairStageFor(previousStage)
		current.CompletedAt = &now
	} else {
		current.RepairStage = ""
		current.CompletedAt = nil
	}
	if err := store.saveLocked(current); err != nil {
		store.writeMu.Unlock()
		return Journal{}, err
	}
	store.publish(journalEventFor(current))
	store.writeMu.Unlock()
	return current, nil
}

// SetMigration records the reviewed migration identity and its durable state
// without changing the execution stage. The identity is copied from the fixed
// release manifest by the Compose executor; callers cannot supply SQL or a
// migration path through this method.
func (store *JournalStore) SetMigration(operationID, manifestDigest, migrationID, checksum, status string) (Journal, error) {
	if store == nil {
		return Journal{}, fmt.Errorf("journal store is nil")
	}
	if err := validateOperationID(operationID); err != nil {
		return Journal{}, err
	}
	if err := validateDigest(manifestDigest); err != nil {
		return Journal{}, err
	}
	if status == "" {
		status = MigrationStatusNotStarted
	}
	switch status {
	case MigrationStatusNotStarted, MigrationStatusRunning, MigrationStatusSucceeded, MigrationStatusFailed, MigrationStatusUnknown:
	default:
		return Journal{}, fmt.Errorf("unsupported migration status %q", status)
	}
	if migrationID != "" && !validJournalToken(migrationID) {
		return Journal{}, fmt.Errorf("migrationId is not canonical")
	}
	if checksum != "" && !digestPattern.MatchString(checksum) {
		return Journal{}, fmt.Errorf("migrationChecksum is not canonical")
	}
	if status != MigrationStatusNotStarted && (migrationID == "" || checksum == "") {
		return Journal{}, fmt.Errorf("migration identity and checksum are required for status %q", status)
	}
	store.writeMu.Lock()
	current, err := store.LoadCurrent()
	if err != nil {
		store.writeMu.Unlock()
		return Journal{}, err
	}
	if current.OperationID != operationID || current.ManifestDigest != manifestDigest {
		store.writeMu.Unlock()
		return Journal{}, ErrReplayDigestMismatch
	}
	if current.MigrationID != "" && current.MigrationID != migrationID {
		store.writeMu.Unlock()
		return Journal{}, ErrMigrationIdentityMismatch
	}
	if current.MigrationChecksum != "" && current.MigrationChecksum != checksum {
		store.writeMu.Unlock()
		return Journal{}, ErrMigrationIdentityMismatch
	}
	if err := ValidateMigrationTransition(current.MigrationStatus, status); err != nil {
		store.writeMu.Unlock()
		return Journal{}, err
	}
	current.MigrationID = migrationID
	current.MigrationChecksum = checksum
	current.MigrationStatus = status
	current.UpdatedAt = time.Now().UTC()
	if err := store.saveLocked(current); err != nil {
		store.writeMu.Unlock()
		return Journal{}, err
	}
	store.publish(journalEventFor(current))
	store.writeMu.Unlock()
	return current, nil
}

func validJournalToken(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("._/-", r) {
			continue
		}
		return false
	}
	return true
}

// ValidateMigrationTransition is intentionally stricter than the stage graph.
// A migration outcome is a one-way evidence record: once the command has
// succeeded, failed, or become uncertain, a replay cannot rewrite it as a
// different result or silently run the migration again.
func ValidateMigrationTransition(from, to string) error {
	if from == "" {
		from = MigrationStatusNotStarted
	}
	if to == "" {
		to = MigrationStatusNotStarted
	}
	valid := func(value string) bool {
		switch value {
		case MigrationStatusNotStarted, MigrationStatusRunning, MigrationStatusSucceeded, MigrationStatusFailed, MigrationStatusUnknown:
			return true
		default:
			return false
		}
	}
	if !valid(from) || !valid(to) {
		return fmt.Errorf("%w: unsupported migration status %q -> %q", ErrMigrationTransition, from, to)
	}
	if from == to {
		return nil
	}
	switch from {
	case MigrationStatusNotStarted:
		if to == MigrationStatusRunning {
			return nil
		}
	case MigrationStatusRunning:
		if to == MigrationStatusSucceeded || to == MigrationStatusFailed || to == MigrationStatusUnknown {
			return nil
		}
	}
	return fmt.Errorf("%w: %s -> %s", ErrMigrationTransition, from, to)
}

// ResetForRepair prepares one terminal operation for an explicit repair
// request. It intentionally bypasses the normal forward-only stage transition
// because the terminal checkpoint is the boundary being reopened. The target
// digest and operation identity remain unchanged; migration-aware RepairStage
// prevents a repair from silently re-running an uncertain migration.
func (store *JournalStore) ResetForRepair(operationID, manifestDigest string) (Journal, error) {
	if store == nil {
		return Journal{}, fmt.Errorf("journal store is nil")
	}
	if err := validateOperationID(operationID); err != nil {
		return Journal{}, err
	}
	if err := validateDigest(manifestDigest); err != nil {
		return Journal{}, err
	}
	store.writeMu.Lock()
	current, currentErr := store.LoadCurrent()
	if currentErr != nil && !errors.Is(currentErr, ErrJournalNotFound) {
		store.writeMu.Unlock()
		return Journal{}, currentErr
	}
	history, historyErr := store.LoadOperation(operationID)
	if historyErr != nil && !errors.Is(historyErr, ErrJournalNotFound) {
		store.writeMu.Unlock()
		return Journal{}, historyErr
	}
	if historyErr == nil && history.ManifestDigest != manifestDigest {
		store.writeMu.Unlock()
		return Journal{}, ErrReplayDigestMismatch
	}
	if currentErr == nil && current.OperationID != operationID {
		// A different current operation may represent a newer deployment. Do not
		// let an old history file replace it through a repair request.
		store.writeMu.Unlock()
		return Journal{}, ErrRepairNotAllowed
	}
	base := history
	if currentErr == nil {
		base = current
	}
	if historyErr == nil && currentErr == nil && current.UpdatedAt.Before(history.UpdatedAt) {
		// Save writes current before history. A newer history file is therefore
		// treated as an externally repaired/tampered state and is not trusted.
		store.writeMu.Unlock()
		return Journal{}, ErrJournalCorrupt
	}
	if base.OperationID != operationID || base.ManifestDigest != manifestDigest || !IsTerminal(base.Stage) || base.Stage == StageSucceeded {
		store.writeMu.Unlock()
		return Journal{}, ErrRepairNotAllowed
	}
	nextStage := base.RepairStage
	if nextStage == "" {
		if base.Stage == StageNeedsRecovery || base.Stage == StageNeedsAttention {
			nextStage = StageRestarting
		} else {
			nextStage = StageQueued
		}
	}
	if !validStage(nextStage) || IsTerminal(nextStage) {
		store.writeMu.Unlock()
		return Journal{}, ErrRepairNotAllowed
	}
	now := time.Now().UTC()
	base.Stage = nextStage
	base.RepairStage = ""
	base.UpdatedAt = now
	base.CompletedAt = nil
	base.ExitCode = nil
	base.Diagnostic = ""
	if err := store.saveLocked(base); err != nil {
		store.writeMu.Unlock()
		return Journal{}, err
	}
	store.publish(journalEventFor(base))
	store.writeMu.Unlock()
	return base, nil
}

func (store *JournalStore) LoadCurrent() (Journal, error) {
	return store.load(store.currentPath())
}

func (store *JournalStore) LoadOperation(operationID string) (Journal, error) {
	if err := validateOperationID(operationID); err != nil {
		return Journal{}, err
	}
	return store.load(store.historyPath(operationID))
}

func (store *JournalStore) ReceiptPath(operationID string) (string, error) {
	if err := validateOperationID(operationID); err != nil {
		return "", err
	}
	return filepath.Join(store.root, ReceiptDirectory, operationID+".json"), nil
}

func (store *JournalStore) ComposeOverridePath(operationID string) (string, error) {
	if store == nil {
		return "", fmt.Errorf("journal store is nil")
	}
	if err := validateOperationID(operationID); err != nil {
		return "", err
	}
	return filepath.Join(store.root, HistoryDirectory, operationID+OverrideFileSuffix), nil
}

func (store *JournalStore) SaveReceipt(receipt Receipt) error {
	if store == nil {
		return fmt.Errorf("journal store is nil")
	}
	if err := receipt.Validate(); err != nil {
		return fmt.Errorf("validate deployment receipt: %w", err)
	}
	store.writeMu.Lock()
	if err := store.validatePrivateLayout(); err != nil {
		store.writeMu.Unlock()
		return err
	}
	current, currentErr := store.LoadCurrent()
	if currentErr != nil {
		store.writeMu.Unlock()
		if errors.Is(currentErr, ErrJournalNotFound) {
			return ErrReceiptMismatch
		}
		return currentErr
	}
	if current.OperationID != receipt.OperationID || current.ManifestDigest != receipt.ManifestDigest {
		store.writeMu.Unlock()
		return ErrReceiptMismatch
	}
	history, historyErr := store.LoadOperation(receipt.OperationID)
	if historyErr != nil {
		store.writeMu.Unlock()
		if errors.Is(historyErr, ErrJournalNotFound) {
			return ErrReceiptMismatch
		}
		return historyErr
	}
	if err := validateJournalPair(current, history, receipt.OperationID, receipt.ManifestDigest); err != nil {
		store.writeMu.Unlock()
		return err
	}
	if err := validateReceiptTime(receipt, current, time.Now().UTC()); err != nil {
		store.writeMu.Unlock()
		return err
	}
	path, err := store.ReceiptPath(receipt.OperationID)
	if err != nil {
		store.writeMu.Unlock()
		return err
	}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		store.writeMu.Unlock()
		return fmt.Errorf("encode deployment receipt: %w", err)
	}
	if err := atomicWrite(path, append(encoded, '\n'), 0o600); err != nil {
		store.writeMu.Unlock()
		return err
	}
	event := JournalEvent{
		OperationID:    receipt.OperationID,
		ManifestDigest: receipt.ManifestDigest,
		UpdatedAt:      receipt.CompletedAt,
		Receipt:        cloneReceipt(&receipt),
	}
	event.Stage = current.Stage
	event.MigrationID = current.MigrationID
	event.MigrationChecksum = current.MigrationChecksum
	event.MigrationStatus = current.MigrationStatus
	store.publish(event)
	store.writeMu.Unlock()
	return nil
}

func (store *JournalStore) LoadReceipt(operationID string) (Receipt, error) {
	if store == nil {
		return Receipt{}, fmt.Errorf("journal store is nil")
	}
	if err := store.validatePrivateLayout(); err != nil {
		return Receipt{}, fmt.Errorf("%w: %v", ErrJournalCorrupt, err)
	}
	path, err := store.ReceiptPath(operationID)
	if err != nil {
		return Receipt{}, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Receipt{}, ErrJournalNotFound
		}
		return Receipt{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return Receipt{}, ErrJournalCorrupt
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Receipt{}, ErrJournalNotFound
		}
		return Receipt{}, err
	}
	var receipt Receipt
	if err := decodeStrict(data, &receipt); err != nil {
		return Receipt{}, fmt.Errorf("%w: %v", ErrJournalCorrupt, err)
	}
	if err := receipt.Validate(); err != nil {
		return Receipt{}, fmt.Errorf("%w: %v", ErrJournalCorrupt, err)
	}
	return receipt, nil
}

// ReconcileReceipt verifies that a host completion receipt belongs to the
// requested operation and target digest. It returns both artifacts for the
// Server to compare; it never upgrades the journal to succeeded because
// Agent, API and database evidence are outside the host receipt's authority.
func (store *JournalStore) ReconcileReceipt(operationID, manifestDigest string) (Journal, Receipt, error) {
	if store == nil {
		return Journal{}, Receipt{}, fmt.Errorf("journal store is nil")
	}
	if err := validateOperationID(operationID); err != nil {
		return Journal{}, Receipt{}, err
	}
	if err := validateDigest(manifestDigest); err != nil {
		return Journal{}, Receipt{}, err
	}
	store.writeMu.Lock()
	defer store.writeMu.Unlock()
	return store.reconcileReceiptLocked(operationID, manifestDigest)
}

// reconcileReceiptLocked reads all three durable artifacts under the same
// write lock. saveLocked writes current.json before the per-operation history
// copy, so a crash can leave a stale history snapshot; that snapshot is safe to
// tolerate only when its identity and timestamp still point at the same
// operation. A newer or differently-identified history file is a replay fence.
func (store *JournalStore) reconcileReceiptLocked(operationID, manifestDigest string) (Journal, Receipt, error) {
	receipt, err := store.LoadReceipt(operationID)
	if err != nil {
		return Journal{}, Receipt{}, err
	}
	if receipt.OperationID != operationID || receipt.ManifestDigest != manifestDigest {
		return Journal{}, Receipt{}, ErrReceiptMismatch
	}

	current, currentErr := store.LoadCurrent()
	history, historyErr := store.LoadOperation(operationID)
	if currentErr != nil && !errors.Is(currentErr, ErrJournalNotFound) {
		return Journal{}, Receipt{}, currentErr
	}
	if historyErr != nil && !errors.Is(historyErr, ErrJournalNotFound) {
		return Journal{}, Receipt{}, historyErr
	}
	if errors.Is(currentErr, ErrJournalNotFound) {
		if errors.Is(historyErr, ErrJournalNotFound) {
			return Journal{}, Receipt{}, ErrReceiptMismatch
		}
		current = history
	} else if errors.Is(historyErr, ErrJournalNotFound) {
		// A receipt is only written after saveLocked has produced both journal
		// copies. Treat a missing history file as an incomplete/tampered handoff
		// instead of silently accepting a single artifact.
		return Journal{}, Receipt{}, ErrReceiptMismatch
	}

	if err := validateJournalPair(current, history, operationID, manifestDigest); err != nil {
		return Journal{}, Receipt{}, err
	}
	if err := validateReceiptTime(receipt, current, time.Now().UTC()); err != nil {
		return Journal{}, Receipt{}, err
	}
	return current, receipt, nil
}

func validateJournalPair(current, history Journal, operationID, manifestDigest string) error {
	if current.OperationID != operationID || current.ManifestDigest != manifestDigest ||
		history.OperationID != operationID || history.ManifestDigest != manifestDigest {
		return ErrReceiptMismatch
	}
	if !current.StartedAt.Equal(history.StartedAt) || history.UpdatedAt.After(current.UpdatedAt) {
		return ErrReceiptMismatch
	}
	if history.UpdatedAt.Equal(current.UpdatedAt) && !journalsSnapshotEqual(current, history) {
		return ErrReceiptMismatch
	}
	return nil
}

func validateReceiptTime(receipt Receipt, journal Journal, now time.Time) error {
	if receipt.CompletedAt.Before(journal.StartedAt) {
		return ErrReceiptMismatch
	}
	if receipt.CompletedAt.After(now.Add(maxReceiptFutureSkew)) {
		return ErrReceiptMismatch
	}
	if IsTerminal(journal.Stage) && journal.CompletedAt != nil && receipt.CompletedAt.After(*journal.CompletedAt) {
		return ErrReceiptMismatch
	}
	return nil
}

func journalsSnapshotEqual(left, right Journal) bool {
	if left.SchemaVersion != right.SchemaVersion ||
		left.OperationID != right.OperationID ||
		left.ManifestDigest != right.ManifestDigest ||
		left.Stage != right.Stage ||
		left.RepairStage != right.RepairStage ||
		left.MigrationID != right.MigrationID ||
		left.MigrationChecksum != right.MigrationChecksum ||
		left.MigrationStatus != right.MigrationStatus ||
		left.ExitCode == nil != (right.ExitCode == nil) ||
		left.Diagnostic != right.Diagnostic ||
		!left.StartedAt.Equal(right.StartedAt) ||
		!left.UpdatedAt.Equal(right.UpdatedAt) {
		return false
	}
	if left.ExitCode != nil && *left.ExitCode != *right.ExitCode {
		return false
	}
	if left.CompletedAt == nil != (right.CompletedAt == nil) {
		return false
	}
	return left.CompletedAt == nil || left.CompletedAt.Equal(*right.CompletedAt)
}

// VerifyReceiptBinding is a compact form for callers that only need the
// operation/digest integrity check.
func (store *JournalStore) VerifyReceiptBinding(operationID, manifestDigest string) error {
	_, _, err := store.ReconcileReceipt(operationID, manifestDigest)
	return err
}

func cloneReceipt(receipt *Receipt) *Receipt {
	if receipt == nil {
		return nil
	}
	copy := *receipt
	copy.Services = append([]string(nil), receipt.Services...)
	copy.ObservedImages = make(map[string]string, len(receipt.ObservedImages))
	for service, digest := range receipt.ObservedImages {
		copy.ObservedImages[service] = digest
	}
	return &copy
}

func (store *JournalStore) currentPath() string {
	return filepath.Join(store.root, CurrentStateFile)
}

func (store *JournalStore) historyPath(operationID string) string {
	return filepath.Join(store.root, HistoryDirectory, operationID+".json")
}

func (store *JournalStore) load(path string) (Journal, error) {
	if store == nil {
		return Journal{}, fmt.Errorf("journal store is nil")
	}
	if err := store.validatePrivateLayout(); err != nil {
		return Journal{}, fmt.Errorf("%w: %v", ErrJournalCorrupt, err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Journal{}, ErrJournalNotFound
		}
		return Journal{}, fmt.Errorf("stat journal: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return Journal{}, ErrJournalCorrupt
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Journal{}, fmt.Errorf("read journal: %w", err)
	}
	var journal Journal
	if err := decodeStrict(data, &journal); err != nil {
		return Journal{}, fmt.Errorf("%w: %v", ErrJournalCorrupt, err)
	}
	if err := journal.Validate(); err != nil {
		return Journal{}, fmt.Errorf("%w: %v", ErrJournalCorrupt, err)
	}
	return journal, nil
}

// validatePrivateLayout is intentionally strict after construction. The
// constructor may establish the private directories, but later reads/writes
// must refuse a chmod'ed directory or symlink instead of silently repairing a
// privileged path that may have been replaced by another process.
func (store *JournalStore) validatePrivateLayout() error {
	if store == nil || store.root == "" {
		return fmt.Errorf("journal store is not configured")
	}
	for _, path := range []string{
		store.root,
		filepath.Join(store.root, HistoryDirectory),
		filepath.Join(store.root, ReceiptDirectory),
	} {
		if err := validatePrivateDirectoryMode(path); err != nil {
			return err
		}
	}
	return nil
}

func validatePrivateDirectoryMode(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return fmt.Errorf("private upgrader directory must be a 0700 directory")
	}
	return nil
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return err
	}
	return nil
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	base := filepath.Base(path)
	temporary, err := os.OpenFile(filepath.Join(directory, "."+base+".tmp"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			// A leftover temporary file is not safe to reuse. Remove only this
			// exact bounded path and retry with a unique temporary name below.
			temporary, err = os.CreateTemp(directory, "."+base+".tmp-")
			if err != nil {
				return err
			}
			if chmodErr := temporary.Chmod(mode); chmodErr != nil {
				_ = temporary.Close()
				_ = os.Remove(temporary.Name())
				return chmodErr
			}
		} else {
			return err
		}
	}
	temporaryName := temporary.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryName)
		}
	}()
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return err
	}
	removeTemporary = false
	// The temporary file was created with the requested mode. Avoid chmod'ing
	// the destination after rename: a hostile replacement between those two
	// syscalls could make chmod follow a symlink outside the private directory.
	return syncDirectory(directory)
}

func ensurePrivateDirectory(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("upgrader directory path is required")
	}
	info, err := os.Lstat(path)
	if err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("upgrader path must be a regular directory")
		}
		if err := os.Chmod(path, 0o700); err != nil {
			return fmt.Errorf("secure upgrader directory: %w", err)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat upgrader directory: %w", err)
	}
	// Create one component at a time after inspecting the parent. MkdirAll can
	// follow a hostile pre-created symlink in a privileged deployment root.
	parent := filepath.Dir(path)
	parentInfo, parentErr := os.Lstat(parent)
	if parentErr != nil {
		return fmt.Errorf("stat upgrader parent: %w", parentErr)
	}
	if !parentInfo.IsDir() || parentInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("upgrader parent must be a regular directory")
	}
	if err := os.Mkdir(path, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("create upgrader directory: %w", err)
	}
	info, err = os.Lstat(path)
	if err != nil {
		return fmt.Errorf("stat upgrader directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("upgrader path must be a regular directory")
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return fmt.Errorf("secure upgrader directory: %w", err)
	}
	return nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return err
	}
	return directory.Close()
}
