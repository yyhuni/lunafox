package dto

import (
	"encoding/json"
	"strings"
	"testing"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

func TestEngineCatalogDTOSeparatesSummaryAndDetail(t *testing.T) {
	item := catalogdomain.EngineCatalogItem{
		EngineID: "engine.lunafox.website_discovery", ManifestVersion: "engine.v5", Publisher: "lunafox",
		PackageVersion: "1.0.0", ArtifactRef: "docker.io/lunafox/lunafox-engine-website-discovery@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		PackageDigest:  "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		EngineAPIMajor: 2, SupportedTargetTypes: []string{"domain"},
		ConfigSections:  []catalogdomain.EngineConfigSection{{ID: "httpx", RequiredEnabled: true}},
		LocaleResources: map[string]map[string]any{"zh": {"engine": map[string]any{"displayName": "站点发现"}}},
	}
	summary := NewEngineCatalogSummaryOutput(&item)
	if summary.Name != "engines/engine.lunafox.website_discovery" || len(summary.Execution.ConfigSections) != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	detail := NewEngineCatalogDetailOutput(&item)
	if len(detail.Execution.ConfigSections) != 1 {
		t.Fatalf("unexpected detail: %+v", detail)
	}
	if !detail.Execution.ConfigSections[0].RequiredEnabled {
		t.Fatalf("detail lost requiredEnabled projection: %+v", detail.Execution.ConfigSections[0])
	}
	payload, err := json.Marshal(detail)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"install", "signature", "rootPath", "artifactPath", "configSchema", "runtimeRef", "inputProfile"} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("detail leaked %q: %s", forbidden, payload)
		}
	}
	if !strings.Contains(string(payload), `"requiredEnabled":true`) {
		t.Fatalf("detail omitted requiredEnabled: %s", payload)
	}
}

func TestEngineCatalogDTOOmitsExecutionInputMembership(t *testing.T) {
	item := catalogdomain.EngineCatalogItem{
		EngineID: "engine.lunafox.demo", ManifestVersion: "engine.v5", Publisher: "lunafox",
		EngineAPIMajor: 2, SupportedTargetTypes: []string{"domain"},
		ExecutionResources: []string{"subfinderProviderConfig"},
		ConfigSections:     []catalogdomain.EngineConfigSection{{ID: "scan"}},
		LocaleResources:    map[string]map[string]any{},
	}
	payload, err := json.Marshal(NewEngineCatalogDetailOutput(&item))
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	for _, required := range []string{`"engineApiMajor":2`, `"executionResources":["subfinderProviderConfig"]`} {
		if !strings.Contains(text, required) {
			t.Fatalf("engine.v5 detail missing %s: %s", required, payload)
		}
	}
	for _, forbidden := range []string{`"runtimeRef"`, `"inputProfile"`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("engine.v5 detail leaked legacy field %s: %s", forbidden, payload)
		}
	}
}
