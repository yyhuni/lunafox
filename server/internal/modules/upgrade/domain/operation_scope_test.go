package domain

import "testing"

func TestValidateExecutionTransitionAllowsQueuedPreflightOnlyForFrontendPlan(t *testing.T) {
	if err := ValidateExecutionTransition(ExecutionModeFull, StatusQueued, StatusPreflight); err == nil {
		t.Fatal("full operation accepted queued -> preflight")
	}
	if err := ValidateExecutionTransition(ExecutionModeFrontendOnly, StatusQueued, StatusPreflight); err != nil {
		t.Fatalf("frontend-only queued -> preflight: %v", err)
	}
	if err := ValidateExecutionTransition(ExecutionModeFrontendOnly, StatusQueued, StatusFailed); err != nil {
		t.Fatalf("frontend-only queued -> failed stale-plan result: %v", err)
	}
	if err := ValidateExecutionTransition(ExecutionModeFull, StatusQueued, StatusFailed); err == nil {
		t.Fatal("full operation accepted queued -> failed")
	}
	if err := ValidateExecutionTransition(ExecutionModeFrontendOnly, StatusPreflight, StatusMigrating); err == nil {
		t.Fatal("frontend-only operation accepted migration stage")
	}
}

func TestValidateJournalRecoveryTransitionAllowsOnlyTrustedForwardCheckpoints(t *testing.T) {
	if err := ValidateJournalRecoveryTransition(ExecutionModeFull, "none", MigrationStatusNotStarted, StatusStopping, StatusVerifying); err != nil {
		t.Fatalf("full stopping -> verifying journal replay: %v", err)
	}
	if err := ValidateJournalRecoveryTransition(ExecutionModeFull, "none", MigrationStatusNotStarted, StatusStopping, StatusSucceeded); err == nil {
		t.Fatal("journal replay accepted terminal success")
	}
	if err := ValidateJournalRecoveryTransition(ExecutionModeFull, "compatible", MigrationStatusRunning, StatusStopping, StatusRestarting); err == nil {
		t.Fatal("journal replay bypassed incomplete migration")
	}
	if err := ValidateJournalRecoveryTransition(ExecutionModeFrontendOnly, "none", MigrationStatusNotStarted, StatusPreflight, StatusAgentVerifying); err == nil {
		t.Fatal("frontend-only journal replay accepted Agent verification")
	}
}

func TestValidateScopePlanRequiresExactFrontendService(t *testing.T) {
	digest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := ValidateScopePlan(ExecutionModeFrontendOnly, PlanSummary{TouchedServices: []string{"frontend"}}, digest, digest, "1.2.3"); err != nil {
		t.Fatalf("valid frontend-only scope plan: %v", err)
	}
	if err := ValidateScopePlan(ExecutionModeFrontendOnly, PlanSummary{TouchedServices: []string{"frontend", "server"}}, digest, digest, "1.2.3"); err == nil {
		t.Fatal("frontend-only plan accepted an additional service")
	}
}

func TestValidateScopePlanKeepsFullPlansFreeOfSelectiveBaselineEvidence(t *testing.T) {
	digest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	services := []string{"agent", "bootstrap", "engine", "engine_package", "engine_runtime", "frontend", "migration", "nginx", "server"}
	if err := ValidateScopePlan(ExecutionModeFull, PlanSummary{TouchedServices: services}, digest, "", ""); err != nil {
		t.Fatalf("valid full scope plan: %v", err)
	}
	if err := ValidateScopePlan(ExecutionModeFull, PlanSummary{TouchedServices: services}, digest, digest, "1.2.3"); err == nil {
		t.Fatal("full scope plan accepted frontend-only baseline evidence")
	}
}
