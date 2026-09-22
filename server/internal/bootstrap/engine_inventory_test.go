package bootstrap

import (
	"errors"
	"strings"
	"testing"

	enginemanifest "github.com/yyhuni/lunafox/contracts/enginemanifest"
	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/packagemanifest"
	"github.com/yyhuni/lunafox/server/internal/installedengines"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type engineInventoryQueryStub struct {
	packages []installedengines.ResolvedInstalledEnginePackage
	err      error
}

func (stub engineInventoryQueryStub) ListInstalledEnginePackages() ([]installedengines.ResolvedInstalledEnginePackage, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return append([]installedengines.ResolvedInstalledEnginePackage(nil), stub.packages...), nil
}

func (stub engineInventoryQueryStub) GetInstalledEnginePackage(string) (installedengines.ResolvedInstalledEnginePackage, error) {
	return installedengines.ResolvedInstalledEnginePackage{}, errors.New("not used by engine inventory")
}

func TestBuildEngineInventoryProjectsVerifiedPackageAndRuntimeDigests(t *testing.T) {
	artifactManifestDigest := "sha256:" + strings.Repeat("a", 64)
	packageDigest := "sha256:" + strings.Repeat("b", 64)
	runtimeDigest := "sha256:" + strings.Repeat("c", 64)
	packageRecord := installedengines.ResolvedInstalledEnginePackage{
		Registration: catalogdomain.Engine{
			EngineID:      "engine.lunafox.port_scan",
			ArtifactRef:   "ghcr.io/yyhuni/lunafox-engine-port-scan@" + artifactManifestDigest,
			PackageDigest: packageDigest,
		},
		Layout: enginepackagecatalog.EnginePackageLayout{Definition: enginepackagecatalog.EnginePackageDefinition{
			PackageManifest: packagemanifest.PackageManifest{
				EngineID: "engine.lunafox.port_scan",
				RuntimeImage: packagemanifest.RuntimeImage{Refs: []string{
					"ghcr.io/yyhuni/lunafox-engine-runtime-port-scan@" + runtimeDigest,
				}},
			},
			EngineDefinition: enginemanifest.EngineDefinition{EngineID: "engine.lunafox.port_scan"},
		}},
	}
	inventory, err := BuildEngineInventory(engineInventoryQueryStub{packages: []installedengines.ResolvedInstalledEnginePackage{packageRecord}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := inventory.Components, []string{"engine.lunafox.port_scan.package=" + artifactManifestDigest, "engine.lunafox.port_scan.runtime=" + runtimeDigest}; len(got) != len(want) || got[0].ID+"="+got[0].Digest != want[0] || got[1].ID+"="+got[1].Digest != want[1] {
		t.Fatalf("inventory components = %#v, want %#v", got, want)
	}
}

func TestBuildEngineInventoryFailsClosedForUnavailableOrInvalidRuntimeEvidence(t *testing.T) {
	if _, err := BuildEngineInventory(engineInventoryQueryStub{err: errors.New("cache mismatch")}); err == nil || !strings.Contains(err.Error(), "cache mismatch") {
		t.Fatalf("query failure = %v, want cache mismatch", err)
	}
	invalid := installedengines.ResolvedInstalledEnginePackage{
		Registration: catalogdomain.Engine{
			EngineID:      "engine.lunafox.port_scan",
			ArtifactRef:   "ghcr.io/yyhuni/lunafox-engine-port-scan@sha256:" + strings.Repeat("a", 64),
			PackageDigest: "sha256:" + strings.Repeat("b", 64),
		},
		Layout: enginepackagecatalog.EnginePackageLayout{Definition: enginepackagecatalog.EnginePackageDefinition{
			PackageManifest: packagemanifest.PackageManifest{RuntimeImage: packagemanifest.RuntimeImage{Refs: []string{"ghcr.io/yyhuni/lunafox-engine-runtime-port-scan:latest"}}},
		}},
	}
	if _, err := BuildEngineInventory(engineInventoryQueryStub{packages: []installedengines.ResolvedInstalledEnginePackage{invalid}}); err == nil {
		t.Fatal("expected invalid runtime reference to fail closed")
	}

	empty, err := BuildEngineInventory(engineInventoryQueryStub{})
	if err != nil {
		t.Fatal(err)
	}
	if empty.Components == nil || len(empty.Components) != 0 {
		t.Fatalf("empty inventory = %#v, want an explicit empty component list", empty)
	}
}
