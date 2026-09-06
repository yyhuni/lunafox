package handler

import (
	"strings"
	"testing"
)

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
	snapshot.CoreSignals.OldestPendingTaskAgeSec = 120

	h.finalizeSnapshotStatus(&snapshot)

	if snapshot.Status != dbHealthStatusDegraded {
		t.Fatalf("expected status degraded, got %s", snapshot.Status)
	}
	if !containsAlertTitle(snapshot.Alerts, "Task backlog age high") {
		t.Fatalf("expected backlog age alert, got %+v", snapshot.Alerts)
	}
}

func TestFinalizeSnapshotStatusOldestPendingTaskAgeBelowTargetStaysOnline(t *testing.T) {
	h := &HealthHandler{}
	snapshot := baseSnapshotForStatusTest()
	snapshot.CoreSignals.OldestPendingTaskAgeSec = 119

	h.finalizeSnapshotStatus(&snapshot)

	if snapshot.Status != dbHealthStatusOnline {
		t.Fatalf("expected status online below pending-task target, got %s", snapshot.Status)
	}
	if containsAlertTitle(snapshot.Alerts, "Task backlog age high") {
		t.Fatalf("did not expect backlog age alert below target, got %+v", snapshot.Alerts)
	}
}

func TestFinalizeSnapshotStatusOldestPendingTaskAgeCriticalUsesCriticalAlert(t *testing.T) {
	h := &HealthHandler{}
	snapshot := baseSnapshotForStatusTest()
	snapshot.CoreSignals.OldestPendingTaskAgeSec = 600

	h.finalizeSnapshotStatus(&snapshot)

	alert := alertByTitle(snapshot.Alerts, "Task backlog age critical")
	if alert == nil {
		t.Fatalf("expected critical backlog age alert, got %+v", snapshot.Alerts)
	}
	if alert.Severity != "critical" {
		t.Fatalf("expected critical severity, got %+v", alert)
	}
	if strings.Contains(alert.Description, "600 seconds") {
		t.Fatalf("expected human-readable threshold description, got %q", alert.Description)
	}
}

func TestBuildDatabaseHealthFindingsHealthySnapshotReturnsEmpty(t *testing.T) {
	h := &HealthHandler{}
	snapshot := baseSnapshotForStatusTest()

	findings := h.buildDatabaseHealthFindings(&snapshot)

	if len(findings) != 0 {
		t.Fatalf("expected no findings for healthy snapshot, got %+v", findings)
	}
}

func TestBuildDatabaseHealthFindingsReturnsActionableEvidence(t *testing.T) {
	h := &HealthHandler{}
	snapshot := baseSnapshotForStatusTest()
	snapshot.CoreSignals.ConnectionUsagePercent = 90
	snapshot.CoreSignals.LockWaitCount = 5
	snapshot.CoreSignals.OldestPendingTaskAgeSec = 900
	snapshot.UnavailableSignals = append(snapshot.UnavailableSignals,
		databaseUnavailableSignalResponse{Name: "qps", Scope: signalScopeOptional, ReasonCode: reasonPermissionDenied},
		databaseUnavailableSignalResponse{Name: "connectionsMax", Scope: signalScopeCore, ReasonCode: reasonUnknown},
	)

	findings := h.buildDatabaseHealthFindings(&snapshot)

	if !containsFindingSignal(findings, "connectionUsagePercent") {
		t.Fatalf("expected established connection finding, got %+v", findings)
	}
	if !containsFindingSignal(findings, "lockWaitCount") {
		t.Fatalf("expected lock wait finding, got %+v", findings)
	}
	if !containsFindingSignal(findings, "oldestPendingTaskAgeSec") {
		t.Fatalf("expected backlog finding, got %+v", findings)
	}
	if containsFindingSignal(findings, "qps") {
		t.Fatalf("optional unavailable signal should not create finding, got %+v", findings)
	}

	coreUnavailable := findingBySignal(findings, "connectionsMax")
	if coreUnavailable == nil {
		t.Fatalf("expected core unavailable finding, got %+v", findings)
	}
	if len(coreUnavailable.Evidence) == 0 || coreUnavailable.Recommendation == "" {
		t.Fatalf("expected finding evidence and recommendation, got %+v", coreUnavailable)
	}
}

func TestBuildDatabaseHealthFindingsUsesPendingTaskTargetAsAlertSource(t *testing.T) {
	h := &HealthHandler{}
	snapshot := baseSnapshotForStatusTest()
	snapshot.CoreSignals.OldestPendingTaskAgeSec = 120

	findings := h.buildDatabaseHealthFindings(&snapshot)

	finding := findingBySignal(findings, "oldestPendingTaskAgeSec")
	if finding == nil {
		t.Fatalf("expected backlog finding at target threshold, got %+v", findings)
	}
	if finding.Severity != "warning" {
		t.Fatalf("expected warning severity at target threshold, got %+v", finding)
	}
	if strings.Contains(finding.Description, "600 seconds") || strings.Contains(finding.Recommendation, "600 seconds") {
		t.Fatalf("expected finding copy to avoid stale 600-second threshold, got %+v", finding)
	}
}

func TestBuildDatabaseHealthFindingsMarksCriticalPendingTaskBacklog(t *testing.T) {
	h := &HealthHandler{}
	snapshot := baseSnapshotForStatusTest()
	snapshot.CoreSignals.OldestPendingTaskAgeSec = 600

	findings := h.buildDatabaseHealthFindings(&snapshot)

	finding := findingBySignal(findings, "oldestPendingTaskAgeSec")
	if finding == nil {
		t.Fatalf("expected backlog finding, got %+v", findings)
	}
	if finding.Severity != "critical" {
		t.Fatalf("expected critical severity above critical threshold, got %+v", finding)
	}
	if !containsEvidence(finding.Evidence, "target<2m") || !containsEvidence(finding.Evidence, "criticalThreshold=10m") {
		t.Fatalf("expected target and critical threshold evidence, got %+v", finding.Evidence)
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
	return alertByTitle(alerts, title) != nil
}

func alertByTitle(alerts []databaseHealthAlertResponse, title string) *databaseHealthAlertResponse {
	for _, alert := range alerts {
		if alert.Title == title {
			return &alert
		}
	}
	return nil
}

func containsFindingSignal(findings []databaseHealthFindingResponse, signal string) bool {
	return findingBySignal(findings, signal) != nil
}

func findingBySignal(findings []databaseHealthFindingResponse, signal string) *databaseHealthFindingResponse {
	for i := range findings {
		if findings[i].Signal == signal {
			return &findings[i]
		}
	}
	return nil
}

func containsEvidence(evidence []string, item string) bool {
	for _, entry := range evidence {
		if entry == item {
			return true
		}
	}
	return false
}
