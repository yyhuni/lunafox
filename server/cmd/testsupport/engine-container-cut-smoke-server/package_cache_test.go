package main

import (
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

func TestValidatePackageReceiptEnvelopeAcceptsSupportedModes(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		mode  string
		count int
	}{
		{name: "development one package", mode: packageReceiptModeDevelopment, count: 1},
		{name: "development four packages", mode: packageReceiptModeDevelopment, count: 4},
		{name: "production one package", mode: packageReceiptModeProduction, count: 1},
		{name: "production four packages", mode: packageReceiptModeProduction, count: 4},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			receipt := packageBuildResults{
				SchemaVersion: packageReceiptSchema,
				Mode:          test.mode,
				Packages:      make([]packageBuildArtifact, test.count),
			}
			if err := validatePackageReceiptEnvelope(receipt); err != nil {
				t.Fatalf("validatePackageReceiptEnvelope(%q, %d): %v", test.mode, test.count, err)
			}
		})
	}
}

func TestValidatePackageReceiptEnvelopeRejectsEmptyPackageSet(t *testing.T) {
	t.Parallel()

	err := validatePackageReceiptEnvelope(packageBuildResults{
		SchemaVersion: packageReceiptSchema,
		Mode:          packageReceiptModeProduction,
	})
	if err == nil || !strings.Contains(err.Error(), "at least one package") {
		t.Fatalf("validatePackageReceiptEnvelope() error = %v; want non-empty package set", err)
	}
}

func TestValidatePackageReceiptEnvelopeRejectsUnknownMode(t *testing.T) {
	t.Parallel()

	err := validatePackageReceiptEnvelope(packageBuildResults{
		SchemaVersion: packageReceiptSchema,
		Mode:          "preview",
		Packages:      make([]packageBuildArtifact, 3),
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported mode") {
		t.Fatalf("validatePackageReceiptEnvelope() error = %v; want unsupported mode", err)
	}
}

func TestSmokeEvidenceRecordsSortedInstalledEngineIDs(t *testing.T) {
	t.Parallel()

	evidence := newSmokeEvidence([]string{
		"engine.lunafox.website_discovery",
		"engine.lunafox.fourth_engine",
		"engine.lunafox.port_scan",
		"engine.lunafox.subdomain_discovery",
	}, packageReceiptModeProduction, nil)
	if got, want := strings.Join(evidence.PackageCacheColdStart.InstalledEngineIDs, ","), "engine.lunafox.fourth_engine,engine.lunafox.port_scan,engine.lunafox.subdomain_discovery,engine.lunafox.website_discovery"; got != want {
		t.Fatalf("installed Engine IDs = %q, want %q", got, want)
	}
	if evidence.PackageCacheColdStart.InstalledPackages != 4 {
		t.Fatalf("installed package count = %d, want 4", evidence.PackageCacheColdStart.InstalledPackages)
	}
}

func TestSortedScenarioEngineIDsDeduplicatesAndOrders(t *testing.T) {
	t.Parallel()

	got := strings.Join(sortedScenarioEngineIDs([]smokePlanRecordSnapshot{
		{engineID: "engine.lunafox.website_discovery"},
		{engineID: "engine.lunafox.port_scan"},
		{engineID: "engine.lunafox.port_scan"},
		{engineID: "engine.lunafox.subdomain_discovery"},
		{},
	}), ",")
	const want = "engine.lunafox.port_scan,engine.lunafox.subdomain_discovery,engine.lunafox.website_discovery"
	if got != want {
		t.Fatalf("scenario Engine IDs = %q, want %q", got, want)
	}
}

func TestOverrideRuntimeImageForSmokeRequiresExactSourceAndCanonicalFixture(t *testing.T) {
	source := "localhost:5000/nuclei@sha256:" + strings.Repeat("a", 64)
	fixture := "localhost:5000/nuclei-fixture@sha256:" + strings.Repeat("b", 64)
	reader := &packageMapReader{packages: map[string]scanapp.PlanTaskPackage{
		engineNucleiID: {
			Identity:         scanapp.PlanTaskPackageIdentity{EngineID: engineNucleiID, PackageDigest: "sha256:" + strings.Repeat("c", 64)},
			RuntimeImageRefs: []string{source},
		},
	}}
	if err := reader.OverrideRuntimeImageForSmoke(engineNucleiID, source, fixture); err != nil {
		t.Fatalf("override fixture image: %v", err)
	}
	if got := reader.packages[engineNucleiID].RuntimeImageRefs; len(got) != 1 || got[0] != fixture {
		t.Fatalf("runtime image refs = %#v, want fixture ref", got)
	}
	if err := reader.OverrideRuntimeImageForSmoke(engineNucleiID, source+" ", fixture); err == nil {
		t.Fatal("source ref with trailing space unexpectedly accepted")
	}
	if err := reader.OverrideRuntimeImageForSmoke(engineNucleiID, source, "localhost:5000/nuclei-fixture:latest"); err == nil {
		t.Fatal("tag-only fixture ref unexpectedly accepted")
	}
	if _, err := ociartifact.ParseDigestReference(fixture); err != nil {
		t.Fatalf("fixture test ref is not canonical: %v", err)
	}
}
