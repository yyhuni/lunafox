package bootstrap

import (
	"os"
	"strings"
	"testing"
)

func TestServerLocationUsesOneSharedCoordinatorAndManagedScheduler(t *testing.T) {
	wiringBytes, err := os.ReadFile("wiring.go")
	if err != nil {
		t.Fatalf("read wiring.go: %v", err)
	}
	runBytes, err := os.ReadFile("run.go")
	if err != nil {
		t.Fatalf("read run.go: %v", err)
	}
	wiring := string(wiringBytes)
	run := string(runBytes)

	if strings.Count(wiring, "geolocation.NewCoordinator(") != 1 {
		t.Fatal("bootstrap must construct exactly one shared geolocation coordinator")
	}
	if strings.Count(wiring, "wireSystemModule(repos, infra, cfg, locationCoordinator)") != 1 ||
		strings.Count(wiring, "wireAgentModule(repos, infra, cfg, scan.scanTaskBridge, grpcPublisher, locationCoordinator, serverLocationReader)") != 1 {
		t.Fatal("Agent and Server location must share the same coordinator")
	}
	if strings.Count(run, "d.serverLocationScheduler.Start(jobCtx)") != 1 {
		t.Fatal("bootstrap must start exactly one Server location scheduler")
	}
}

func TestLocationLifecyclesStopBeforeSharedCoordinatorAndDatabase(t *testing.T) {
	runBytes, err := os.ReadFile("run.go")
	if err != nil {
		t.Fatalf("read run.go: %v", err)
	}
	run := string(runBytes)

	grpcStop := strings.Index(run, "agentPlaneServer.Shutdown(shutdownCtx)")
	agentObserverStop := strings.Index(run, "d.agentLocationObserver.Shutdown(shutdownCtx)")
	serverSchedulerJoin := strings.Index(run, `waitForManagedBackgroundJob(shutdownCtx, "Server location scheduler"`)
	coordinatorStop := strings.Index(run, "d.geolocationCoordinator.Shutdown(shutdownCtx)")
	databaseClose := strings.Index(run, "sqlDB.Close()")
	if grpcStop < 0 || agentObserverStop < 0 || serverSchedulerJoin < 0 || coordinatorStop < 0 || databaseClose < 0 {
		t.Fatal("location shutdown contract markers are incomplete")
	}
	if !(grpcStop < agentObserverStop && agentObserverStop < serverSchedulerJoin && serverSchedulerJoin < coordinatorStop && coordinatorStop < databaseClose) {
		t.Fatal("location shutdown order must be gRPC, Agent observer, Server scheduler, coordinator, then database")
	}
}
