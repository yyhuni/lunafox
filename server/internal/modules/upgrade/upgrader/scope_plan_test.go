package upgrader

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
)

func nginxScopeObservation(configDigest string, healthy bool) RuntimeObservation {
	return RuntimeObservation{
		Images: map[string]string{
			runtimeAgentComponent:      testManifestDigest,
			runtimeBootstrapComponent:  testManifestDigest,
			"engine.port-scan.package": testManifestDigest,
			"engine.port-scan.runtime": testManifestDigest,
			runtimeFrontendComponent:   testManifestDigest,
			runtimeNginxComponent:      testManifestDigest,
			runtimeServerComponent:     testManifestDigest,
		},
		NginxHealthy:      healthy,
		NginxConfigDigest: configDigest,
		ObservedAt:        time.Now().UTC(),
	}
}

func frontendScopePlannerFixture(t *testing.T, observer RuntimeObserver) (*HostScopePlanner, *JournalStore, CandidateDeployment) {
	t.Helper()
	root := newComposeRoot(t)
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	baselineDigest := testManifestDigest
	targetDigest := "sha256:" + strings.Repeat("e", 64)
	components := []DeploymentComponent{
		{ID: runtimeAgentComponent, Digest: baselineDigest},
		{ID: runtimeBootstrapComponent, Digest: baselineDigest},
		{ID: "engine.port-scan.package", Digest: baselineDigest},
		{ID: "engine.port-scan.runtime", Digest: baselineDigest},
		{ID: runtimeFrontendComponent, Digest: targetDigest},
		{ID: runtimeNginxComponent, Digest: baselineDigest},
		{ID: runtimeServerComponent, Digest: baselineDigest},
	}
	sortDeploymentComponents(components)
	candidate := CandidateDeployment{
		ReleaseVersion:    "1.1.0",
		CompositionDigest: "sha256:" + strings.Repeat("f", 64),
		Components:        components,
		Capabilities:      DeploymentCapabilities{DynamicFrontendUpstream: true},
	}
	baselineComponents := append([]DeploymentComponent(nil), components...)
	for index := range baselineComponents {
		if baselineComponents[index].ID == runtimeFrontendComponent {
			baselineComponents[index].Digest = baselineDigest
		}
	}
	state := ConfirmedDeploymentState{
		SchemaVersion:     ConfirmedStateSchema,
		OperationID:       "baseline-operation",
		ManifestDigest:    testManifestDigest,
		CompositionDigest: "sha256:" + strings.Repeat("a", 64),
		ReleaseVersion:    "1.0.0",
		Components:        baselineComponents,
		Capabilities:      DeploymentCapabilities{DynamicFrontendUpstream: true},
		NginxConfigDigest: "sha256:" + strings.Repeat("1", 64),
		ConfirmedAt:       time.Now().UTC(),
	}
	state.StateDigest, err = state.derivedDigest()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConfirmedDeploymentState(state); err != nil {
		t.Fatal(err)
	}
	source := CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
		return candidate, nil
	})
	return NewHostScopePlanner(store, source, observer), store, candidate
}

func TestHostScopePlannerRejectsCandidateOlderThanConfirmedBeforeAnyPlan(t *testing.T) {
	for _, requireFull := range []bool{false, true} {
		t.Run(fmt.Sprintf("requireFull=%t", requireFull), func(t *testing.T) {
			planner, _, candidate := frontendScopePlannerFixture(t, RuntimeObserverFunc(func(context.Context) (RuntimeObservation, error) {
				return nginxScopeObservation("sha256:"+strings.Repeat("1", 64), true), nil
			}))
			candidate.ReleaseVersion = "0.9.0"
			planner.candidate = CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
				return candidate, nil
			})
			if _, err := planner.Plan(context.Background(), "older-candidate", testManifestDigest, requireFull); !errors.Is(err, ErrCandidateReleaseOlderThanConfirmed) {
				t.Fatalf("Plan() error = %v, want ErrCandidateReleaseOlderThanConfirmed", err)
			}
		})
	}
}

func TestHostScopePlannerDoesNotSelectFrontendOnlyForEqualConfirmedRelease(t *testing.T) {
	t.Run("manifest conflict", func(t *testing.T) {
		planner, _, candidate := frontendScopePlannerFixture(t, RuntimeObserverFunc(func(context.Context) (RuntimeObservation, error) {
			return nginxScopeObservation("sha256:"+strings.Repeat("1", 64), true), nil
		}))
		candidate.ReleaseVersion = "1.0.0"
		planner.candidate = CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
			return candidate, nil
		})
		otherManifestDigest := "sha256:" + strings.Repeat("d", 64)
		if _, err := planner.Plan(context.Background(), "equal-conflict", otherManifestDigest, false); !errors.Is(err, ErrCandidateReleaseConflictsWithConfirmed) {
			t.Fatalf("Plan() error = %v, want ErrCandidateReleaseConflictsWithConfirmed", err)
		}
	})

	t.Run("same identity with inconsistent components falls back full", func(t *testing.T) {
		planner, _, candidate := frontendScopePlannerFixture(t, RuntimeObserverFunc(func(context.Context) (RuntimeObservation, error) {
			return nginxScopeObservation("sha256:"+strings.Repeat("1", 64), true), nil
		}))
		candidate.ReleaseVersion = "1.0.0"
		planner.candidate = CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
			return candidate, nil
		})
		plan, err := planner.Plan(context.Background(), "equal-same-identity", testManifestDigest, false)
		if err != nil {
			t.Fatal(err)
		}
		if plan.ExecutionMode != ExecutionModeFull {
			t.Fatalf("equal release mode = %q, want full", plan.ExecutionMode)
		}
		if !strings.Contains(plan.Diagnostic, "already confirmed") {
			t.Fatalf("equal release diagnostic = %q, want already-confirmed explanation", plan.Diagnostic)
		}
	})
}

func TestHostScopePlannerFallsBackToFullWhenCompositionCacheIsMissing(t *testing.T) {
	root := newComposeRoot(t)
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := releasemanifest.Load(filepath.Join(root, defaultManifestName))
	if err != nil {
		t.Fatal(err)
	}
	missingComposition := CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
		return CandidateDeployment{}, ErrCompositionUnavailable
	})
	planner := NewHostScopePlanner(store, missingComposition, nil)
	plan, err := planner.Plan(context.Background(), "missing-composition", manifest.Digest(), false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ExecutionMode != ExecutionModeFull {
		t.Fatalf("fallback execution mode = %q, want full", plan.ExecutionMode)
	}
	if plan.Candidate.CompositionDigest != "" {
		t.Fatalf("fallback plan unexpectedly retained composition digest %q", plan.Candidate.CompositionDigest)
	}
}

func TestHostScopePlannerDoesNotDowngradeInvalidComposition(t *testing.T) {
	root := newComposeRoot(t)
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := releasemanifest.Load(filepath.Join(root, defaultManifestName))
	if err != nil {
		t.Fatal(err)
	}
	invalid := CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
		return CandidateDeployment{}, errors.Join(ErrCompositionInvalid, errors.New("digest mismatch"))
	})
	planner := NewHostScopePlanner(store, invalid, nil)
	if _, err := planner.Plan(context.Background(), "invalid-composition", manifest.Digest(), false); !errors.Is(err, ErrCompositionInvalid) {
		t.Fatalf("invalid composition error = %v, want ErrCompositionInvalid", err)
	}
}

func TestHostScopePlannerRejectsNginxConfigurationDriftBeforeFrontendOnlyPlan(t *testing.T) {
	planner, _, _ := frontendScopePlannerFixture(t, RuntimeObserverFunc(func(context.Context) (RuntimeObservation, error) {
		return nginxScopeObservation("", true), nil
	}))
	plan, err := planner.Plan(context.Background(), "nginx-preflight-drift", testManifestDigest, false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ExecutionMode != ExecutionModeFull {
		t.Fatalf("drifted nginx preflight mode = %q, want full", plan.ExecutionMode)
	}
	if !strings.Contains(plan.Diagnostic, "runtime observation") {
		t.Fatalf("drifted nginx diagnostic = %q", plan.Diagnostic)
	}
}

func TestHostScopePlannerRejectsNginxConfigurationDriftDuringRevalidation(t *testing.T) {
	configDigest := "sha256:" + strings.Repeat("1", 64)
	observation := nginxScopeObservation(configDigest, true)
	planner, _, _ := frontendScopePlannerFixture(t, RuntimeObserverFunc(func(context.Context) (RuntimeObservation, error) {
		return observation, nil
	}))
	plan, err := planner.Plan(context.Background(), "nginx-lock-drift", testManifestDigest, false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ExecutionMode != ExecutionModeFrontendOnly {
		t.Fatalf("initial plan mode = %q, want frontend_only", plan.ExecutionMode)
	}
	observation.NginxConfigDigest = "sha256:" + strings.Repeat("2", 64)
	if err := planner.Revalidate(context.Background(), plan); !errors.Is(err, ErrScopePlanStale) {
		t.Fatalf("revalidation error = %v, want ErrScopePlanStale", err)
	}
}

func TestHostScopePlannerFallsBackToFullWhenObservationMissesConfirmedComponents(t *testing.T) {
	observation := nginxScopeObservation("sha256:"+strings.Repeat("1", 64), true)
	delete(observation.Images, runtimeBootstrapComponent)
	delete(observation.Images, "engine.port-scan.package")
	delete(observation.Images, "engine.port-scan.runtime")
	planner, _, _ := frontendScopePlannerFixture(t, RuntimeObserverFunc(func(context.Context) (RuntimeObservation, error) {
		return observation, nil
	}))

	plan, err := planner.Plan(context.Background(), "incomplete-runtime-observation", testManifestDigest, false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ExecutionMode != ExecutionModeFull {
		t.Fatalf("incomplete observation mode = %q, want full", plan.ExecutionMode)
	}
	if !strings.Contains(plan.Diagnostic, "runtime containers drift") {
		t.Fatalf("incomplete observation diagnostic = %q", plan.Diagnostic)
	}
}

func TestHostScopePlannerFallsBackToFullWhenBootstrapOrEngineDrifts(t *testing.T) {
	tests := []struct {
		name        string
		componentID string
	}{
		{name: "bootstrap", componentID: runtimeBootstrapComponent},
		{name: "engine runtime", componentID: "engine.port-scan.runtime"},
		{name: "engine package", componentID: "engine.port-scan.package"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			observation := nginxScopeObservation("sha256:"+strings.Repeat("1", 64), true)
			observation.Images[test.componentID] = "sha256:" + strings.Repeat("2", 64)
			planner, _, _ := frontendScopePlannerFixture(t, RuntimeObserverFunc(func(context.Context) (RuntimeObservation, error) {
				return observation, nil
			}))

			plan, err := planner.Plan(context.Background(), "runtime-component-drift", testManifestDigest, false)
			if err != nil {
				t.Fatal(err)
			}
			if plan.ExecutionMode != ExecutionModeFull {
				t.Fatalf("drifted %s mode = %q, want full", test.componentID, plan.ExecutionMode)
			}
		})
	}
}

func TestHostScopePlannerRevalidationRejectsEngineDrift(t *testing.T) {
	observation := nginxScopeObservation("sha256:"+strings.Repeat("1", 64), true)
	planner, _, _ := frontendScopePlannerFixture(t, RuntimeObserverFunc(func(context.Context) (RuntimeObservation, error) {
		return observation, nil
	}))
	plan, err := planner.Plan(context.Background(), "engine-lock-drift", testManifestDigest, false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ExecutionMode != ExecutionModeFrontendOnly {
		t.Fatalf("initial plan mode = %q, want frontend_only", plan.ExecutionMode)
	}

	observation.Images["engine.port-scan.runtime"] = "sha256:" + strings.Repeat("2", 64)
	if err := planner.Revalidate(context.Background(), plan); !errors.Is(err, ErrScopePlanStale) {
		t.Fatalf("revalidation error = %v, want ErrScopePlanStale", err)
	}
}

func TestHostScopePlannerFullConfirmationRequiresExactLiveInventory(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	otherDigest := "sha256:" + strings.Repeat("b", 64)
	components := []DeploymentComponent{
		{ID: runtimeAgentComponent, Digest: digest},
		{ID: runtimeBootstrapComponent, Digest: digest},
		{ID: "engine.port-scan.package", Digest: digest},
		{ID: "engine.port-scan.runtime", Digest: digest},
		{ID: runtimeFrontendComponent, Digest: digest},
		{ID: runtimeNginxComponent, Digest: digest},
		{ID: runtimeServerComponent, Digest: digest},
	}
	sortDeploymentComponents(components)
	candidate := CandidateDeployment{
		ReleaseVersion:    "1.1.0",
		CompositionDigest: "sha256:" + strings.Repeat("c", 64),
		Components:        components,
		Capabilities:      DeploymentCapabilities{DynamicFrontendUpstream: true},
	}
	observation := nginxScopeObservation(digest, true)
	planner := NewHostScopePlanner(nil, CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
		return candidate, nil
	}), RuntimeObserverFunc(func(context.Context) (RuntimeObservation, error) {
		return observation, nil
	}))
	plan := ScopePlan{
		SchemaVersion:   ScopePlanSchema,
		OperationID:     "full-confirmation",
		ManifestDigest:  testManifestDigest,
		ExecutionMode:   ExecutionModeFull,
		TouchedServices: fullTouchedServices(),
		Candidate:       candidate,
		GeneratedAt:     time.Now().UTC(),
	}
	digestPlan, err := plan.derivedDigest()
	if err != nil {
		t.Fatal(err)
	}
	plan.PlanDigest = digestPlan
	if _, err := planner.ObserveFullDeployment(context.Background(), plan); err != nil {
		t.Fatalf("matching live inventory rejected: %v", err)
	}

	observation.Images["engine.port-scan.runtime"] = otherDigest
	if _, err := planner.ObserveFullDeployment(context.Background(), plan); err == nil || !strings.Contains(err.Error(), "does not match candidate") {
		t.Fatalf("engine inventory drift error = %v, want exact candidate mismatch", err)
	}

	observation.Images["engine.port-scan.runtime"] = digest
	observation.Images[runtimeBootstrapComponent] = otherDigest
	if _, err := planner.ObserveFullDeployment(context.Background(), plan); err == nil || !strings.Contains(err.Error(), "does not match candidate") {
		t.Fatalf("bootstrap drift error = %v, want exact candidate mismatch", err)
	}
}

func availabilityPlannerFixture(t *testing.T) (*HostScopePlanner, *JournalStore, CandidateDeployment, *releasemanifest.Manifest) {
	t.Helper()
	store, err := NewPublicJournalStore(newShortRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	raw := releaseManifestFixtureBytes(t)
	manifest, err := releasemanifest.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath, err := store.ManifestPath(manifest.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	candidate, err := CandidateDeploymentFromManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	candidate.ReleaseVersion = "1.1.0"
	candidate.CompositionDigest = manifest.RuntimeComposition.SHA256
	candidate.Capabilities = DeploymentCapabilities{DynamicFrontendUpstream: true}
	state := confirmedAvailabilityState(t, manifest.Digest(), candidate, "1.0.0")
	if err := store.SaveConfirmedDeploymentState(state); err != nil {
		t.Fatal(err)
	}
	planner := NewHostScopePlanner(store, CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
		return candidate, nil
	}), nil)
	return planner, store, candidate, manifest
}

func confirmedAvailabilityState(t *testing.T, manifestDigest string, candidate CandidateDeployment, releaseVersion string) ConfirmedDeploymentState {
	t.Helper()
	state := ConfirmedDeploymentState{
		SchemaVersion:     ConfirmedStateSchema,
		OperationID:       "availability-baseline",
		ManifestDigest:    manifestDigest,
		CompositionDigest: candidate.CompositionDigest,
		ReleaseVersion:    releaseVersion,
		Components:        append([]DeploymentComponent(nil), candidate.Components...),
		Capabilities:      candidate.Capabilities,
		NginxConfigDigest: testManifestDigest,
		ConfirmedAt:       time.Now().UTC(),
	}
	digest, err := state.derivedDigest()
	if err != nil {
		t.Fatal(err)
	}
	state.StateDigest = digest
	return state
}

func TestHostScopePlannerCandidateAvailabilityComparesCompleteInventory(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*CandidateDeployment)
		decision CandidateAvailabilityDecision
	}{
		{name: "metadata-only release is already applied", decision: CandidateAvailabilityAlreadyApplied},
		{
			name: "server change is available",
			mutate: func(candidate *CandidateDeployment) {
				for index := range candidate.Components {
					if candidate.Components[index].ID == runtimeServerComponent {
						candidate.Components[index].Digest = "sha256:" + strings.Repeat("c", 64)
					}
				}
			},
			decision: CandidateAvailabilityAvailable,
		},
		{
			name: "bootstrap change is available",
			mutate: func(candidate *CandidateDeployment) {
				for index := range candidate.Components {
					if candidate.Components[index].ID == runtimeBootstrapComponent {
						candidate.Components[index].Digest = "sha256:" + strings.Repeat("c", 64)
					}
				}
			},
			decision: CandidateAvailabilityAvailable,
		},
		{
			name: "engine package change is available",
			mutate: func(candidate *CandidateDeployment) {
				for index := range candidate.Components {
					if strings.HasPrefix(candidate.Components[index].ID, "engine.") {
						candidate.Components[index].Digest = "sha256:" + strings.Repeat("d", 64)
					}
				}
			},
			decision: CandidateAvailabilityAvailable,
		},
		{
			name: "deployment capability change is available",
			mutate: func(candidate *CandidateDeployment) {
				candidate.Capabilities.DynamicFrontendUpstream = false
			},
			decision: CandidateAvailabilityAvailable,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			planner, _, candidate, manifest := availabilityPlannerFixture(t)
			if test.mutate != nil {
				test.mutate(&candidate)
			}
			planner.candidate = CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
				return candidate, nil
			})
			availability, err := planner.CandidateAvailability(context.Background(), manifest.Digest())
			if err != nil {
				t.Fatal(err)
			}
			if availability.Decision != test.decision {
				t.Fatalf("CandidateAvailability() decision = %q, want %q", availability.Decision, test.decision)
			}
			if availability.ConfirmedDeploymentVersion != "1.0.0" || availability.BaselineStateDigest == "" {
				t.Fatalf("CandidateAvailability() omitted confirmed evidence: %#v", availability)
			}
		})
	}
}

func TestHostScopePlannerCandidateAvailabilityTreatsMigrationDifferenceAsAvailable(t *testing.T) {
	planner, store, candidate, _ := availabilityPlannerFixture(t)
	raw := releaseManifestFixtureBytes(t)
	raw = []byte(strings.Replace(string(raw), "hasDatabaseMigration: false", "hasDatabaseMigration: true", 1))
	raw = []byte(strings.Replace(string(raw), "migrationType: \"none\"", "migrationType: \"compatible\"", 1))
	raw = []byte(strings.Replace(string(raw), `migrationId: ""
    checksum: ""`, `migrationId: 000002_upgrade
    checksum: sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc`, 1))
	manifest, err := releasemanifest.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath, err := store.ManifestPath(manifest.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	candidate.CompositionDigest = manifest.RuntimeComposition.SHA256
	planner.candidate = CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
		return candidate, nil
	})
	availability, err := planner.CandidateAvailability(context.Background(), manifest.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if availability.Decision != CandidateAvailabilityAvailable {
		t.Fatalf("migration-changing candidate decision = %q, want available", availability.Decision)
	}
}

func TestHostScopePlannerCandidateAvailabilityResolvesVersionBounds(t *testing.T) {
	t.Run("older candidate is not newer", func(t *testing.T) {
		planner, _, candidate, manifest := availabilityPlannerFixture(t)
		candidate.ReleaseVersion = "0.9.0"
		planner.candidate = CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
			return candidate, nil
		})
		availability, err := planner.CandidateAvailability(context.Background(), manifest.Digest())
		if err != nil {
			t.Fatal(err)
		}
		if availability.Decision != CandidateAvailabilityNotNewer {
			t.Fatalf("older candidate decision = %q, want not_newer", availability.Decision)
		}
	})

	t.Run("same release with different manifest conflicts", func(t *testing.T) {
		planner, _, candidate, _ := availabilityPlannerFixture(t)
		candidate.ReleaseVersion = "1.0.0"
		planner.candidate = CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
			return candidate, nil
		})
		availability, err := planner.CandidateAvailability(context.Background(), "sha256:"+strings.Repeat("d", 64))
		if err != nil {
			t.Fatal(err)
		}
		if availability.Decision != CandidateAvailabilityConflict {
			t.Fatalf("same-version conflict decision = %q, want conflict", availability.Decision)
		}
	})
}

func TestHostScopePlannerCandidateAvailabilityFallsBackOnlyForMissingEvidence(t *testing.T) {
	t.Run("missing cached candidate manifest", func(t *testing.T) {
		planner, store, _, manifest := availabilityPlannerFixture(t)
		manifestPath, err := store.ManifestPath(manifest.Digest())
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(manifestPath); err != nil {
			t.Fatal(err)
		}
		planner.candidate = NewCompositionCandidateDeploymentSource(store)
		availability, err := planner.CandidateAvailability(context.Background(), manifest.Digest())
		if err != nil {
			t.Fatal(err)
		}
		if availability.Decision != CandidateAvailabilityFallback {
			t.Fatalf("missing manifest decision = %q, want fallback", availability.Decision)
		}
		if _, err := store.LoadCurrent(); !errors.Is(err, ErrJournalNotFound) {
			t.Fatalf("availability probe created a journal: %v", err)
		}
		entries, err := os.ReadDir(filepath.Join(store.Directory(), PlanDirectory))
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Fatalf("availability probe persisted scope plans: %#v", entries)
		}
	})

	t.Run("missing composition", func(t *testing.T) {
		planner, _, _, manifest := availabilityPlannerFixture(t)
		planner.candidate = CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
			return CandidateDeployment{}, ErrCompositionUnavailable
		})
		availability, err := planner.CandidateAvailability(context.Background(), manifest.Digest())
		if err != nil {
			t.Fatal(err)
		}
		if availability.Decision != CandidateAvailabilityFallback {
			t.Fatalf("missing composition decision = %q, want fallback", availability.Decision)
		}
	})

	t.Run("missing confirmed baseline", func(t *testing.T) {
		planner, store, _, manifest := availabilityPlannerFixture(t)
		if err := os.Remove(store.ConfirmedDeploymentStatePath()); err != nil {
			t.Fatal(err)
		}
		availability, err := planner.CandidateAvailability(context.Background(), manifest.Digest())
		if err != nil {
			t.Fatal(err)
		}
		if availability.Decision != CandidateAvailabilityFallback {
			t.Fatalf("missing baseline decision = %q, want fallback", availability.Decision)
		}
	})

	t.Run("missing confirmed manifest", func(t *testing.T) {
		planner, store, candidate, manifest := availabilityPlannerFixture(t)
		state, err := store.LoadConfirmedDeploymentState()
		if err != nil {
			t.Fatal(err)
		}
		state.ManifestDigest = "sha256:" + strings.Repeat("d", 64)
		state.StateDigest, err = state.derivedDigest()
		if err != nil {
			t.Fatal(err)
		}
		if err := store.SaveConfirmedDeploymentState(state); err != nil {
			t.Fatal(err)
		}
		planner.candidate = CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
			return candidate, nil
		})
		availability, err := planner.CandidateAvailability(context.Background(), manifest.Digest())
		if err != nil {
			t.Fatal(err)
		}
		if availability.Decision != CandidateAvailabilityFallback {
			t.Fatalf("missing confirmed manifest decision = %q, want fallback", availability.Decision)
		}
	})

	t.Run("invalid composition fails closed", func(t *testing.T) {
		planner, _, _, manifest := availabilityPlannerFixture(t)
		planner.candidate = CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
			return CandidateDeployment{}, errors.Join(ErrCompositionInvalid, errors.New("digest mismatch"))
		})
		if _, err := planner.CandidateAvailability(context.Background(), manifest.Digest()); !errors.Is(err, ErrCompositionInvalid) {
			t.Fatalf("invalid composition error = %v, want ErrCompositionInvalid", err)
		}
	})
}

func TestHostScopePlannerCandidateAvailabilityUsesVersionBoundsBeforeInventory(t *testing.T) {
	t.Run("older candidate is not newer", func(t *testing.T) {
		planner, _, candidate, manifest := availabilityPlannerFixture(t)
		candidate.ReleaseVersion = "0.9.0"
		planner.candidate = CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
			return candidate, nil
		})
		availability, err := planner.CandidateAvailability(context.Background(), manifest.Digest())
		if err != nil {
			t.Fatal(err)
		}
		if availability.Decision != CandidateAvailabilityNotNewer || availability.ConfirmedDeploymentVersion != "1.0.0" || availability.BaselineStateDigest == "" {
			t.Fatalf("older candidate availability = %#v", availability)
		}
	})

	t.Run("same version with a different manifest conflicts", func(t *testing.T) {
		planner, _, candidate, _ := availabilityPlannerFixture(t)
		candidate.ReleaseVersion = "1.0.0"
		planner.candidate = CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
			return candidate, nil
		})
		availability, err := planner.CandidateAvailability(context.Background(), "sha256:"+strings.Repeat("d", 64))
		if err != nil {
			t.Fatal(err)
		}
		if availability.Decision != CandidateAvailabilityConflict || availability.BaselineStateDigest == "" {
			t.Fatalf("conflicting manifest availability = %#v", availability)
		}
	})

	t.Run("missing baseline falls back without evidence", func(t *testing.T) {
		planner, store, _, manifest := availabilityPlannerFixture(t)
		if err := os.Remove(store.ConfirmedDeploymentStatePath()); err != nil {
			t.Fatal(err)
		}
		availability, err := planner.CandidateAvailability(context.Background(), manifest.Digest())
		if err != nil {
			t.Fatal(err)
		}
		if availability.Decision != CandidateAvailabilityFallback || availability.BaselineStateDigest != "" || availability.ConfirmedDeploymentVersion != "" {
			t.Fatalf("missing baseline availability = %#v", availability)
		}
	})

	t.Run("legacy store falls back", func(t *testing.T) {
		store, err := NewJournalStore(newShortRoot(t))
		if err != nil {
			t.Fatal(err)
		}
		planner := NewHostScopePlanner(store, CandidateDeploymentSourceFunc(func(context.Context, string) (CandidateDeployment, error) {
			t.Fatal("legacy store must not compare inventory")
			return CandidateDeployment{}, nil
		}), nil)
		availability, err := planner.CandidateAvailability(context.Background(), "sha256:"+strings.Repeat("e", 64))
		if err != nil {
			t.Fatal(err)
		}
		if availability.Decision != CandidateAvailabilityFallback || availability.BaselineStateDigest != "" {
			t.Fatalf("legacy store availability = %#v", availability)
		}
	})
}
