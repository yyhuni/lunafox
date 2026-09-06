package containercontract_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	enginecontract "github.com/yyhuni/lunafox/contracts/enginemanifest"
)

func TestBuiltinEngineDefinitionsUseIndependentApplicabilityAndInputs(t *testing.T) {
	cases := []struct {
		name           string
		targetTypes    []string
		platforms      []string
		resourceParams map[string]string
	}{
		{
			name:        "subdomain_discovery",
			targetTypes: []string{"domain"},
			platforms:   []string{"subfinderProviderConfig"},
			resourceParams: map[string]string{
				"bruteforce.wordlist":  "subdomains-top1million-110000.txt",
				"bruteforce.resolvers": "resolvers.txt",
				"resolve.resolvers":    "resolvers.txt",
			},
		},
		{
			name:        "port_scan",
			targetTypes: []string{"domain", "ip", "cidr"},
		},
		{
			name:        "website_discovery",
			targetTypes: []string{"domain", "ip", "cidr"},
		},
		{
			name:        "url_collection",
			targetTypes: []string{"domain", "ip", "cidr"},
		},
		{
			name:        "directory_scan",
			targetTypes: []string{"domain", "ip", "cidr"},
			resourceParams: map[string]string{
				"ffuf.wordlist": "dir_default.txt",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join("..", tc.name, "engine.json")
			payload, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			manifest, err := enginecontract.DecodeRootManifest(payload, path)
			if err != nil {
				t.Fatalf("DecodeRootManifest() error = %v", err)
			}
			if manifest.Execution.EngineAPIMajor != 2 {
				t.Fatalf("engineApiMajor = %d, want 2", manifest.Execution.EngineAPIMajor)
			}
			if got, want := manifest.Execution.SupportedTargetTypes, tc.targetTypes; !equalStrings(got, want) {
				t.Fatalf("supportedTargetTypes = %#v, want %#v", got, want)
			}
			if got, want := manifest.Execution.ExecutionResources, tc.platforms; !equalStrings(got, want) {
				t.Fatalf("executionResources = %#v, want %#v", got, want)
			}
			for key, wantDefault := range tc.resourceParams {
				sectionID, paramKey := splitResourceKey(t, key)
				found := false
				for _, section := range manifest.Execution.ConfigSections {
					if section.ID != sectionID {
						continue
					}
					for _, param := range section.Params {
						if param.Key == paramKey {
							found = true
							if param.Resource == nil || param.Resource.Kind != "wordlist" {
								t.Fatalf("%s resource declaration = %#v", key, param.Resource)
							}
							if got, ok := param.Default.(string); !ok || got != wantDefault {
								t.Fatalf("%s default = %#v, want %q", key, param.Default, wantDefault)
							}
						}
					}
				}
				if !found {
					t.Fatalf("missing resource param %s", key)
				}
			}
		})
	}
}

func TestDirectoryScanManifestDefinesExactClosedFFUFSurface(t *testing.T) {
	path := filepath.Join("..", "directory_scan", "engine.json")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	definition, err := enginecontract.DecodeEngineDefinition(payload, path)
	if err != nil {
		t.Fatalf("DecodeEngineDefinition() error = %v", err)
	}
	if definition.EngineID != "engine.lunafox.directory_scan" || definition.Execution.EngineAPIMajor != 2 {
		t.Fatalf("Directory identity = %#v", definition)
	}
	if got, want := definition.Execution.SupportedTargetTypes, []string{"domain", "ip", "cidr"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("supportedTargetTypes = %#v, want %#v", got, want)
	}

	type expectedParam struct {
		key          string
		kind         string
		defaultValue any
		minimum      *int
		maximum      *int
		enum         []string
		resourceKind string
	}
	want := []expectedParam{
		{key: "wordlist", kind: "string", defaultValue: "dir_default.txt", resourceKind: "wordlist"},
		{key: "recursion", kind: "boolean", defaultValue: false},
		{key: "recursion-depth", kind: "integer", defaultValue: 1, minimum: intPointer(1), maximum: intPointer(5)},
		{key: "recursion-strategy", kind: "string", defaultValue: "default", enum: []string{"default", "greedy"}},
		{key: "auto-calibration", kind: "boolean", defaultValue: true},
		{key: "auto-calibration-mode", kind: "string", defaultValue: "ac", enum: []string{"ac", "ach"}},
		{key: "match-codes", kind: "string", defaultValue: "200-299,301,302,307,401,403,405,500"},
		{key: "concurrency", kind: "integer", defaultValue: 5, minimum: intPointer(1), maximum: intPointer(20)},
		{key: "threads", kind: "integer", defaultValue: 10, minimum: intPointer(1), maximum: intPointer(100)},
		{key: "rate", kind: "integer", defaultValue: 0, minimum: intPointer(0), maximum: intPointer(1000)},
		{key: "delay", kind: "string", defaultValue: "0.1-2.0"},
		{key: "request-timeout", kind: "integer", defaultValue: 10, minimum: intPointer(1), maximum: intPointer(120)},
		{key: "timeout", kind: "integer", defaultValue: 86400, minimum: intPointer(60), maximum: intPointer(604800)},
		{key: "follow-redirects", kind: "boolean", defaultValue: false},
		{key: "http2", kind: "boolean", defaultValue: false},
	}
	sections := definition.Execution.ConfigSections
	if len(sections) != 1 || sections[0].ID != "ffuf" || !sections[0].DefaultEnabled || len(sections[0].Params) != len(want) {
		t.Fatalf("Directory config sections = %#v", sections)
	}
	for index, expected := range want {
		param := sections[0].Params[index]
		resourceKind := ""
		if param.Resource != nil {
			resourceKind = param.Resource.Kind
		}
		if param.Key != expected.key || param.Type != expected.kind || !reflect.DeepEqual(param.Default, expected.defaultValue) ||
			!sameIntPointer(param.Minimum, expected.minimum) || !sameIntPointer(param.Maximum, expected.maximum) ||
			!reflect.DeepEqual(param.Enum, expected.enum) || resourceKind != expected.resourceKind {
			t.Fatalf("ffuf param[%d] = %#v, want %#v", index, param, expected)
		}
		if param.MinLength != nil || param.MaxLength != nil || param.Pattern != "" {
			t.Fatalf("ffuf.%s added an undeclared string constraint: %#v", param.Key, param)
		}
	}

	normalized, err := engineexecution.NormalizeAndValidateConfig(map[string]any{
		"ffuf": map[string]any{},
	}, definition.Execution)
	if err != nil {
		t.Fatalf("NormalizeAndValidateConfig() error = %v", err)
	}
	ffuf := normalized["ffuf"].(map[string]any)
	if ffuf["enabled"] != true || ffuf["wordlist"] != "dir_default.txt" ||
		ffuf["match-codes"] != "200-299,301,302,307,401,403,405,500" || ffuf["delay"] != "0.1-2.0" {
		t.Fatalf("materialized Directory defaults = %#v", ffuf)
	}

	for _, control := range []string{"headers", "authentication", "arbitrary-flags", "ignore-body"} {
		t.Run("reject_"+control, func(t *testing.T) {
			_, err := engineexecution.NormalizeAndValidateConfig(map[string]any{
				"ffuf": map[string]any{"enabled": true, control: "forbidden"},
			}, definition.Execution)
			if err == nil || !strings.Contains(err.Error(), "is not allowed") {
				t.Fatalf("undeclared control %q error = %v", control, err)
			}
		})
	}
}

func TestURLCollectionManifestExposesOnlyConfirmedConfiguration(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "url_collection", "engine.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := enginecontract.DecodeRootManifest(payload, "url_collection/engine.json")
	if err != nil {
		t.Fatal(err)
	}
	type expectedParam struct {
		key          string
		kind         string
		defaultValue any
		minimum      *int
		maximum      *int
		enum         []string
	}
	expected := []struct {
		id      string
		enabled bool
		params  []expectedParam
	}{
		{"waymore", true, []expectedParam{{"timeout", "integer", "3600", intPointer(60), intPointer(604800), nil}}},
		{"katana", true, []expectedParam{
			{"timeout", "integer", "3600", intPointer(60), intPointer(604800), nil},
			{"depth", "integer", "3", intPointer(1), intPointer(10), nil},
			{"concurrency", "integer", "10", intPointer(1), intPointer(100), nil},
			{"rate-limit", "integer", "30", intPointer(1), intPointer(500), nil},
			{"request-timeout", "integer", "10", intPointer(1), intPointer(120), nil},
			{"retries", "integer", "1", intPointer(0), intPointer(5), nil},
			{"delay", "integer", "0", intPointer(0), intPointer(30), nil},
		}},
		{"uro", true, []expectedParam{
			{"timeout", "integer", "600", intPointer(60), intPointer(21600), nil},
			{"whitelist", "stringArray", []any{}, nil, nil, nil},
			{"blacklist", "stringArray", []any{}, nil, nil, nil},
			{"filters", "stringArray", []any{}, nil, nil, []string{"hasparams", "noparams", "hasext", "noext", "allexts", "keepcontent", "keepslash", "vuln"}},
		}},
		{"httpx", true, []expectedParam{
			{"timeout", "integer", "3600", intPointer(60), intPointer(604800), nil},
			{"threads", "integer", "25", intPointer(1), intPointer(200), nil},
			{"rate-limit", "integer", "150", intPointer(1), intPointer(1000), nil},
			{"request-timeout", "integer", "10", intPointer(1), intPointer(120), nil},
			{"retries", "integer", "1", intPointer(0), intPointer(5), nil},
		}},
	}
	if len(manifest.Execution.ConfigSections) != len(expected) {
		t.Fatalf("config section count = %d, want %d", len(manifest.Execution.ConfigSections), len(expected))
	}
	for index, wantSection := range expected {
		gotSection := manifest.Execution.ConfigSections[index]
		if gotSection.ID != wantSection.id || gotSection.DefaultEnabled != wantSection.enabled || len(gotSection.Params) != len(wantSection.params) {
			t.Fatalf("section[%d] = %#v, want %s with %d params", index, gotSection, wantSection.id, len(wantSection.params))
		}
		for paramIndex, wantParam := range wantSection.params {
			gotParam := gotSection.Params[paramIndex]
			if gotParam.Key != wantParam.key || gotParam.Type != wantParam.kind || fmt.Sprint(gotParam.Default) != fmt.Sprint(wantParam.defaultValue) || !sameIntPointer(gotParam.Minimum, wantParam.minimum) || !sameIntPointer(gotParam.Maximum, wantParam.maximum) || !reflect.DeepEqual(gotParam.Enum, wantParam.enum) {
				t.Fatalf("%s.%s = %#v, want %#v", wantSection.id, wantParam.key, gotParam, wantParam)
			}
			if gotParam.MinLength != nil || gotParam.MaxLength != nil || gotParam.Pattern != "" || gotParam.Resource != nil {
				t.Fatalf("%s.%s added project-specific restrictions: %#v", wantSection.id, wantParam.key, gotParam)
			}
		}
	}
}

func intPointer(value int) *int { return &value }

func sameIntPointer(left, right *int) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func splitResourceKey(t *testing.T, value string) (string, string) {
	t.Helper()
	for index, character := range value {
		if character == '.' {
			return value[:index], value[index+1:]
		}
	}
	t.Fatalf("resource key %q is missing section separator", value)
	return "", ""
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
