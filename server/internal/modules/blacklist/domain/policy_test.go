package domain

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestCanonicalizePolicyPatterns(t *testing.T) {
	patterns, err := CanonicalizePolicyPatterns([]string{
		" EXAMPLE.com. ", "*.Example.com.", "192.0.2.1", "192.0.2.19/24", "example.com",
	})
	if err != nil {
		t.Fatalf("CanonicalizePolicyPatterns() error = %v", err)
	}
	want := []string{"*.example.com", "192.0.2.0/24", "192.0.2.1", "example.com"}
	if fmt.Sprint(patterns) != fmt.Sprint(want) {
		t.Fatalf("patterns = %v, want %v", patterns, want)
	}
}

func TestCanonicalizePolicyPatternsRejectsUnsupportedSyntax(t *testing.T) {
	for _, pattern := range []string{
		"*keyword*", "api.*.example.com", "example.*", "*.example.*", "2001:db8::1", "2001:db8::/32", "https://example.com", "localhost", "", "*.192.0.2.1",
	} {
		t.Run(pattern, func(t *testing.T) {
			_, err := CanonicalizePolicyPatterns([]string{pattern})
			if !errors.Is(err, ErrInvalidPattern) {
				t.Fatalf("error = %v, want ErrInvalidPattern", err)
			}
		})
	}
}

func TestMatcherCoversDomainIPv4URLAndHostPortIdentities(t *testing.T) {
	patterns, err := CanonicalizeEffectivePatterns([]string{"example.com", "*.example.com", "192.0.2.1", "198.51.100.7/24"})
	if err != nil {
		t.Fatal(err)
	}
	matcher, err := CompileMatcher(patterns)
	if err != nil {
		t.Fatal(err)
	}
	for host, want := range map[string]bool{
		"example.com": true, "api.example.com": true, "a.b.example.com": true,
		"badexample.com": false, "example.com.evil": false, "198.51.100.99": true,
		"192.0.2.1": true, "192.0.2.2": false,
	} {
		if got := matcher.MatchesHostname(host); got != want {
			t.Errorf("MatchesHostname(%q) = %t, want %t", host, got, want)
		}
	}
	if !matcher.MatchesURL("https://api.example.com:8443/path?q=x#fragment") {
		t.Fatal("URL hostname should match")
	}
	if !matcher.MatchesURL("HTTPS://api.example.com:443/path%zz?payload=%0d%0a#fragment") {
		t.Fatal("raw URL authority should match even when later payload syntax is parser-hostile")
	}
	if matcher.MatchesURL("https://unmatched.example.net/path") {
		t.Fatal("unexpected URL match")
	}
	if !matcher.MatchesHostPort("unmatched.example.net", "198.51.100.9") {
		t.Fatal("HostPort IP identity should match")
	}
	if !matcher.MatchesHostPort("api.example.com", "203.0.113.2") {
		t.Fatal("HostPort hostname identity should match")
	}
}

func TestPolicyLimitsAndCanonicalETag(t *testing.T) {
	within := make([]string, MaxPolicyPatterns)
	for index := range within {
		within[index] = fmt.Sprintf("%d.example.com", index)
	}
	if _, err := CanonicalizePolicyPatterns(within); err != nil {
		t.Fatalf("exact pattern limit rejected: %v", err)
	}
	tooMany := append(append([]string{}, within...), "overflow.example.com")
	if _, err := CanonicalizePolicyPatterns(tooMany); !errors.Is(err, ErrPolicyLimitExceeded) {
		t.Fatalf("over-limit error = %v, want ErrPolicyLimitExceeded", err)
	}
	tooLong := strings.Repeat("a", 64) + "." + strings.Repeat("b", 64) + "." + strings.Repeat("c", 64) + "." + strings.Repeat("d", 64) + ".com"
	if _, err := CanonicalizePolicyPatterns([]string{tooLong}); !errors.Is(err, ErrInvalidPattern) && !errors.Is(err, ErrPolicyLimitExceeded) {
		t.Fatalf("long pattern error = %v", err)
	}

	canonical, err := CanonicalizePolicyPatterns([]string{"example.com", "192.0.2.1"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := ETag(canonical)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ETag(canonical)
	if err != nil || first != second || len(first) != 64 {
		t.Fatalf("etag = %q, err = %v", first, err)
	}
	if err := ValidateCanonicalPolicyPatterns([]string{"Example.com"}); !errors.Is(err, ErrNonCanonicalPatterns) {
		t.Fatalf("stored noncanonical patterns error = %v, want ErrNonCanonicalPatterns", err)
	}
}
