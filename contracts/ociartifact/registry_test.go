package ociartifact

import (
	"reflect"
	"strings"
	"testing"
)

func TestRegistryAuthority(t *testing.T) {
	for _, registry := range []string{"docker.io", "ghcr.io", "registry.company.cn", "registry.company.cn:5443", "localhost:5000", "127.0.0.1:5000"} {
		if err := ValidateRegistryAuthority(registry); err != nil {
			t.Errorf("%s: %v", registry, err)
		}
	}
	for _, registry := range []string{"", " registry.cn", "registry.cn ", "https://registry.cn", "registry.cn/team", "user@registry.cn", "registry.cn?x", "registry.cn#x", "registry.cn:0", "registry.cn:65536", "registry..cn", "registry.-cn"} {
		if err := ValidateRegistryAuthority(registry); err == nil {
			t.Errorf("accepted %q", registry)
		}
	}
}

func TestRegistryCandidatesPreserveContentIdentityWithoutOfficialFallback(t *testing.T) {
	suffix := "/yyhuni/lunafox-engine@sha256:" + strings.Repeat("a", 64)
	mapped, err := RegistryCandidates([]string{"docker.io" + suffix, "ghcr.io" + suffix}, "registry.company.cn")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(mapped, []string{"registry.company.cn" + suffix}) {
		t.Fatal(mapped)
	}
	for _, ref := range []string{"docker.io/repo:latest", " docker.io" + suffix, "docker.io" + suffix + " "} {
		if _, err := ReplaceRegistry(ref, "registry.company.cn"); err == nil {
			t.Errorf("accepted %q", ref)
		}
	}
}
