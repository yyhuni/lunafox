package upgrader

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const testManifestDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestJournalStoreUsesPrivateAtomicFiles(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	j := Journal{SchemaVersion: JournalSchema, OperationID: "op-1", ManifestDigest: testManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}
	if err := store.Save(j); err != nil {
		t.Fatal(err)
	}
	rootInfo, err := os.Stat(store.Directory())
	if err != nil {
		t.Fatal(err)
	}
	if got := rootInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("journal directory mode = %o, want 700", got)
	}
	for _, path := range []string{filepath.Join(store.Directory(), CurrentStateFile), filepath.Join(store.Directory(), HistoryDirectory, "op-1.json")} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("journal file %s mode = %o, want 600", path, got)
		}
	}
	got, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if got.OperationID != j.OperationID || got.ManifestDigest != j.ManifestDigest {
		t.Fatalf("loaded journal = %#v, want %#v", got, j)
	}
}

func TestJournalStoreRejectsTruncatedAndUnknownJSON(t *testing.T) {
	store := newTestStore(t)
	path := filepath.Join(store.Directory(), CurrentStateFile)
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"operationId":"op-1"`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadCurrent(); !errors.Is(err, ErrJournalCorrupt) {
		t.Fatalf("truncated journal error = %v, want ErrJournalCorrupt", err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"operationId":"op-1","manifestDigest":"`+testManifestDigest+`","stage":"queued","startedAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","unexpected":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadCurrent(); !errors.Is(err, ErrJournalCorrupt) {
		t.Fatalf("unknown-field journal error = %v, want ErrJournalCorrupt", err)
	}
}

func TestJournalStoreRejectsCredentialDiagnostics(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	j := Journal{SchemaVersion: JournalSchema, OperationID: "op-1", ManifestDigest: testManifestDigest, Stage: StageFailed, StartedAt: now, UpdatedAt: now, CompletedAt: &now, Diagnostic: "Authorization: Bearer eyJsecret"}
	if err := store.Save(j); err == nil || !strings.Contains(err.Error(), "forbidden credential") {
		t.Fatalf("Save() error = %v, want credential rejection", err)
	}
}

func TestJournalStoreRejectsUnsafeOperationID(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	for _, operationID := range []string{"../escape", "", "op/child", " op-1"} {
		j := Journal{SchemaVersion: JournalSchema, OperationID: operationID, ManifestDigest: testManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}
		if err := store.Save(j); err == nil {
			t.Fatalf("Save(%q) unexpectedly succeeded", operationID)
		}
	}
}

func TestFileLockAllowsOnlyOneOwner(t *testing.T) {
	store := newTestStore(t)
	first, err := acquireFileLock(store.LockPath())
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := acquireFileLock(store.LockPath())
	if !errors.Is(err, ErrAlreadyRunning) {
		if second != nil {
			_ = second.Close()
		}
		t.Fatalf("second lock error = %v, want ErrAlreadyRunning", err)
	}
}

func TestFileLockRejectsSymlink(t *testing.T) {
	store := newTestStore(t)
	target := filepath.Join(t.TempDir(), "outside-lock")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, store.LockPath()); err != nil {
		t.Fatal(err)
	}
	if _, err := acquireFileLock(store.LockPath()); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("acquireFileLock() error = %v, want symlink rejection", err)
	}
}

func TestJournalStoreConcurrentSavesRemainValid(t *testing.T) {
	store := newTestStore(t)
	var wait sync.WaitGroup
	for i := 0; i < 8; i++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			now := time.Now().UTC().Add(time.Duration(index) * time.Microsecond)
			_ = store.Save(Journal{SchemaVersion: JournalSchema, OperationID: "op-1", ManifestDigest: testManifestDigest, Stage: StageUpdating, StartedAt: now, UpdatedAt: now})
		}(i)
	}
	wait.Wait()
	if _, err := store.LoadCurrent(); err != nil {
		t.Fatalf("LoadCurrent() after concurrent writes = %v", err)
	}
}

func TestJournalStoreRejectsWeakPrivateFileModes(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	journal := Journal{SchemaVersion: JournalSchema, OperationID: "op-mode", ManifestDigest: testManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}
	currentPath := filepath.Join(store.Directory(), CurrentStateFile)
	if err := os.Chmod(currentPath, 0o400); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadCurrent(); !errors.Is(err, ErrJournalCorrupt) {
		t.Fatalf("weak journal mode error = %v, want ErrJournalCorrupt", err)
	}
}

func TestJournalStoreRejectsUnsafePrivateDirectoryModes(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	journal := Journal{SchemaVersion: JournalSchema, OperationID: "op-directory-mode", ManifestDigest: testManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}

	for _, directory := range []string{
		filepath.Join(store.Directory(), HistoryDirectory),
		filepath.Join(store.Directory(), ReceiptDirectory),
	} {
		if err := os.Chmod(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := store.LoadCurrent(); !errors.Is(err, ErrJournalCorrupt) {
			t.Fatalf("LoadCurrent() with directory %s mode = %v, want ErrJournalCorrupt", directory, err)
		}
		if err := store.Save(journal); err == nil || !strings.Contains(err.Error(), "0700") {
			t.Fatalf("Save() with directory %s mode = %v, want private-directory rejection", directory, err)
		}
		if err := os.Chmod(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
}

func TestJournalStoreRejectsPrivateDirectorySymlink(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	journal := Journal{SchemaVersion: JournalSchema, OperationID: "op-directory-link", ManifestDigest: testManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(store.Directory(), ReceiptDirectory)
	outside := t.TempDir()
	if err := os.RemoveAll(directory); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, directory); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadCurrent(); !errors.Is(err, ErrJournalCorrupt) {
		t.Fatalf("LoadCurrent() with receipt symlink = %v, want ErrJournalCorrupt", err)
	}
	if err := store.Save(journal); err == nil || !strings.Contains(err.Error(), "0700 directory") {
		t.Fatalf("Save() with receipt symlink = %v, want symlink rejection", err)
	}
}

func TestJournalStoreResetForRepairPreservesTargetAndUsesSafeStage(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	terminal := Journal{
		SchemaVersion: JournalSchema, OperationID: "op-repair", ManifestDigest: testManifestDigest,
		Stage: StageNeedsRecovery, RepairStage: StageRestarting, StartedAt: now, UpdatedAt: now, CompletedAt: &now,
	}
	if err := store.Save(terminal); err != nil {
		t.Fatal(err)
	}
	repaired, err := store.ResetForRepair(terminal.OperationID, terminal.ManifestDigest)
	if err != nil {
		t.Fatal(err)
	}
	if repaired.Stage != StageRestarting || repaired.RepairStage != "" || repaired.CompletedAt != nil {
		t.Fatalf("repaired journal = %#v", repaired)
	}
	if repaired.OperationID != terminal.OperationID || repaired.ManifestDigest != terminal.ManifestDigest || !repaired.StartedAt.Equal(terminal.StartedAt) {
		t.Fatalf("repair changed immutable identity/timestamp = %#v", repaired)
	}
}

func TestJournalStoreDoesNotResetSucceededOperation(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	if err := store.Save(Journal{
		SchemaVersion: JournalSchema, OperationID: "op-success", ManifestDigest: testManifestDigest,
		Stage: StageSucceeded, StartedAt: now, UpdatedAt: now, CompletedAt: &now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ResetForRepair("op-success", testManifestDigest); !errors.Is(err, ErrRepairNotAllowed) {
		t.Fatalf("ResetForRepair() error = %v, want ErrRepairNotAllowed", err)
	}
}

func TestJournalStoreReconcilesReceiptByOperationAndDigest(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	journal := Journal{SchemaVersion: JournalSchema, OperationID: "op-receipt", ManifestDigest: testManifestDigest, Stage: StageVerifying, StartedAt: now, UpdatedAt: now}
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}
	receipt := Receipt{
		SchemaVersion: JournalSchema, OperationID: journal.OperationID, ManifestDigest: journal.ManifestDigest,
		CompletedAt: now.Add(time.Second), Services: []string{"server"}, ObservedImages: map[string]string{"server": testManifestDigest},
	}
	if err := store.SaveReceipt(receipt); err != nil {
		t.Fatal(err)
	}
	gotJournal, gotReceipt, err := store.ReconcileReceipt(journal.OperationID, journal.ManifestDigest)
	if err != nil {
		t.Fatal(err)
	}
	if gotJournal.OperationID != journal.OperationID || gotReceipt.OperationID != receipt.OperationID {
		t.Fatalf("reconciled artifacts = journal=%#v receipt=%#v", gotJournal, gotReceipt)
	}
	if err := store.VerifyReceiptBinding(journal.OperationID, "sha256:"+strings.Repeat("b", 64)); !errors.Is(err, ErrReceiptMismatch) {
		t.Fatalf("VerifyReceiptBinding() error = %v, want ErrReceiptMismatch", err)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageVerifying {
		t.Fatalf("receipt reconciliation changed journal stage to %q, want verifying", current.Stage)
	}
}

func TestJournalStoreReceiptTimeBoundary(t *testing.T) {
	store := newTestStore(t)
	startedAt := time.Now().UTC()
	journal := Journal{
		SchemaVersion: JournalSchema, OperationID: "op-receipt-time", ManifestDigest: testManifestDigest,
		Stage: StageVerifying, StartedAt: startedAt, UpdatedAt: startedAt,
	}
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}
	newReceipt := func(completedAt time.Time) Receipt {
		return Receipt{
			SchemaVersion: JournalSchema, OperationID: journal.OperationID, ManifestDigest: journal.ManifestDigest,
			CompletedAt: completedAt, Services: []string{"server"}, ObservedImages: map[string]string{"server": testManifestDigest},
		}
	}
	if err := store.SaveReceipt(newReceipt(startedAt.Add(-time.Second))); !errors.Is(err, ErrReceiptMismatch) {
		t.Fatalf("receipt before journal start error = %v, want ErrReceiptMismatch", err)
	}
	if err := store.SaveReceipt(newReceipt(time.Now().UTC().Add(maxReceiptFutureSkew + time.Second))); !errors.Is(err, ErrReceiptMismatch) {
		t.Fatalf("receipt beyond future skew error = %v, want ErrReceiptMismatch", err)
	}
	if err := store.SaveReceipt(newReceipt(time.Now().UTC().Add(maxReceiptFutureSkew - time.Second))); err != nil {
		t.Fatalf("receipt within future skew error = %v", err)
	}
}

func TestJournalStoreReceiptCannotOutliveTerminalJournal(t *testing.T) {
	store := newTestStore(t)
	startedAt := time.Now().UTC()
	completedAt := startedAt.Add(time.Minute)
	journal := Journal{
		SchemaVersion: JournalSchema, OperationID: "op-receipt-terminal", ManifestDigest: testManifestDigest,
		Stage: StageFailed, StartedAt: startedAt, UpdatedAt: completedAt, CompletedAt: &completedAt,
	}
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}
	receipt := Receipt{
		SchemaVersion: JournalSchema, OperationID: journal.OperationID, ManifestDigest: journal.ManifestDigest,
		CompletedAt: completedAt.Add(time.Second), Services: []string{"server"}, ObservedImages: map[string]string{"server": testManifestDigest},
	}
	if err := store.SaveReceipt(receipt); !errors.Is(err, ErrReceiptMismatch) {
		t.Fatalf("receipt after terminal completion error = %v, want ErrReceiptMismatch", err)
	}
}

func TestJournalStoreReconcileRejectsFutureReceipt(t *testing.T) {
	store := newTestStore(t)
	startedAt := time.Now().UTC()
	journal := Journal{
		SchemaVersion: JournalSchema, OperationID: "op-reconcile-future", ManifestDigest: testManifestDigest,
		Stage: StageVerifying, StartedAt: startedAt, UpdatedAt: startedAt,
	}
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}
	receipt := Receipt{
		SchemaVersion: JournalSchema, OperationID: journal.OperationID, ManifestDigest: journal.ManifestDigest,
		CompletedAt: time.Now().UTC().Add(maxReceiptFutureSkew + time.Second), Services: []string{"server"},
		ObservedImages: map[string]string{"server": testManifestDigest},
	}
	path, err := store.ReceiptPath(receipt.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(path, append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.ReconcileReceipt(journal.OperationID, journal.ManifestDigest); !errors.Is(err, ErrReceiptMismatch) {
		t.Fatalf("future receipt reconciliation error = %v, want ErrReceiptMismatch", err)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageVerifying {
		t.Fatalf("future receipt changed journal stage to %q, want verifying", current.Stage)
	}
}

func TestJournalStoreReconcileRequiresConsistentHistoryIdentityAndTime(t *testing.T) {
	store := newTestStore(t)
	startedAt := time.Now().UTC()
	journal := Journal{
		SchemaVersion: JournalSchema, OperationID: "op-reconcile-history", ManifestDigest: testManifestDigest,
		Stage: StageVerifying, StartedAt: startedAt, UpdatedAt: startedAt,
	}
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}
	receipt := Receipt{
		SchemaVersion: JournalSchema, OperationID: journal.OperationID, ManifestDigest: journal.ManifestDigest,
		CompletedAt: startedAt.Add(time.Second), Services: []string{"server"}, ObservedImages: map[string]string{"server": testManifestDigest},
	}
	if err := store.SaveReceipt(receipt); err != nil {
		t.Fatal(err)
	}
	historyPath := filepath.Join(store.Directory(), HistoryDirectory, journal.OperationID+".json")
	tests := []struct {
		name   string
		mutate func(*Journal)
	}{
		{
			name: "history starts at a different time",
			mutate: func(history *Journal) {
				history.StartedAt = startedAt.Add(time.Second)
				history.UpdatedAt = history.StartedAt
			},
		},
		{
			name:   "history is newer than current",
			mutate: func(history *Journal) { history.UpdatedAt = startedAt.Add(time.Second) },
		},
		{
			name:   "same timestamp snapshot drifts",
			mutate: func(history *Journal) { history.Stage = StageUpdating },
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			history := journal
			test.mutate(&history)
			encoded, err := json.Marshal(history)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(historyPath, append(encoded, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := store.SaveReceipt(receipt); !errors.Is(err, ErrReceiptMismatch) {
				t.Fatalf("history drift receipt save error = %v, want ErrReceiptMismatch", err)
			}
			if _, _, err := store.ReconcileReceipt(journal.OperationID, journal.ManifestDigest); !errors.Is(err, ErrReceiptMismatch) {
				t.Fatalf("history drift reconciliation error = %v, want ErrReceiptMismatch", err)
			}
			if err := os.WriteFile(historyPath, mustMarshalJournal(t, journal), 0o600); err != nil {
				t.Fatal(err)
			}
		})
	}
	if err := os.Remove(historyPath); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReceipt(receipt); !errors.Is(err, ErrReceiptMismatch) {
		t.Fatalf("missing history receipt save error = %v, want ErrReceiptMismatch", err)
	}
	if _, _, err := store.ReconcileReceipt(journal.OperationID, journal.ManifestDigest); !errors.Is(err, ErrReceiptMismatch) {
		t.Fatalf("missing history reconciliation error = %v, want ErrReceiptMismatch", err)
	}
}

func mustMarshalJournal(t *testing.T, journal Journal) []byte {
	t.Helper()
	encoded, err := json.Marshal(journal)
	if err != nil {
		t.Fatal(err)
	}
	return append(encoded, '\n')
}

func TestJournalStoreRejectsReceiptWhenCurrentOperationChanges(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	first := Journal{SchemaVersion: JournalSchema, OperationID: "op-receipt-old", ManifestDigest: testManifestDigest, Stage: StageVerifying, StartedAt: now, UpdatedAt: now}
	if err := store.Save(first); err != nil {
		t.Fatal(err)
	}
	receipt := Receipt{
		SchemaVersion: JournalSchema, OperationID: first.OperationID, ManifestDigest: first.ManifestDigest,
		CompletedAt: now.Add(time.Second), Services: []string{"server"}, ObservedImages: map[string]string{"server": testManifestDigest},
	}
	if err := store.SaveReceipt(receipt); err != nil {
		t.Fatal(err)
	}
	second := Journal{SchemaVersion: JournalSchema, OperationID: "op-receipt-new", ManifestDigest: testManifestDigest, Stage: StageQueued, StartedAt: now.Add(2 * time.Second), UpdatedAt: now.Add(2 * time.Second)}
	if err := store.Save(second); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.ReconcileReceipt(first.OperationID, first.ManifestDigest); !errors.Is(err, ErrReceiptMismatch) {
		t.Fatalf("ReconcileReceipt() error = %v, want ErrReceiptMismatch after current operation changed", err)
	}
}

func TestJournalStoreRejectsReceiptWhenHistoryIdentityDrifts(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	journal := Journal{SchemaVersion: JournalSchema, OperationID: "op-receipt-history", ManifestDigest: testManifestDigest, Stage: StageVerifying, StartedAt: now, UpdatedAt: now}
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}
	receipt := Receipt{
		SchemaVersion: JournalSchema, OperationID: journal.OperationID, ManifestDigest: journal.ManifestDigest,
		CompletedAt: now.Add(time.Second), Services: []string{"server"}, ObservedImages: map[string]string{"server": testManifestDigest},
	}
	if err := store.SaveReceipt(receipt); err != nil {
		t.Fatal(err)
	}
	historyPath := filepath.Join(store.Directory(), HistoryDirectory, journal.OperationID+".json")
	tampered := Journal{SchemaVersion: JournalSchema, OperationID: journal.OperationID, ManifestDigest: "sha256:" + strings.Repeat("b", 64), Stage: StageVerifying, StartedAt: now, UpdatedAt: now}
	data, err := json.Marshal(tampered)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(historyPath, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.ReconcileReceipt(journal.OperationID, journal.ManifestDigest); !errors.Is(err, ErrReceiptMismatch) {
		t.Fatalf("ReconcileReceipt() error = %v, want ErrReceiptMismatch for drifted history", err)
	}
}

func TestJournalStoreReceiptEventIsBestEffortAndCarriesStage(t *testing.T) {
	store := newTestStore(t)
	events := make(chan JournalEvent, 2)
	store.SetEventSink(EventSinkFunc(func(_ context.Context, event JournalEvent) error {
		events <- event
		return errors.New("server is restarting")
	}))
	now := time.Now().UTC()
	journal := Journal{SchemaVersion: JournalSchema, OperationID: "op-receipt-event", ManifestDigest: testManifestDigest, Stage: StageVerifying, StartedAt: now, UpdatedAt: now}
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}
	receipt := Receipt{
		SchemaVersion: JournalSchema, OperationID: journal.OperationID, ManifestDigest: journal.ManifestDigest,
		CompletedAt: now.Add(time.Second), Services: []string{"server"}, ObservedImages: map[string]string{"server": testManifestDigest},
	}
	if err := store.SaveReceipt(receipt); err != nil {
		t.Fatal(err)
	}
	foundReceiptEvent := false
	deadline := time.After(2 * time.Second)
	for !foundReceiptEvent {
		select {
		case event := <-events:
			if event.Receipt != nil {
				foundReceiptEvent = true
				if event.Stage != StageVerifying || event.Receipt.OperationID != receipt.OperationID {
					t.Fatalf("receipt event = %#v", event)
				}
			}
		case <-deadline:
			t.Fatal("receipt event was not published")
		}
	}
}

func TestJournalStoreMigrationStatusIsMonotonicAndIdentityBound(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	if err := store.Save(Journal{
		SchemaVersion: JournalSchema, OperationID: "op-migration-state", ManifestDigest: testManifestDigest,
		Stage: StageMigrating, StartedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	migrationID := "000002_upgrade"
	checksum := "sha256:" + strings.Repeat("b", 64)
	if _, err := store.SetMigration("op-migration-state", testManifestDigest, migrationID, checksum, MigrationStatusRunning); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetMigration("op-migration-state", testManifestDigest, migrationID, checksum, MigrationStatusSucceeded); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetMigration("op-migration-state", testManifestDigest, migrationID, checksum, MigrationStatusRunning); !errors.Is(err, ErrMigrationTransition) {
		t.Fatalf("backward migration transition = %v, want ErrMigrationTransition", err)
	}
	if _, err := store.SetMigration("op-migration-state", testManifestDigest, "000003_other", checksum, MigrationStatusSucceeded); !errors.Is(err, ErrMigrationIdentityMismatch) {
		t.Fatalf("identity replay = %v, want ErrMigrationIdentityMismatch", err)
	}
}

func TestJournalStoreRejectsOrphanReceipt(t *testing.T) {
	store := newTestStore(t)
	receipt := Receipt{
		SchemaVersion: JournalSchema, OperationID: "op-orphan-receipt", ManifestDigest: testManifestDigest,
		CompletedAt: time.Now().UTC(), Services: []string{"server"}, ObservedImages: map[string]string{"server": testManifestDigest},
	}
	if err := store.SaveReceipt(receipt); !errors.Is(err, ErrReceiptMismatch) {
		t.Fatalf("orphan receipt error = %v, want ErrReceiptMismatch", err)
	}
}

func TestJournalStoreEventSinkPreservesCheckpointOrder(t *testing.T) {
	store := newTestStore(t)
	events := make(chan JournalEvent, 4)
	store.SetEventSink(EventSinkFunc(func(_ context.Context, event JournalEvent) error {
		events <- event
		return nil
	}))
	now := time.Now().UTC()
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: "op-order", ManifestDigest: testManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Checkpoint("op-order", testManifestDigest, StageStopping, "", nil); err != nil {
		t.Fatal(err)
	}
	select {
	case first := <-events:
		if first.Stage != StageQueued {
			t.Fatalf("first event stage = %q, want queued", first.Stage)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first journal event was not delivered")
	}
	select {
	case second := <-events:
		if second.Stage != StageStopping {
			t.Fatalf("second event stage = %q, want stopping", second.Stage)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second journal event was not delivered")
	}
}

func TestJournalStoreEventSinkPanicDoesNotBreakPersistence(t *testing.T) {
	store := newTestStore(t)
	called := make(chan struct{})
	store.SetEventSink(EventSinkFunc(func(context.Context, JournalEvent) error {
		close(called)
		panic("control-plane adapter failed")
	}))
	now := time.Now().UTC()
	journal := Journal{SchemaVersion: JournalSchema, OperationID: "op-event-panic", ManifestDigest: testManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}
	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatal("panicking event sink was not invoked")
	}
	if got, err := store.LoadCurrent(); err != nil || got.OperationID != journal.OperationID {
		t.Fatalf("journal after panicking event sink = %#v, %v", got, err)
	}
}

func newTestStore(t *testing.T) *JournalStore {
	t.Helper()
	// macOS limits filesystem UDS paths to roughly 104 bytes. Keep the test
	// deployment root short so it exercises the real `.lunafox/upgrade` path
	// instead of failing before the protocol is reached.
	root, err := os.MkdirTemp("/tmp", "lfu-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	return store
}
