package handler

import "testing"

func TestFinalizeSnapshotStatusOptionalUnavailableDoesNotSetOffline(t *testing.T) {
	h := &HealthHandler{}
	snapshot := baseSnapshotForStatusTest()
	snapshot.UnavailableSignals = append(snapshot.UnavailableSignals, databaseUnavailableSignalResponse{
		Name:       "qps",
		Scope:      signalScopeOptional,
		ReasonCode: reasonPermissionDenied,
	})

	h.finalizeSnapshotStatus(&snapshot)

	if snapshot.Status == dbHealthStatusOffline {
		t.Fatalf("optional signal unavailable should not set status offline")
	}
	if snapshot.Status != dbHealthStatusOnline {
		t.Fatalf("expected status online, got %s", snapshot.Status)
	}
}

func TestFinalizeSnapshotStatusCoreUnavailableSetsDegraded(t *testing.T) {
	h := &HealthHandler{}
	snapshot := baseSnapshotForStatusTest()
	snapshot.UnavailableSignals = append(snapshot.UnavailableSignals, databaseUnavailableSignalResponse{
		Name:       "connectionsMax",
		Scope:      signalScopeCore,
		ReasonCode: reasonUnknown,
	})

	h.finalizeSnapshotStatus(&snapshot)

	if snapshot.Status != dbHealthStatusDegraded {
		t.Fatalf("expected status degraded, got %s", snapshot.Status)
	}
	if !containsAlertTitle(snapshot.Alerts, "Core signals unavailable") {
		t.Fatalf("expected core unavailable alert, got %+v", snapshot.Alerts)
	}
}

func TestFinalizeSnapshotStatusLockWaitHighSetsDegraded(t *testing.T) {
	h := &HealthHandler{}
	snapshot := baseSnapshotForStatusTest()
	snapshot.CoreSignals.LockWaitCount = 5

	h.finalizeSnapshotStatus(&snapshot)

	if snapshot.Status != dbHealthStatusDegraded {
		t.Fatalf("expected status degraded, got %s", snapshot.Status)
	}
	if !containsAlertTitle(snapshot.Alerts, "Lock waits high") {
		t.Fatalf("expected lock wait alert, got %+v", snapshot.Alerts)
	}
}

func TestFinalizeSnapshotStatusDeadlocksHighSetsDegraded(t *testing.T) {
	h := &HealthHandler{}
	snapshot := baseSnapshotForStatusTest()
	snapshot.CoreSignals.Deadlocks1h = 1

	h.finalizeSnapshotStatus(&snapshot)

	if snapshot.Status != dbHealthStatusDegraded {
		t.Fatalf("expected status degraded, got %s", snapshot.Status)
	}
	if !containsAlertTitle(snapshot.Alerts, "Deadlocks detected") {
		t.Fatalf("expected deadlocks alert, got %+v", snapshot.Alerts)
	}
}

func TestFinalizeSnapshotStatusLongTransactionsHighSetsDegraded(t *testing.T) {
	h := &HealthHandler{}
	snapshot := baseSnapshotForStatusTest()
	snapshot.CoreSignals.LongTransactionCount = 5

	h.finalizeSnapshotStatus(&snapshot)

	if snapshot.Status != dbHealthStatusDegraded {
		t.Fatalf("expected status degraded, got %s", snapshot.Status)
	}
	if !containsAlertTitle(snapshot.Alerts, "Long transactions high") {
		t.Fatalf("expected long transaction alert, got %+v", snapshot.Alerts)
	}
}

func TestFinalizeSnapshotStatusOldestPendingTaskAgeHighSetsDegraded(t *testing.T) {
	h := &HealthHandler{}
	snapshot := baseSnapshotForStatusTest()
	snapshot.CoreSignals.OldestPendingTaskAgeSec = 600

	h.finalizeSnapshotStatus(&snapshot)

	if snapshot.Status != dbHealthStatusDegraded {
		t.Fatalf("expected status degraded, got %s", snapshot.Status)
	}
	if !containsAlertTitle(snapshot.Alerts, "Task backlog age high") {
		t.Fatalf("expected backlog age alert, got %+v", snapshot.Alerts)
	}
}

func baseSnapshotForStatusTest() databaseHealthSnapshotResponse {
	return databaseHealthSnapshotResponse{
		Status: dbHealthStatusOnline,
		Role:   "primary",
		CoreSignals: databaseCoreSignalsResponse{
			ProbeLatencyMs:          20,
			ConnectionsUsed:         10,
			ConnectionsMax:          100,
			ConnectionUsagePercent:  10,
			LockWaitCount:           0,
			Deadlocks1h:             0,
			LongTransactionCount:    0,
			OldestPendingTaskAgeSec: 0,
		},
		UnavailableSignals: make([]databaseUnavailableSignalResponse, 0),
		Alerts:             make([]databaseHealthAlertResponse, 0),
	}
}

func containsAlertTitle(alerts []databaseHealthAlertResponse, title string) bool {
	for _, alert := range alerts {
		if alert.Title == title {
			return true
		}
	}
	return false
}
