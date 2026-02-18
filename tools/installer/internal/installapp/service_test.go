package installapp

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
	"github.com/yyhuni/lunafox/tools/installer/internal/execx"
	"github.com/yyhuni/lunafox/tools/installer/internal/ui"
)

type stubRunner struct{}

func (stubRunner) LookPath(string) (string, error) { return "", errors.New("not implemented") }
func (stubRunner) Run(context.Context, execx.Command) (execx.Result, error) {
	return execx.Result{}, errors.New("not implemented")
}

type fakeInstallerFactory struct {
	build func(printer *ui.Printer) Runner
}

func (factory fakeInstallerFactory) New(_ cli.Options, _ execx.Runner, printer *ui.Printer) Runner {
	return factory.build(printer)
}

type fakeInstaller struct {
	run func(context.Context) error
}

func (installer fakeInstaller) Run(ctx context.Context) error {
	return installer.run(ctx)
}

func waitSnapshot(t *testing.T, service *Service, jobID string, expect InstallState) InstallStateSnapshot {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snapshot, err := service.Snapshot(jobID)
		if err == nil && snapshot.State == string(expect) {
			return snapshot
		}
		time.Sleep(20 * time.Millisecond)
	}
	snapshot, _ := service.Snapshot(jobID)
	t.Fatalf("snapshot not reached expected state %s: %+v", expect, snapshot)
	return InstallStateSnapshot{}
}

func TestServiceStartConflict(t *testing.T) {
	service := NewService(stubRunner{}, io.Discard, io.Discard)
	block := make(chan struct{})
	service.installerFactory = fakeInstallerFactory{
		build: func(_ *ui.Printer) Runner {
			return fakeInstaller{run: func(context.Context) error {
				<-block
				return nil
			}}
		},
	}

	firstJob, err := service.Start(cli.Options{})
	if err != nil {
		t.Fatalf("start first: %v", err)
	}
	_, err = service.Start(cli.Options{})
	if err == nil {
		t.Fatalf("expected conflict error")
	}
	runningID, ok := IsJobRunning(err)
	if !ok {
		t.Fatalf("expected running error, got %v", err)
	}
	if runningID != firstJob {
		t.Fatalf("unexpected running job id: %s", runningID)
	}

	close(block)
	waitSnapshot(t, service, firstJob, StateSucceeded)
}

func TestServiceSnapshotAndEvents(t *testing.T) {
	service := NewService(stubRunner{}, io.Discard, io.Discard)
	var runCount atomic.Int64
	service.installerFactory = fakeInstallerFactory{
		build: func(printer *ui.Printer) Runner {
			return fakeInstaller{run: func(context.Context) error {
				runCount.Add(1)
				printer.Step(1, 7, "系统环境校验")
				printer.Info("hello")
				return nil
			}}
		},
	}

	jobID, err := service.Start(cli.Options{})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	snapshot := waitSnapshot(t, service, jobID, StateSucceeded)
	if snapshot.CurrentStep != 1 {
		t.Fatalf("expected step=1, got %d", snapshot.CurrentStep)
	}
	if snapshot.TotalSteps != 7 {
		t.Fatalf("expected total step 7, got %d", snapshot.TotalSteps)
	}
	if snapshot.Logs == "" {
		t.Fatalf("expected logs not empty")
	}
	if runCount.Load() != 1 {
		t.Fatalf("expected one run, got %d", runCount.Load())
	}

	events, cancel, err := service.Subscribe(jobID, 0)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer cancel()

	timeout := time.After(2 * time.Second)
	lastEventID := int64(0)
	gotDone := false
	for !gotDone {
		select {
		case event, ok := <-events:
			if !ok {
				t.Fatalf("event channel closed unexpectedly")
			}
			if event.ID <= lastEventID {
				t.Fatalf("event id not increasing: %d <= %d", event.ID, lastEventID)
			}
			lastEventID = event.ID
			if event.Type == EventDone {
				gotDone = true
			}
		case <-timeout:
			t.Fatalf("timeout waiting for done event")
		}
	}

	replay, replayCancel, err := service.Subscribe(jobID, 1)
	if err != nil {
		t.Fatalf("replay subscribe: %v", err)
	}
	defer replayCancel()
	select {
	case event := <-replay:
		if event.ID <= 1 {
			t.Fatalf("expected replay event id > 1, got %d", event.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting replay event")
	}
}

func TestServiceSnapshotNotFound(t *testing.T) {
	service := NewService(stubRunner{}, io.Discard, io.Discard)
	_, err := service.Snapshot("missing")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !IsJobNotFound(err) {
		t.Fatalf("expected not found error")
	}
}
