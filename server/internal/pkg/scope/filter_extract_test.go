package scope

import "testing"

func TestExtractField(t *testing.T) {
	remaining, matches, err := ExtractField(`websiteUrl=="https://api.acme.com" && status=="200" || status=="301"`, "websiteUrl")
	if err != nil {
		t.Fatalf("ExtractField returned error: %v", err)
	}
	if len(matches) != 1 || matches[0].Filter.Operator != "==" {
		t.Fatalf("unexpected matches: %+v", matches)
	}
	if remaining != `status=="200" || status=="301"` {
		t.Fatalf("unexpected remaining filter: %q", remaining)
	}
}

func TestExtractFieldRejectsMalformedExpression(t *testing.T) {
	if _, _, err := ExtractField(`websiteUrl=="https://example.com" &&`, "websiteUrl"); err == nil {
		t.Fatal("expected malformed filter error")
	}
}

func TestExtractFieldPreservesAnUnstructuredRawURL(t *testing.T) {
	rawURL := "https://example.test/path with=literal&&payload "
	remaining, matches, err := ExtractField(rawURL, "websiteUrl")
	if err != nil {
		t.Fatalf("ExtractField returned error: %v", err)
	}
	if len(matches) != 0 || remaining != rawURL {
		t.Fatalf("ExtractField changed raw URL: remaining=%q matches=%+v", remaining, matches)
	}
}
