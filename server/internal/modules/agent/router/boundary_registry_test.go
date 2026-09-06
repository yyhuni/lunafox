package router

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

type agentBoundaryRegistry struct {
	HTTPCanonicalRouteAllowlist []agentCanonicalRouteEntry `json:"httpCanonicalRouteAllowlist"`
}

type agentCanonicalRouteEntry struct {
	ID      string   `json:"id"`
	Class   string   `json:"class"`
	File    string   `json:"file"`
	Method  string   `json:"method"`
	Path    string   `json:"path"`
	Reasons []string `json:"reasons"`
}

func TestAgentFixedViewsHaveExactCanonicalBoundaryRegistryEntries(t *testing.T) {
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
	var registry agentBoundaryRegistry
	if err := json.Unmarshal(payload, &registry); err != nil {
		t.Fatalf("parse boundary registry: %v", err)
	}

	expected := map[string]string{
		"http.canonical.agent.cluster_summaries.current": "/admin/agentClusterSummaries/current",
		"http.canonical.agent.location_maps.current":     "/admin/agentLocationMaps/current",
	}
	found := make(map[string]bool, len(expected))
	for _, entry := range registry.HTTPCanonicalRouteAllowlist {
		path, belongsToAgentFixedViews := expected[entry.ID]
		if !belongsToAgentFixedViews {
			continue
		}
		if found[entry.ID] {
			t.Fatalf("duplicate boundary registry entry %q", entry.ID)
		}
		found[entry.ID] = true
		if entry.Class != "http.canonical.report_view" || entry.File != "server/internal/modules/agent/router/routes.go" || entry.Method != "GET" || entry.Path != path || !slices.Equal(entry.Reasons, []string{"semantic_classification_required"}) {
			t.Fatalf("boundary registry entry %q does not match the fixed-view contract: %#v", entry.ID, entry)
		}
	}
	for id := range expected {
		if !found[id] {
			t.Errorf("missing boundary registry entry %q", id)
		}
	}
}
