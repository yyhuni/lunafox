package enginemanifest

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestDecodeRootManifestProducesCompleteNormalizedExecutionDefinition(t *testing.T) {
	payload := validRootManifestPayload()

	manifest, err := DecodeRootManifest(payload, "engine.json")
	if err != nil {
		t.Fatalf("DecodeRootManifest failed: %v", err)
	}
	if err := ValidateRootManifest(manifest); err != nil {
		t.Fatalf("ValidateRootManifest failed: %v", err)
	}

	definition, err := NormalizeRootManifest(manifest)
	if err != nil {
		t.Fatalf("NormalizeRootManifest failed: %v", err)
	}
	if definition.EngineID != "engine.lunafox.subdomain_discovery" || definition.Execution.EngineAPIMajor != 2 {
		t.Fatalf("identity or Engine API major was lost: %+v", definition)
	}
	if got, want := definition.Execution.SupportedTargetTypes, []string{"domain", "ip", "cidr"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("supported target types = %#v, want %#v", got, want)
	}
	if got, want := definition.Execution.ExecutionResources, []string{"subfinderProviderConfig"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("execution resources = %#v, want %#v", got, want)
	}
	resource := definition.Execution.ConfigSections[0].Params[0].Resource
	if resource == nil || resource.Kind != "wordlist" {
		t.Fatalf("config resource metadata was lost: %#v", resource)
	}
	if !definition.Execution.ConfigSections[0].RequiredEnabled {
		t.Fatalf("requiredEnabled declaration was lost: %#v", definition.Execution.ConfigSections[0])
	}
}

func TestDecodeRootManifestAcceptsAndCanonicalizesFingerprintLibraries(t *testing.T) {
	payload := strings.Replace(
		string(validRootManifestPayload()),
		`"executionResources": ["subfinderProviderConfig"]`,
		`"executionResources": ["fingerprintLibraryFingerPrintHub"]`,
		1,
	)
	definition, err := DecodeEngineDefinition([]byte(payload), "engine.json")
	if err != nil {
		t.Fatalf("DecodeEngineDefinition() error = %v", err)
	}
	want := []string{
		EnginePlatformResourceFingerprintLibraryFingerPrintHub,
	}
	if !reflect.DeepEqual(definition.Execution.ExecutionResources, want) {
		t.Fatalf("fingerprint execution resources = %#v, want %#v", definition.Execution.ExecutionResources, want)
	}
}

func TestDecodeRootManifestRejectsNonCanonicalDeclarationSets(t *testing.T) {
	cases := map[string]string{
		"missing engine api major":       `"engineApiMajor": 0`,
		"retired engine api major 1":     `"engineApiMajor": 1`,
		"unsupported engine api major 3": `"engineApiMajor": 3`,
		"retired inputs field":           `"inputs": ["subdomains"]`,
		"unknown target type":            `"supportedTargetTypes": ["url"]`,
		"target type whitespace":         `"supportedTargetTypes": ["domain "]`,
		"target type case":               `"supportedTargetTypes": ["DOMAIN"]`,
		"duplicate target type":          `"supportedTargetTypes": ["domain", "domain"]`,
		"unknown execution resource":     `"executionResources": ["provider"]`,
		"execution whitespace":           `"executionResources": ["subfinderProviderConfig "]`,
		"duplicate execution resource":   `"executionResources": ["subfinderProviderConfig", "subfinderProviderConfig"]`,
	}
	base := string(validRootManifestPayload())
	for name, replacement := range cases {
		t.Run(name, func(t *testing.T) {
			payload := []byte(strings.Replace(base, `"engineApiMajor": 2`, replacement, 1))
			if name == "missing engine api major" {
				payload = []byte(strings.Replace(base, `"engineApiMajor": 2,`, "", 1))
			}
			if name != "missing engine api major" {
				if strings.Contains(name, "retired inputs") {
					payload = []byte(strings.Replace(base, `"engineApiMajor": 2,`, `"inputs": ["subdomains"], "engineApiMajor": 2,`, 1))
				} else {
					payload = []byte(strings.Replace(base, `"engineApiMajor": 2`, replacement, 1))
				}
				if strings.Contains(name, "target type") {
					payload = []byte(strings.Replace(base, `"supportedTargetTypes": ["domain", "ip", "cidr"]`, replacement, 1))
				}
				if strings.Contains(name, "execution") {
					payload = []byte(strings.Replace(base, `"executionResources": ["subfinderProviderConfig"]`, replacement, 1))
				}
			}
			if _, err := DecodeRootManifest(payload, "engine.json"); err == nil {
				t.Fatalf("expected strict declaration rejection for %s", name)
			}
		})
	}
}

func TestDecodeRootManifestRejectsEveryNonCurrentEngineAPIMajor(t *testing.T) {
	base := string(validRootManifestPayload())
	for _, major := range []string{"1", "3", "99"} {
		t.Run(major, func(t *testing.T) {
			payload := strings.Replace(base, `"engineApiMajor": 2`, `"engineApiMajor": `+major, 1)
			if _, err := DecodeEngineDefinition([]byte(payload), "stale-engine.json"); err == nil || !strings.Contains(err.Error(), "unsupported Engine API major") {
				t.Fatalf("DecodeEngineDefinition() error = %v, want stale major rejection", err)
			}
		})
	}
}

func TestDecodeRootManifestRejectsRetiredInputsField(t *testing.T) {
	payload := strings.Replace(string(validRootManifestPayload()), `"engineApiMajor": 2,`, `"inputs": [], "engineApiMajor": 2,`, 1)
	if _, err := DecodeRootManifest([]byte(payload), "engine.json"); err == nil || !strings.Contains(err.Error(), "inputs") {
		t.Fatalf("retired execution.inputs field was accepted: %v", err)
	}
}

func TestDecodeRootManifestRejectsRetiredExecutionResourceFieldsAndID(t *testing.T) {
	base := string(validRootManifestPayload())
	for _, field := range []string{"platformResources", "runtimeArtifacts"} {
		t.Run(field, func(t *testing.T) {
			payload := strings.Replace(base, `"executionResources": ["subfinderProviderConfig"]`, `"`+field+`": ["subfinderProviderConfig"]`, 1)
			if _, err := DecodeRootManifest([]byte(payload), "engine.json"); err == nil || !strings.Contains(err.Error(), field) {
				t.Fatalf("retired field %q was accepted: %v", field, err)
			}
		})
	}
	payload := strings.Replace(base, "subfinderProviderConfig", "runtimeTemplateBundle", 1)
	if _, err := DecodeRootManifest([]byte(payload), "engine.json"); err == nil || !strings.Contains(err.Error(), "runtimeTemplateBundle") {
		t.Fatalf("retired runtimeTemplateBundle ID was accepted: %v", err)
	}
}

func TestDecodeRootManifestRejectsInputMembershipAliases(t *testing.T) {
	for _, field := range []string{"executionInputs", "inputRoles", "inputAllowSet", "inputAllowlist", "inputProfile"} {
		t.Run(field, func(t *testing.T) {
			payload := strings.Replace(
				string(validRootManifestPayload()),
				`"engineApiMajor": 2,`,
				`"`+field+`": [], "engineApiMajor": 2,`,
				1,
			)
			if _, err := DecodeRootManifest([]byte(payload), "engine.json"); err == nil || !strings.Contains(err.Error(), field) {
				t.Fatalf("retired input membership alias %q was accepted: %v", field, err)
			}
		})
	}
}

func TestDecodeRootManifestRejectsLegacyUnknownAndTrailingFields(t *testing.T) {
	base := string(validRootManifestPayload())
	cases := []struct {
		name     string
		payload  string
		contains string
	}{
		{name: "v4 version", payload: strings.Replace(base, `"engine.v5"`, `"engine.v4"`, 1), contains: "manifestVersion"},
		{name: "runtime ref", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"runtimeRef":"runtime.demo","engineApiMajor": 2,`, 1), contains: "runtimeRef"},
		{name: "runtime source", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"runtimeSource":"runtime.demo","engineApiMajor": 2,`, 1), contains: "runtimeSource"},
		{name: "input profile", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"inputProfile":"target","engineApiMajor": 2,`, 1), contains: "inputProfile"},
		{name: "target declaration", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"target":{},"engineApiMajor": 2,`, 1), contains: "target"},
		{name: "resource requirements", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"resourceRequirements":[],"engineApiMajor": 2,`, 1), contains: "resourceRequirements"},
		{name: "result types", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"resultTypes":[],"engineApiMajor": 2,`, 1), contains: "resultTypes"},
		{name: "output allow set", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"outputs":[],"engineApiMajor": 2,`, 1), contains: "outputs"},
		{name: "result schema reference", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"resultSchemaRef":"schema.result.demo.v1","engineApiMajor": 2,`, 1), contains: "resultSchemaRef"},
		{name: "result schema list", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"resultSchemas":[],"engineApiMajor": 2,`, 1), contains: "resultSchemas"},
		{name: "result schema", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"resultSchema":{"ref":"schema.result.demo.v1"},"engineApiMajor": 2,`, 1), contains: "resultSchema"},
		{name: "allowed result types", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"allowedResultTypes":[],"engineApiMajor": 2,`, 1), contains: "allowedResultTypes"},
		{name: "output authorization", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"outputAuthorization":{},"engineApiMajor": 2,`, 1), contains: "outputAuthorization"},
		{name: "output allowlist", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"outputAllowlist":[],"engineApiMajor": 2,`, 1), contains: "outputAllowlist"},
		{name: "entrypoint override", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"entrypoint":["/engine"],"engineApiMajor": 2,`, 1), contains: "entrypoint"},
		{name: "command override", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"command":["scan"],"engineApiMajor": 2,`, 1), contains: "command"},
		{name: "cmd override", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"cmd":["scan"],"engineApiMajor": 2,`, 1), contains: "cmd"},
		{name: "args override", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"args":["--target","example.com"],"engineApiMajor": 2,`, 1), contains: "args"},
		{name: "image identity", payload: strings.Replace(base, `"publisher": "lunafox",`, `"publisher":"lunafox","runtimeImage":{"refs":["registry/demo@sha256:abc"]},`, 1), contains: "runtimeImage"},
		{name: "resource source", payload: strings.Replace(base, `"resource": {"kind": "wordlist"}`, `"resource":{"kind":"wordlist","source":"catalog"}`, 1), contains: "source"},
		{name: "resource path", payload: strings.Replace(base, `"resource": {"kind": "wordlist"}`, `"resource":{"kind":"wordlist","path":"/tmp/list"}`, 1), contains: "path"},
		{name: "resource condition", payload: strings.Replace(base, `"resource": {"kind": "wordlist"}`, `"resource":{"kind":"wordlist","when":"enabled"}`, 1), contains: "when"},
		{name: "resource required", payload: strings.Replace(base, `"resource": {"kind": "wordlist"}`, `"resource":{"kind":"wordlist","required":true}`, 1), contains: "required"},
		{name: "null execution resources", payload: strings.Replace(base, `"executionResources": ["subfinderProviderConfig"]`, `"executionResources":null`, 1), contains: "executionResources"},
		{name: "duplicate execution field", payload: strings.Replace(base, `"engineApiMajor": 2,`, `"engineApiMajor":2,"engineApiMajor":2,`, 1), contains: "duplicate JSON field"},
		{name: "duplicate resource field", payload: strings.Replace(base, `"resource": {"kind": "wordlist"}`, `"resource":{"kind":"wordlist","kind":"wordlist"}`, 1), contains: "duplicate JSON field"},
		{name: "trailing json", payload: base + `{}`, contains: "trailing"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeRootManifest([]byte(tc.payload), "engine.json")
			if err == nil || !strings.Contains(err.Error(), tc.contains) {
				t.Fatalf("expected error containing %q, got %v", tc.contains, err)
			}
		})
	}
}

func TestDecodeRootManifestAcceptsDeclaredNonFirstPartyPublisher(t *testing.T) {
	payload := strings.Replace(string(validRootManifestPayload()), `"publisher": "lunafox"`, `"publisher": "example"`, 1)
	payload = strings.Replace(payload, "engine.lunafox.subdomain_discovery", "engine.example.subdomain_discovery", 1)
	manifest, err := DecodeRootManifest([]byte(payload), "engine.json")
	if err != nil {
		t.Fatalf("DecodeRootManifest() error = %v", err)
	}
	if manifest.EngineID != "engine.example.subdomain_discovery" || manifest.Publisher != "example" {
		t.Fatalf("unexpected generic identity: %#v", manifest)
	}
}

func TestDecodeRootManifestAcceptsDynamicallyNamedFirstPartyBuiltinIdentity(t *testing.T) {
	payload := strings.Replace(
		string(validRootManifestPayload()),
		"engine.lunafox.subdomain_discovery",
		"engine.lunafox.http2_probe",
		1,
	)
	definition, err := DecodeRootManifest([]byte(payload), "engine.json")
	if err != nil {
		t.Fatalf("DecodeRootManifest() error = %v", err)
	}
	if definition.EngineID != "engine.lunafox.http2_probe" {
		t.Fatalf("DecodeRootManifest() engineId = %q", definition.EngineID)
	}
}

func TestDecodeRootManifestRejectsReleaseScopedBuiltinIdentity(t *testing.T) {
	base := string(validRootManifestPayload())
	for _, engineID := range []string{
		"engine.lunafox.subdomain_discovery_v2",
		"engine.lunafox.subdomain_discovery_sha256_" + strings.Repeat("a", 64),
		"engine.lunafox.subdomain_discovery_deadbee",
		"engine.lunafox.subdomain_discovery_release_01j2x3k4m5n6p7q8r9s0t1v2w3",
		"engine.lunafox.subdomain_discovery_42",
	} {
		t.Run(engineID, func(t *testing.T) {
			payload := strings.Replace(base, "engine.lunafox.subdomain_discovery", engineID, 1)
			_, err := DecodeRootManifest([]byte(payload), "engine.json")
			if err == nil || !strings.Contains(err.Error(), "release-scoped engineId") {
				t.Fatalf("DecodeRootManifest() error = %v, want stable identity rejection", err)
			}
		})
	}
}

func TestDecodeRootManifestAcceptsGenericIdentityAndRejectsMalformedIdentity(t *testing.T) {
	base := string(validRootManifestPayload())
	for _, tc := range []struct {
		engineID string
		wantErr  bool
	}{
		{engineID: "engine.example.subdomain_discovery"},
		{engineID: "engine.lunafox.subdomain-discovery"},
	} {
		t.Run(tc.engineID, func(t *testing.T) {
			payload := strings.Replace(base, "engine.lunafox.subdomain_discovery", tc.engineID, 1)
			_, err := DecodeRootManifest([]byte(payload), "engine.json")
			if tc.wantErr && err == nil {
				t.Fatalf("DecodeRootManifest() error = nil, want invalid identity rejection")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("DecodeRootManifest() error = %v, want generic identity acceptance", err)
			}
		})
	}
}

func TestDecodeRootManifestRejectsLegacyTrustField(t *testing.T) {
	payload := strings.Replace(string(validRootManifestPayload()), `"publisher": "lunafox",`, `"publisher": "lunafox", "trust": {"class":"builtin"},`, 1)
	if _, err := DecodeRootManifest([]byte(payload), "engine.json"); err == nil || !strings.Contains(err.Error(), "trust") {
		t.Fatalf("DecodeRootManifest() error = %v, want legacy trust field rejection", err)
	}
}

func TestEngineDefinitionCloneAndRoundTripPreserveAllDeclarations(t *testing.T) {
	definition, err := DecodeEngineDefinition(validRootManifestPayload(), "engine.json")
	if err != nil {
		t.Fatalf("DecodeEngineDefinition failed: %v", err)
	}

	cloned := CloneEngineDefinition(definition)
	cloned.Execution.SupportedTargetTypes[0] = "cidr"
	cloned.Execution.ConfigSections[0].ID = "changed"
	cloned.Execution.ConfigSections[0].Params[0].Default = "other"
	cloned.Execution.ConfigSections[0].Params[0].Resource.Kind = "other"
	cloned.Execution.ExecutionResources[0] = "other"

	if definition.Execution.SupportedTargetTypes[0] != "domain" {
		t.Fatalf("clone aliased declaration slices: %+v", definition.Execution)
	}
	param := definition.Execution.ConfigSections[0].Params[0]
	if definition.Execution.ConfigSections[0].ID != "scan" || param.Default != "subdomains" || param.Resource.Kind != "wordlist" {
		t.Fatalf("clone aliased config definition: %+v", definition.Execution.ConfigSections)
	}
	if definition.Execution.ExecutionResources[0] != "subfinderProviderConfig" {
		t.Fatalf("clone aliased execution resources: %#v", definition.Execution.ExecutionResources)
	}

	payload, err := json.Marshal(definition)
	if err != nil {
		t.Fatalf("marshal normalized definition: %v", err)
	}
	roundTripped, err := DecodeEngineDefinition(payload, "normalized-engine.json")
	if err != nil {
		t.Fatalf("decode normalized definition: %v", err)
	}
	if !reflect.DeepEqual(roundTripped, definition) {
		t.Fatalf("normalized roundtrip lost declarations:\n got: %#v\nwant: %#v", roundTripped, definition)
	}
}

func validRootManifestPayload() []byte {
	return []byte(`{
  "manifestVersion": "engine.v5",
  "engineId": "engine.lunafox.subdomain_discovery",
  "publisher": "lunafox",
  "execution": {
    "engineApiMajor": 2,
    "supportedTargetTypes": ["domain", "ip", "cidr"],
    "configSections": [{
      "id": "scan",
      "defaultEnabled": true,
      "requiredEnabled": true,
      "params": [{
        "key": "wordlist",
        "type": "string",
        "default": "subdomains",
        "minLength": 1,
        "resource": {"kind": "wordlist"}
      }]
    }],
    "executionResources": ["subfinderProviderConfig"]
  }
}`)
}
