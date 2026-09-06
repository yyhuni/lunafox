package webscope

import "testing"

func TestExtractFilterScope(t *testing.T) {
	remaining, parsed, err := ExtractFilterScope(`websiteUrl=="https://api.acme.com" && status=="200" || status=="301"`)
	if err != nil {
		t.Fatalf("ExtractFilterScope returned error: %v", err)
	}
	if parsed == nil || remaining != `status=="200" || status=="301"` {
		t.Fatalf("unexpected scope result: parsed=%v remaining=%q", parsed, remaining)
	}
}

func TestExtractFilterScopeAcceptsRawPayloadURLWithoutChangingItsIdentity(t *testing.T) {
	remaining, parsed, err := ExtractFilterScope(`websiteUrl=="HTTPS://api.acme.com:443/a%zz?payload=%0d%0a#fragment" && status=="200"`)
	if err != nil {
		t.Fatalf("ExtractFilterScope returned error: %v", err)
	}
	if parsed == nil || remaining != `status=="200"` || !parsed.Contains("https://api.acme.com/a%zz/child?payload=%00") {
		t.Fatalf("unexpected raw scope result: parsed=%v remaining=%q", parsed, remaining)
	}
}

func TestExtractFilterScopePreservesUnstructuredRawURL(t *testing.T) {
	rawURL := "https://api.acme.com/payload%zz "
	remaining, parsed, err := ExtractFilterScope(rawURL)
	if err != nil {
		t.Fatalf("ExtractFilterScope returned error: %v", err)
	}
	if parsed != nil || remaining != rawURL {
		t.Fatalf("unexpected unstructured result: parsed=%v remaining=%q", parsed, remaining)
	}
}

func TestExtractFilterScopeRejectsUnsafeForms(t *testing.T) {
	for _, filter := range []string{
		`websiteUrl="https://example.com"`,
		`websiteUrl=="https://example.com" || status=="200"`,
		`websiteUrl=="https://example.com" websiteUrl=="https://other.example"`,
		`websiteUrl=="not-a-url"`,
	} {
		if _, _, err := ExtractFilterScope(filter); err == nil {
			t.Fatalf("ExtractFilterScope(%q) succeeded, want error", filter)
		}
	}
}
