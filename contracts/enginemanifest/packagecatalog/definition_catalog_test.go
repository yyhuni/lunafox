package packagecatalog

import (
	"reflect"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/contracts/enginemanifest/repositoryname"
)

func TestDecodeEnginePackageDefinitionPreservesNormalizedDeclarations(t *testing.T) {
	definition, err := DecodeEnginePackageDefinition(
		validPackageManifestForCatalogTest("engine.lunafox.subdomain_discovery"),
		validEngineManifestForCatalogTest("engine.lunafox.subdomain_discovery"),
		"package.json",
		"engine.json",
	)
	if err != nil {
		t.Fatalf("DecodeEnginePackageDefinition failed: %v", err)
	}

	execution := definition.EngineDefinition.Execution
	if execution.EngineAPIMajor != 2 {
		t.Fatalf("engineApiMajor = %d, want 2", execution.EngineAPIMajor)
	}
	if got, want := execution.SupportedTargetTypes, []string{"domain", "ip", "cidr"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("supportedTargetTypes = %#v, want %#v", got, want)
	}
	if got := execution.ExecutionResources; len(got) != 1 || got[0] != "subfinderProviderConfig" {
		t.Fatalf("executionResources = %#v", got)
	}
	resource := execution.ConfigSections[0].Params[0].Resource
	if resource == nil || resource.Kind != "wordlist" {
		t.Fatalf("config resource metadata was lost: %#v", resource)
	}
	if !execution.ConfigSections[0].RequiredEnabled {
		t.Fatalf("requiredEnabled declaration was lost: %#v", execution.ConfigSections[0])
	}
}

func TestDecodeEnginePackageDefinitionRejectsIdentityDrift(t *testing.T) {
	_, err := DecodeEnginePackageDefinition(
		validPackageManifestForCatalogTest("engine.lunafox.subdomain_discovery"),
		validEngineManifestForCatalogTest("engine.lunafox.port_scan"),
		"package.json",
		"engine.json",
	)
	if err == nil || !strings.Contains(err.Error(), "identity mismatch") {
		t.Fatalf("expected identity mismatch, got %v", err)
	}
}

func TestDecodeEnginePackageDefinitionRejectsIndependentRuntimeMetadata(t *testing.T) {
	tests := []struct {
		name   string
		field  string
		value  string
		inRoot bool
	}{
		{name: "runtime ref", field: "runtimeRef", value: `"runtime.subdomain_discovery"`, inRoot: true},
		{name: "runtime source", field: "runtimeSource", value: `"runtime.subdomain_discovery"`, inRoot: true},
		{name: "runtime image sibling", field: "runtimeImages", value: `[]`, inRoot: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			packagePayload := validPackageManifestForCatalogTest("engine.lunafox.subdomain_discovery")
			enginePayload := validEngineManifestForCatalogTest("engine.lunafox.subdomain_discovery")
			if test.inRoot {
				text := strings.TrimSuffix(strings.TrimSpace(string(enginePayload)), "}")
				enginePayload = []byte(text + `,"` + test.field + `":` + test.value + `}`)
			} else {
				text := strings.TrimSuffix(strings.TrimSpace(string(packagePayload)), "}")
				packagePayload = []byte(text + `,"` + test.field + `":` + test.value + `}`)
			}
			_, err := DecodeEnginePackageDefinition(packagePayload, enginePayload, "package.json", "engine.json")
			if err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("DecodeEnginePackageDefinition() error = %v, want strict legacy metadata rejection", err)
			}
		})
	}
}

func TestEngineDefinitionRegistryReturnsDetachedDefinitions(t *testing.T) {
	definition, err := DecodeEnginePackageDefinition(
		validPackageManifestForCatalogTest("engine.lunafox.subdomain_discovery"),
		validEngineManifestForCatalogTest("engine.lunafox.subdomain_discovery"),
		"package.json",
		"engine.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	registry := NewEngineDefinitionRegistry()
	if err := registry.Register(definition); err != nil {
		t.Fatal(err)
	}
	definition.PackageManifest.RuntimeImage.Refs[0] = "mutated"
	definition.EngineDefinition.Execution.ConfigSections[0].Params[0].Resource.Kind = "mutated"

	loaded, found := registry.Get("engine.lunafox.subdomain_discovery")
	if !found {
		t.Fatal("registered definition not found")
	}
	if strings.Contains(loaded.PackageManifest.RuntimeImage.Refs[0], "mutated") {
		t.Fatalf("registry stored aliased definition: %#v", loaded)
	}
	if loaded.EngineDefinition.Execution.ConfigSections[0].Params[0].Resource.Kind != "wordlist" {
		t.Fatalf("registry stored aliased resource metadata: %#v", loaded.EngineDefinition.Execution.ConfigSections)
	}
	loaded.EngineDefinition.Execution.SupportedTargetTypes[0] = "mutated-again"
	reloaded, _ := registry.Get("engine.lunafox.subdomain_discovery")
	if reloaded.EngineDefinition.Execution.SupportedTargetTypes[0] != "domain" {
		t.Fatal("registry Get returned an aliased definition")
	}
}

func TestEngineDefinitionRegistryRejectsHandConstructedInvalidDefinition(t *testing.T) {
	registry := NewEngineDefinitionRegistry()
	definition, err := DecodeEnginePackageDefinition(
		validPackageManifestForCatalogTest("engine.lunafox.subdomain_discovery"),
		validEngineManifestForCatalogTest("engine.lunafox.subdomain_discovery"),
		"package.json",
		"engine.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	definition.EngineDefinition.Execution.EngineAPIMajor = 0
	if err := registry.Register(definition); err == nil {
		t.Fatal("expected registry to reject an unvalidated definition")
	}
}

func TestEngineDefinitionRegistryRejectsHandConstructedInvalidPackageManifest(t *testing.T) {
	registry := NewEngineDefinitionRegistry()
	definition, err := DecodeEnginePackageDefinition(
		validPackageManifestForCatalogTest("engine.lunafox.subdomain_discovery"),
		validEngineManifestForCatalogTest("engine.lunafox.subdomain_discovery"),
		"package.json",
		"engine.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	definition.PackageManifest.RuntimeImage.Refs = []string{"docker.io/lunafox/lunafox-engine-runtime-subdomain-discovery:latest"}
	if err := registry.Register(definition); err == nil || !strings.Contains(err.Error(), "package manifest") {
		t.Fatalf("expected registry to reject invalid package refs, got %v", err)
	}
}

func TestEngineDefinitionRegistryCanonicalizesStructuredDefinitions(t *testing.T) {
	registry := NewEngineDefinitionRegistry()
	definition, err := DecodeEnginePackageDefinition(
		validPackageManifestForCatalogTest("engine.lunafox.subdomain_discovery"),
		validEngineManifestForCatalogTest("engine.lunafox.subdomain_discovery"),
		"package.json",
		"engine.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	definition.EngineDefinition.Execution.SupportedTargetTypes = []string{"cidr", "domain", "ip"}

	if err := registry.Register(definition); err != nil {
		t.Fatal(err)
	}
	registered, found := registry.Get("engine.lunafox.subdomain_discovery")
	if !found {
		t.Fatal("registered definition not found")
	}
	if got, want := registered.EngineDefinition.Execution.SupportedTargetTypes, []string{"domain", "ip", "cidr"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("supported target types = %#v, want %#v", got, want)
	}
}

func TestEngineDefinitionRegistryZeroValueInitializesStorage(t *testing.T) {
	var registry EngineDefinitionRegistry
	definition, err := DecodeEnginePackageDefinition(
		validPackageManifestForCatalogTest("engine.lunafox.subdomain_discovery"),
		validEngineManifestForCatalogTest("engine.lunafox.subdomain_discovery"),
		"package.json",
		"engine.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(definition); err != nil {
		t.Fatalf("zero-value registry should be usable: %v", err)
	}
}

func validPackageManifestForCatalogTest(engineID string) []byte {
	repository, err := repositoryname.FirstPartyRuntimeImageRepositoryName(engineID)
	if err != nil {
		panic(err)
	}
	return []byte(`{
  "packageFormatVersion":"lunafox.engine-package.v2",
  "engineId":"` + engineID + `",
  "engineVersion":"1.2.3",
  "runtimeImage":{"refs":["docker.io/lunafox/` + repository + `@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"]}
}`)
}

func validEngineManifestForCatalogTest(engineID string) []byte {
	return []byte(`{
  "manifestVersion":"engine.v5",
  "engineId":"` + engineID + `",
  "publisher":"lunafox",
  "execution":{
    "engineApiMajor":2,
    "supportedTargetTypes":["cidr","domain","ip"],
    "configSections":[{"id":"scan","defaultEnabled":true,"requiredEnabled":true,"params":[{"key":"wordlist","type":"string","default":"subdomains.txt","resource":{"kind":"wordlist"}}]}],
    "executionResources":["subfinderProviderConfig"]
  }
}`)
}
