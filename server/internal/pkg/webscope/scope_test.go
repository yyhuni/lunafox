package webscope

import (
	"strings"
	"testing"
)

func TestScopeContains(t *testing.T) {
	tests := []struct {
		name      string
		website   string
		candidate string
		want      bool
	}{
		{name: "root includes descendant", website: "https://api.acme.com", candidate: "https://api.acme.com/v1/products", want: true},
		{name: "root ignores query and fragment", website: "https://api.acme.com/?source=scan#section", candidate: "https://api.acme.com/v1/products?query=1#fragment", want: true},
		{name: "root includes default explicit port", website: "https://api.acme.com", candidate: "https://api.acme.com:443/v1/products", want: true},
		{name: "explicit default port includes omitted port", website: "https://api.acme.com:443", candidate: "https://api.acme.com/v1/products", want: true},
		{name: "explicit nondefault port includes same port", website: "https://api.acme.com:8443", candidate: "https://api.acme.com:8443/v1/products", want: true},
		{name: "explicit nondefault port rejects omitted port", website: "https://api.acme.com:8443", candidate: "https://api.acme.com/v1/products", want: false},
		{name: "nested includes itself", website: "https://api.acme.com/a", candidate: "https://api.acme.com/a", want: true},
		{name: "nested includes trailing slash", website: "https://api.acme.com/a", candidate: "https://api.acme.com/a/", want: true},
		{name: "nested includes descendant", website: "https://api.acme.com/a/", candidate: "https://api.acme.com/a/child?query=1#fragment", want: true},
		{name: "nested rejects sibling prefix", website: "https://api.acme.com/a", candidate: "https://api.acme.com/ab", want: false},
		{name: "rejects lookalike host", website: "https://api.acme.com", candidate: "https://api.acme.com.evil/a", want: false},
		{name: "rejects scheme mismatch", website: "https://api.acme.com", candidate: "http://api.acme.com/a", want: false},
		{name: "rejects port mismatch", website: "https://api.acme.com", candidate: "https://api.acme.com:8443/a", want: false},
		{name: "preserves escaped path boundary", website: "https://api.acme.com/a%2Fb", candidate: "https://api.acme.com/a%2Fb/c", want: true},
		{name: "escaped path rejects decoded lookalike", website: "https://api.acme.com/a%2Fb", candidate: "https://api.acme.com/a/b/c", want: false},
		{name: "parser-hostile payload remains scopeable", website: "HTTPS://api.acme.com:443/a%zz?payload=%0d%0a#fragment", candidate: "https://api.acme.com/a%zz/child?payload=%00", want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scope, err := Parse(test.website)
			if err != nil {
				t.Fatalf("Parse(%q) returned error: %v", test.website, err)
			}
			if got := scope.Contains(test.candidate); got != test.want {
				t.Fatalf("Contains(%q) = %t, want %t", test.candidate, got, test.want)
			}
		})
	}
}

func TestParseRejectsInvalidWebsiteURLs(t *testing.T) {
	for _, rawURL := range []string{
		"",
		"api.acme.com",
		"ftp://api.acme.com",
		"https:///missing-host",
		"https://user:pass@api.acme.com",
		" https://api.acme.com",
		"https://api.acme.com/\n",
	} {
		if _, err := Parse(rawURL); err == nil {
			t.Fatalf("Parse(%q) succeeded, want error", rawURL)
		}
	}
}

func TestScopeSQLPredicateUsesOriginAndPathBoundary(t *testing.T) {
	scope, err := Parse("https://api.acme.com/a%2Fb")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	predicate, args := scope.SQLPredicate("url")
	for _, expected := range []string{"ILIKE ? ESCAPE E'\\\\'", "LOWER(substring(url", "chr(63)", "LIKE ? ESCAPE E'\\\\'"} {
		if !strings.Contains(predicate, expected) {
			t.Fatalf("predicate %q does not include %q", predicate, expected)
		}
	}
	if len(args) != 7 ||
		args[0] != "https://api.acme.com/a\\%2Fb%" ||
		args[1] != "https://api.acme.com:443/a\\%2Fb%" ||
		args[2] != "api.acme.com" ||
		args[3] != "api.acme.com:443" ||
		args[4] != "https" ||
		args[5] != "/a%2Fb" ||
		args[6] != "/a\\%2Fb/%" {
		t.Fatalf("unexpected predicate args: %#v", args)
	}
}

func TestScopeSQLPredicateHasOneBindMarkerPerArgument(t *testing.T) {
	for _, rawURL := range []string{
		"https://api.acme.com",
		"https://api.acme.com/a",
		"https://api.acme.com:8443/a",
	} {
		scope, err := Parse(rawURL)
		if err != nil {
			t.Fatalf("Parse(%q) returned error: %v", rawURL, err)
		}
		predicate, args := scope.SQLPredicate("url")
		if got, want := strings.Count(predicate, "?"), len(args); got != want {
			t.Fatalf("SQLPredicate(%q) has %d bind markers for %d arguments: %s", rawURL, got, want, predicate)
		}
	}
}
