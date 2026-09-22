package upgrader

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
)

func closeTestDaemon(t *testing.T, daemon *Daemon) {
	t.Helper()
	if err := daemon.Close(); err != nil {
		t.Errorf("close daemon: %v", err)
	}
}

func TestVerifyDeploymentManifestAcceptsOnlyPinnedLegacyAlpha114(t *testing.T) {
	path := legacyAlpha114ManifestFixturePath(t)
	if err := VerifyDeploymentManifest(path, releasemanifest.LegacyAlpha114ManifestDigest); err != nil {
		t.Fatalf("VerifyDeploymentManifest() error = %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tamperedPath := filepath.Join(t.TempDir(), "release.manifest.yaml")
	tampered := strings.Replace(string(raw), "maintenanceWindowMinutes: 15", "maintenanceWindowMinutes: 16", 1)
	if err := os.WriteFile(tamperedPath, []byte(tampered), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerifyDeploymentManifest(tamperedPath, releasemanifest.LegacyAlpha114ManifestDigest); err == nil {
		t.Fatal("VerifyDeploymentManifest accepted a tampered legacy manifest")
	}
}

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

func TestDaemonStopWithoutRunningExecutorWritesTerminalAttentionCheckpoint(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-stop-idle", Action: ActionStop, ManifestDigest: testManifestDigest}
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	daemon := NewDaemon(store, nil)
	response := daemon.accept(request)
	if !response.Accepted || response.Journal.Stage != StageNeedsAttention {
		t.Fatalf("idle stop response = %#v, want accepted needs_attention", response)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageNeedsAttention || current.Diagnostic != "upgrade stopped by operator" {
		t.Fatalf("idle stop journal = %#v", current)
	}
	// Repeating the same stop is a replay and cannot alter the terminal result.
	replayed := daemon.accept(request)
	if !replayed.Accepted || !replayed.Replayed || replayed.Journal.Stage != StageNeedsAttention {
		t.Fatalf("repeated idle stop response = %#v", replayed)
	}
}

func TestDaemonStopRaceDoesNotDowngradeTerminalCheckpoint(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-stop-race", Action: ActionStop, ManifestDigest: testManifestDigest}
	terminal := Journal{
		SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest,
		Stage: StageSucceeded, StartedAt: now, UpdatedAt: now, CompletedAt: &now,
	}
	if err := store.Save(terminal); err != nil {
		t.Fatal(err)
	}
	daemon := NewDaemon(store, nil)
	updated, err := daemon.stopJournal(request, terminal)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Stage != StageSucceeded {
		t.Fatalf("terminal stop race changed stage to %q", updated.Stage)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageSucceeded {
		t.Fatalf("terminal stop race persisted stage %q", current.Stage)
	}
}

func TestDaemonStopCancelsExecutorAndUsesMigrationRecoveryFence(t *testing.T) {
	store := newTestStore(t)
	started := make(chan struct{})
	finished := make(chan struct{})
	executor := ExecutorFunc(func(ctx context.Context, _ Request, _ *JournalStore) error {
		close(started)
		<-ctx.Done()
		close(finished)
		return ctx.Err()
	})
	daemon := NewDaemon(store, executor)
	daemon.SetManifestVerifier(func(string, string) error { return nil })
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-stop-running", Action: ActionStart, ManifestDigest: testManifestDigest}
	if response := daemon.accept(request); !response.Accepted {
		t.Fatalf("start response = %#v", response)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("executor did not start")
	}
	if _, err := store.Checkpoint(request.OperationID, request.ManifestDigest, StageMigrating, "", nil); err != nil {
		t.Fatal(err)
	}
	stopRequest := request
	stopRequest.Action = ActionStop
	stopResponse := daemon.accept(stopRequest)
	if !stopResponse.Accepted {
		t.Fatalf("stop response = %#v", stopResponse)
	}
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("executor did not observe cancellation")
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		current, err := store.LoadCurrent()
		if err == nil && current.Stage == StageNeedsRecovery {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("cancelled migration journal did not converge: current=%#v err=%v", current, err)
		}
		time.Sleep(10 * time.Millisecond)
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

func TestDaemonServeEstablishesRecoveryFenceBeforeAccepting(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	request, plan := staleFrontendOnlyRequestAndPlan(t, now)
	if err := store.SaveScopePlan(plan); err != nil {
		t.Fatal(err)
	}
	journal := journalForRequest(request, now)
	journal.Stage = StageVerifying
	journal.StageUpdatedAt = now
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}

	lockDirectory := filepath.Join(store.DeploymentRoot(), DeploymentLockDirectory)
	if err := os.Mkdir(lockDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lockDirectory, DeploymentLockMetadataFile), []byte("schema=1\nowner=lifecycle\ncommand=install\noperation_id=\ncreated_at=1700000000\nrecovery_fence=false\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	daemon := NewDaemon(store, nil)
	serveDone := make(chan error, 1)
	go func() { serveDone <- daemon.Serve(ctx) }()
	select {
	case err := <-serveDone:
		if err == nil || !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
			t.Fatalf("Serve() error = %v, want cancellable recovery-fence wait", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not stop while recovery fence was blocked")
	}
	if daemon.deploymentLock != nil {
		t.Fatalf("daemon retained a lock after cancelled fence acquisition: %#v", daemon.deploymentLock)
	}
	if _, exists, err := ReadDeploymentLock(store.DeploymentRoot()); err != nil || !exists {
		t.Fatalf("lifecycle lock was changed during blocked startup exists=%t err=%v", exists, err)
	}
}

func TestDaemonConfirmationRequiresCurrentDeploymentLockOwner(t *testing.T) {
	for _, test := range []struct {
		name       string
		staleOwner bool
	}{
		{name: "missing lock"},
		{name: "lifecycle owner", staleOwner: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := newTestStore(t)
			now := time.Now().UTC()
			request, plan := fullConfirmationRequestAndPlan(t, now)
			if err := store.SaveScopePlan(plan); err != nil {
				t.Fatal(err)
			}
			journal := journalForRequest(request, now)
			journal.Stage = StageVerifying
			journal.StageUpdatedAt = now
			if err := store.Save(journal); err != nil {
				t.Fatal(err)
			}
			if err := store.SaveReceipt(receiptForRequest(request, now.Add(time.Millisecond), []string{"server", "frontend", "nginx", "agent"}, map[string]string{
				"server": testManifestDigest, "frontend": testManifestDigest, "nginx": testManifestDigest, "agent": testManifestDigest,
			})); err != nil {
				t.Fatal(err)
			}
			daemon := NewDaemon(store, nil)
			if test.staleOwner {
				lockDirectory := filepath.Join(store.DeploymentRoot(), DeploymentLockDirectory)
				if err := os.Mkdir(lockDirectory, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(lockDirectory, DeploymentLockMetadataFile), []byte("schema=1\nowner=lifecycle\ncommand=install\noperation_id=\ncreated_at=1700000000\nrecovery_fence=false\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				daemon.deploymentLock = &DeploymentLock{directory: lockDirectory, operationID: request.OperationID}
			}
			response := daemon.accept(request)
			if response.Accepted || response.Error != ErrDeploymentLockNotOwned.Error() {
				t.Fatalf("confirmation response = %#v, want lock-ownership rejection", response)
			}
			if _, err := store.LoadConfirmedDeploymentState(); !errors.Is(err, ErrConfirmedStateNotFound) {
				t.Fatalf("confirmation mutated baseline despite missing ownership: %v", err)
			}
		})
	}
}

type scopePlannerWithoutAvailabilityStub struct {
	capabilities HostCapabilities
}

func (planner *scopePlannerWithoutAvailabilityStub) Capabilities() HostCapabilities {
	return planner.capabilities
}

func (*scopePlannerWithoutAvailabilityStub) Plan(context.Context, string, string, bool) (ScopePlan, error) {
	return ScopePlan{}, errors.New("unexpected plan call")
}

func (*scopePlannerWithoutAvailabilityStub) Revalidate(context.Context, ScopePlan) error {
	return nil
}

type candidateAvailabilityPlannerStub struct {
	capabilities HostCapabilities
	availability CandidateAvailability
	err          error
	calls        int
}

func (planner *candidateAvailabilityPlannerStub) Capabilities() HostCapabilities {
	return planner.capabilities
}

func (*candidateAvailabilityPlannerStub) Plan(context.Context, string, string, bool) (ScopePlan, error) {
	return ScopePlan{}, errors.New("unexpected plan call")
}

func (*candidateAvailabilityPlannerStub) Revalidate(context.Context, ScopePlan) error {
	return nil
}

func (planner *candidateAvailabilityPlannerStub) CandidateAvailability(_ context.Context, _ string) (CandidateAvailability, error) {
	planner.calls++
	if planner.err != nil {
		return CandidateAvailability{}, planner.err
	}
	return planner.availability, nil
}

type fullConfirmationPlannerStub struct {
	observation RuntimeObservation
	called      bool
	err         error
}

func (planner *fullConfirmationPlannerStub) Capabilities() HostCapabilities {
	return DefaultHostCapabilities()
}

func (*fullConfirmationPlannerStub) Plan(context.Context, string, string, bool) (ScopePlan, error) {
	return ScopePlan{}, errors.New("unexpected plan call")
}

func (*fullConfirmationPlannerStub) Revalidate(context.Context, ScopePlan) error {
	return nil
}

func (planner *fullConfirmationPlannerStub) ObserveFullDeployment(_ context.Context, plan ScopePlan) (RuntimeObservation, error) {
	planner.called = true
	if planner.err != nil {
		return RuntimeObservation{}, planner.err
	}
	if !runtimeObservationMatchesComponents(planner.observation, plan.Candidate.Components) {
		return RuntimeObservation{}, errors.New("live deployment inventory does not match candidate components")
	}
	return planner.observation, nil
}

func TestDaemonFullConfirmationRechecksLiveInventoryBeforeSavingBaseline(t *testing.T) {
	for _, test := range []struct {
		name       string
		mutate     func(map[string]string)
		wantAccept bool
	}{
		{name: "matching inventory", wantAccept: true},
		{name: "engine drift", mutate: func(images map[string]string) { images[runtimeServerComponent] = "sha256:" + strings.Repeat("f", 64) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := newTestStore(t)
			now := time.Now().UTC()
			request, plan := fullConfirmationRequestAndPlan(t, now)
			if err := store.SaveScopePlan(plan); err != nil {
				t.Fatal(err)
			}
			journal := journalForRequest(request, now)
			journal.Stage = StageVerifying
			journal.StageUpdatedAt = now
			if err := store.Save(journal); err != nil {
				t.Fatal(err)
			}
			if err := store.SaveReceipt(receiptForRequest(request, now.Add(time.Millisecond), []string{"server", "frontend", "nginx", "agent"}, map[string]string{
				"server": testManifestDigest, "frontend": testManifestDigest, "nginx": testManifestDigest, "agent": testManifestDigest,
			})); err != nil {
				t.Fatal(err)
			}
			images := map[string]string{
				runtimeAgentComponent:     testManifestDigest,
				runtimeBootstrapComponent: testManifestDigest,
				runtimeFrontendComponent:  testManifestDigest,
				runtimeNginxComponent:     testManifestDigest,
				runtimeServerComponent:    testManifestDigest,
			}
			if test.mutate != nil {
				test.mutate(images)
			}
			planner := &fullConfirmationPlannerStub{observation: RuntimeObservation{
				Images: images, NginxHealthy: true,
				NginxConfigDigest: "sha256:" + strings.Repeat("1", 64), ObservedAt: now,
			}}
			daemon := NewDaemon(store, nil)
			daemon.SetScopePlanner(planner)
			lock, err := AcquireDeploymentLock(store.DeploymentRoot(), request.OperationID)
			if err != nil {
				t.Fatal(err)
			}
			daemon.deploymentLock = lock
			response := daemon.accept(request)
			if response.Accepted != test.wantAccept {
				t.Fatalf("confirmation accepted=%t response=%#v, want %t", response.Accepted, response, test.wantAccept)
			}
			if !planner.called {
				t.Fatal("full confirmation did not invoke live deployment observer")
			}
			state, stateErr := store.LoadConfirmedDeploymentState()
			if test.wantAccept {
				if stateErr != nil {
					t.Fatal(stateErr)
				}
				if state.NginxConfigDigest != "sha256:"+strings.Repeat("1", 64) {
					t.Fatalf("confirmed nginx config digest = %q", state.NginxConfigDigest)
				}
			} else if !errors.Is(stateErr, ErrConfirmedStateNotFound) {
				t.Fatalf("live inventory mismatch saved baseline: %v", stateErr)
			}
			if !test.wantAccept {
				_ = lock.Release()
			}
		})
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
	defer closeTestDaemon(t, daemon)
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
	defer closeTestDaemon(t, first)
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
	defer func() {
		if err := connection.Close(); err != nil {
			t.Errorf("close connection: %v", err)
		}
	}()
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

func TestDaemonCapabilityResponsesProjectSchemaV3Fields(t *testing.T) {
	planner := &candidateAvailabilityPlannerStub{
		capabilities: DefaultHostCapabilities(),
		availability: CandidateAvailability{
			ManifestDigest:             testManifestDigest,
			Decision:                   CandidateAvailabilityAlreadyApplied,
			BaselineStateDigest:        testManifestDigest,
			ConfirmedDeploymentVersion: "1.2.3",
		},
	}
	daemon := NewDaemon(newTestStore(t), nil)
	daemon.SetScopePlanner(planner)

	v3 := daemon.accept(Request{SchemaVersion: RequestSchemaV3, Action: ActionCapabilities})
	if !v3.Accepted || v3.Capabilities == nil || !v3.Capabilities.SupportsV3CandidateInventoryAvailability() {
		t.Fatalf("schema-v3 capabilities = %#v", v3)
	}
	v2 := daemon.accept(Request{SchemaVersion: ScopedRequestSchema, Action: ActionCapabilities})
	if !v2.Accepted || v2.Capabilities == nil {
		t.Fatalf("schema-v2 capabilities = %#v", v2)
	}
	if v2.Capabilities.CandidateInventoryAvailability || containsSchema(v2.Capabilities.SchemaVersions, RequestSchemaV3) {
		t.Fatalf("schema-v2 capabilities leaked schema-v3 data: %#v", v2.Capabilities)
	}
	encoded, err := json.Marshal(v2)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("candidateInventoryAvailability")) {
		t.Fatalf("schema-v2 capability JSON leaked schema-v3 field: %s", encoded)
	}
	leaked := Response{SchemaVersion: ScopedRequestSchema, Accepted: true, Capabilities: &planner.capabilities}
	if err := leaked.ValidateFor(Request{SchemaVersion: ScopedRequestSchema, Action: ActionCapabilities}); err == nil {
		t.Fatal("schema-v2 capability response accepted schema-v3 data")
	}
}

func TestDaemonDoesNotMutateCustomPlannerCapabilitiesDuringNegotiation(t *testing.T) {
	planner := &scopePlannerWithoutAvailabilityStub{capabilities: DefaultHostCapabilities()}
	daemon := NewDaemon(newTestStore(t), nil)
	daemon.SetScopePlanner(planner)
	response := daemon.accept(Request{SchemaVersion: RequestSchemaV3, Action: ActionCapabilities})
	if !response.Accepted || response.Capabilities == nil || response.Capabilities.CandidateInventoryAvailability || containsSchema(response.Capabilities.SchemaVersions, RequestSchemaV3) {
		t.Fatalf("custom planner capability response = %#v", response)
	}
	if !planner.capabilities.CandidateInventoryAvailability || !containsSchema(planner.capabilities.SchemaVersions, RequestSchemaV3) {
		t.Fatalf("daemon mutated custom planner capabilities: %#v", planner.capabilities)
	}
}

func TestDaemonCandidateAvailabilityIsReadOnlyAndStrictlyBound(t *testing.T) {
	store := newTestStore(t)
	planner := &candidateAvailabilityPlannerStub{
		capabilities: DefaultHostCapabilities(),
		availability: CandidateAvailability{
			ManifestDigest:             testManifestDigest,
			Decision:                   CandidateAvailabilityAlreadyApplied,
			BaselineStateDigest:        testManifestDigest,
			ConfirmedDeploymentVersion: "1.2.3",
		},
	}
	daemon := NewDaemon(store, nil)
	daemon.SetScopePlanner(planner)
	request := Request{SchemaVersion: RequestSchemaV3, Action: ActionCandidateAvailability, ManifestDigest: testManifestDigest}
	response := daemon.accept(request)
	if !response.Accepted || response.CandidateAvailability == nil || response.CandidateAvailability.Decision != CandidateAvailabilityAlreadyApplied {
		t.Fatalf("candidate availability response = %#v", response)
	}
	if planner.calls != 1 {
		t.Fatalf("availability planner calls = %d, want 1", planner.calls)
	}
	if _, err := store.LoadCurrent(); !errors.Is(err, ErrJournalNotFound) {
		t.Fatalf("availability request created a journal: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(store.Directory(), PlanDirectory))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("availability request persisted scope plans: %#v", entries)
	}
	if _, exists, err := ReadDeploymentLock(store.DeploymentRoot()); err != nil || exists {
		t.Fatalf("availability request acquired deployment lock exists=%t err=%v", exists, err)
	}

	invalid := request
	invalid.OperationID = "must-not-be-present"
	if rejected := daemon.accept(invalid); rejected.Accepted || planner.calls != 1 {
		t.Fatalf("operation-bound availability request = %#v, planner calls=%d", rejected, planner.calls)
	}
	wrongDigest := response
	wrongDigest.CandidateAvailability = &CandidateAvailability{
		ManifestDigest:             "sha256:" + strings.Repeat("b", 64),
		Decision:                   CandidateAvailabilityAlreadyApplied,
		BaselineStateDigest:        testManifestDigest,
		ConfirmedDeploymentVersion: "1.2.3",
	}
	if err := wrongDigest.ValidateFor(request); !errors.Is(err, ErrReplayDigestMismatch) {
		t.Fatalf("wrong digest availability response error = %v, want ErrReplayDigestMismatch", err)
	}

	executable := Request{SchemaVersion: RequestSchemaV3, OperationID: "not-allowed", Action: ActionStart, ManifestDigest: testManifestDigest}
	if err := executable.Validate(); err == nil {
		t.Fatal("schema-v3 executable request was accepted")
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

func TestDaemonResumeKeepsVerifiedFrontendOnlyJournalWaitingForConfirmation(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	request, plan := staleFrontendOnlyRequestAndPlan(t, now)
	if err := store.SaveScopePlan(plan); err != nil {
		t.Fatal(err)
	}
	journal := journalForRequest(request, now)
	journal.Stage = StageVerifying
	journal.StageUpdatedAt = now
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReceipt(receiptForRequest(
		request,
		now.Add(time.Millisecond),
		[]string{FrontendOnlyService},
		map[string]string{FrontendOnlyService: testManifestDigest},
	)); err != nil {
		t.Fatal(err)
	}

	executed := make(chan struct{}, 1)
	revalidated := make(chan struct{}, 1)
	daemon := NewDaemon(store, ExecutorFunc(func(context.Context, Request, *JournalStore) error {
		executed <- struct{}{}
		return nil
	}))
	daemon.SetManifestVerifier(func(string, string) error { return nil })
	daemon.SetScopePlanner(staleFrontendOnlyScopePlanner{revalidated: revalidated})
	if err := daemon.resumeCurrent(context.Background()); err != nil {
		t.Fatalf("resumeCurrent() = %v", err)
	}
	select {
	case <-revalidated:
		t.Fatal("verified frontend-only journal was revalidated")
	case <-time.After(100 * time.Millisecond):
	}
	select {
	case <-executed:
		t.Fatal("verified frontend-only journal was executed again")
	default:
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageVerifying {
		t.Fatalf("journal stage after restart = %q, want verifying until confirm", current.Stage)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	daemon.ensureRecoveryFence(ctx)
	deadline := time.Now().Add(2 * time.Second)
	for {
		metadata, exists, err := ReadDeploymentLock(store.DeploymentRoot())
		if err != nil {
			t.Fatal(err)
		}
		if exists && metadata.OperationID == request.OperationID && metadata.RecoveryFence {
			break
		}
		if exists && metadata.OperationID != request.OperationID {
			t.Fatalf("confirmation fence belongs to another operation = %#v", metadata)
		}
		if time.Now().After(deadline) {
			t.Fatal("confirmation fence was not restored")
		}
		time.Sleep(10 * time.Millisecond)
	}
	defer daemon.releaseDeploymentLock()
}

func TestDaemonReplayDoesNotRelaunchVerifiedFrontendOnlyJournal(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	request, plan := staleFrontendOnlyRequestAndPlan(t, now)
	request.Action = ActionResume
	if err := store.SaveScopePlan(plan); err != nil {
		t.Fatal(err)
	}
	journal := journalForRequest(request, now)
	journal.Stage = StageVerifying
	journal.StageUpdatedAt = now
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReceipt(receiptForRequest(request, now.Add(time.Millisecond), []string{FrontendOnlyService}, map[string]string{FrontendOnlyService: testManifestDigest})); err != nil {
		t.Fatal(err)
	}
	executed := make(chan struct{}, 1)
	revalidated := make(chan struct{}, 1)
	daemon := NewDaemon(store, ExecutorFunc(func(context.Context, Request, *JournalStore) error {
		executed <- struct{}{}
		return nil
	}))
	daemon.SetManifestVerifier(func(string, string) error { return nil })
	daemon.SetScopePlanner(staleFrontendOnlyScopePlanner{revalidated: revalidated})
	response := daemon.accept(request)
	if !response.Accepted || !response.Replayed || response.Journal.Stage != StageVerifying {
		t.Fatalf("replayed verified frontend-only response = %#v", response)
	}
	select {
	case <-executed:
		t.Fatal("replayed verified frontend-only journal launched executor")
	default:
	}
	select {
	case <-revalidated:
		t.Fatal("replayed verified frontend-only journal revalidated scope")
	default:
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
	if current.Diagnostic != "upgrade execution failed" {
		t.Fatalf("diagnostic = %q, want fixed safe message", current.Diagnostic)
	}
}

func TestDaemonFailureDoesNotPersistExecutorOutput(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	if err := store.Save(Journal{
		SchemaVersion: JournalSchema, OperationID: "op-safe-diagnostic", ManifestDigest: testManifestDigest,
		Stage: StagePreflight, StartedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-safe-diagnostic", Action: ActionResume, ManifestDigest: testManifestDigest}
	if err := NewDaemon(store, nil).failJournal(request, errors.New("docker compose failed at /srv/lunafox/.env TOKEN=secret")); err != nil {
		t.Fatal(err)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Diagnostic != "upgrade execution failed" || strings.Contains(current.Diagnostic, "/srv") || strings.Contains(strings.ToLower(current.Diagnostic), "token") {
		t.Fatalf("unsafe executor output persisted: %q", current.Diagnostic)
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

func TestDaemonStaleFrontendOnlyRecheckFailsQueuedJournalAndReleasesDeploymentLock(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	request, plan := staleFrontendOnlyRequestAndPlan(t, now)
	if err := store.SaveScopePlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(journalForRequest(request, now)); err != nil {
		t.Fatal(err)
	}
	executed := make(chan struct{}, 1)
	daemon := NewDaemon(store, ExecutorFunc(func(context.Context, Request, *JournalStore) error {
		executed <- struct{}{}
		return nil
	}))
	daemon.SetScopePlanner(staleFrontendOnlyScopePlanner{})

	daemon.mu.Lock()
	daemon.launchExecutionLocked(context.Background(), request)
	daemon.mu.Unlock()
	done := make(chan struct{})
	go func() {
		daemon.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("stale frontend-only recheck did not finish")
	}
	select {
	case <-executed:
		t.Fatal("stale frontend-only recheck invoked the executor")
	default:
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageFailed || current.CompletedAt == nil || current.Diagnostic != "frontend-only scope plan requires recheck" {
		t.Fatalf("stale frontend-only journal = %#v", current)
	}
	if daemon.deploymentLock != nil {
		t.Fatalf("stale frontend-only recheck retained daemon lock = %#v", daemon.deploymentLock)
	}
	if _, exists, err := ReadDeploymentLock(store.DeploymentRoot()); err != nil || exists {
		t.Fatalf("stale frontend-only recheck retained deployment lock exists=%t err=%v", exists, err)
	}
}

type staleFrontendOnlyScopePlanner struct {
	revalidated chan<- struct{}
}

func (staleFrontendOnlyScopePlanner) Capabilities() HostCapabilities {
	return DefaultHostCapabilities()
}

func (staleFrontendOnlyScopePlanner) Plan(context.Context, string, string, bool) (ScopePlan, error) {
	return ScopePlan{}, errors.New("unexpected plan call")
}

func (planner staleFrontendOnlyScopePlanner) Revalidate(context.Context, ScopePlan) error {
	if planner.revalidated != nil {
		select {
		case planner.revalidated <- struct{}{}:
		default:
		}
	}
	return ErrScopePlanStale
}

func staleFrontendOnlyRequestAndPlan(t *testing.T, now time.Time) (Request, ScopePlan) {
	t.Helper()
	operationID := "op-stale-frontend"
	compositionDigest := "sha256:" + strings.Repeat("b", 64)
	baselineDigest := "sha256:" + strings.Repeat("c", 64)
	components := []DeploymentComponent{
		{ID: runtimeAgentComponent, Digest: testManifestDigest},
		{ID: runtimeBootstrapComponent, Digest: testManifestDigest},
		{ID: runtimeFrontendComponent, Digest: testManifestDigest},
		{ID: runtimeNginxComponent, Digest: testManifestDigest},
		{ID: runtimeServerComponent, Digest: testManifestDigest},
	}
	observed := map[string]string{
		runtimeAgentComponent: testManifestDigest, runtimeFrontendComponent: testManifestDigest,
		runtimeNginxComponent: testManifestDigest, runtimeServerComponent: testManifestDigest,
	}
	plan := ScopePlan{
		SchemaVersion: ScopePlanSchema, OperationID: operationID, ManifestDigest: testManifestDigest,
		ExecutionMode: ExecutionModeFrontendOnly, BaselineStateDigest: baselineDigest,
		TouchedServices: []string{FrontendOnlyService}, ConfirmedDeploymentVersion: "1.0.0",
		Candidate: CandidateDeployment{
			ReleaseVersion: "1.1.0", CompositionDigest: compositionDigest, Components: components,
			Capabilities: DeploymentCapabilities{DynamicFrontendUpstream: true},
		},
		ObservedRuntimeImages: observed, ObservedNginxHealthy: true,
		ObservedNginxConfigDigest: "sha256:" + strings.Repeat("d", 64), GeneratedAt: now,
	}
	digest, err := plan.derivedDigest()
	if err != nil {
		t.Fatal(err)
	}
	plan.PlanDigest = digest
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
	request := Request{
		SchemaVersion: ScopedRequestSchema, OperationID: operationID, Action: ActionStart, ManifestDigest: testManifestDigest,
		ExecutionMode: ExecutionModeFrontendOnly, PlanDigest: plan.PlanDigest, BaselineStateDigest: baselineDigest,
		TouchedServices: []string{FrontendOnlyService}, ConfirmedDeploymentVersion: "1.0.0",
	}
	return request, plan
}

func fullConfirmationRequestAndPlan(t *testing.T, now time.Time) (Request, ScopePlan) {
	t.Helper()
	operationID := "op-confirm-lock"
	components := []DeploymentComponent{
		{ID: runtimeAgentComponent, Digest: testManifestDigest},
		{ID: runtimeBootstrapComponent, Digest: testManifestDigest},
		{ID: runtimeFrontendComponent, Digest: testManifestDigest},
		{ID: runtimeNginxComponent, Digest: testManifestDigest},
		{ID: runtimeServerComponent, Digest: testManifestDigest},
	}
	sortDeploymentComponents(components)
	plan := ScopePlan{
		SchemaVersion: ScopePlanSchema, OperationID: operationID, ManifestDigest: testManifestDigest,
		ExecutionMode: ExecutionModeFull, TouchedServices: fullTouchedServices(),
		Candidate: CandidateDeployment{
			ReleaseVersion: "1.1.0", CompositionDigest: "sha256:" + strings.Repeat("e", 64),
			Components: components,
		},
		GeneratedAt: now,
	}
	digest, err := plan.derivedDigest()
	if err != nil {
		t.Fatal(err)
	}
	plan.PlanDigest = digest
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
	request := Request{
		SchemaVersion: ScopedRequestSchema, OperationID: operationID, Action: ActionConfirm,
		ManifestDigest: testManifestDigest, ExecutionMode: ExecutionModeFull,
		PlanDigest: digest, TouchedServices: fullTouchedServices(),
	}
	return request, plan
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
	defer closeTestDaemon(t, daemon)
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
