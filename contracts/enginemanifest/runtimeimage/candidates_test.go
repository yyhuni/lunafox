package runtimeimage

import (
	"strings"
	"testing"
)

const (
	testDigestA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testDigestB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func TestParseCandidatesPreservesArbitraryLocationsAndSharedDigest(t *testing.T) {
	refs := []string{
		"docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@" + testDigestA,
		"ghcr.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@" + testDigestA,
	}

	parsed, err := ParseCandidates(refs)
	if err != nil {
		t.Fatalf("ParseFirstPartyCandidates() error = %v", err)
	}
	if parsed.RuntimeImageDigest != testDigestA {
		t.Fatalf("parsed runtime image digest = %q, want %q", parsed.RuntimeImageDigest, testDigestA)
	}
	if len(parsed.References) != 2 {
		t.Fatalf("parsed reference count = %d, want 2", len(parsed.References))
	}
	if parsed.References[0].Registry != "docker.io" || parsed.References[0].Repository != "yyhuni/lunafox-engine-runtime-subdomain-discovery" {
		t.Fatalf("unexpected Docker Hub reference: %#v", parsed.References[0])
	}
	if parsed.References[1].Registry != "ghcr.io" || parsed.References[1].Repository != "yyhuni/lunafox-engine-runtime-subdomain-discovery" {
		t.Fatalf("unexpected GHCR reference: %#v", parsed.References[1])
	}
	if parsed.References[0].String() != refs[0] || parsed.References[1].String() != refs[1] {
		t.Fatalf("candidate order or canonical text changed: %#v", parsed.References)
	}
}

func TestParseCandidatesAcceptsOneLocalRegistryLocation(t *testing.T) {
	ref := "registry.local:5000/lunafox/lunafox-engine-runtime-port-scan@" + testDigestA

	parsed, err := ParseCandidates([]string{ref})
	if err != nil {
		t.Fatalf("ParseFirstPartyCandidates() error = %v", err)
	}
	if len(parsed.References) != 1 || parsed.References[0].String() != ref || parsed.RuntimeImageDigest != testDigestA {
		t.Fatalf("unexpected parsed candidates: %#v", parsed)
	}
}

func TestParseCandidatesAcceptsRepositoryUnrelatedToEngineIdentity(t *testing.T) {
	ref := "docker.io/example/scanner-runtime@" + testDigestA

	parsed, err := ParseCandidates([]string{ref})
	if err != nil {
		t.Fatalf("ParseFirstPartyCandidates() error = %v", err)
	}
	if got, want := parsed.References[0].String(), ref; got != want {
		t.Fatalf("candidate ref = %q, want %q", got, want)
	}
}

func TestParseFirstPartyCandidatesRequiresEngineIDDerivedRepository(t *testing.T) {
	valid := "docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@" + testDigestA
	parsed, err := ParseFirstPartyCandidates("engine.lunafox.subdomain_discovery", []string{valid})
	if err != nil {
		t.Fatalf("ParseFirstPartyCandidates() error = %v", err)
	}
	if len(parsed.References) != 1 || parsed.References[0].String() != valid {
		t.Fatalf("unexpected first-party candidates: %#v", parsed)
	}

	_, err = ParseFirstPartyCandidates("engine.lunafox.subdomain_discovery", []string{
		"docker.io/yyhuni/lunafox-engine-subdomain-discovery@" + testDigestA,
	})
	if err == nil || !strings.Contains(err.Error(), "engineId-derived repository") {
		t.Fatalf("expected engineId-derived repository rejection, got %v", err)
	}
}

func TestParseCandidatesRejectsInvalidCandidateSet(t *testing.T) {
	valid := "docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@" + testDigestA
	tests := []struct {
		name string
		refs []string
		want string
	}{
		{name: "empty candidates", want: "candidates are required"},
		{name: "empty candidate", refs: []string{""}, want: "non-empty canonical OCI digest reference"},
		{name: "tag only", refs: []string{"docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery:latest"}, want: "digest separator"},
		{name: "non-canonical whitespace", refs: []string{" " + valid}, want: "canonical OCI digest reference"},
		{name: "duplicate location", refs: []string{valid, valid}, want: "duplicate runtimeImage candidate location"},
		{name: "digest drift", refs: []string{valid, "ghcr.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@" + testDigestB}, want: "same OCI image digest"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseCandidates(test.refs)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ParseFirstPartyCandidates() error = %v, want error containing %q", err, test.want)
			}
		})
	}
}
