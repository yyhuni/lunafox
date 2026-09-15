package upgrader

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDaemonAcceptsFixedRequestAndIdempotentlyReplays(t *testing.T) {
	store := newTestStore(t)
	daemon := NewDaemon(store, nil)
	daemon.SetManifestVerifier(func(string, string) error { return nil })
	if err := daemon.Listen(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serveDone := make(chan error, 1)
	go func() { serveDone <- daemon.Serve(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-serveDone:
		case <-time.After(2 * time.Second):
			t.Error("daemon did not stop")
		}
	})
	client := &Client{store: store}
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-1", Action: ActionStart, ManifestDigest: testManifestDigest}
	first, err := client.Send(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Accepted || first.Replayed || first.Journal.Stage != StageQueued {
		t.Fatalf("first response = %#v", first)
	}
	second, err := client.Send(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Accepted || !second.Replayed || second.Journal.OperationID != request.OperationID {
		t.Fatalf("replay response = %#v", second)
	}
}

func TestDaemonRejectsDigestReplayAndArbitraryAction(t *testing.T) {
	store := newTestStore(t)
	daemon := NewDaemon(store, nil)
	daemon.SetManifestVerifier(func(string, string) error { return nil })
	if err := daemon.Listen(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serveDone := make(chan error, 1)
	go func() { serveDone <- daemon.Serve(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-serveDone:
		case <-time.After(2 * time.Second):
			t.Error("daemon did not stop")
		}
	})
	client := &Client{store: store}
	base := Request{SchemaVersion: RequestSchema, OperationID: "op-1", Action: ActionStart, ManifestDigest: testManifestDigest}
	if _, err := client.Send(context.Background(), base); err != nil {
		t.Fatal(err)
	}
	_, err := client.Send(context.Background(), Request{SchemaVersion: RequestSchema, OperationID: "op-1", Action: ActionStart, ManifestDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"})
	if !errors.Is(err, ErrReplayDigestMismatch) && !strings.Contains(err.Error(), ErrReplayDigestMismatch.Error()) {
		t.Fatalf("digest replay error = %v", err)
	}
	_, err = client.Send(context.Background(), Request{SchemaVersion: RequestSchema, OperationID: "op-2", Action: Action("start;touch /tmp/pwned"), ManifestDigest: testManifestDigest})
	if err == nil || !strings.Contains(err.Error(), "unsupported upgrader action") {
		t.Fatalf("arbitrary action error = %v", err)
	}
}

func TestDaemonRejectsResumingNonTerminalHistoryAfterNewerTerminalOperation(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	old := Journal{
		SchemaVersion: JournalSchema, OperationID: "op-old", ManifestDigest: testManifestDigest,
		Stage: StageUpdating, StartedAt: now, UpdatedAt: now,
	}
	if err := store.Save(old); err != nil {
		t.Fatal(err)
	}
	completedAt := now.Add(time.Second)
	newer := Journal{
		SchemaVersion: JournalSchema, OperationID: "op-new", ManifestDigest: testManifestDigest,
		Stage: StageFailed, RepairStage: StageQueued, StartedAt: completedAt, UpdatedAt: completedAt, CompletedAt: &completedAt,
	}
	if err := store.Save(newer); err != nil {
		t.Fatal(err)
	}
	// Restore the old history snapshot while keeping current.json on the newer
	// terminal operation. This models a process interruption between the two
	// journal writes and must not reopen the stale operation.
	oldPath := filepath.Join(store.Directory(), HistoryDirectory, old.OperationID+".json")
	encoded, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldPath, append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	// A nil executor is sufficient here: the acceptance response itself must be
	// rejected before any launch decision can be made.
	daemon := NewDaemon(store, nil)
	daemon.SetManifestVerifier(func(string, string) error { return nil })
	response := daemon.accept(Request{SchemaVersion: RequestSchema, OperationID: old.OperationID, Action: ActionResume, ManifestDigest: old.ManifestDigest})
	if response.Accepted || response.Error != ErrOperationInProgress.Error() {
		t.Fatalf("stale resume response = %#v, want operation-in-progress rejection", response)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.OperationID != newer.OperationID || current.Stage != newer.Stage {
		t.Fatalf("current journal changed after stale resume = %#v, want newer terminal operation", current)
	}
}

func TestDaemonRejectsCorruptJournalBeforeServing(t *testing.T) {
	store := newTestStore(t)
	path := filepath.Join(store.Directory(), CurrentStateFile)
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1`), 0o600); err != nil {
		t.Fatal(err)
	}
	daemon := NewDaemon(store, nil)
	daemon.SetManifestVerifier(func(string, string) error { return nil })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := daemon.Serve(ctx); !errors.Is(err, ErrJournalCorrupt) {
		t.Fatalf("Serve() error = %v, want ErrJournalCorrupt", err)
	}
}

func TestDaemonRejectsRequestsWhenCurrentJournalIsCorrupt(t *testing.T) {
	store := newTestStore(t)
	if err := os.WriteFile(filepath.Join(store.Directory(), CurrentStateFile), []byte(`{"schemaVersion":1`), 0o600); err != nil {
		t.Fatal(err)
	}
	daemon := NewDaemon(store, nil)
	daemon.SetManifestVerifier(func(string, string) error { return nil })
	if err := daemon.Listen(); err != nil {
		t.Fatal(err)
	}
	defer daemon.Close()
	response := daemon.accept(Request{SchemaVersion: RequestSchema, OperationID: "op-corrupt", Action: ActionStart, ManifestDigest: testManifestDigest})
	if response.Accepted || response.Error == "" {
		t.Fatalf("accept response = %#v, want corrupt-journal rejection", response)
	}
	if _, err := store.LoadOperation("op-corrupt"); !errors.Is(err, ErrJournalNotFound) {
		t.Fatalf("request created history despite corrupt current journal: %v", err)
	}
}

func TestDaemonSecondProcessCannotBindSocket(t *testing.T) {
	store := newTestStore(t)
	first := NewDaemon(store, nil)
	if err := first.Listen(); err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second := NewDaemon(store, nil)
	if err := second.Listen(); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second Listen() error = %v, want ErrAlreadyRunning", err)
	}
}

func TestDaemonRejectsSocketPathHijack(t *testing.T) {
	store := newTestStore(t)
	if err := os.WriteFile(store.SocketPath(), []byte("not a socket"), 0o600); err != nil {
		t.Fatal(err)
	}
	daemon := NewDaemon(store, nil)
	if err := daemon.Listen(); err == nil || !strings.Contains(err.Error(), "non-socket") {
		t.Fatalf("Listen() error = %v, want non-socket rejection", err)
	}
}

func TestDaemonRejectsOverlongSocketPathBeforeLock(t *testing.T) {
	base := newTestStore(t)
	// Use a synthetic store value so the test does not depend on the host's
	// sockaddr_un implementation while still exercising the production guard.
	overlongRoot := strings.Repeat("x", maxUnixSocketPathBytes)
	overlong := &JournalStore{deploymentRoot: base.DeploymentRoot(), root: overlongRoot}
	daemon := NewDaemon(overlong, nil)
	if err := daemon.Listen(); err == nil || !strings.Contains(err.Error(), "socket path exceeds") {
		t.Fatalf("Listen() error = %v, want overlong socket path rejection", err)
	}
	if _, err := os.Stat(overlong.LockPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lock should not be acquired before path validation: %v", err)
	}
}

func TestDaemonUDSUsesPrivateModeAndStrictWireSchema(t *testing.T) {
	store := newTestStore(t)
	daemon := NewDaemon(store, nil)
	daemon.SetManifestVerifier(func(string, string) error { return nil })
	if err := daemon.Listen(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serveDone := make(chan error, 1)
	go func() { serveDone <- daemon.Serve(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-serveDone:
		case <-time.After(2 * time.Second):
			t.Error("daemon did not stop")
		}
	})
	info, err := os.Stat(store.SocketPath())
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("socket mode = %o, want 600", got)
	}
	connection, err := net.Dial("unix", store.SocketPath())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	unknown := map[string]any{"schemaVersion": RequestSchema, "operationId": "op-1", "action": "start", "manifestDigest": testManifestDigest, "command": "rm -rf /"}
	encoded, _ := json.Marshal(unknown)
	_, _ = connection.Write(append(encoded, '\n'))
	var response Response
	if err := json.NewDecoder(connection).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Error == "" {
		t.Fatal("unknown command field was accepted")
	}
}

func TestDaemonResumesNonTerminalCurrentJournal(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: "op-resume", ManifestDigest: testManifestDigest, Stage: StageUpdating, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	started := make(chan Request, 1)
	executor := ExecutorFunc(func(_ context.Context, request Request, _ *JournalStore) error {
		started <- request
		return nil
	})
	daemon := NewDaemon(store, executor)
	daemon.SetManifestVerifier(func(string, string) error { return nil })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serveDone := make(chan error, 1)
	go func() { serveDone <- daemon.Serve(ctx) }()
	select {
	case request := <-started:
		if request.Action != ActionResume || request.OperationID != "op-resume" {
			t.Fatalf("resume request = %#v", request)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("resume executor was not invoked")
	}
	cancel()
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("daemon did not stop")
	}
}

func TestDaemonFailurePreservesPostMigrationRecoveryBoundary(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: "op-failure", ManifestDigest: testManifestDigest, Stage: StageMigrating, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	daemon := NewDaemon(store, nil)
	if err := daemon.failJournal(Request{SchemaVersion: RequestSchema, OperationID: "op-failure", Action: ActionResume, ManifestDigest: testManifestDigest}, errors.New("executor stopped")); err != nil {
		t.Fatal(err)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageNeedsRecovery {
		t.Fatalf("failure stage = %q, want needs_recovery", current.Stage)
	}
	if !current.CompletedAt.After(now) {
		t.Fatalf("completedAt = %v, want after start %v", current.CompletedAt, now)
	}
}

func TestDaemonFailureKeepsNoMigrationVerificationRetryable(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	if err := store.Save(Journal{
		SchemaVersion: JournalSchema, OperationID: "op-code-receipt-failure", ManifestDigest: testManifestDigest,
		Stage: StageVerifying, MigrationStatus: MigrationStatusNotStarted, StartedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	daemon := NewDaemon(store, nil)
	if err := daemon.failJournal(Request{SchemaVersion: RequestSchema, OperationID: "op-code-receipt-failure", Action: ActionResume, ManifestDigest: testManifestDigest}, errors.New("receipt write failed")); err != nil {
		t.Fatal(err)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageFailed {
		t.Fatalf("no-migration verification failure stage = %q, want failed", current.Stage)
	}
	if current.RepairStage != StageRestarting {
		t.Fatalf("no-migration repair stage = %q, want restarting", current.RepairStage)
	}
}

func TestDaemonFailureDoesNotOverwriteDifferentCurrentOperation(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	current := Journal{
		SchemaVersion: JournalSchema, OperationID: "op-current", ManifestDigest: testManifestDigest,
		Stage: StageFailed, RepairStage: StageQueued, StartedAt: now, UpdatedAt: now, CompletedAt: &now,
	}
	if err := store.Save(current); err != nil {
		t.Fatal(err)
	}
	oldRequest := Request{SchemaVersion: RequestSchema, OperationID: "op-old", Action: ActionResume, ManifestDigest: testManifestDigest}
	if err := NewDaemon(store, nil).failJournal(oldRequest, errors.New("stale executor failure")); !errors.Is(err, ErrReplayDigestMismatch) {
		t.Fatalf("failJournal() error = %v, want ErrReplayDigestMismatch", err)
	}
	got, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if got.OperationID != current.OperationID || got.Stage != current.Stage {
		t.Fatalf("current journal overwritten by stale failure = %#v, want %#v", got, current)
	}
}

func TestDaemonFailureDoesNotOverwriteCorruptCurrentJournal(t *testing.T) {
	store := newTestStore(t)
	path := filepath.Join(store.Directory(), CurrentStateFile)
	corrupt := []byte(`{"schemaVersion":1`)
	if err := os.WriteFile(path, corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-corrupt-current", Action: ActionResume, ManifestDigest: testManifestDigest}
	if err := NewDaemon(store, nil).failJournal(request, errors.New("executor failure")); !errors.Is(err, ErrJournalCorrupt) {
		t.Fatalf("failJournal() error = %v, want ErrJournalCorrupt", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(corrupt) {
		t.Fatalf("corrupt current journal was overwritten: %q", got)
	}
}

func TestDaemonTerminalResumeRequiresExplicitRepair(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	if err := store.Save(Journal{
		SchemaVersion: JournalSchema, OperationID: "op-terminal-resume", ManifestDigest: testManifestDigest,
		Stage: StageFailed, RepairStage: StageQueued, StartedAt: now, UpdatedAt: now, CompletedAt: &now,
	}); err != nil {
		t.Fatal(err)
	}
	daemon := NewDaemon(store, nil)
	response := daemon.accept(Request{SchemaVersion: RequestSchema, OperationID: "op-terminal-resume", Action: ActionResume, ManifestDigest: testManifestDigest})
	if response.Accepted || response.Error != ErrRepairRequired.Error() {
		t.Fatalf("terminal resume response = %#v, want explicit repair rejection", response)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageFailed || current.RepairStage != StageQueued {
		t.Fatalf("terminal resume changed journal = %#v", current)
	}
}

func TestDaemonExplicitRepairResetsTerminalJournalAndLaunchesExecutor(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	if err := store.Save(Journal{
		SchemaVersion: JournalSchema, OperationID: "op-repair", ManifestDigest: testManifestDigest,
		Stage: StageNeedsRecovery, RepairStage: StageRestarting, StartedAt: now, UpdatedAt: now, CompletedAt: &now,
	}); err != nil {
		t.Fatal(err)
	}
	started := make(chan Request, 1)
	seenStage := make(chan Stage, 1)
	executor := ExecutorFunc(func(_ context.Context, request Request, store *JournalStore) error {
		started <- request
		journal, err := store.LoadCurrent()
		if err != nil {
			return err
		}
		seenStage <- journal.Stage
		return nil
	})
	daemon := NewDaemon(store, executor)
	daemon.SetManifestVerifier(func(string, string) error { return nil })
	if err := daemon.Listen(); err != nil {
		t.Fatal(err)
	}
	defer daemon.Close()
	response := daemon.accept(Request{SchemaVersion: RequestSchema, OperationID: "op-repair", Action: ActionRepair, ManifestDigest: testManifestDigest})
	if !response.Accepted || !response.Repaired || response.Replayed || response.Journal.Stage != StageRestarting {
		t.Fatalf("repair response = %#v", response)
	}
	select {
	case request := <-started:
		if request.Action != ActionRepair || request.ManifestDigest != testManifestDigest {
			t.Fatalf("executor request = %#v", request)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("repair executor was not launched")
	}
	select {
	case stage := <-seenStage:
		if stage != StageRestarting {
			t.Fatalf("repaired journal stage = %q, want restarting", stage)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("executor did not observe repaired stage")
	}
}

func TestDaemonDoesNotRepairSucceededJournal(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	if err := store.Save(Journal{
		SchemaVersion: JournalSchema, OperationID: "op-succeeded", ManifestDigest: testManifestDigest,
		Stage: StageSucceeded, StartedAt: now, UpdatedAt: now, CompletedAt: &now,
	}); err != nil {
		t.Fatal(err)
	}
	daemon := NewDaemon(store, nil)
	daemon.SetManifestVerifier(func(string, string) error { return nil })
	response := daemon.accept(Request{SchemaVersion: RequestSchema, OperationID: "op-succeeded", Action: ActionRepair, ManifestDigest: testManifestDigest})
	if response.Accepted || response.Error != ErrRepairNotAllowed.Error() {
		t.Fatalf("succeeded repair response = %#v, want rejection", response)
	}
}

func TestDaemonEventSinkReceivesRepairCheckpoint(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	if err := store.Save(Journal{
		SchemaVersion: JournalSchema, OperationID: "op-event", ManifestDigest: testManifestDigest,
		Stage: StageFailed, RepairStage: StageQueued, StartedAt: now, UpdatedAt: now, CompletedAt: &now,
	}); err != nil {
		t.Fatal(err)
	}
	events := make(chan JournalEvent, 4)
	daemon := NewDaemon(store, nil)
	daemon.SetManifestVerifier(func(string, string) error { return nil })
	daemon.SetEventSink(EventSinkFunc(func(_ context.Context, event JournalEvent) error {
		events <- event
		return errors.New("control plane unavailable")
	}))
	response := daemon.accept(Request{SchemaVersion: RequestSchema, OperationID: "op-event", Action: ActionRepair, ManifestDigest: testManifestDigest})
	if !response.Accepted || !response.Repaired {
		t.Fatalf("repair response = %#v", response)
	}
	select {
	case event := <-events:
		if event.OperationID != "op-event" || event.ManifestDigest != testManifestDigest || event.Stage != StageQueued {
			t.Fatalf("repair event = %#v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("repair checkpoint event was not published")
	}
}
