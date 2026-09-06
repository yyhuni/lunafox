package application

import (
	"errors"
	"strings"
	"testing"
)

func TestParseGlobalAssetSearchQuerySupportsPlainURLAndStrictStructuredDSL(t *testing.T) {
	plain, err := ParseGlobalAssetSearchQuery("https://Example.test/Admin")
	if err != nil {
		t.Fatalf("parse plain URL: %v", err)
	}
	if plain.Mode != GlobalAssetSearchModePlainURL || plain.PlainURL != "https://Example.test/Admin" || len(plain.Conditions) != 0 {
		t.Fatalf("unexpected plain AST: %+v", plain)
	}
	urlWithQuery, err := ParseGlobalAssetSearchQuery("https://example.test/?foo=bar%zz ")
	if err != nil || urlWithQuery.Mode != GlobalAssetSearchModePlainURL {
		t.Fatalf("ordinary URL query strings must remain plain URL searches: ast=%+v err=%v", urlWithQuery, err)
	}
	if urlWithQuery.PlainURL != "https://example.test/?foo=bar%zz " {
		t.Fatalf("plain URL bytes must not be trimmed: %q", urlWithQuery.PlainURL)
	}
	urlWithDSLBytes := "HTTPS://example.test/path with=literal&&payload"
	plainWithDSLBytes, err := ParseGlobalAssetSearchQuery(urlWithDSLBytes)
	if err != nil || plainWithDSLBytes.Mode != GlobalAssetSearchModePlainURL || plainWithDSLBytes.PlainURL != urlWithDSLBytes {
		t.Fatalf("HTTP(S)-prefixed raw URL must win over DSL detection: ast=%+v err=%v", plainWithDSLBytes, err)
	}

	structured, err := ParseGlobalAssetSearchQuery(`host="api" && title=="Admin \"Gateway\"" && tech="nginx" && tech=="react"`)
	if err != nil {
		t.Fatalf("parse structured query: %v", err)
	}
	if structured.Mode != GlobalAssetSearchModeStructured || len(structured.Conditions) != 4 {
		t.Fatalf("unexpected structured AST: %+v", structured)
	}
	if got := structured.Conditions[1]; got.Field != GlobalAssetSearchFieldTitle || got.Operator != GlobalAssetSearchOperatorExact || got.Text != `Admin "Gateway"` {
		t.Fatalf("escaped quoted value lost: %+v", got)
	}
	if structured.Conditions[2].Field != GlobalAssetSearchFieldTech || structured.Conditions[3].Field != GlobalAssetSearchFieldTech {
		t.Fatalf("repeated fields must remain separate conditions: %+v", structured.Conditions)
	}
	urlCondition, err := ParseGlobalAssetSearchQuery(`url="HTTPS://Example.test/%00?x=%zz"`)
	if err != nil {
		t.Fatalf("parse raw URL condition: %v", err)
	}
	if got := urlCondition.Conditions[0]; got.Operator != GlobalAssetSearchOperatorExact || got.Text != "HTTPS://Example.test/%00?x=%zz" {
		t.Fatalf("URL condition must retain exact raw bytes: %+v", got)
	}
}

func TestParseGlobalAssetSearchQuerySupportsTenConditions(t *testing.T) {
	conditions := make([]string, 0, 10)
	for index := 0; index < 10; index++ {
		conditions = append(conditions, `tech="nginx"`)
	}
	ast, err := ParseGlobalAssetSearchQuery(strings.Join(conditions, " && "))
	if err != nil {
		t.Fatalf("ten conditions should be accepted: %v", err)
	}
	if len(ast.Conditions) != 10 {
		t.Fatalf("expected ten conditions, got %d", len(ast.Conditions))
	}
}

func TestParseGlobalAssetSearchQueryRejectsUnsafeOrMalformedSyntax(t *testing.T) {
	tooLong := strings.Repeat("a", globalAssetSearchMaxQueryBytes+1)
	elevenConditions := strings.TrimSuffix(strings.Repeat(`tech="nginx" && `, 11), " && ")
	cases := []string{
		" ",
		"ab",
		tooLong,
		elevenConditions,
		`responseBody="password"`,
		`url!="admin"`,
		`url="admin" || host="api"`,
		`url="admin" and host="api"`,
		`url="admin" or host="api"`,
		`(url="admin")`,
		`url="admin" &&`,
		`plain text host="api"`,
		`url="unterminated`,
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			if _, err := ParseGlobalAssetSearchQuery(raw); !errors.Is(err, ErrInvalidGlobalAssetSearchQuery) {
				t.Fatalf("expected invalid query for %q, got %v", raw, err)
			}
		})
	}
}

func TestParseGlobalAssetSearchQueryUsesFieldSpecificTypedSemantics(t *testing.T) {
	for _, raw := range []string{`statusCode="200"`, `statusCode=="200"`, `tech="Nginx"`, `tech=="Nginx"`, `title=="A"`} {
		if _, err := ParseGlobalAssetSearchQuery(raw); err != nil {
			t.Fatalf("query %q should be accepted: %v", raw, err)
		}
	}
	for _, raw := range []string{`statusCode="2xx"`, `statusCode="-1"`, `statusCode="+200"`, `statusCode="200.0"`} {
		if _, err := ParseGlobalAssetSearchQuery(raw); !errors.Is(err, ErrInvalidGlobalAssetSearchQuery) {
			t.Fatalf("query %q should be rejected: %v", raw, err)
		}
	}
}
