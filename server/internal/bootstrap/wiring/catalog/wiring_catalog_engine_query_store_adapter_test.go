package catalogwiring

import (
	"context"
	"errors"
	"testing"

	engineexecution "github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	enginemanifest "github.com/yyhuni/lunafox/contracts/enginemanifest"
	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/server/internal/installedengines"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type engineCatalogPackageReaderStub struct {
	packages []installedengines.ResolvedInstalledEnginePackage
	err      error
}

func (stub engineCatalogPackageReaderStub) ListInstalledEnginePackages() ([]installedengines.ResolvedInstalledEnginePackage, error) {
	return stub.packages, stub.err
}

func (stub engineCatalogPackageReaderStub) GetInstalledEnginePackage(engineID string) (installedengines.ResolvedInstalledEnginePackage, error) {
	if stub.err != nil {
		return installedengines.ResolvedInstalledEnginePackage{}, stub.err
	}
	for _, resolved := range stub.packages {
		if resolved.Registration.EngineID == engineID {
			return resolved, nil
		}
	}
	return installedengines.ResolvedInstalledEnginePackage{}, installedengines.ErrEnginePackageNotFound
}

func TestEngineCatalogAdapterSeparatesSummaryAndDetail(t *testing.T) {
	resolvedPackage := testResolvedEnginePackage(t)
	record := testInstalledEngineRecord()
	adapter := newEngineCatalogQueryStoreAdapter(engineCatalogPackageReaderStub{packages: []installedengines.ResolvedInstalledEnginePackage{resolvedPackage}})
	items, err := adapter.ListEngines(context.Background())
	if err != nil || len(items) != 1 {
		t.Fatalf("ListEngines() = %+v, %v", items, err)
	}
	if len(items[0].ConfigSections) != 0 {
		t.Fatalf("summary leaked detail: %+v", items[0])
	}
	if items[0].EngineAPIMajor != 2 || len(items[0].SupportedTargetTypes) != 1 || items[0].SupportedTargetTypes[0] != engineexecution.TargetTypeDomain {
		t.Fatalf("summary lost v2 execution declarations: %+v", items[0])
	}
	if _, exists := items[0].LocaleResources["zh"]["sections"]; exists {
		t.Fatalf("summary leaked section locale resources: %+v", items[0].LocaleResources)
	}

	item, err := adapter.GetEngineByID(context.Background(), "engine.lunafox.website_discovery")
	if err != nil {
		t.Fatal(err)
	}
	if len(item.ConfigSections) != 1 {
		t.Fatalf("detail missing config projection: %+v", item)
	}
	if !item.ConfigSections[0].RequiredEnabled {
		t.Fatalf("detail lost requiredEnabled projection: %+v", item.ConfigSections[0])
	}
	if _, exists := item.LocaleResources["zh"]["sections"]; !exists {
		t.Fatalf("detail missing locale resources: %+v", item.LocaleResources)
	}
	if item.PackageDigest != record.PackageDigest || item.ArtifactRef != record.ArtifactRef {
		t.Fatalf("catalog item did not project installed package identity: %+v", item)
	}
}

func TestEngineCatalogAdapterMapsUnknownEngineToNotFound(t *testing.T) {
	adapter := newEngineCatalogQueryStoreAdapter(engineCatalogPackageReaderStub{})
	if _, err := adapter.GetEngineByID(context.Background(), "engine.lunafox.unknown"); !errors.Is(err, catalogapp.ErrEngineNotFound) {
		t.Fatalf("GetEngineByID() error = %v", err)
	}
}

func testInstalledEngineRecord() catalogdomain.Engine {
	return catalogdomain.Engine{
		EngineID: "engine.lunafox.website_discovery", Publisher: "lunafox", PackageVersion: "1.0.0",
		ArtifactRef:   "docker.io/lunafox/lunafox-engine-website-discovery@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		PackageDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
}

func testResolvedEnginePackage(t *testing.T) installedengines.ResolvedInstalledEnginePackage {
	t.Helper()
	locale := func(name string) map[string]any {
		return map[string]any{
			"engine":   map[string]any{"displayName": name, "description": "desc"},
			"sections": map[string]any{"httpx": map[string]any{"name": "HTTPX", "description": "desc", "params": map[string]any{"timeout": map[string]any{"description": "desc"}}}},
		}
	}
	minimum := 1
	return installedengines.ResolvedInstalledEnginePackage{
		Registration: testInstalledEngineRecord(),
		Layout: enginepackagecatalog.EnginePackageLayout{
			Definition: enginepackagecatalog.EnginePackageDefinition{
				EngineDefinition: enginemanifest.EngineDefinition{
					ManifestVersion: enginemanifest.SupportedRootManifestVersion,
					EngineID:        "engine.lunafox.website_discovery",
					Publisher:       "lunafox",
					Execution: engineexecution.ExecutionDefinition{
						EngineAPIMajor:       2,
						SupportedTargetTypes: []string{engineexecution.TargetTypeDomain},
						ConfigSections: []engineexecution.ConfigSectionDefinition{{
							ID:              "httpx",
							DefaultEnabled:  true,
							RequiredEnabled: true,
							Params: []engineexecution.ParamDefinition{{
								Key:     "timeout",
								Type:    engineexecution.ParamTypeInteger,
								Default: 30,
								Minimum: &minimum,
							}},
						}},
					},
				},
			},
			LocaleResources: map[string]map[string]any{"zh": locale("站点发现"), "en": locale("Website Discovery")},
		},
	}
}
