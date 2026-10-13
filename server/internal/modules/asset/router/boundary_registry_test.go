package router

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

type assetBoundaryRegistry struct {
	HTTPCanonicalRouteAllowlist []assetCanonicalRouteEntry `json:"httpCanonicalRouteAllowlist"`
}

type assetCanonicalRouteEntry struct {
	ID               string   `json:"id"`
	Class            string   `json:"class"`
	File             string   `json:"file"`
	Method           string   `json:"method"`
	Path             string   `json:"path"`
	Reasons          []string `json:"reasons"`
	VerificationNote string   `json:"verificationNote"`
}

func TestGlobalAssetExportRoutesHaveExactCanonicalBoundaryRegistryEntries(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve boundary registry test path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../../../.."))
	if _, err := os.Stat(filepath.Join(repositoryRoot, "PUBLIC_PROVENANCE.json")); err == nil {
		t.Skip("private OpenSpec boundary registry is outside the public projection")
	}
	payload, err := os.ReadFile(filepath.Join(repositoryRoot, "openspec/specs/boundary-contract-standard-precedence/legacy-exception-registry.json"))
	if err != nil {
		t.Fatalf("read boundary registry: %v", err)
	}
	var registry assetBoundaryRegistry
	if err := json.Unmarshal(payload, &registry); err != nil {
		t.Fatalf("parse boundary registry: %v", err)
	}

	expected := map[string]struct {
		class   string
		method  string
		path    string
		reasons []string
	}{
		"http.canonical.asset.global_export.grant": {
			class:   "http.canonical.custom_method_route",
			method:  "POST",
			path:    "/assets:export",
			reasons: []string{"global_asset_export_grant_custom_method"},
		},
		"http.canonical.asset.global_export.download": {
			class:   "http.canonical.export_route",
			method:  "GET",
			path:    "/assets/exportFiles/current",
			reasons: []string{"semantic_classification_required"},
		},
	}
	found := make(map[string]bool, len(expected))
	for _, entry := range registry.HTTPCanonicalRouteAllowlist {
		contract, ok := expected[entry.ID]
		if !ok {
			continue
		}
		if found[entry.ID] {
			t.Fatalf("duplicate boundary registry entry %q", entry.ID)
		}
		found[entry.ID] = true
		if entry.Class != contract.class || entry.File != "server/internal/modules/asset/router/routes.go" || entry.Method != contract.method || entry.Path != contract.path || !slices.Equal(entry.Reasons, contract.reasons) || entry.VerificationNote == "" {
			t.Fatalf("boundary registry entry %q does not match the global export contract: %#v", entry.ID, entry)
		}
	}
	for id := range expected {
		if !found[id] {
			t.Errorf("missing boundary registry entry %q", id)
		}
	}
}
