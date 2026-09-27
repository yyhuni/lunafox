package upgrader

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

func TestControlledHostActionCoversEveryFixedActionAndLeavesNoActivityAfterReturn(t *testing.T) {
	tests := []struct {
		name   string
		action domain.HostAction
		stage  Stage
		mode   ExecutionMode
	}{
		{name: "preflight", action: domain.HostActionPreflight, stage: StagePreflight, mode: ExecutionModeFull},
		{name: "pull images", action: domain.HostActionPullImages, stage: StageUpdating, mode: ExecutionModeFull},
		{name: "update services", action: domain.HostActionUpdateServices, stage: StageUpdating, mode: ExecutionModeFrontendOnly},
		{name: "migration", action: domain.HostActionDatabaseMigration, stage: StageMigrating, mode: ExecutionModeFull},
		{name: "resident Agent", action: domain.HostActionUpdateResidentAgent, stage: StageRestarting, mode: ExecutionModeFull},
		{name: "full health", action: domain.HostActionWaitForServiceHealth, stage: StageRestarting, mode: ExecutionModeFull},
		{name: "frontend health", action: domain.HostActionWaitForServiceHealth, stage: StageRestarting, mode: ExecutionModeFrontendOnly},
		{name: "runtime images", action: domain.HostActionVerifyRuntimeImages, stage: StageVerifying, mode: ExecutionModeFull},
		{name: "frontend container", action: domain.HostActionVerifyFrontendContainer, stage: StageVerifying, mode: ExecutionModeFrontendOnly},
		{name: "frontend edge", action: domain.HostActionVerifyFrontendEdge, stage: StageVerifying, mode: ExecutionModeFrontendOnly},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newTestStore(t)
			request, journal := hostActivityRequestAndJournal(test.name, test.stage, test.mode)
			if err := store.Save(journal); err != nil {
				t.Fatal(err)
			}
			executor := NewComposeExecutor(&recordingRunner{})
			if err := executor.runControlledHostAction(context.Background(), store, request, test.stage, test.action, func() error { return nil }); err != nil {
				t.Fatal(err)
			}
			current, err := store.LoadCurrent()
			if err != nil {
				t.Fatal(err)
			}
			if current.HostActivity != nil {
				t.Fatalf("action return retained host activity: %#v", current.HostActivity)
			}
			if len(current.ProgressEvents) != 2 {
				t.Fatalf("progress boundaries = %#v", current.ProgressEvents)
			}
			for index, key := range []string{ProgressHostActionStarted, ProgressHostActionCompleted} {
				if event := current.ProgressEvents[index]; event.MessageKey != key || event.Metadata["action"] != string(test.action) {
					t.Fatalf("progress event %d = %#v", index, event)
				}
			}
		})
	}
}

func TestControlledHostActionHeartbeatsWithoutWritingLogRowsAndCleansUpAfterCancellation(t *testing.T) {
	store := newTestStore(t)
	request, journal := hostActivityRequestAndJournal("heartbeat", StageUpdating, ExecutionModeFull)
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}
	executor := NewComposeExecutor(&recordingRunner{})
	executor.hostActivityHeartbeatInterval = time.Millisecond
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- executor.runControlledHostAction(context.Background(), store, request, StageUpdating, domain.HostActionPullImages, func() error {
			close(started)
			<-release
			return context.Canceled
		})
	}()
	<-started

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		current, err := store.LoadCurrent()
		if err != nil {
			t.Fatal(err)
		}
		if current.HostActivity != nil && current.HostActivity.LastHeartbeatAt.After(current.HostActivity.StartedAt) {
			if len(current.ProgressEvents) != 1 || current.ProgressEvents[0].MessageKey != ProgressHostActionStarted {
				t.Fatalf("heartbeat wrote an event row: %#v", current.ProgressEvents)
			}
			if !current.StageUpdatedAt.Equal(journal.StageUpdatedAt) {
				t.Fatalf("heartbeat advanced stage timestamp: %s", current.StageUpdatedAt)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("host activity heartbeat was not observed")
		}
		time.Sleep(time.Millisecond)
	}

	close(release)
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("controlled action error = %v", err)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.HostActivity != nil || len(current.ProgressEvents) != 2 {
		t.Fatalf("cancellation did not clean up bounded observation: %#v", current)
	}
	failure := current.ProgressEvents[1]
	if failure.MessageKey != ProgressHostActionFailed || failure.Metadata["reason"] != string(domain.HostActionFailureCommandCancelled) {
		t.Fatalf("cancellation boundary = %#v", failure)
	}
}

func TestControlledHostActionHeartbeatIntervalIsCapped(t *testing.T) {
	executor := NewComposeExecutor(&recordingRunner{})
	executor.hostActivityHeartbeatInterval = 30 * time.Second
	if interval := executor.hostActivityInterval(); interval != hostActivityHeartbeatInterval {
		t.Fatalf("heartbeat interval = %s, want cap %s", interval, hostActivityHeartbeatInterval)
	}
	executor.hostActivityHeartbeatInterval = time.Second
	if interval := executor.hostActivityInterval(); interval != time.Second {
		t.Fatalf("test heartbeat interval = %s, want 1s", interval)
	}
}

func TestControlledHostActionFailureIsSafeAndObservationFailureDoesNotBlockRunner(t *testing.T) {
	store := newTestStore(t)
	request, journal := hostActivityRequestAndJournal("safe-failure", StageUpdating, ExecutionModeFull)
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}
	executor := NewComposeExecutor(&recordingRunner{})
	runnerErr := errors.New("docker compose stderr: secret token at /deployment/.env")
	if err := executor.runControlledHostAction(context.Background(), store, request, StageUpdating, domain.HostActionPullImages, func() error { return runnerErr }); !errors.Is(err, runnerErr) {
		t.Fatalf("controlled action error = %v", err)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.HostActivity != nil || len(current.ProgressEvents) != 2 {
		t.Fatalf("failed action observation = %#v", current)
	}
	failure := current.ProgressEvents[1]
	if failure.MessageKey != ProgressHostActionFailed || failure.Metadata["action"] != string(domain.HostActionPullImages) || failure.Metadata["reason"] != string(domain.HostActionFailureCommandFailed) {
		t.Fatalf("failure boundary = %#v", failure)
	}
	if strings.Contains(strings.Join([]string{failure.Message, failure.Metadata["action"], failure.Metadata["reason"]}, " "), "secret") || strings.Contains(failure.Message, "stderr") {
		t.Fatalf("Runner output leaked into failure boundary: %#v", failure)
	}

	called := false
	executor.hostActivityNow = func() time.Time { return time.Time{} }
	if err := executor.runControlledHostAction(context.Background(), store, request, StageUpdating, domain.HostActionPullImages, func() error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("activity write failure changed Runner result: %v", err)
	}
	if !called {
		t.Fatal("Runner was skipped after activity observation failed")
	}
}

func TestLocalHostActionFailureUsesTheSafeCatalogWithoutActivity(t *testing.T) {
	store := newTestStore(t)
	request, journal := hostActivityRequestAndJournal("local-failure", StagePreflight, ExecutionModeFull)
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}
	tryAppendHostActionLocalFailure(store, request, StagePreflight, domain.HostActionPreflight)
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.HostActivity != nil || len(current.ProgressEvents) != 1 {
		t.Fatalf("local failure observation = %#v", current)
	}
	event := current.ProgressEvents[0]
	if event.MessageKey != ProgressHostActionFailed || event.Metadata["reason"] != string(domain.HostActionFailureLocalActionFailed) {
		t.Fatalf("local failure event = %#v", event)
	}
}

func hostActivityRequestAndJournal(suffix string, stage Stage, mode ExecutionMode) (Request, Journal) {
	base := time.Now().UTC().Add(-time.Second).Truncate(time.Microsecond)
	operationID := "op-host-activity-" + strings.ReplaceAll(suffix, " ", "-")
	request := Request{SchemaVersion: RequestSchema, OperationID: operationID, Action: ActionStart, ManifestDigest: testManifestDigest}
	journal := Journal{
		SchemaVersion: JournalSchema, OperationID: operationID, ManifestDigest: testManifestDigest,
		Stage: stage, StartedAt: base, UpdatedAt: base, StageUpdatedAt: base,
	}
	if mode == ExecutionModeFrontendOnly {
		request.SchemaVersion = ScopedRequestSchema
		request.ExecutionMode = ExecutionModeFrontendOnly
		request.PlanDigest = "sha256:" + strings.Repeat("1", 64)
		request.BaselineStateDigest = "sha256:" + strings.Repeat("2", 64)
		request.TouchedServices = []string{FrontendOnlyService}
		request.ConfirmedDeploymentVersion = "1.2.2"
		journal.SchemaVersion = ScopedJournalSchema
		journal.ExecutionMode = ExecutionModeFrontendOnly
		journal.PlanDigest = request.PlanDigest
		journal.BaselineStateDigest = request.BaselineStateDigest
		journal.TouchedServices = append([]string(nil), request.TouchedServices...)
		journal.ConfirmedDeploymentVersion = request.ConfirmedDeploymentVersion
	}
	return request, journal
}
