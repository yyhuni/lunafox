package engineinstall

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
)

const (
	inventoryDigestA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	inventoryDigestB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func TestLoadInventoryReturnsTypedCanonicalCandidatesWithoutPackageIdentity(t *testing.T) {
	ref := "localhost:5000/development/engine-package@" + inventoryDigestA
	path := writeInventory(t, `enginePackages:
  - refs:
      - `+ref+`
`)

	inventory, err := LoadInventory(path, InventoryModeDevelopment)
	if err != nil {
		t.Fatalf("LoadInventory() error = %v", err)
	}
	if len(inventory.EnginePackages) != 1 {
		t.Fatalf("engine package count = %d, want 1", len(inventory.EnginePackages))
	}
	item := inventory.EnginePackages[0]
	if item.Candidates.ArtifactManifestDigest != ociartifact.ArtifactManifestDigest(inventoryDigestA) {
		t.Fatalf("artifactManifestDigest = %q, want %q", item.Candidates.ArtifactManifestDigest, inventoryDigestA)
	}
	if len(item.Candidates.References) != 1 || item.Candidates.References[0].String() != ref {
		t.Fatalf("typed candidates = %#v, want %q", item.Candidates.References, ref)
	}
}

func TestLoadInventoryProductionPreservesDeclaredCandidateOrder(t *testing.T) {
	refs := []string{
		"docker.io/yyhuni/lunafox-engine-website-discovery@" + inventoryDigestA,
		"ghcr.io/yyhuni/lunafox-engine-website-discovery@" + inventoryDigestA,
	}
	path := writeInventory(t, `enginePackages:
  - refs:
      - `+refs[0]+`
      - `+refs[1]+`
`)

	inventory, err := LoadInventory(path, InventoryModeProduction)
	if err != nil {
		t.Fatalf("LoadInventory() error = %v", err)
	}
	got := make([]string, 0, len(inventory.EnginePackages[0].Candidates.References))
	for _, reference := range inventory.EnginePackages[0].Candidates.References {
		got = append(got, reference.String())
	}
	if !reflect.DeepEqual(got, refs) {
		t.Fatalf("candidate order = %#v, want %#v", got, refs)
	}
}

func TestLoadSelectedRegistryInventoryAcceptsOnlyTheExplicitCanonicalRegistry(t *testing.T) {
	for _, test := range []struct {
		name     string
		registry string
		ref      string
		want     string
	}{
		{
			name:     "Docker Hub",
			registry: "docker.io",
			ref:      "docker.io/yyhuni/lunafox-engine-runtime-port-scan@" + inventoryDigestA,
		},
		{
			name:     "GHCR",
			registry: "ghcr.io",
			ref:      "ghcr.io/yyhuni/lunafox-engine-runtime-port-scan@" + inventoryDigestA,
		},
		{
			name:     "mismatched selected registry",
			registry: "docker.io",
			ref:      "ghcr.io/yyhuni/lunafox-engine-runtime-port-scan@" + inventoryDigestA,
			want:     "want \"docker.io\"",
		},
		{
			name:     "third-party registry",
			registry: "registry.example",
			ref:      "registry.example/yyhuni/lunafox-engine-runtime-port-scan@" + inventoryDigestA,
			want:     "must be docker.io or ghcr.io",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := writeInventory(t, "enginePackages:\n  - refs:\n      - "+test.ref+"\n")
			inventory, err := LoadSelectedRegistryInventory(path, test.registry)
			if test.want != "" {
				if err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("LoadSelectedRegistryInventory() error = %v, want %q", err, test.want)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadSelectedRegistryInventory() error = %v", err)
			}
			if got := inventory.EnginePackages[0].Candidates.References[0].Registry; got != test.registry {
				t.Fatalf("selected Registry = %q, want %q", got, test.registry)
			}
		})
	}
}

func TestLoadInventoryCloudflareAccelerationRequiresCFDockerHubAndGHCRSequence(t *testing.T) {
	refs := []string{
		"docker.lunafox.cc.cd/yyhuni/lunafox-engine-website-discovery@" + inventoryDigestA,
		"docker.io/yyhuni/lunafox-engine-website-discovery@" + inventoryDigestA,
		"ghcr.io/yyhuni/lunafox-engine-website-discovery@" + inventoryDigestA,
	}
	path := writeInventory(t, `enginePackages:
  - refs:
      - `+refs[0]+`
      - `+refs[1]+`
      - `+refs[2]+`
`)

	inventory, err := LoadInventory(path, InventoryModeCloudflareAccelerated)
	if err != nil {
		t.Fatalf("LoadInventory() error = %v", err)
	}
	got := make([]string, 0, len(inventory.EnginePackages[0].Candidates.References))
	for _, reference := range inventory.EnginePackages[0].Candidates.References {
		got = append(got, reference.String())
	}
	if !reflect.DeepEqual(got, refs) {
		t.Fatalf("accelerated candidate order = %#v, want %#v", got, refs)
	}

	invalidPath := writeInventory(t, `enginePackages:
  - refs:
      - `+refs[1]+`
      - `+refs[0]+`
      - `+refs[2]+`
`)
	if _, err := LoadInventory(invalidPath, InventoryModeCloudflareAccelerated); err == nil || !strings.Contains(err.Error(), "Docker Hub followed by GHCR") {
		t.Fatalf("invalid accelerated inventory error = %v", err)
	}
}

func TestLoadInventoryRejectsInvalidDocumentOrCandidates(t *testing.T) {
	validRef := "docker.io/yyhuni/lunafox-engine-subdomain-discovery@" + inventoryDigestA
	tests := []struct {
		name, payload, want string
		mode                InventoryMode
	}{
		{name: "unknown mode", mode: InventoryMode("staging"), payload: "enginePackages: []\n", want: "unsupported Engine Package v2 inventory mode"},
		{name: "empty inventory", mode: InventoryModeDevelopment, payload: "enginePackages: []\n", want: "cannot be empty"},
		{name: "package identity is rejected", mode: InventoryModeDevelopment, payload: `enginePackages:
  - engineId: engine.example.scanner
    refs: [` + validRef + `]
`, want: "field engineId not found"},
		{name: "unknown document field", mode: InventoryModeDevelopment, payload: `enginePackages:
  - refs: [` + validRef + `]
unexpected: true
`, want: "field unexpected not found"},
		{name: "unknown package field", mode: InventoryModeDevelopment, payload: `enginePackages:
  - refs: [` + validRef + `]
    digest: ` + inventoryDigestA + `
`, want: "field digest not found"},
		{name: "trailing yaml document", mode: InventoryModeDevelopment, payload: `enginePackages:
  - refs: [` + validRef + `]
---
enginePackages: []
`, want: "exactly one YAML document"},
		{name: "empty candidates", mode: InventoryModeDevelopment, payload: "enginePackages:\n  - refs: []\n", want: "artifact candidates are required"},
		{name: "tag only", mode: InventoryModeDevelopment, payload: "enginePackages:\n  - refs: [docker.io/yyhuni/lunafox-engine-subdomain-discovery:latest]\n", want: "digest separator"},
		{name: "noncanonical candidate whitespace", mode: InventoryModeDevelopment, payload: "enginePackages:\n  - refs: [\" " + validRef + "\"]\n", want: "canonical OCI digest reference"},
		{name: "duplicate candidate location", mode: InventoryModeProduction, payload: "enginePackages:\n  - refs:\n      - " + validRef + "\n      - " + validRef + "\n", want: "duplicate Engine Package v2 artifact candidate location"},
		{name: "candidate digest drift", mode: InventoryModeProduction, payload: "enginePackages:\n  - refs:\n      - " + validRef + "\n      - ghcr.io/yyhuni/lunafox-engine-subdomain-discovery@" + inventoryDigestB + "\n", want: "same OCI artifact manifest digest"},
		{name: "development candidate count", mode: InventoryModeDevelopment, payload: "enginePackages:\n  - refs:\n      - " + validRef + "\n      - ghcr.io/yyhuni/lunafox-engine-subdomain-discovery@" + inventoryDigestA + "\n", want: "must contain exactly 1 candidate"},
		{name: "production candidate count", mode: InventoryModeProduction, payload: "enginePackages:\n  - refs: [" + validRef + "]\n", want: "must contain exactly 2 candidates"},
		{name: "selected Registry candidate count", mode: InventoryModeSelectedRegistry, payload: "enginePackages:\n  - refs:\n      - " + validRef + "\n      - ghcr.io/yyhuni/lunafox-engine-subdomain-discovery@" + inventoryDigestA + "\n", want: "must contain exactly 1 candidate"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadInventory(writeInventory(t, test.payload), test.mode)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("LoadInventory() error = %v, want error containing %q", err, test.want)
			}
		})
	}
}

func writeInventory(t *testing.T, payload string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "engine-inventory-v2.yaml")
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("write inventory fixture: %v", err)
	}
	return path
}
