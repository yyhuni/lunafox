package application

import (
	"context"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
)

func TestValidateSourceURLAcceptsAnonymousHTTPSAndNormalizesHost(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: " https://EXAMPLE.com:0443/org/templates.git ", want: "https://example.com/org/templates.git"},
		{input: "https://10.0.0.8:8443/org/templates.git", want: "https://10.0.0.8:8443/org/templates.git"},
		{input: "https://[fd00::8]:8443/org/templates.git", want: "https://[fd00::8]:8443/org/templates.git"},
	}
	for _, test := range tests {
		got, err := ValidateSourceURL(domain.SourceTypeGit, test.input)
		if err != nil {
			t.Fatalf("validate source %q: %v", test.input, err)
		}
		if got != test.want {
			t.Fatalf("normalized URL=%q want=%q", got, test.want)
		}
	}
}

func TestPublicSourceResolverDoesNotResolveOrProbeHosts(t *testing.T) {
	resolver := NewPublicSourceResolver()
	for _, rawURL := range []string{
		"https://does-not-resolve.invalid/org/templates.git",
		"https://198.18.15.249/org/templates.git",
		"https://10.0.0.8:8443/org/templates.git",
		"https://localhost/org/templates.git",
		"https://intranet/org/templates.git",
	} {
		if _, err := resolver.Validate(context.Background(), domain.SourceTypeCustom, rawURL); err != nil {
			t.Fatalf("proxy/internal URL %q was rejected before Git: %v", rawURL, err)
		}
	}
}

func TestValidateSourceURLRejectsCredentialsAndUnsupportedShapes(t *testing.T) {
	tests := []struct {
		name       string
		sourceType domain.SourceType
		url        string
	}{
		{name: "http", sourceType: domain.SourceTypeCustom, url: "http://example.com/repo.git"},
		{name: "ssh", sourceType: domain.SourceTypeCustom, url: "ssh://example.com/repo.git"},
		{name: "userinfo", sourceType: domain.SourceTypeCustom, url: "https://user:secret@example.com/repo.git"},
		{name: "file", sourceType: domain.SourceTypeCustom, url: "file:///tmp/repo"},
		{name: "query", sourceType: domain.SourceTypeCustom, url: "https://example.com/repo.git?ref=main"},
		{name: "path traversal", sourceType: domain.SourceTypeCustom, url: "https://example.com/org/%2e%2e/repo.git"},
		{name: "invalid port", sourceType: domain.SourceTypeCustom, url: "https://example.com:not-a-port/repo.git"},
		{name: "empty port", sourceType: domain.SourceTypeCustom, url: "https://example.com:/repo.git"},
		{name: "zero port", sourceType: domain.SourceTypeCustom, url: "https://example.com:0/repo.git"},
		{name: "out of range port", sourceType: domain.SourceTypeCustom, url: "https://example.com:65536/repo.git"},
		{name: "invalid host", sourceType: domain.SourceTypeCustom, url: "https://bad_host/repo.git"},
		{name: "bracketed non IP host", sourceType: domain.SourceTypeCustom, url: "https://[intranet]/repo.git"},
		{name: "gitee mismatch", sourceType: domain.SourceTypeGitee, url: "https://github.com/org/repo.git"},
		{name: "missing path", sourceType: domain.SourceTypeCustom, url: "https://example.com/"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ValidateSourceURL(test.sourceType, test.url); err == nil {
				t.Fatal("expected source URL to be rejected")
			}
		})
	}
}
