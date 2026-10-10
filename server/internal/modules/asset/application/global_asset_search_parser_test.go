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
	if urlWithQuery.PlainURL != "https://example.test/?foo=bar%zz" {
		t.Fatalf("plain URL search term must trim presentation whitespace: %q", urlWithQuery.PlainURL)
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
	urlCondition, err := ParseGlobalAssetSearchQuery(`url="jd"`)
	if err != nil {
		t.Fatalf("parse URL contains condition: %v", err)
	}
	if got := urlCondition.Conditions[0]; got.Operator != GlobalAssetSearchOperatorContains || got.Text != "jd" {
		t.Fatalf("URL single-equals condition must remain contains: %+v", got)
	}
	exactURL, err := ParseGlobalAssetSearchQuery(`url=="HTTPS://Example.test/%00?x=%zz"`)
	if err != nil {
		t.Fatalf("parse exact raw URL condition: %v", err)
	}
	if got := exactURL.Conditions[0]; got.Operator != GlobalAssetSearchOperatorExact || got.Text != "HTTPS://Example.test/%00?x=%zz" {
		t.Fatalf("URL double-equals condition must retain exact raw bytes: %+v", got)
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

func TestParseGlobalAssetSearchQueryAcceptsAndAsConjunction(t *testing.T) {
	baseline, err := ParseGlobalAssetSearchQuery(`url="admin" && host="api" && title=="Login"`)
	if err != nil {
		t.Fatalf("baseline: %v", err)
	}
	for _, raw := range []string{
		`url="admin" and host="api" and title=="Login"`,
		`url="admin" AND host="api" And title=="Login"`,
		`url="admin"&&host="api" and title=="Login"`,
	} {
		got, err := ParseGlobalAssetSearchQuery(raw)
		if err != nil {
			t.Fatalf("parse %q: %v", raw, err)
		}
		if got.Mode != baseline.Mode || got.Combinator != baseline.Combinator || len(got.Conditions) != len(baseline.Conditions) {
			t.Fatalf("%q AST = %+v, want %+v", raw, got, baseline)
		}
		for index := range baseline.Conditions {
			if got.Conditions[index] != baseline.Conditions[index] {
				t.Fatalf("%q condition %d = %+v, want %+v", raw, index, got.Conditions[index], baseline.Conditions[index])
			}
		}
	}

	plain, err := ParseGlobalAssetSearchQuery("login and admin")
	if err != nil || plain.Mode != GlobalAssetSearchModePlainURL || plain.PlainURL != "login and admin" {
		t.Fatalf("plain text containing and = %+v, err=%v", plain, err)
	}
	quoted, err := ParseGlobalAssetSearchQuery(`title="foo and bar"`)
	if err != nil || quoted.Mode != GlobalAssetSearchModeStructured || len(quoted.Conditions) != 1 || quoted.Conditions[0].Text != "foo and bar" {
		t.Fatalf("quoted and = %+v, err=%v", quoted, err)
	}
	for _, raw := range []string{`url="admin" and`, `host="api" android="x"`} {
		if _, err := ParseGlobalAssetSearchQuery(raw); !errors.Is(err, ErrInvalidGlobalAssetSearchQuery) {
			t.Fatalf("expected invalid query for %q, got %v", raw, err)
		}
	}
}

func TestParseGlobalAssetSearchQueryAcceptsOrAsDisjunction(t *testing.T) {
	baseline, err := ParseGlobalAssetSearchQuery(`url="admin" || host="api" || title=="Login"`)
	if err != nil {
		t.Fatalf("baseline: %v", err)
	}
	if baseline.Combinator != GlobalAssetSearchCombinatorOr {
		t.Fatalf("|| combinator = %q", baseline.Combinator)
	}
	for _, raw := range []string{
		`url="admin" or host="api" or title=="Login"`,
		`url="admin" OR host="api" Or title=="Login"`,
		`url="admin"||host="api" or title=="Login"`,
	} {
		got, err := ParseGlobalAssetSearchQuery(raw)
		if err != nil {
			t.Fatalf("parse %q: %v", raw, err)
		}
		if got.Combinator != baseline.Combinator || len(got.Conditions) != len(baseline.Conditions) {
			t.Fatalf("%q AST = %+v, want %+v", raw, got, baseline)
		}
		for index := range baseline.Conditions {
			if got.Conditions[index] != baseline.Conditions[index] {
				t.Fatalf("%q condition %d = %+v, want %+v", raw, index, got.Conditions[index], baseline.Conditions[index])
			}
		}
		if globalAssetSearchQueryDigest(got) != globalAssetSearchQueryDigest(baseline) {
			t.Fatalf("%q digest differs from ||", raw)
		}
	}

	andAST, err := ParseGlobalAssetSearchQuery(`url="admin" && host="api" && title=="Login"`)
	if err != nil {
		t.Fatalf("and baseline: %v", err)
	}
	if globalAssetSearchQueryDigest(andAST) == globalAssetSearchQueryDigest(baseline) {
		t.Fatal("AND and OR of the same conditions must not share a page token digest")
	}

	plain, err := ParseGlobalAssetSearchQuery("login or admin")
	if err != nil || plain.Mode != GlobalAssetSearchModePlainURL || plain.PlainURL != "login or admin" {
		t.Fatalf("plain text containing or = %+v, err=%v", plain, err)
	}
	quoted, err := ParseGlobalAssetSearchQuery(`title="foo or bar"`)
	if err != nil || quoted.Mode != GlobalAssetSearchModeStructured || len(quoted.Conditions) != 1 || quoted.Conditions[0].Text != "foo or bar" {
		t.Fatalf("quoted or = %+v, err=%v", quoted, err)
	}
	for _, raw := range []string{
		`url="admin" or`,
		`url="admin" ||`,
		`host="api" origin="x"`,
		`url="admin" && host="api" or title=="Login"`,
		`url="admin" or host="api" and title=="Login"`,
	} {
		if _, err := ParseGlobalAssetSearchQuery(raw); !errors.Is(err, ErrInvalidGlobalAssetSearchQuery) {
			t.Fatalf("expected invalid query for %q, got %v", raw, err)
		}
	}
}

func TestParseGlobalAssetSearchQueryRejectsUnsafeOrMalformedSyntax(t *testing.T) {
	tooLong := strings.Repeat("a", globalAssetSearchMaxQueryBytes+1)
	elevenConditions := strings.TrimSuffix(strings.Repeat(`tech="nginx" && `, 11), " && ")
	cases := []string{
		" ",
		tooLong,
		elevenConditions,
		`responseBody="password"`,
		`url!="admin"`,
		`url="admin" && host="api" || title=="Login"`,
		`(url="admin")`,
		`url="admin" &&`,
		`plain text host="api"`,
		`url="unterminated`,
		`url="a"`,
		`host="a"`,
		`title="a"`,
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
	for _, raw := range []string{`jd`, `url="jd"`, `url=="j"`} {
		if _, err := ParseGlobalAssetSearchQuery(raw); err != nil {
			t.Fatalf("query %q should be accepted: %v", raw, err)
		}
	}
	for _, raw := range []string{`host="ab"`, `title="登录"`} {
		if _, err := ParseGlobalAssetSearchQuery(raw); err != nil {
			t.Fatalf("two-character contains query %q should be accepted: %v", raw, err)
		}
	}
	for _, raw := range []string{`statusCode="2xx"`, `statusCode="-1"`, `statusCode="+200"`, `statusCode="200.0"`} {
		if _, err := ParseGlobalAssetSearchQuery(raw); !errors.Is(err, ErrInvalidGlobalAssetSearchQuery) {
			t.Fatalf("query %q should be rejected: %v", raw, err)
		}
	}
}

func TestParseGlobalAssetSearchQuerySupportsHasScreenshotBoolean(t *testing.T) {
	for _, raw := range []string{`hasScreenshot="true"`, `hasScreenshot=="true"`} {
		ast, err := ParseGlobalAssetSearchQuery(raw)
		if err != nil {
			t.Fatalf("query %q should be accepted: %v", raw, err)
		}
		condition := ast.Conditions[0]
		if condition.Field != GlobalAssetSearchFieldHasScreenshot || condition.HasScreenshot == nil || !*condition.HasScreenshot {
			t.Fatalf("query %q must parse into a true hasScreenshot condition: %+v", raw, condition)
		}
	}
	falseAST, err := ParseGlobalAssetSearchQuery(`hasScreenshot="false"`)
	if err != nil {
		t.Fatalf("parse false polarity: %v", err)
	}
	if condition := falseAST.Conditions[0]; condition.HasScreenshot == nil || *condition.HasScreenshot {
		t.Fatalf("false polarity must be preserved verbatim: %+v", condition)
	}

	combined, err := ParseGlobalAssetSearchQuery(`host="api" && hasScreenshot=="false"`)
	if err != nil {
		t.Fatalf("parse combined boolean query: %v", err)
	}
	if len(combined.Conditions) != 2 || combined.Conditions[1].Field != GlobalAssetSearchFieldHasScreenshot || *combined.Conditions[1].HasScreenshot {
		t.Fatalf("hasScreenshot must compose with flat AND conditions: %+v", combined.Conditions)
	}

	for _, raw := range []string{
		`hasScreenshot=="TRUE"`,
		`hasScreenshot="True"`,
		`hasScreenshot="1"`,
		`hasScreenshot==""`,
		`hasScreenshot=" true "`,
	} {
		if _, err := ParseGlobalAssetSearchQuery(raw); !errors.Is(err, ErrInvalidGlobalAssetSearchQuery) {
			t.Fatalf("query %q should be rejected without boolean normalization: %v", raw, err)
		}
	}
}
