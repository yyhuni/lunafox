package repositoryname

import (
	"strings"
	"testing"
)

func TestFirstPartyOCIRepositoryName(t *testing.T) {
	repository, err := FirstPartyOCIRepositoryName("engine.lunafox.website_discovery")
	if err != nil {
		t.Fatalf("FirstPartyOCIRepositoryName() error = %v", err)
	}
	if want := "lunafox-engine-runtime-website-discovery"; repository != want {
		t.Fatalf("FirstPartyOCIRepositoryName() = %q, want %q", repository, want)
	}
}

func TestFirstPartyOCIRepositoryNameRejectsNonCanonicalID(t *testing.T) {
	if _, err := FirstPartyOCIRepositoryName("engine.lunafox.website-discovery"); err == nil {
		t.Fatal("expected non-canonical first-party engine ID to be rejected")
	}
}

func TestFirstPartyRuntimeImageRepositoryName(t *testing.T) {
	repository, err := FirstPartyRuntimeImageRepositoryName("engine.lunafox.website_discovery")
	if err != nil {
		t.Fatalf("FirstPartyRuntimeImageRepositoryName() error = %v", err)
	}
	if want := "lunafox-engine-runtime-website-discovery"; repository != want {
		t.Fatalf("FirstPartyRuntimeImageRepositoryName() = %q, want %q", repository, want)
	}
}

func TestFirstPartyRuntimeImageRepositoryNameRejectsNonCanonicalID(t *testing.T) {
	if _, err := FirstPartyRuntimeImageRepositoryName("engine.lunafox.website-discovery"); err == nil {
		t.Fatal("expected non-canonical first-party engine ID to be rejected")
	}
}

func TestFirstPartyRepositoryNamesDeriveDynamicallyNamedEngine(t *testing.T) {
	const engineID = "engine.lunafox.http2_probe"

	packageRepository, err := FirstPartyOCIRepositoryName(engineID)
	if err != nil {
		t.Fatalf("FirstPartyOCIRepositoryName() error = %v", err)
	}
	if want := "lunafox-engine-runtime-http2-probe"; packageRepository != want {
		t.Fatalf("FirstPartyOCIRepositoryName() = %q, want %q", packageRepository, want)
	}

	runtimeRepository, err := FirstPartyRuntimeImageRepositoryName(engineID)
	if err != nil {
		t.Fatalf("FirstPartyRuntimeImageRepositoryName() error = %v", err)
	}
	if want := "lunafox-engine-runtime-http2-probe"; runtimeRepository != want {
		t.Fatalf("FirstPartyRuntimeImageRepositoryName() = %q, want %q", runtimeRepository, want)
	}
	if packageRepository != runtimeRepository {
		t.Fatalf("first-party package and runtime repositories must agree: package=%q runtime=%q", packageRepository, runtimeRepository)
	}
	if err := ValidateFirstPartyBuiltinEngineID(engineID); err != nil {
		t.Fatalf("ValidateFirstPartyBuiltinEngineID() error = %v", err)
	}
}

func TestFirstPartyRuntimeImageRepositoryNameRejectsInvalidLocalName(t *testing.T) {
	tests := []string{
		"engine.lunafox.",
		"engine.lunafox._probe",
		"engine.lunafox.probe-2",
		"engine.lunafox.probe.name",
		"engine.lunafox.Probe",
		" engine.lunafox.probe",
	}

	for _, engineID := range tests {
		t.Run(engineID, func(t *testing.T) {
			if _, err := FirstPartyRuntimeImageRepositoryName(engineID); err == nil {
				t.Fatalf("expected invalid engine ID %q to be rejected", engineID)
			}
		})
	}
}

func TestFirstPartyRuntimeImageRepositoryNameRejectsReleaseScopedIdentity(t *testing.T) {
	tests := []string{
		"engine.lunafox.website_discovery_v2",
		"engine.lunafox.website_discovery_v1_2_3",
		"engine.lunafox.website_discovery_sha256_" + strings.Repeat("a", 64),
		"engine.lunafox.website_discovery_deadbee",
		"engine.lunafox.website_discovery_release_01j2x3k4m5n6p7q8r9s0t1v2w3",
		"engine.lunafox.website_discovery_42",
		"engine.lunafox.website_discovery_550e8400e29b41d4a716446655440000",
	}

	for _, engineID := range tests {
		t.Run(engineID, func(t *testing.T) {
			if _, err := FirstPartyRuntimeImageRepositoryName(engineID); err == nil {
				t.Fatalf("expected release-scoped engine ID %q to be rejected", engineID)
			}
		})
	}
}

func TestFirstPartyRuntimeImageRepositoryNameRejectsNonFirstPartyNamespace(t *testing.T) {
	if _, err := FirstPartyRuntimeImageRepositoryName("engine.example.website_discovery"); err == nil {
		t.Fatal("expected third-party engine ID to be rejected by first-party derivation")
	}
}

func TestValidateFirstPartyBuiltinEngineIDAcceptsCanonicalFirstPartyIdentities(t *testing.T) {
	for _, engineID := range []string{
		FirstPartyEngineIDSubdomainDiscovery,
		FirstPartyEngineIDPortScan,
		FirstPartyEngineIDWebsiteDiscovery,
		"engine.lunafox.http2_probe",
	} {
		if err := ValidateFirstPartyBuiltinEngineID(engineID); err != nil {
			t.Fatalf("ValidateFirstPartyBuiltinEngineID(%q) error = %v", engineID, err)
		}
	}

	if err := ValidateFirstPartyBuiltinEngineID("engine.example.http2_probe"); err == nil {
		t.Fatal("expected a non-first-party identity to be rejected")
	}
}
