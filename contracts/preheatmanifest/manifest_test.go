package preheatmanifest

import (
	"encoding/json"
	"strings"
	"testing"
)

const testDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func canonicalTestManifest(t *testing.T) []byte {
	t.Helper()
	entry := Entry{
		Candidates:        []string{"docker.io/yyhuni/lunafox-engine-runtime-port-scan@" + testDigest, "ghcr.io/yyhuni/lunafox-engine-runtime-port-scan@" + testDigest},
		Digest:            testDigest,
		IdentityReference: "ghcr.io/yyhuni/lunafox-engine-runtime-port-scan@" + testDigest,
		Platforms:         []string{PlatformLinuxAMD64, PlatformLinuxARM64},
		Profiles:          []string{ProfileEmbedded, ProfileExternal},
		Repository:        "yyhuni/lunafox-engine-runtime-port-scan",
		Sources:           []LogicalSource{{Kind: "engine-runtime", Name: "engine.lunafox.port_scan"}},
		Trust:             "first-party",
	}
	closures := make([]ProfileClosure, 0, 2)
	for _, profile := range []string{ProfileEmbedded, ProfileExternal} {
		entries := []string{entryIdentity(entry)}
		digest, err := closureDigest(profile, entries)
		if err != nil {
			t.Fatal(err)
		}
		closures = append(closures, ProfileClosure{Profile: profile, Entries: entries, Digest: digest})
	}
	manifest := Manifest{
		Entries:         []Entry{entry},
		Kind:            Kind,
		ProfileClosures: closures,
		Release: ReleaseBinding{
			ComposeDigest:          testDigest,
			CompositionDigest:      testDigest,
			ManifestDigest:         testDigest,
			Tag:                    "v1.2.3",
			ThirdPartyPolicyDigest: testDigest,
		},
		SchemaVersion: SchemaVersion,
	}
	core, err := canonicalJSON(manifest.corePayload())
	if err != nil {
		t.Fatal(err)
	}
	manifest.ManifestDigest = sha256Digest(core)
	bytes, err := canonicalJSON(map[string]any{
		"entries":         manifest.Entries,
		"kind":            manifest.Kind,
		"manifestDigest":  manifest.ManifestDigest,
		"profileClosures": manifest.ProfileClosures,
		"release":         manifest.Release,
		"schemaVersion":   manifest.SchemaVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}

func TestParseAcceptsCanonicalManifestAndSelectsPlatformClosure(t *testing.T) {
	raw := canonicalTestManifest(t)
	manifest, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	entries, err := manifest.EntriesForPlatform(ProfileEmbedded, PlatformLinuxAMD64)
	if err != nil {
		t.Fatalf("EntriesForPlatform() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Repository != "yyhuni/lunafox-engine-runtime-port-scan" {
		t.Fatalf("EntriesForPlatform() = %#v", entries)
	}
}

func TestParseRejectsReformattedUnknownAndDuplicateInput(t *testing.T) {
	raw := canonicalTestManifest(t)
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	reformatted, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(reformatted); err == nil || !strings.Contains(err.Error(), "canonical JSON") {
		t.Fatalf("Parse(reformatted) error = %v, want canonical-byte failure", err)
	}
	if _, err := Parse([]byte(`{"entries":[],"entries":[],"kind":"lunafox.preheat-manifest","manifestDigest":"` + testDigest + `","profileClosures":[],"release":{},"schemaVersion":1}`)); err == nil || !strings.Contains(err.Error(), "duplicate key") {
		t.Fatalf("Parse(duplicate) error = %v, want duplicate-key failure", err)
	}
	value["unexpected"] = true
	unknown, err := canonicalJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(unknown); err == nil || !strings.Contains(err.Error(), "unknown or missing") {
		t.Fatalf("Parse(unknown) error = %v, want strict-key failure", err)
	}
}

func TestEntriesForPlatformFailsWhenNoEngineRuntimeSupportsHost(t *testing.T) {
	raw := canonicalTestManifest(t)
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	entries := value["entries"].([]any)
	entry := entries[0].(map[string]any)
	entry["platforms"] = []any{PlatformLinuxARM64}
	core := map[string]any{
		"entries":         value["entries"],
		"kind":            value["kind"],
		"profileClosures": value["profileClosures"],
		"release":         value["release"],
		"schemaVersion":   value["schemaVersion"],
	}
	canonicalCore, err := canonicalJSON(core)
	if err != nil {
		t.Fatal(err)
	}
	value["manifestDigest"] = sha256Digest(canonicalCore)
	mutated, err := canonicalJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := Parse(mutated)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if _, err := manifest.EntriesForPlatform(ProfileEmbedded, PlatformLinuxAMD64); err == nil || !strings.Contains(err.Error(), "no supported Engine Runtime") {
		t.Fatalf("EntriesForPlatform() error = %v, want unsupported Engine Runtime failure", err)
	}
}

func TestDecodeRuntimeCompositionBindingAcceptsTheFullCompositionEnvelope(t *testing.T) {
	raw, err := canonicalJSON(map[string]any{
		"capabilities":         map[string]any{"dynamicFrontendUpstream": true},
		"components":           []any{map[string]any{"id": "runtime.agent"}},
		"compositionDigest":    testDigest,
		"kind":                 "lunafox.runtime-composition",
		"manifestBinding":      map[string]any{"manifestDigest": testDigest},
		"publicMergeCommit":    "b" + strings.Repeat("a", 39),
		"releaseTag":           "v1.2.3",
		"schemaVersion":        1,
		"sourceRevisionDigest": testDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	binding, err := decodeRuntimeCompositionBinding(raw)
	if err != nil {
		t.Fatalf("decodeRuntimeCompositionBinding() error = %v", err)
	}
	if binding.CompositionDigest != testDigest || binding.ManifestDigest != testDigest || binding.ReleaseTag != "v1.2.3" {
		t.Fatalf("binding = %#v", binding)
	}
}

func TestDecodeRuntimeCompositionBindingFailsClosedForMissingOrUnknownFields(t *testing.T) {
	base := map[string]any{
		"capabilities":      map[string]any{},
		"components":        []any{map[string]any{"id": "runtime.agent"}},
		"compositionDigest": testDigest,
		"kind":              "lunafox.runtime-composition",
		"manifestBinding":   map[string]any{"manifestDigest": testDigest},
		"releaseTag":        "v1.2.3",
		"schemaVersion":     1,
	}
	for name, mutate := range map[string]func(map[string]any){
		"missing binding": func(value map[string]any) { delete(value, "manifestBinding") },
		"unknown field":   func(value map[string]any) { value["unexpected"] = true },
		"bad binding": func(value map[string]any) {
			value["manifestBinding"] = map[string]any{"manifestDigest": testDigest, "extra": true}
		},
	} {
		t.Run(name, func(t *testing.T) {
			value := make(map[string]any, len(base))
			for key, item := range base {
				value[key] = item
			}
			mutate(value)
			raw, err := canonicalJSON(value)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := decodeRuntimeCompositionBinding(raw); err == nil {
				t.Fatal("decodeRuntimeCompositionBinding() accepted invalid composition")
			}
		})
	}
}
